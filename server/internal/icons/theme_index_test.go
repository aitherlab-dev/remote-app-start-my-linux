package icons

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeIndexTheme(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.theme"), []byte(body), 0o644); err != nil {
		t.Fatalf("write index.theme: %v", err)
	}
}

func TestLoadThemeIndex_Minimal(t *testing.T) {
	dir := t.TempDir()
	writeIndexTheme(t, dir, "[Icon Theme]\nName=Test\n")

	idx, err := LoadThemeIndex(dir)
	if err != nil {
		t.Fatalf("LoadThemeIndex: %v", err)
	}
	if idx.Name != "Test" {
		t.Errorf("Name = %q, want %q", idx.Name, "Test")
	}
	if idx.Inherits != nil {
		t.Errorf("Inherits = %v, want nil", idx.Inherits)
	}
}

func TestLoadThemeIndex_WithInherits(t *testing.T) {
	dir := t.TempDir()
	writeIndexTheme(t, dir, "[Icon Theme]\nName=Child\nInherits=Adwaita,hicolor\n")

	idx, err := LoadThemeIndex(dir)
	if err != nil {
		t.Fatalf("LoadThemeIndex: %v", err)
	}
	want := []string{"Adwaita", "hicolor"}
	if !equalStrings(idx.Inherits, want) {
		t.Errorf("Inherits = %v, want %v", idx.Inherits, want)
	}
}

func TestLoadThemeIndex_WithInheritsTrim(t *testing.T) {
	dir := t.TempDir()
	writeIndexTheme(t, dir, "[Icon Theme]\nName=Child\nInherits=Adwaita, hicolor , gnome\n")

	idx, err := LoadThemeIndex(dir)
	if err != nil {
		t.Fatalf("LoadThemeIndex: %v", err)
	}
	want := []string{"Adwaita", "hicolor", "gnome"}
	if !equalStrings(idx.Inherits, want) {
		t.Errorf("Inherits = %v, want %v", idx.Inherits, want)
	}
}

func TestLoadThemeIndex_NoFile(t *testing.T) {
	dir := t.TempDir()

	_, err := LoadThemeIndex(dir)
	if !errors.Is(err, ErrThemeIndexNotFound) {
		t.Errorf("err = %v, want ErrThemeIndexNotFound", err)
	}
}

func TestLoadThemeIndex_NoSection(t *testing.T) {
	dir := t.TempDir()
	writeIndexTheme(t, dir, "Name=Orphan\nInherits=hicolor\n")

	_, err := LoadThemeIndex(dir)
	if err == nil {
		t.Fatal("LoadThemeIndex: want error, got nil")
	}
}

func TestLoadThemeIndex_NoName(t *testing.T) {
	dir := t.TempDir()
	writeIndexTheme(t, dir, "[Icon Theme]\nComment=no name here\nInherits=hicolor\n")

	_, err := LoadThemeIndex(dir)
	if err == nil {
		t.Fatal("LoadThemeIndex: want error, got nil")
	}
}

func TestLoadThemeIndex_IgnoresLocalized(t *testing.T) {
	dir := t.TempDir()
	writeIndexTheme(t, dir, "[Icon Theme]\nName[ru]=Тест\nName=Test\n")

	idx, err := LoadThemeIndex(dir)
	if err != nil {
		t.Fatalf("LoadThemeIndex: %v", err)
	}
	if idx.Name != "Test" {
		t.Errorf("Name = %q, want %q", idx.Name, "Test")
	}
}

func TestLoadThemeIndex_IgnoresOtherSections(t *testing.T) {
	dir := t.TempDir()
	body := "[Icon Theme]\nName=A\nInherits=hicolor\n[16x16/apps]\nSize=16\nContext=Applications\n"
	writeIndexTheme(t, dir, body)

	idx, err := LoadThemeIndex(dir)
	if err != nil {
		t.Fatalf("LoadThemeIndex: %v", err)
	}
	if idx.Name != "A" {
		t.Errorf("Name = %q, want %q", idx.Name, "A")
	}
	if !equalStrings(idx.Inherits, []string{"hicolor"}) {
		t.Errorf("Inherits = %v, want [hicolor]", idx.Inherits)
	}
}

func TestLoadThemeIndex_InheritsEmptyAndAllBlank(t *testing.T) {
	dir := t.TempDir()
	writeIndexTheme(t, dir, "[Icon Theme]\nName=A\nInherits=\n")

	idx, err := LoadThemeIndex(dir)
	if err != nil {
		t.Fatalf("LoadThemeIndex empty: %v", err)
	}
	if idx.Inherits != nil {
		t.Errorf("Inherits = %v, want nil for empty value", idx.Inherits)
	}

	dir2 := t.TempDir()
	writeIndexTheme(t, dir2, "[Icon Theme]\nName=B\nInherits= , ,\n")

	idx2, err := LoadThemeIndex(dir2)
	if err != nil {
		t.Fatalf("LoadThemeIndex all-blank: %v", err)
	}
	if idx2.Inherits != nil {
		t.Errorf("Inherits = %v, want nil for all-blank value", idx2.Inherits)
	}
}

func TestLoadThemeIndex_RejectsLineWithoutEquals(t *testing.T) {
	dir := t.TempDir()
	writeIndexTheme(t, dir, "[Icon Theme]\nNameTest\n")

	_, err := LoadThemeIndex(dir)
	if err == nil {
		t.Fatal("want error for missing '=' separator")
	}
}

func TestLoadThemeIndex_RejectsEmptyKey(t *testing.T) {
	dir := t.TempDir()
	writeIndexTheme(t, dir, "[Icon Theme]\n=value\n")

	_, err := LoadThemeIndex(dir)
	if err == nil {
		t.Fatal("want error for empty key")
	}
}

func TestLoadThemeIndex_IgnoresCommentsAndBlankLines(t *testing.T) {
	dir := t.TempDir()
	body := "# header comment\n\n[Icon Theme]\n# inside comment\nName=A\n\nInherits=hicolor\n"
	writeIndexTheme(t, dir, body)

	idx, err := LoadThemeIndex(dir)
	if err != nil {
		t.Fatalf("LoadThemeIndex: %v", err)
	}
	if idx.Name != "A" {
		t.Errorf("Name = %q, want %q", idx.Name, "A")
	}
	if !equalStrings(idx.Inherits, []string{"hicolor"}) {
		t.Errorf("Inherits = %v, want [hicolor]", idx.Inherits)
	}
}
