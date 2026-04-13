// Package desktop parses freedesktop Desktop Entry files.
//
// See: https://specifications.freedesktop.org/desktop-entry-spec/latest/
package desktop

import "errors"

const (
	TypeApplication = "Application"
	TypeLink        = "Link"
	TypeDirectory   = "Directory"
)

var (
	ErrMissingDesktopSection = errors.New("missing [Desktop Entry] section")
	ErrMissingType           = errors.New("missing Type key")
	ErrInvalidType           = errors.New("invalid Type value")
)

type Entry struct {
	// ID is populated by the directory scanner, not by Parse.
	// It is derived from the file path relative to the scan root
	// (see scanner.go) and stays empty for entries produced by Parse directly.
	ID             string
	Name           string
	GenericName    string
	Comment        string
	Exec           string
	TryExec        string
	Icon           string
	Terminal       bool
	Type           string
	Categories     []string
	Keywords       []string
	NoDisplay      bool
	Hidden         bool
	OnlyShowIn     []string
	NotShowIn      []string
	StartupNotify  bool
	StartupWMClass string
	Path           string
	URL            string
}
