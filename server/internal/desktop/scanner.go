package desktop

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const desktopExt = ".desktop"

// ScanError is a non-fatal error associated with a single .desktop file.
// Scan collects these without aborting the rest of the walk.
type ScanError struct {
	Path string
	Err  error
}

func (e *ScanError) Error() string {
	return fmt.Sprintf("desktop scan: %s: %v", e.Path, e.Err)
}

func (e *ScanError) Unwrap() error {
	return e.Err
}

// DefaultPaths returns the XDG-aware list of directories that typically
// contain .desktop files, ordered from lowest to highest priority. Scan
// uses that ordering so the last path wins on ID collisions, matching the
// freedesktop rule "user overrides system".
//
// Order:
//  1. /usr/share/applications
//  2. /usr/local/share/applications
//  3. $XDG_DATA_DIRS split on ':' (each entry gets /applications appended)
//  4. $XDG_DATA_HOME/applications (or ~/.local/share/applications)
//
// Duplicate paths are deduplicated, keeping the first occurrence.
func DefaultPaths() []string {
	var paths []string

	paths = append(paths, "/usr/share/applications")
	paths = append(paths, "/usr/local/share/applications")

	if dirs := os.Getenv("XDG_DATA_DIRS"); dirs != "" {
		for _, d := range strings.Split(dirs, ":") {
			if d == "" {
				continue
			}
			paths = append(paths, filepath.Join(d, "applications"))
		}
	}

	if home := os.Getenv("XDG_DATA_HOME"); home != "" {
		paths = append(paths, filepath.Join(home, "applications"))
	} else if h := os.Getenv("HOME"); h != "" {
		paths = append(paths, filepath.Join(h, ".local/share/applications"))
	}

	return dedupe(paths)
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

// Scan walks every directory in paths, parses .desktop files and returns
// the resulting entries.
//
// Filtering rules:
//   - Files whose extension is not .desktop are ignored.
//   - Entries with Hidden or NoDisplay set to true are dropped.
//   - Entries whose Type is not Application are dropped.
//   - Parse errors, stat errors and WalkDir errors become ScanError and
//     do not abort the walk.
//   - A non-existent path in paths is silently skipped (not an error).
//
// Priority:
//
//	paths is consumed in order. When an ID collides with one seen earlier
//	the later entry replaces it. Callers must therefore pass directories
//	in order of increasing priority (system first, user last) — exactly
//	what DefaultPaths returns.
//
// The returned []Entry is sorted by ID.
func Scan(paths []string) ([]Entry, []ScanError) {
	if len(paths) == 0 {
		return nil, nil
	}

	byID := make(map[string]Entry)
	var errs []ScanError

	for _, root := range paths {
		info, err := os.Stat(root)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			errs = append(errs, ScanError{Path: root, Err: err})
			continue
		}
		if !info.IsDir() {
			continue
		}

		walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				errs = append(errs, ScanError{Path: path, Err: walkErr})
				if d != nil && d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			if d.IsDir() {
				return nil
			}
			if !strings.EqualFold(filepath.Ext(d.Name()), desktopExt) {
				return nil
			}

			entry, parseErr := parseFile(path)
			if parseErr != nil {
				errs = append(errs, ScanError{Path: path, Err: parseErr})
				return nil
			}
			if entry.Hidden || entry.NoDisplay {
				return nil
			}
			if entry.Type != TypeApplication {
				return nil
			}

			id, idErr := computeID(root, path)
			if idErr != nil {
				errs = append(errs, ScanError{Path: path, Err: idErr})
				return nil
			}
			entry.ID = id
			byID[id] = *entry
			return nil
		})
		if walkErr != nil {
			errs = append(errs, ScanError{Path: root, Err: walkErr})
		}
	}

	entries := make([]Entry, 0, len(byID))
	for _, e := range byID {
		entries = append(entries, e)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].ID < entries[j].ID
	})

	return entries, errs
}

func parseFile(path string) (*Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f)
}

func computeID(root, path string) (string, error) {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return "", err
	}
	rel = strings.TrimSuffix(rel, desktopExt)
	return strings.ReplaceAll(rel, string(filepath.Separator), "-"), nil
}
