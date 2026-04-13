package desktop

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	desktopEntrySection = "Desktop Entry"
	maxLineSize         = 1 << 20
	scannerInitialBuf   = 64 * 1024
)

func Parse(r io.Reader) (*Entry, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, scannerInitialBuf), maxLineSize)

	entry := &Entry{}
	var (
		lineNum            int
		currentSection     string
		seenDesktopSection bool
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
			if currentSection == desktopEntrySection {
				seenDesktopSection = true
			}
			continue
		}

		if currentSection == "" {
			return nil, fmt.Errorf("parse desktop entry: line %d: key outside of any section", lineNum)
		}

		eq := strings.IndexByte(line, '=')
		if eq == -1 {
			return nil, fmt.Errorf("parse desktop entry: line %d: missing '=' separator", lineNum)
		}

		key := strings.TrimSpace(line[:eq])
		value := strings.TrimSpace(line[eq+1:])

		if currentSection != desktopEntrySection {
			continue
		}

		if key == "" {
			return nil, fmt.Errorf("parse desktop entry: line %d: empty key", lineNum)
		}

		if strings.IndexByte(key, '[') >= 0 {
			continue
		}

		if err := assignKey(entry, key, value); err != nil {
			return nil, fmt.Errorf("parse desktop entry: line %d: %s: %w", lineNum, key, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("parse desktop entry: read: %w", err)
	}

	if !seenDesktopSection {
		return nil, ErrMissingDesktopSection
	}
	if entry.Type == "" {
		return nil, ErrMissingType
	}

	switch entry.Type {
	case TypeApplication, TypeLink, TypeDirectory:
	default:
		return nil, fmt.Errorf("unsupported type %q: %w", entry.Type, ErrInvalidType)
	}

	return entry, nil
}

func assignKey(e *Entry, key, value string) error {
	switch key {
	case "Name":
		s, err := unescapeString(value)
		if err != nil {
			return err
		}
		e.Name = s
	case "GenericName":
		s, err := unescapeString(value)
		if err != nil {
			return err
		}
		e.GenericName = s
	case "Comment":
		s, err := unescapeString(value)
		if err != nil {
			return err
		}
		e.Comment = s
	case "Exec":
		s, err := unescapeString(value)
		if err != nil {
			return err
		}
		e.Exec = s
	case "TryExec":
		s, err := unescapeString(value)
		if err != nil {
			return err
		}
		e.TryExec = s
	case "Icon":
		s, err := unescapeString(value)
		if err != nil {
			return err
		}
		e.Icon = s
	case "Type":
		s, err := unescapeString(value)
		if err != nil {
			return err
		}
		e.Type = s
	case "Path":
		s, err := unescapeString(value)
		if err != nil {
			return err
		}
		e.Path = s
	case "URL":
		s, err := unescapeString(value)
		if err != nil {
			return err
		}
		e.URL = s
	case "StartupWMClass":
		s, err := unescapeString(value)
		if err != nil {
			return err
		}
		e.StartupWMClass = s
	case "Terminal":
		b, err := parseBool(value)
		if err != nil {
			return err
		}
		e.Terminal = b
	case "NoDisplay":
		b, err := parseBool(value)
		if err != nil {
			return err
		}
		e.NoDisplay = b
	case "Hidden":
		b, err := parseBool(value)
		if err != nil {
			return err
		}
		e.Hidden = b
	case "StartupNotify":
		b, err := parseBool(value)
		if err != nil {
			return err
		}
		e.StartupNotify = b
	case "Categories":
		list, err := parseStringList(value)
		if err != nil {
			return err
		}
		e.Categories = list
	case "Keywords":
		list, err := parseStringList(value)
		if err != nil {
			return err
		}
		e.Keywords = list
	case "OnlyShowIn":
		list, err := parseStringList(value)
		if err != nil {
			return err
		}
		e.OnlyShowIn = list
	case "NotShowIn":
		list, err := parseStringList(value)
		if err != nil {
			return err
		}
		e.NotShowIn = list
	}
	return nil
}

func parseBool(s string) (bool, error) {
	switch s {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean %q (expected true or false)", s)
	}
}

func unescapeString(s string) (string, error) {
	if !strings.ContainsRune(s, '\\') {
		return s, nil
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != '\\' {
			b.WriteByte(c)
			continue
		}
		i++
		if i >= len(s) {
			return "", errors.New("trailing backslash")
		}
		switch s[i] {
		case '\\':
			b.WriteByte('\\')
		case 's':
			b.WriteByte(' ')
		case 'n':
			b.WriteByte('\n')
		case 't':
			b.WriteByte('\t')
		case 'r':
			b.WriteByte('\r')
		default:
			return "", fmt.Errorf("invalid escape sequence \\%c", s[i])
		}
	}
	return b.String(), nil
}

func parseStringList(s string) ([]string, error) {
	if s == "" {
		return nil, nil
	}
	var (
		result []string
		cur    strings.Builder
	)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\\' {
			i++
			if i >= len(s) {
				return nil, errors.New("trailing backslash")
			}
			switch s[i] {
			case '\\':
				cur.WriteByte('\\')
			case ';':
				cur.WriteByte(';')
			case 's':
				cur.WriteByte(' ')
			case 'n':
				cur.WriteByte('\n')
			case 't':
				cur.WriteByte('\t')
			case 'r':
				cur.WriteByte('\r')
			default:
				return nil, fmt.Errorf("invalid escape sequence \\%c", s[i])
			}
			continue
		}
		if c == ';' {
			result = append(result, cur.String())
			cur.Reset()
			continue
		}
		cur.WriteByte(c)
	}
	if cur.Len() > 0 {
		result = append(result, cur.String())
	}
	return result, nil
}
