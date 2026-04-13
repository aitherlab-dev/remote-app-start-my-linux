package desktop

import (
	"errors"
	"reflect"
	"testing"
)

func TestSplitExec(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      []string
		wantErrIs error
	}{
		{
			name:  "empty string",
			input: "",
			want:  []string{},
		},
		{
			name:  "only spaces",
			input: "   \t  ",
			want:  []string{},
		},
		{
			name:  "single word",
			input: "firefox",
			want:  []string{"firefox"},
		},
		{
			name:  "several words",
			input: "a b c",
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "thunar with flag",
			input: "thunar --bulk-rename %F",
			want:  []string{"thunar", "--bulk-rename", "%F"},
		},
		{
			name:  "double quoted path with space",
			input: `"/opt/foo bar/bin" --flag`,
			want:  []string{"/opt/foo bar/bin", "--flag"},
		},
		{
			name:  "escaped double quote inside quotes",
			input: `sh -c "echo \"hi\""`,
			want:  []string{"sh", "-c", `echo "hi"`},
		},
		{
			name:  "escaped backslash inside quotes",
			input: `prog "a\\b"`,
			want:  []string{"prog", `a\b`},
		},
		{
			name:  "escaped backtick inside quotes",
			input: "prog \"a\\`b\"",
			want:  []string{"prog", "a`b"},
		},
		{
			name:  "escaped dollar inside quotes",
			input: `prog "a\$b"`,
			want:  []string{"prog", "a$b"},
		},
		{
			name:  "double percent stays literal",
			input: "env VAR=x prog %%percent",
			want:  []string{"env", "VAR=x", "prog", "%%percent"},
		},
		{
			name:  "multiple spaces as separators",
			input: "a    b\t\tc",
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "single quote is not special",
			input: "it's fine",
			want:  []string{"it's", "fine"},
		},
		{
			name:  "empty quoted string yields empty token",
			input: `foo "" bar`,
			want:  []string{"foo", "", "bar"},
		},
		{
			name:  "adjacent quoted and unquoted concatenate",
			input: `pre"fix post"suffix`,
			want:  []string{"prefix postsuffix"},
		},
		{
			name:  "unescaped backslash outside quotes is literal via \\\\",
			input: `a\\b`,
			want:  []string{`a\b`},
		},
		{
			name:      "unterminated quote",
			input:     `"foo`,
			wantErrIs: ErrUnterminatedQuote,
		},
		{
			name:      "invalid escape inside quotes",
			input:     `"foo\q"`,
			wantErrIs: ErrInvalidEscape,
		},
		{
			name:      "trailing backslash inside quotes",
			input:     `"foo\`,
			wantErrIs: ErrInvalidEscape,
		},
		{
			name:      "lone backslash outside quotes is invalid",
			input:     `foo \n`,
			wantErrIs: ErrInvalidEscape,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SplitExec(tt.input)
			if tt.wantErrIs != nil {
				if err == nil {
					t.Fatalf("SplitExec(%q) expected error %v, got nil (result=%#v)", tt.input, tt.wantErrIs, got)
				}
				if !errors.Is(err, tt.wantErrIs) {
					t.Fatalf("SplitExec(%q) error = %v, want errors.Is(%v)", tt.input, err, tt.wantErrIs)
				}
				return
			}
			if err != nil {
				t.Fatalf("SplitExec(%q) unexpected error: %v", tt.input, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("SplitExec(%q) = %#v, want %#v", tt.input, got, tt.want)
			}
		})
	}
}

func TestExpand(t *testing.T) {
	tests := []struct {
		name   string
		tokens []string
		entry  *Entry
		files  []string
		want   []string
	}{
		{
			name:   "no fields unchanged",
			tokens: []string{"firefox", "--safe-mode"},
			want:   []string{"firefox", "--safe-mode"},
		},
		{
			name:   "%f without files removed",
			tokens: []string{"firefox", "%f"},
			want:   []string{"firefox"},
		},
		{
			name:   "%f with file substituted",
			tokens: []string{"firefox", "%f"},
			files:  []string{"/tmp/x"},
			want:   []string{"firefox", "/tmp/x"},
		},
		{
			name:   "%F with two files expands",
			tokens: []string{"thunar", "--bulk-rename", "%F"},
			files:  []string{"/a", "/b"},
			want:   []string{"thunar", "--bulk-rename", "/a", "/b"},
		},
		{
			name:   "%F without files removed",
			tokens: []string{"thunar", "%F"},
			want:   []string{"thunar"},
		},
		{
			name:   "%u behaves like %f",
			tokens: []string{"firefox", "%u"},
			files:  []string{"https://example.com"},
			want:   []string{"firefox", "https://example.com"},
		},
		{
			name:   "%u without files removed",
			tokens: []string{"firefox", "%u"},
			want:   []string{"firefox"},
		},
		{
			name:   "%U behaves like %F",
			tokens: []string{"browser", "%U"},
			files:  []string{"https://a", "https://b"},
			want:   []string{"browser", "https://a", "https://b"},
		},
		{
			name:   "%i with icon expands to --icon pair",
			tokens: []string{"prog", "%i"},
			entry:  &Entry{Icon: "foo"},
			want:   []string{"prog", "--icon", "foo"},
		},
		{
			name:   "%i without icon removed",
			tokens: []string{"prog", "%i"},
			entry:  &Entry{},
			want:   []string{"prog"},
		},
		{
			name:   "%c replaced by Name",
			tokens: []string{"prog", "%c"},
			entry:  &Entry{Name: "Test"},
			want:   []string{"prog", "Test"},
		},
		{
			name:   "%c with empty name removed",
			tokens: []string{"prog", "%c"},
			entry:  &Entry{},
			want:   []string{"prog"},
		},
		{
			name:   "%k removed",
			tokens: []string{"prog", "%k"},
			want:   []string{"prog"},
		},
		{
			name:   "literal %% becomes %",
			tokens: []string{"echo", "100%%"},
			want:   []string{"echo", "100%"},
		},
		{
			name:   "standalone %% becomes %",
			tokens: []string{"echo", "%%"},
			want:   []string{"echo", "%"},
		},
		{
			name:   "%%f is literal percent then f, not file",
			tokens: []string{"echo", "%%f"},
			files:  []string{"/should/not/appear"},
			want:   []string{"echo", "%f"},
		},
		{
			name:   "deprecated %v removed",
			tokens: []string{"prog", "%v"},
			want:   []string{"prog"},
		},
		{
			name:   "deprecated fields removed",
			tokens: []string{"prog", "%m", "%d", "%D", "%n", "%N"},
			want:   []string{"prog"},
		},
		{
			name:   "embedded %f in token",
			tokens: []string{"--opt=%f", "file"},
			files:  []string{"/tmp/x"},
			want:   []string{"--opt=/tmp/x", "file"},
		},
		{
			name:   "embedded %f in token without files keeps prefix",
			tokens: []string{"--opt=%f"},
			want:   []string{"--opt="},
		},
		{
			name:   "embedded %c in token",
			tokens: []string{"--name=%c"},
			entry:  &Entry{Name: "App"},
			want:   []string{"--name=App"},
		},
		{
			name:   "nil entry is safe",
			tokens: []string{"prog", "%c", "%i"},
			want:   []string{"prog"},
		},
		{
			name:   "combination of fields",
			tokens: []string{"prog", "%c", "--files", "%F"},
			entry:  &Entry{Name: "Foo"},
			files:  []string{"a", "b"},
			want:   []string{"prog", "Foo", "--files", "a", "b"},
		},
		{
			name:   "unknown field preserved as literal",
			tokens: []string{"prog", "%z"},
			want:   []string{"prog", "%z"},
		},
		{
			name:   "trailing percent preserved",
			tokens: []string{"prog", "50%"},
			want:   []string{"prog", "50%"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Expand(tt.tokens, tt.entry, tt.files)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("Expand(%#v, %#v, %#v) = %#v, want %#v",
					tt.tokens, tt.entry, tt.files, got, tt.want)
			}
		})
	}
}
