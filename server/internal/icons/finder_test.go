package icons

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

const fakeIcons = "testdata/fake_icons"

func TestFinder_ExactSize(t *testing.T) {
	f := New([]string{fakeIcons}, "Adwaita")

	path, format, err := f.Find("firefox", 48)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if format != FormatPNG {
		t.Errorf("format = %q, want %q", format, FormatPNG)
	}
	want := filepath.Join(fakeIcons, "Adwaita", "48x48", "apps", "firefox.png")
	if path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
}

func TestFinder_PreferScalableForUnusualSize(t *testing.T) {
	f := New([]string{fakeIcons}, "Adwaita")

	path, format, err := f.Find("firefox", 256)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if format != FormatSVG {
		t.Errorf("format = %q, want %q", format, FormatSVG)
	}
	want := filepath.Join(fakeIcons, "Adwaita", "scalable", "apps", "firefox.svg")
	if path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
}

// themeWithSizes builds a single-theme base directory inside dir with
// the given raster sizes (no scalable). iconName is used for every file
// so a Find call can locate it.
func themeWithSizes(t *testing.T, dir, theme, iconName string, sizes ...int) {
	t.Helper()
	for _, s := range sizes {
		sub := filepath.Join(dir, theme, sizeDirName(s), "apps")
		if err := os.MkdirAll(sub, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		file := filepath.Join(sub, iconName+".png")
		if err := os.WriteFile(file, []byte("raster"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
}

func sizeDirName(n int) string {
	s := ""
	if n == 0 {
		return "0x0"
	}
	for v := n; v > 0; v /= 10 {
		s = string(rune('0'+v%10)) + s
	}
	return s + "x" + s
}

func TestFinder_FallbackToBiggerRaster(t *testing.T) {
	base := t.TempDir()
	themeWithSizes(t, base, "MyTheme", "tool", 32, 64)

	f := New([]string{base}, "MyTheme")

	// Asking for 16 — no exact, no scalable, nearest *above* is 32.
	path, format, err := f.Find("tool", 16)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if format != FormatPNG {
		t.Errorf("format = %q, want %q", format, FormatPNG)
	}
	want := filepath.Join(base, "MyTheme", "32x32", "apps", "tool.png")
	if path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
}

func TestFinder_FallbackToSmallerRaster(t *testing.T) {
	base := t.TempDir()
	themeWithSizes(t, base, "MyTheme", "tool", 32, 64)

	f := New([]string{base}, "MyTheme")

	// Asking for 512 — no exact, no scalable, no bigger. Nearest
	// *below* is 64.
	path, format, err := f.Find("tool", 512)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if format != FormatPNG {
		t.Errorf("format = %q, want %q", format, FormatPNG)
	}
	want := filepath.Join(base, "MyTheme", "64x64", "apps", "tool.png")
	if path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
}

func TestFinder_FallbackToHicolor(t *testing.T) {
	f := New([]string{fakeIcons}, "Adwaita")

	path, format, err := f.Find("vlc", 64)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if format != FormatPNG {
		t.Errorf("format = %q, want %q", format, FormatPNG)
	}
	want := filepath.Join(fakeIcons, "hicolor", "64x64", "apps", "vlc.png")
	if path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
}

func TestFinder_FallbackToPixmaps(t *testing.T) {
	f := New([]string{fakeIcons}, "Adwaita")

	path, format, err := f.Find("legacy", 32)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if format != FormatXPM {
		t.Errorf("format = %q, want %q", format, FormatXPM)
	}
	want := filepath.Join(fakeIcons, "pixmaps", "legacy.xpm")
	if path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
}

func TestFinder_NotFound(t *testing.T) {
	f := New([]string{fakeIcons}, "Adwaita")

	_, _, err := f.Find("nonexistent", 48)
	if !errors.Is(err, ErrIconNotFound) {
		t.Errorf("err = %v, want ErrIconNotFound", err)
	}
}

func TestFinder_AbsolutePathIcon(t *testing.T) {
	dir := t.TempDir()
	abs := filepath.Join(dir, "explicit.png")
	if err := os.WriteFile(abs, []byte("raster"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	f := New([]string{fakeIcons}, "Adwaita")

	path, format, err := f.Find(abs, 0)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if path != abs {
		t.Errorf("path = %q, want %q", path, abs)
	}
	if format != FormatPNG {
		t.Errorf("format = %q, want %q", format, FormatPNG)
	}
}

func TestFinder_AbsolutePathIconUnknownExt(t *testing.T) {
	dir := t.TempDir()
	abs := filepath.Join(dir, "explicit.bin")
	if err := os.WriteFile(abs, []byte("blob"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	f := New([]string{fakeIcons}, "Adwaita")

	path, format, err := f.Find(abs, 0)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if path != abs {
		t.Errorf("path = %q, want %q", path, abs)
	}
	if format != FormatUnknown {
		t.Errorf("format = %q, want %q", format, FormatUnknown)
	}
}

func TestFinder_AbsolutePathMissing(t *testing.T) {
	f := New([]string{fakeIcons}, "Adwaita")

	_, _, err := f.Find("/definitely/not/here.png", 0)
	if !errors.Is(err, ErrIconNotFound) {
		t.Errorf("err = %v, want ErrIconNotFound", err)
	}
}

func TestFinder_RelativePathWithSlash(t *testing.T) {
	f := New([]string{fakeIcons}, "Adwaita")

	_, _, err := f.Find("subdir/icon.png", 0)
	if !errors.Is(err, ErrIconNotFound) {
		t.Errorf("err = %v, want ErrIconNotFound", err)
	}
}

func TestFinder_AbsolutePathDirectoryIsMiss(t *testing.T) {
	dir := t.TempDir()

	f := New([]string{fakeIcons}, "Adwaita")

	_, _, err := f.Find(dir, 0)
	if !errors.Is(err, ErrIconNotFound) {
		t.Errorf("err = %v, want ErrIconNotFound", err)
	}
}

func TestFinder_EmptyNameIsMiss(t *testing.T) {
	f := New([]string{fakeIcons}, "Adwaita")

	_, _, err := f.Find("", 48)
	if !errors.Is(err, ErrIconNotFound) {
		t.Errorf("err = %v, want ErrIconNotFound", err)
	}
}

func TestFinder_ThemeFromConfig(t *testing.T) {
	// With Theme=Papirus the Papirus icon is reachable.
	f := New([]string{fakeIcons}, "Papirus")
	path, format, err := f.Find("papirus-only", 22)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if format != FormatPNG {
		t.Errorf("format = %q, want %q", format, FormatPNG)
	}
	want := filepath.Join(fakeIcons, "Papirus", "22x22", "apps", "papirus-only.png")
	if path != want {
		t.Errorf("path = %q, want %q", path, want)
	}

	// Without it, the default "hicolor" theme cannot see Papirus.
	f2 := New([]string{fakeIcons}, "")
	if _, _, err := f2.Find("papirus-only", 22); !errors.Is(err, ErrIconNotFound) {
		t.Errorf("err = %v, want ErrIconNotFound", err)
	}
}

func TestFinder_HicolorOnly(t *testing.T) {
	f := New([]string{fakeIcons}, "hicolor")

	path, format, err := f.Find("inkscape", 64)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if format != FormatSVG {
		t.Errorf("format = %q, want %q", format, FormatSVG)
	}
	want := filepath.Join(fakeIcons, "hicolor", "scalable", "apps", "inkscape.svg")
	if path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
}

func TestFinder_UserBaseOverridesSystemBase(t *testing.T) {
	user := t.TempDir()
	system := t.TempDir()

	// System has 48x48 firefox.
	themeWithSizes(t, system, "Adwaita", "firefox", 48)
	// User has a different 48x48 firefox in the same theme.
	themeWithSizes(t, user, "Adwaita", "firefox", 48)
	userFile := filepath.Join(user, "Adwaita", "48x48", "apps", "firefox.png")
	if err := os.WriteFile(userFile, []byte("user"), 0o644); err != nil {
		t.Fatalf("rewrite user: %v", err)
	}

	// user is first → wins.
	f := New([]string{user, system}, "Adwaita")
	path, _, err := f.Find("firefox", 48)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if path != userFile {
		t.Errorf("path = %q, want %q (user base must win)", path, userFile)
	}
}

func TestFinder_RawFileInBaseFallback(t *testing.T) {
	// A base directory may hold the icon directly (no pixmaps, no theme).
	base := t.TempDir()
	raw := filepath.Join(base, "loose.png")
	if err := os.WriteFile(raw, []byte("raster"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	f := New([]string{base}, "Adwaita")
	path, format, err := f.Find("loose", 0)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if path != raw {
		t.Errorf("path = %q, want %q", path, raw)
	}
	if format != FormatPNG {
		t.Errorf("format = %q, want %q", format, FormatPNG)
	}
}

func TestFinder_SymbolicIsSkipped(t *testing.T) {
	base := t.TempDir()

	// A symbolic-only theme — icons live under symbolic/apps, no size dir.
	sub := filepath.Join(base, "OnlySym", "symbolic", "apps")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sub, "sym.svg"), []byte("svg"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	f := New([]string{base}, "OnlySym")
	if _, _, err := f.Find("sym", 48); !errors.Is(err, ErrIconNotFound) {
		t.Errorf("err = %v, want ErrIconNotFound (symbolic must be ignored)", err)
	}
}

func TestFinder_ScaledDirSuffixAccepted(t *testing.T) {
	// 48x48@2 is HiDPI notation for 48 logical pixels at 2x — we treat
	// it as size 48 for selection purposes.
	base := t.TempDir()
	sub := filepath.Join(base, "Hi", "48x48@2", "apps")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	file := filepath.Join(sub, "tool.png")
	if err := os.WriteFile(file, []byte("raster"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	f := New([]string{base}, "Hi")
	path, _, err := f.Find("tool", 48)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if path != file {
		t.Errorf("path = %q, want %q", path, file)
	}
}

func TestDefaultBaseDirs_RespectsXDG(t *testing.T) {
	t.Setenv("HOME", "/tmp/h")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_DATA_DIRS", "/foo:/bar")

	got := DefaultBaseDirs()

	want := []string{
		"/tmp/h/.icons",
		"/tmp/h/.local/share/icons",
		"/foo/icons",
		"/bar/icons",
		"/usr/share/icons",
		"/usr/share/pixmaps",
	}
	if !equalStrings(got, want) {
		t.Errorf("DefaultBaseDirs() = %v, want %v", got, want)
	}
}

func TestDefaultBaseDirs_XDGDataHomeWins(t *testing.T) {
	t.Setenv("HOME", "/tmp/h")
	t.Setenv("XDG_DATA_HOME", "/tmp/xdh")
	t.Setenv("XDG_DATA_DIRS", "")

	got := DefaultBaseDirs()

	// /tmp/xdh/icons must be present, /tmp/h/.local/share/icons must not.
	if !containsString(got, "/tmp/xdh/icons") {
		t.Errorf("missing XDG_DATA_HOME entry in %v", got)
	}
	if containsString(got, "/tmp/h/.local/share/icons") {
		t.Errorf("HOME fallback should not appear when XDG_DATA_HOME set: %v", got)
	}
}

func TestDefaultBaseDirs_Dedupe(t *testing.T) {
	// XDG_DATA_DIRS contains /usr/share which duplicates /usr/share/icons.
	t.Setenv("HOME", "/tmp/h")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_DATA_DIRS", "/usr/share")

	got := DefaultBaseDirs()

	seen := make(map[string]int, len(got))
	for _, p := range got {
		seen[p]++
	}
	for p, n := range seen {
		if n > 1 {
			t.Errorf("path %q appears %d times in %v", p, n, got)
		}
	}
}

func TestDefaultBaseDirs_IgnoresEmptyXDGEntries(t *testing.T) {
	t.Setenv("HOME", "/tmp/h")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_DATA_DIRS", "::/foo::")

	got := DefaultBaseDirs()

	for _, p := range got {
		if p == "/icons" {
			t.Errorf("empty XDG entry leaked into %v", got)
		}
	}
	if !containsString(got, "/foo/icons") {
		t.Errorf("missing /foo/icons in %v", got)
	}
}

func TestDefaultBaseDirs_NoHome(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_DATA_DIRS", "")

	got := DefaultBaseDirs()

	for _, p := range got {
		if p == ".icons" || p == ".local/share/icons" {
			t.Errorf("HOME-relative entry leaked into %v", got)
		}
	}
	if !containsString(got, "/usr/share/icons") {
		t.Errorf("missing /usr/share/icons in %v", got)
	}
}

func TestNew_Defaults(t *testing.T) {
	t.Setenv("HOME", "/tmp/h")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_DATA_DIRS", "")

	f := New(nil, "")
	if f.Theme != "hicolor" {
		t.Errorf("Theme = %q, want hicolor", f.Theme)
	}
	if len(f.BaseDirs) == 0 {
		t.Error("BaseDirs empty, expected DefaultBaseDirs fallback")
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
