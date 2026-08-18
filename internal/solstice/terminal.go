package solstice

import (
	"image"
	"image/color"
	"strings"
	"unicode"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/mitchellh/go-wordwrap"
)

const (
	terminalWrapWidth   = 36
	terminalMaxHistory  = 300
	terminalVisibleRows = 28 // 224 / 8 = 28 lines
)

var defaultTerminal *Terminal

// GetTerminal returns the global terminal instance.
func GetTerminal() *Terminal {
	return defaultTerminal
}

// SetTerminal sets the global terminal instance.
func SetTerminal(t *Terminal) {
	defaultTerminal = t
}

// TerminalLine represents a single line of text in the terminal log with an optional color.
type TerminalLine struct {
	Text  string
	Color color.Color
}

// SavedTerminalLine represents a serialized terminal line with color support.
type SavedTerminalLine struct {
	Text  string      `json:"text"`
	Color *color.RGBA `json:"color,omitempty"`
}

// ColorToRGBA converts any color.Color to *color.RGBA, or nil if c is nil.
func ColorToRGBA(c color.Color) *color.RGBA {
	if c == nil {
		return nil
	}
	r, g, b, a := c.RGBA()
	return &color.RGBA{
		R: uint8(r >> 8),
		G: uint8(g >> 8),
		B: uint8(b >> 8),
		A: uint8(a >> 8),
	}
}

// ToTerminalLine converts a SavedTerminalLine back to a TerminalLine.
func (s *SavedTerminalLine) ToTerminalLine() TerminalLine {
	var c color.Color
	if s.Color != nil {
		c = *s.Color
	}
	return TerminalLine{
		Text:  s.Text,
		Color: c,
	}
}

// Terminal manages the terminal UI, log message history with per-line colors, input mode, scrolling, and rendering.
type Terminal struct {
	lines            []TerminalLine
	scrollOffset     int
	inputMode        bool
	inputText        string
	ignoreFrameInput bool
}

// NewTerminal creates a new Terminal.
func NewTerminal() *Terminal {
	t := &Terminal{
		lines: make([]TerminalLine, 0, terminalMaxHistory),
	}
	defaultTerminal = t
	return t
}

// SetInputMode enables or disables terminal input mode.
func (t *Terminal) SetInputMode(enabled bool) {
	t.inputMode = enabled
	t.inputText = ""
	if enabled {
		t.ignoreFrameInput = true
	}
}

// IsInputMode returns true if terminal input mode is active.
func (t *Terminal) IsInputMode() bool {
	return t.inputMode
}

// AddMessage adds a message to the terminal log with default white color.
func (t *Terminal) AddMessage(msg string) {
	t.AddMessageColored(msg, nil)
}

// AddMessageColored adds a message to the terminal log with a specific text color.
// Messages are word-wrapped to 36 characters and up to 300 lines of history are retained.
func (t *Terminal) AddMessageColored(msg string, c color.Color) {
	wrapped := wordwrap.WrapString(msg, terminalWrapWidth)
	newLines := strings.Split(wrapped, "\n")
	for _, line := range newLines {
		t.lines = append(t.lines, TerminalLine{
			Text:  line,
			Color: c,
		})
	}

	if len(t.lines) > terminalMaxHistory {
		t.lines = t.lines[len(t.lines)-terminalMaxHistory:]
	}
}

// Clear removes all lines from the terminal history and resets scroll offset.
func (t *Terminal) Clear() {
	t.lines = t.lines[:0]
	t.scrollOffset = 0
}

// GetLines returns all terminal lines with their text and color metadata.
func (t *Terminal) GetLines() []TerminalLine {
	return t.lines
}

// GetLineTexts returns a slice of strings containing the raw text of all terminal lines.
func (t *Terminal) GetLineTexts() []string {
	res := make([]string, len(t.lines))
	for i, l := range t.lines {
		res[i] = l.Text
	}
	return res
}

// GetSavedLines returns a slice of serializable SavedTerminalLine structs from the current terminal history.
func (t *Terminal) GetSavedLines() []SavedTerminalLine {
	if t == nil {
		return nil
	}
	res := make([]SavedTerminalLine, len(t.lines))
	for i, l := range t.lines {
		res[i] = SavedTerminalLine{
			Text:  l.Text,
			Color: ColorToRGBA(l.Color),
		}
	}
	return res
}

// RestoreSavedLines restores the terminal history from a slice of SavedTerminalLine structs.
func (t *Terminal) RestoreSavedLines(saved []SavedTerminalLine) {
	if t == nil {
		return
	}
	t.lines = make([]TerminalLine, 0, len(saved))
	for _, sl := range saved {
		t.lines = append(t.lines, sl.ToTerminalLine())
	}
	if len(t.lines) > terminalMaxHistory {
		t.lines = t.lines[len(t.lines)-terminalMaxHistory:]
	}
	t.scrollOffset = 0
}

// HandleInputScrollOnly handles Page Up and Page Down scrolling.
func (t *Terminal) HandleInputScrollOnly() {
	if inpututil.IsKeyJustPressed(ebiten.KeyPageUp) {
		t.scrollOffset += 20
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyPageDown) {
		t.scrollOffset -= 20
	}

	visibleRows := terminalVisibleRows
	if t.inputMode {
		visibleRows = terminalVisibleRows - 1
	}

	maxScroll := len(t.lines) - visibleRows
	if maxScroll < 0 {
		maxScroll = 0
	}

	if t.scrollOffset > maxScroll {
		t.scrollOffset = maxScroll
	}
	if t.scrollOffset < 0 {
		t.scrollOffset = 0
	}
}

// HandleInput handles standard terminal keyboard input (scrolling when not in input mode).
func (t *Terminal) HandleInput() {
	t.HandleInputScrollOnly()
}

// HandleInputMode processes character entry, backspace, submit (Enter), and cancel (Escape) in terminal input mode.
// Converts all entered characters to upper-case. Returns (submittedText, submitted, canceled).
func (t *Terminal) HandleInputMode() (string, bool, bool) {
	if !t.inputMode {
		return "", false, false
	}

	t.HandleInputScrollOnly()

	if t.ignoreFrameInput {
		t.ignoreFrameInput = false
		return "", false, false
	}

	// Cancel input mode on Escape key press
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		t.SetInputMode(false)
		return "", false, true
	}

	// Backspace support
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		if len(t.inputText) > 0 {
			runes := []rune(t.inputText)
			t.inputText = string(runes[:len(runes)-1])
		}
	}

	// Read typed characters
	var typedRunes []rune
	typedRunes = ebiten.AppendInputChars(typedRunes)
	for _, r := range typedRunes {
		if r >= 32 && r != 127 {
			upperR := unicode.ToUpper(r)
			if len([]rune(t.inputText)) < terminalWrapWidth-3 { // Leave room for prompt "> " and cursor
				t.inputText += string(upperR)
			}
		}
	}

	// Confirm on Enter
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyKPEnter) {
		text := t.inputText
		t.inputText = ""
		return text, true, false
	}

	return "", false, false
}

// Draw renders the terminal log to screen at position (352, 128) with size 288x224 pixels.
// In input mode, the bottom line displays prompt "> ", typed input text, and animated cursor all in bright blue.
func (t *Terminal) Draw(dst *ebiten.Image, assets *Assets) {
	if assets == nil || dst == nil {
		return
	}

	termArea := dst.SubImage(image.Rect(352, 128, 352+288, 128+224)).(*ebiten.Image)

	logRows := terminalVisibleRows
	if t.inputMode {
		logRows = terminalVisibleRows - 1
	}

	N := len(t.lines)
	if N > 0 {
		bottomLineIdx := (N - 1) - t.scrollOffset
		for r := 0; r < logRows; r++ {
			lineIdx := bottomLineIdx - ((logRows - 1) - r)
			if lineIdx >= 0 && lineIdx < N {
				lineObj := t.lines[lineIdx]
				// cellX = 352 / 8 = 44, cellY = 128 / 8 + r = 16 + r
				if lineObj.Color != nil {
					assets.DrawString8x8Colored(termArea, lineObj.Text, 44, 16+r, lineObj.Color)
				} else {
					assets.DrawString8x8(termArea, lineObj.Text, 44, 16+r)
				}
			}
		}
	}

	if t.inputMode {
		promptRow := 16 + (terminalVisibleRows - 1) // Row 27
		promptStr := "> " + t.inputText
		brightBlue := VGAPalette16[9] // Bright blue from 16 VGA palette

		// Draw prompt and input line text in bright blue
		assets.DrawString8x8Colored(termArea, promptStr, 44, promptRow, brightBlue)

		// Draw animated cursor as glyphs 5-8 in bright blue
		cursorCol := 44 + len([]rune(promptStr))
		cursorGlyph := 5 + (GetAnimFrame() % 4)
		assets.DrawGlyph8x8Colored(termArea, cursorGlyph, cursorCol, promptRow, brightBlue)
	}
}
