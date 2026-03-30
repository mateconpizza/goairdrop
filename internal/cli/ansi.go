package cli

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

type ansi string

const (
	Reset ansi = "\x1b[0m" // Reset all attributes

	// Standard foreground colors (30-37).
	Black   ansi = "\x1b[30m"
	Red     ansi = "\x1b[31m"
	Green   ansi = "\x1b[32m"
	Yellow  ansi = "\x1b[33m"
	Blue    ansi = "\x1b[34m"
	Magenta ansi = "\x1b[35m"
	Cyan    ansi = "\x1b[36m"
	White   ansi = "\x1b[37m"

	// Bright foreground colors (90-97).
	BrightBlack   ansi = "\x1b[90m"
	BrightRed     ansi = "\x1b[91m"
	BrightGreen   ansi = "\x1b[92m"
	BrightYellow  ansi = "\x1b[93m"
	BrightBlue    ansi = "\x1b[94m"
	BrightMagenta ansi = "\x1b[95m"
	BrightCyan    ansi = "\x1b[96m"
	BrightWhite   ansi = "\x1b[97m"

	// Text styles.
	Bold          ansi = "\x1b[1m"   // Bold
	Dim           ansi = "\x1b[2m"   // Faint or dim
	Italic        ansi = "\x1b[3m"   // Italic
	Underline     ansi = "\x1b[4m"   // Underline
	Undercurl     ansi = "\x1b[4:3m" // Undercurl
	Blink         ansi = "\x1b[5m"   // Slow blink
	BlinkRapid    ansi = "\x1b[6m"   // Rapid blink
	Inverse       ansi = "\x1b[7m"   // Inverse/reverse video
	Hidden        ansi = "\x1b[8m"   // Conceal/hidden
	Strikethrough ansi = "\x1b[9m"   // Crossed-out/strikethrough
)

// AnsiRemover removes ANSI codes from a given string.
func AnsiRemover(s string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return re.ReplaceAllString(s, "")
}

// Wrap wraps the given text with the provided styles and resets afterwards.
func (s ansi) Wrap(text string, styles ...ansi) string {
	if colorDisabled() {
		return text
	}

	return string(s) + combine(styles...) + text + string(Reset)
}

// With combines the receiver style with additional styles and returns a new
// ansi value.
func (s ansi) With(styles ...ansi) ansi {
	return ansi(string(s) + combine(styles...))
}

// Sprint wraps the formatted text with the receiver style and returns it as a
// string.
func (s ansi) Sprint(a ...any) string {
	return s.Wrap(fmt.Sprint(a...))
}

// Sprintf wraps the formatted text using the provided format string with the
// receiver style and returns it as a string.
func (s ansi) Sprintf(f string, a ...any) string {
	return s.Wrap(fmt.Sprintf(f, a...))
}

// Print prints styled text to the standard output.
func (s ansi) Print(a ...any) {
	fmt.Print(s.Wrap(s.Sprint(a...)))
}

// Println prints styled text with a newline.
func (s ansi) Println(a ...any) {
	fmt.Println(s.Wrap(s.Sprint(a...)))
}

// Printf prints styled text using a format string.
func (s ansi) Printf(format string, a ...any) {
	fmt.Print(s.Wrap(fmt.Sprintf(format, a...)))
}

// combine merges multiple ansi codes into a single string.
func combine(codes ...ansi) string {
	var sb strings.Builder
	for _, code := range codes {
		sb.WriteString(string(code))
	}
	return sb.String()
}

// colorDisabled disables color output if the NO_COLOR environment variable is
// set.
func colorDisabled() bool {
	// https://no-color.org
	const noColorEnv string = "NO_COLOR"
	_, ok := os.LookupEnv(noColorEnv)
	return ok
}
