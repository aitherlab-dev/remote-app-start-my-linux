package desktop

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrUnterminatedQuote = errors.New("unterminated double quote")
	ErrInvalidEscape     = errors.New("invalid escape sequence")
)

func SplitExec(exec string) ([]string, error) {
	result := []string{}
	var (
		cur     strings.Builder
		inQuote bool
		inToken bool
	)

	flush := func() {
		if inToken {
			result = append(result, cur.String())
			cur.Reset()
			inToken = false
		}
	}

	for i := 0; i < len(exec); i++ {
		c := exec[i]

		if inQuote {
			switch c {
			case '"':
				inQuote = false
			case '\\':
				if i+1 >= len(exec) {
					return nil, fmt.Errorf("split exec: pos %d: %w", i, ErrInvalidEscape)
				}
				next := exec[i+1]
				switch next {
				case '\\', '"', '`', '$':
					cur.WriteByte(next)
					i++
				default:
					return nil, fmt.Errorf("split exec: pos %d: %w", i, ErrInvalidEscape)
				}
			default:
				cur.WriteByte(c)
			}
			continue
		}

		switch c {
		case ' ', '\t':
			flush()
		case '"':
			inQuote = true
			inToken = true
		case '\\':
			if i+1 >= len(exec) {
				return nil, fmt.Errorf("split exec: pos %d: %w", i, ErrInvalidEscape)
			}
			if exec[i+1] != '\\' {
				return nil, fmt.Errorf("split exec: pos %d: %w", i, ErrInvalidEscape)
			}
			cur.WriteByte('\\')
			inToken = true
			i++
		default:
			cur.WriteByte(c)
			inToken = true
		}
	}

	if inQuote {
		return nil, fmt.Errorf("split exec: pos %d: %w", len(exec), ErrUnterminatedQuote)
	}
	flush()

	return result, nil
}

func Expand(tokens []string, entry *Entry, files []string) []string {
	var icon, name string
	if entry != nil {
		icon = entry.Icon
		name = entry.Name
	}

	result := make([]string, 0, len(tokens))
	for _, tok := range tokens {
		switch tok {
		case "%F", "%U":
			result = append(result, files...)
			continue
		case "%f", "%u":
			if len(files) > 0 {
				result = append(result, files[0])
			}
			continue
		case "%i":
			if icon != "" {
				result = append(result, "--icon", icon)
			}
			continue
		case "%c":
			if name != "" {
				result = append(result, name)
			}
			continue
		case "%k", "%v", "%m", "%d", "%D", "%n", "%N":
			continue
		}

		if !strings.ContainsRune(tok, '%') {
			result = append(result, tok)
			continue
		}

		var b strings.Builder
		b.Grow(len(tok))
		for i := 0; i < len(tok); i++ {
			if tok[i] != '%' || i+1 >= len(tok) {
				b.WriteByte(tok[i])
				continue
			}
			next := tok[i+1]
			switch next {
			case '%':
				b.WriteByte('%')
			case 'f', 'u':
				if len(files) > 0 {
					b.WriteString(files[0])
				}
			case 'F', 'U':
			case 'i':
			case 'c':
				b.WriteString(name)
			case 'k':
			case 'v', 'm', 'd', 'D', 'n', 'N':
			default:
				b.WriteByte('%')
				b.WriteByte(next)
			}
			i++
		}
		if b.Len() > 0 {
			result = append(result, b.String())
		}
	}
	return result
}
