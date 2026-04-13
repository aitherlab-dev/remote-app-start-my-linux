package icons

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ErrThemeIndexNotFound is returned by LoadThemeIndex when the theme
// directory does not contain an index.theme file. It wraps fs.ErrNotExist
// so callers can match either with errors.Is.
var ErrThemeIndexNotFound = errors.New("theme index not found")

// ThemeIndex is the parsed subset of <theme>/index.theme that the icon
// finder cares about. Only the [Icon Theme] section is consulted, and
// only the Name and Inherits keys are kept; every other field (Comment,
// Directories, per-directory Size/Type/Context blocks) is intentionally
// ignored — the finder walks the theme tree recursively and infers sizes
// from directory names instead.
type ThemeIndex struct {
	Name     string
	Inherits []string
}

const iconThemeSection = "Icon Theme"

// LoadThemeIndex reads <themeDir>/index.theme and returns the parsed
// [Icon Theme] section. A missing file yields ErrThemeIndexNotFound (via
// %w-wrapping) so callers can distinguish "no theme metadata here" from
// other I/O errors. A present file with no [Icon Theme] section, or a
// section without a Name= key, is reported as a parse error.
func LoadThemeIndex(themeDir string) (*ThemeIndex, error) {
	path := filepath.Join(themeDir, "index.theme")

	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("%s: %w", path, ErrThemeIndexNotFound)
		}
		return nil, fmt.Errorf("open theme index %s: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, scannerInitialBuf), maxLineSize)

	idx := &ThemeIndex{}
	var (
		lineNum         int
		currentSection  string
		seenIconSection bool
	)

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if lineNum == 1 {
			line = strings.TrimPrefix(line, "\uFEFF")
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			currentSection = trimmed[1 : len(trimmed)-1]
			if currentSection == iconThemeSection {
				seenIconSection = true
			}
			continue
		}

		if currentSection != iconThemeSection {
			continue
		}

		eq := strings.IndexByte(line, '=')
		if eq == -1 {
			return nil, fmt.Errorf("parse theme index %s: line %d: missing '=' separator", path, lineNum)
		}

		key := strings.TrimSpace(line[:eq])
		value := strings.TrimSpace(line[eq+1:])

		if key == "" {
			return nil, fmt.Errorf("parse theme index %s: line %d: empty key", path, lineNum)
		}

		if strings.IndexByte(key, '[') >= 0 {
			continue
		}

		switch key {
		case "Name":
			idx.Name = value
		case "Inherits":
			idx.Inherits = splitInheritsList(value)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read theme index %s: %w", path, err)
	}

	if !seenIconSection {
		return nil, fmt.Errorf("parse theme index %s: missing [Icon Theme] section", path)
	}
	if idx.Name == "" {
		return nil, fmt.Errorf("parse theme index %s: missing Name key", path)
	}

	return idx, nil
}

// splitInheritsList parses a comma-separated Inherits= value into trimmed,
// non-empty parent theme names. Order from the file is preserved.
func splitInheritsList(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		name := strings.TrimSpace(p)
		if name == "" {
			continue
		}
		out = append(out, name)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// scannerInitialBuf and maxLineSize are reused from desktop-style INI parsing.
// They are package-private constants so tests can pin their values.
const (
	scannerInitialBuf = 64 * 1024
	maxLineSize       = 1 << 20
)
