package sshadmin

import "fmt"

type ProfileInfo struct {
	// Term is the TERM environment variable value.
	Term string
	// termenv color profile
	ColorProfile string
	Width        uint
	Height       uint
	// Whether terminal is rendered on a dark background
	IsDarkMode bool
}

func (p ProfileInfo) String() string {
	colorScheme := "Light mode"
	if p.IsDarkMode {
		colorScheme = "Dark mode"
	}
	return fmt.Sprintf("%s %s %s [w,h][%d,%d]", p.Term, p.ColorProfile, colorScheme, p.Width, p.Height)

}
