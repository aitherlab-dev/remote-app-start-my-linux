package desktop

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		want           *Entry
		wantErrIs      error
		wantErrContain []string
	}{
		{
			name:  "minimal application",
			input: "[Desktop Entry]\nType=Application\nName=Test\nExec=/bin/test\n",
			want: &Entry{
				Type: "Application",
				Name: "Test",
				Exec: "/bin/test",
			},
		},
		{
			name: "full set of fields",
			input: `[Desktop Entry]
Type=Application
Name=Full
GenericName=Generic
Comment=A comment
Exec=/bin/full --arg
TryExec=/bin/full
Icon=full-icon
Terminal=true
NoDisplay=true
Hidden=false
StartupNotify=true
StartupWMClass=Full
Path=/tmp
Categories=Utility;System;
Keywords=foo;bar;
OnlyShowIn=GNOME;KDE;
NotShowIn=XFCE;
`,
			want: &Entry{
				Type:           "Application",
				Name:           "Full",
				GenericName:    "Generic",
				Comment:        "A comment",
				Exec:           "/bin/full --arg",
				TryExec:        "/bin/full",
				Icon:           "full-icon",
				Terminal:       true,
				NoDisplay:      true,
				Hidden:         false,
				StartupNotify:  true,
				StartupWMClass: "Full",
				Path:           "/tmp",
				Categories:     []string{"Utility", "System"},
				Keywords:       []string{"foo", "bar"},
				OnlyShowIn:     []string{"GNOME", "KDE"},
				NotShowIn:      []string{"XFCE"},
			},
		},
		{
			name: "comments and blank lines ignored",
			input: `# header comment
[Desktop Entry]
# inside comment

Type=Application
Name=CommentTest
Exec=/bin/x
# trailing comment
`,
			want: &Entry{
				Type: "Application",
				Name: "CommentTest",
				Exec: "/bin/x",
			},
		},
		{
			name:  "BOM at start",
			input: "\uFEFF[Desktop Entry]\nType=Application\nName=BomTest\nExec=/bin/x\n",
			want: &Entry{
				Type: "Application",
				Name: "BomTest",
				Exec: "/bin/x",
			},
		},
		{
			name: "localized keys ignored",
			input: `[Desktop Entry]
Type=Application
Name[ru]=Тест
Name=Test
GenericName[de]=Eintrag
Exec=/bin/test
`,
			want: &Entry{
				Type: "Application",
				Name: "Test",
				Exec: "/bin/test",
			},
		},
		{
			name: "actions section ignored, does not overwrite Entry",
			input: `[Desktop Entry]
Type=Application
Name=Main
Exec=/bin/main

[Desktop Action foo]
Name=Foo
Exec=/bin/foo

[Desktop Action bar]
Name=Bar
Exec=/bin/bar
`,
			want: &Entry{
				Type: "Application",
				Name: "Main",
				Exec: "/bin/main",
			},
		},
		{
			name:  "escapes in string and list",
			input: "[Desktop Entry]\nType=Application\nName=Hello\\sWorld\\\\\nCategories=Foo\\;Bar;Baz;\nComment=line1\\nline2\n",
			want: &Entry{
				Type:       "Application",
				Name:       "Hello World\\",
				Categories: []string{"Foo;Bar", "Baz"},
				Comment:    "line1\nline2",
			},
		},
		{
			name:  "escape sequences inside list",
			input: "[Desktop Entry]\nType=Application\nName=X\nKeywords=a\\sb;c\\nd;e\\tf;g\\rh;i\\\\j;\n",
			want: &Entry{
				Type:     "Application",
				Name:     "X",
				Keywords: []string{"a b", "c\nd", "e\tf", "g\rh", "i\\j"},
			},
		},
		{
			name:  "list without trailing semicolon is tolerated",
			input: "[Desktop Entry]\nType=Application\nName=X\nCategories=Foo;Bar\n",
			want: &Entry{
				Type:       "Application",
				Name:       "X",
				Categories: []string{"Foo", "Bar"},
			},
		},
		{
			name: "type Link is valid",
			input: `[Desktop Entry]
Type=Link
Name=A Link
URL=https://example.com
`,
			want: &Entry{
				Type: "Link",
				Name: "A Link",
				URL:  "https://example.com",
			},
		},
		{
			name: "type Directory is valid",
			input: `[Desktop Entry]
Type=Directory
Name=A Dir
`,
			want: &Entry{
				Type: "Directory",
				Name: "A Dir",
			},
		},
		{
			name:      "missing desktop entry section",
			input:     "[Other]\nFoo=bar\n",
			wantErrIs: ErrMissingDesktopSection,
		},
		{
			name:      "empty file",
			input:     "",
			wantErrIs: ErrMissingDesktopSection,
		},
		{
			name:      "missing type",
			input:     "[Desktop Entry]\nName=Foo\nExec=/bin/x\n",
			wantErrIs: ErrMissingType,
		},
		{
			name:      "unknown type wraps ErrInvalidType",
			input:     "[Desktop Entry]\nType=Unknown\nName=Foo\n",
			wantErrIs: ErrInvalidType,
		},
		{
			name: "invalid boolean reports field and line",
			input: `[Desktop Entry]
Type=Application
Name=BadBool
Terminal=yes
Exec=/bin/x
`,
			wantErrContain: []string{"Terminal", "line 4", "yes"},
		},
		{
			name:  "duplicate key, last wins",
			input: "[Desktop Entry]\nType=Application\nName=A\nName=B\nExec=/bin/x\n",
			want: &Entry{
				Type: "Application",
				Name: "B",
				Exec: "/bin/x",
			},
		},
		{
			name:  "spaces around equals are trimmed",
			input: "[Desktop Entry]\nType = Application\nName = Test\nExec = /bin/x\n",
			want: &Entry{
				Type: "Application",
				Name: "Test",
				Exec: "/bin/x",
			},
		},
		{
			name:           "key outside of any section",
			input:          "Foo=bar\n[Desktop Entry]\nType=Application\nName=X\n",
			wantErrContain: []string{"line 1", "outside"},
		},
		{
			name: "missing equals separator",
			input: `[Desktop Entry]
Type=Application
NoEqualsHere
Name=X
`,
			wantErrContain: []string{"line 3", "missing"},
		},
		{
			name: "empty key",
			input: `[Desktop Entry]
Type=Application
Name=X
=oops
`,
			wantErrContain: []string{"line 4", "empty key"},
		},
		{
			name:           "trailing backslash in string",
			input:          "[Desktop Entry]\nType=Application\nName=Bad\\\n",
			wantErrContain: []string{"Name", "line 3", "trailing backslash"},
		},
		{
			name:           "invalid escape sequence in string",
			input:          "[Desktop Entry]\nType=Application\nName=Bad\\x\n",
			wantErrContain: []string{"Name", "line 3", "invalid escape"},
		},
		{
			name:           "trailing backslash in list",
			input:          "[Desktop Entry]\nType=Application\nName=X\nCategories=Foo;Bar\\\n",
			wantErrContain: []string{"Categories", "line 4", "trailing backslash"},
		},
		{
			name:           "invalid escape sequence in list",
			input:          "[Desktop Entry]\nType=Application\nName=X\nCategories=Foo;Bar\\q\n",
			wantErrContain: []string{"Categories", "line 4", "invalid escape"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse(strings.NewReader(tc.input))

			if tc.wantErrIs != nil {
				if !errors.Is(err, tc.wantErrIs) {
					t.Fatalf("err = %v, want errors.Is(_, %v)", err, tc.wantErrIs)
				}
				if got != nil {
					t.Fatalf("entry on error must be nil, got %+v", got)
				}
				return
			}

			if len(tc.wantErrContain) > 0 {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				msg := err.Error()
				for _, sub := range tc.wantErrContain {
					if !strings.Contains(msg, sub) {
						t.Errorf("error %q does not contain %q", msg, sub)
					}
				}
				if got != nil {
					t.Fatalf("entry on error must be nil, got %+v", got)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("entry mismatch\n got: %+v\nwant: %+v", got, tc.want)
			}
		})
	}
}

func TestParse_RealFile(t *testing.T) {
	f, err := os.Open("testdata/real_firefox.desktop")
	if err != nil {
		t.Fatalf("open testdata: %v", err)
	}
	defer f.Close()

	entry, err := Parse(f)
	if err != nil {
		t.Fatalf("parse real desktop file: %v", err)
	}
	if entry.Name == "" {
		t.Error("entry.Name is empty")
	}
	if entry.Exec == "" {
		t.Error("entry.Exec is empty")
	}
	if entry.Type != TypeApplication {
		t.Errorf("entry.Type = %q, want %q", entry.Type, TypeApplication)
	}
}
