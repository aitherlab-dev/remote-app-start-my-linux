// Package icons locates freedesktop icon files on disk.
//
// This is a deliberately simplified XDG icon-theme lookup: it walks the
// configured theme directories recursively and picks the best match by
// size, but it does not parse index.theme or follow Inherits= chains.
// Theme inheritance is handled by a separate, optional step (S2.3) of the
// plan. The finder only resolves absolute paths, theme-relative names and
// a legacy pixmaps fallback, which is enough for parsing Icon= fields in
// .desktop files and serving the resulting bytes over HTTP.
package icons

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// ErrIconNotFound is returned by Finder.Find when no icon matches.
var ErrIconNotFound = errors.New("icon not found")

// Supported file formats. Returned by Find as the second value.
const (
	FormatSVG     = "svg"
	FormatPNG     = "png"
	FormatXPM     = "xpm"
	FormatUnknown = "unknown"
)

const (
	hicolorTheme = "hicolor"
	pixmapsDir   = "pixmaps"

	// scalableSize is the sentinel size used for svg files living in a
	// "scalable" directory. It never equals a real preferred size.
	scalableSize = -1
)

// Finder resolves icon names to filesystem paths.
//
// BaseDirs lists the XDG icon base directories in priority order (first
// entry wins). Theme is the primary theme to search; "hicolor" is always
// consulted as a fallback, and an empty Theme is treated as "hicolor".
type Finder struct {
	BaseDirs []string
	Theme    string
}

// New constructs a Finder. Nil or empty baseDirs selects DefaultBaseDirs.
// Empty theme means "hicolor".
func New(baseDirs []string, theme string) *Finder {
	f := &Finder{
		BaseDirs: baseDirs,
		Theme:    theme,
	}
	if len(f.BaseDirs) == 0 {
		f.BaseDirs = DefaultBaseDirs()
	}
	if f.Theme == "" {
		f.Theme = hicolorTheme
	}
	return f
}

// DefaultBaseDirs returns the XDG icon base directories ordered from the
// highest-priority (user) to the lowest-priority (system).
//
// Order:
//  1. $HOME/.icons
//  2. $XDG_DATA_HOME/icons (fallback $HOME/.local/share/icons)
//  3. each entry of $XDG_DATA_DIRS with /icons appended
//  4. /usr/share/icons
//  5. /usr/share/pixmaps
//
// Duplicate paths are removed, keeping the first (highest priority)
// occurrence. Entries are returned unconditionally — callers that care
// about existence must stat the result themselves.
func DefaultBaseDirs() []string {
	var dirs []string

	home := os.Getenv("HOME")

	if home != "" {
		dirs = append(dirs, filepath.Join(home, ".icons"))
	}

	if xdh := os.Getenv("XDG_DATA_HOME"); xdh != "" {
		dirs = append(dirs, filepath.Join(xdh, "icons"))
	} else if home != "" {
		dirs = append(dirs, filepath.Join(home, ".local/share/icons"))
	}

	if xdd := os.Getenv("XDG_DATA_DIRS"); xdd != "" {
		for _, d := range strings.Split(xdd, ":") {
			if d == "" {
				continue
			}
			dirs = append(dirs, filepath.Join(d, "icons"))
		}
	}

	dirs = append(dirs, "/usr/share/icons", "/usr/share/pixmaps")

	return dedupe(dirs)
}

func dedupe(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, p := range in {
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

// Find locates an icon by name and returns its absolute path together
// with the detected format. It returns ErrIconNotFound if nothing matches.
//
// Name handling:
//   - If iconName is absolute or contains "/" it is used verbatim after a
//     stat. The format is derived from the extension, or FormatUnknown
//     when the extension is not one of svg/png/xpm.
//   - Otherwise the file is looked up inside f.BaseDirs, first under
//     f.Theme, then under "hicolor". An empty iconName is a miss.
//
// Size-selection waterfall (per theme):
//  1. exact match for preferredSize
//  2. scalable svg
//  3. the raster whose size is just above preferredSize
//  4. the raster whose size is just below preferredSize
//
// When no theme candidate wins, Find falls back to a pixmaps lookup:
// <base>/pixmaps/<name>.{svg,png,xpm} and <base>/<name>.{svg,png,xpm} for
// each base directory.
func (f *Finder) Find(iconName string, preferredSize int) (path, format string, err error) {
	if iconName == "" {
		return "", "", ErrIconNotFound
	}

	if filepath.IsAbs(iconName) || strings.ContainsRune(iconName, '/') {
		return lookupExplicit(iconName)
	}

	themes := []string{f.Theme}
	if f.Theme != hicolorTheme {
		themes = append(themes, hicolorTheme)
	}

	for _, theme := range themes {
		cands := f.collectCandidates(theme, iconName)
		if p, fmtOut, ok := pickCandidate(cands, preferredSize); ok {
			return p, fmtOut, nil
		}
	}

	if p, fmtOut, ok := f.pixmapsFallback(iconName); ok {
		return p, fmtOut, nil
	}

	return "", "", ErrIconNotFound
}

// lookupExplicit resolves a user-supplied absolute or slashed path.
// Directories and missing files become ErrIconNotFound.
func lookupExplicit(iconName string) (string, string, error) {
	info, err := os.Stat(iconName)
	if err != nil || info.IsDir() {
		return "", "", ErrIconNotFound
	}
	return iconName, formatFromExt(iconName), nil
}

// iconCandidate is a single file found during a theme walk.
type iconCandidate struct {
	path    string
	format  string
	size    int // scalableSize for svg inside a "scalable" dir
	baseIdx int // index into BaseDirs; smaller means higher priority
}

// collectCandidates walks every base directory under the given theme and
// returns files whose basename matches "<iconName>.{svg,png,xpm}". Files
// living under a "symbolic" directory are ignored.
func (f *Finder) collectCandidates(theme, iconName string) []iconCandidate {
	var out []iconCandidate

	wanted := map[string]struct{}{
		iconName + ".svg": {},
		iconName + ".png": {},
		iconName + ".xpm": {},
	}

	for idx, base := range f.BaseDirs {
		themeDir := filepath.Join(base, theme)
		info, err := os.Stat(themeDir)
		if err != nil || !info.IsDir() {
			continue
		}

		_ = filepath.WalkDir(themeDir, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				if d != nil && d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			if d.IsDir() {
				return nil
			}
			if _, ok := wanted[d.Name()]; !ok {
				return nil
			}
			rel, relErr := filepath.Rel(themeDir, path)
			if relErr != nil {
				return nil
			}
			parts := strings.Split(filepath.ToSlash(filepath.Dir(rel)), "/")
			size, ok := sizeFromParts(parts)
			if !ok {
				return nil
			}
			format := formatFromExt(d.Name())
			if format == FormatUnknown {
				return nil
			}
			out = append(out, iconCandidate{
				path:    path,
				format:  format,
				size:    size,
				baseIdx: idx,
			})
			return nil
		})
	}

	return out
}

// sizeFromParts inspects the theme-relative directory components and
// determines which size bucket the icon file belongs to.
//
// Recognized segments:
//
//	<N>x<N>          → exact size N (e.g. 48x48)
//	<N>x<N>@<scale>  → exact size N (the @scale suffix is accepted and
//	                   ignored — HiDPI themes use "48x48@2")
//	scalable         → scalableSize sentinel
//	symbolic         → skipped entirely (second return value is false)
//
// The first recognized segment wins, so "48x48/apps" still yields 48.
func sizeFromParts(parts []string) (int, bool) {
	for _, p := range parts {
		if p == "symbolic" {
			return 0, false
		}
	}
	for _, p := range parts {
		if p == "scalable" {
			return scalableSize, true
		}
		if n := parseSizeDir(p); n > 0 {
			return n, true
		}
	}
	return 0, false
}

func parseSizeDir(name string) int {
	if at := strings.IndexByte(name, '@'); at != -1 {
		name = name[:at]
	}
	x := strings.IndexByte(name, 'x')
	if x <= 0 {
		return 0
	}
	a, err := strconv.Atoi(name[:x])
	if err != nil || a <= 0 {
		return 0
	}
	b, err := strconv.Atoi(name[x+1:])
	if err != nil || b != a {
		return 0
	}
	return a
}

// pickCandidate chooses the best candidate from a theme walk, applying
// the four-step size waterfall. It returns ok=false when the candidate
// slice is empty or contains only entries that fail every step (which
// should not happen in practice).
//
// Candidates are stable-sorted by baseIdx first, so user bases win ties
// against system bases on every step.
func pickCandidate(cands []iconCandidate, preferredSize int) (string, string, bool) {
	if len(cands) == 0 {
		return "", "", false
	}

	sort.SliceStable(cands, func(i, j int) bool {
		return cands[i].baseIdx < cands[j].baseIdx
	})

	for _, c := range cands {
		if c.size == preferredSize {
			return c.path, c.format, true
		}
	}

	for _, c := range cands {
		if c.size == scalableSize {
			return c.path, c.format, true
		}
	}

	bestBig := -1
	for i, c := range cands {
		if c.size <= preferredSize {
			continue
		}
		if bestBig == -1 || c.size < cands[bestBig].size {
			bestBig = i
		}
	}
	if bestBig != -1 {
		c := cands[bestBig]
		return c.path, c.format, true
	}

	bestSmall := -1
	for i, c := range cands {
		if c.size <= 0 || c.size >= preferredSize {
			continue
		}
		if bestSmall == -1 || c.size > cands[bestSmall].size {
			bestSmall = i
		}
	}
	if bestSmall != -1 {
		c := cands[bestSmall]
		return c.path, c.format, true
	}

	return "", "", false
}

// pixmapsFallback tries the legacy locations for an icon: a "pixmaps"
// subdirectory under each base, and the base itself. It walks bases in
// priority order and returns the first file that exists.
func (f *Finder) pixmapsFallback(iconName string) (string, string, bool) {
	exts := []string{".svg", ".png", ".xpm"}

	for _, base := range f.BaseDirs {
		for _, ext := range exts {
			p := filepath.Join(base, pixmapsDir, iconName+ext)
			if statIsFile(p) {
				return p, formatFromExt(p), true
			}
		}
		for _, ext := range exts {
			p := filepath.Join(base, iconName+ext)
			if statIsFile(p) {
				return p, formatFromExt(p), true
			}
		}
	}

	return "", "", false
}

func statIsFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func formatFromExt(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".svg":
		return FormatSVG
	case ".png":
		return FormatPNG
	case ".xpm":
		return FormatXPM
	default:
		return FormatUnknown
	}
}
