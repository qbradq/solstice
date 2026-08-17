package solstice

import (
	"image"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// TengoTerminalState defines the visibility/height state of the Tengo pull-down terminal.
type TengoTerminalState int

const (
	TengoTerminalStateHidden TengoTerminalState = iota // Hidden (default) - normal game update/draw path
	TengoTerminalStateHalf                             // Half - covers top half of screen
	TengoTerminalStateFull                             // Full - covers entire screen
)

var defaultTengoTerm *TengoTerminal

// GetTengoTerminal returns the global TengoTerminal instance.
func GetTengoTerminal() *TengoTerminal {
	return defaultTengoTerm
}

// SetTengoTerminal sets the global TengoTerminal instance.
func SetTengoTerminal(t *TengoTerminal) {
	defaultTengoTerm = t
}

// TengoTerminal manages the Quake-style pull-down REPL console UI, text editing, scrolling, and rendering.
type TengoTerminal struct {
	repl         *TengoREPL
	state        TengoTerminalState
	inputRunes   []rune
	cursorPos    int
	historyIndex int // -1 when typing draft line, otherwise index into repl.commandHistory
	draftInput   string
	scrollOffset int
	termBuffer   *ebiten.Image
}

// NewTengoTerminal creates and initializes a new TengoTerminal.
func NewTengoTerminal(repl *TengoREPL) *TengoTerminal {
	if repl == nil {
		repl = GetTengoREPL()
		if repl == nil {
			repl = NewTengoREPL()
		}
	}

	t := &TengoTerminal{
		repl:         repl,
		state:        TengoTerminalStateHidden,
		historyIndex: -1,
		termBuffer:   ebiten.NewImage(screenWidth, screenHeight),
	}
	SetTengoTerminal(t)
	return t
}

// GetState returns the current terminal display state.
func (t *TengoTerminal) GetState() TengoTerminalState {
	return t.state
}

// SetState sets the terminal display state.
func (t *TengoTerminal) SetState(state TengoTerminalState) {
	t.state = state
}

// CycleState advances the terminal state: Hidden -> Half -> Full -> Hidden.
func (t *TengoTerminal) CycleState() {
	switch t.state {
	case TengoTerminalStateHidden:
		t.state = TengoTerminalStateHalf
	case TengoTerminalStateHalf:
		t.state = TengoTerminalStateFull
	case TengoTerminalStateFull:
		t.state = TengoTerminalStateHidden
	default:
		t.state = TengoTerminalStateHidden
	}
}

// GetInputText returns the current typed input text.
func (t *TengoTerminal) GetInputText() string {
	return string(t.inputRunes)
}

// SetInputText sets the input text and moves cursor to the end.
func (t *TengoTerminal) SetInputText(s string) {
	t.inputRunes = []rune(s)
	t.cursorPos = len(t.inputRunes)
	t.historyIndex = -1
}

// GetCursorPos returns the current cursor index within the input text.
func (t *TengoTerminal) GetCursorPos() int {
	return t.cursorPos
}

// GetScrollOffset returns the current output log scroll offset.
func (t *TengoTerminal) GetScrollOffset() int {
	return t.scrollOffset
}

// SetScrollOffset sets the output log scroll offset.
func (t *TengoTerminal) SetScrollOffset(offset int) {
	t.scrollOffset = offset
}

func (t *TengoTerminal) getVisibleLogRows() int {
	numInputLines := t.getInputLineCount()
	totalRows := 45
	if t.state == TengoTerminalStateHalf {
		totalRows = 22
	}
	visible := totalRows - numInputLines
	if visible < 1 {
		visible = 1
	}
	return visible
}

func (t *TengoTerminal) getInputLineCount() int {
	totalInputRunes := len(t.inputRunes)
	if totalInputRunes <= 78 {
		return 1
	}
	return 1 + ((totalInputRunes - 78 + 79) / 80)
}

// Update handles text input, cursor navigation, history traversal, scrolling, and submission.
func (t *TengoTerminal) Update(g *Game) error {
	// Advance animated cursor frame ticker
	UpdateAnimTicker()

	// 1. Output scrolling: Page Up / Page Down
	visibleRows := t.getVisibleLogRows()
	outputLen := 0
	if t.repl != nil {
		outputLen = len(t.repl.GetOutputHistory())
	}

	maxScroll := outputLen - visibleRows
	if maxScroll < 0 {
		maxScroll = 0
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyPageUp) {
		t.scrollOffset += 20
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyPageDown) {
		t.scrollOffset -= 20
	}

	if t.scrollOffset > maxScroll {
		t.scrollOffset = maxScroll
	}
	if t.scrollOffset < 0 {
		t.scrollOffset = 0
	}

	// 2. Cursor movement: Left, Right, Home, End
	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		if t.cursorPos > 0 {
			t.cursorPos--
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		if t.cursorPos < len(t.inputRunes) {
			t.cursorPos++
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyHome) {
		t.cursorPos = 0
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnd) {
		t.cursorPos = len(t.inputRunes)
	}

	// 3. Command history: Up, Down arrows
	if t.repl != nil {
		history := t.repl.GetCommandHistory()
		historyLen := len(history)

		if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
			if historyLen > 0 {
				if t.historyIndex == -1 {
					t.draftInput = string(t.inputRunes)
					t.historyIndex = historyLen - 1
					t.inputRunes = []rune(history[t.historyIndex])
					t.cursorPos = len(t.inputRunes)
				} else if t.historyIndex > 0 {
					t.historyIndex--
					t.inputRunes = []rune(history[t.historyIndex])
					t.cursorPos = len(t.inputRunes)
				}
			}
		}

		if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
			if t.historyIndex != -1 {
				if t.historyIndex < historyLen-1 {
					t.historyIndex++
					t.inputRunes = []rune(history[t.historyIndex])
					t.cursorPos = len(t.inputRunes)
				} else {
					t.historyIndex = -1
					t.inputRunes = []rune(t.draftInput)
					t.cursorPos = len(t.inputRunes)
				}
			}
		}
	}

	// 4. Backspace and Delete
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		if t.cursorPos > 0 {
			t.inputRunes = append(t.inputRunes[:t.cursorPos-1], t.inputRunes[t.cursorPos:]...)
			t.cursorPos--
			t.historyIndex = -1
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDelete) {
		if t.cursorPos < len(t.inputRunes) {
			t.inputRunes = append(t.inputRunes[:t.cursorPos], t.inputRunes[t.cursorPos+1:]...)
			t.historyIndex = -1
		}
	}

	// 5. Character entry
	var typedRunes []rune
	typedRunes = ebiten.AppendInputChars(typedRunes)
	for _, r := range typedRunes {
		// Ignore control characters, DEL, and tilde/backquote (grave accent used for cycling)
		if r >= 32 && r != 127 && r != '`' && r != '~' {
			t.inputRunes = append(t.inputRunes[:t.cursorPos], append([]rune{r}, t.inputRunes[t.cursorPos:]...)...)
			t.cursorPos++
			t.historyIndex = -1
		}
	}

	// 6. Enter / KPEnter: Submit command line
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyKPEnter) {
		line := string(t.inputRunes)
		brightBlue := VGAPalette16[9]
		brightWhite := VGAPalette16[15]

		if t.repl != nil {
			if strings.TrimSpace(line) != "" {
				t.repl.AddCommand(line)
			}

			// Append input lines exactly (not word-wrapped) to the output history in bright VGA blue
			firstChunkLen := len(t.inputRunes)
			if firstChunkLen > 78 {
				firstChunkLen = 78
			}
			promptLine0 := "> " + string(t.inputRunes[:firstChunkLen])
			t.repl.AddRawOutputColored(promptLine0, brightBlue)

			remaining := len(t.inputRunes) - 78
			offset := 78
			for remaining > 0 {
				chunkSize := remaining
				if chunkSize > 80 {
					chunkSize = 80
				}
				chunkStr := string(t.inputRunes[offset : offset+chunkSize])
				t.repl.AddRawOutputColored(chunkStr, brightBlue)
				offset += chunkSize
				remaining -= chunkSize
			}

			// Execute statement and report error in white if returned
			if strings.TrimSpace(line) != "" {
				if err := t.repl.Execute(line); err != nil {
					t.repl.AddOutputColored(err.Error(), brightWhite)
				}
			}
		}
		t.inputRunes = nil
		t.cursorPos = 0
		t.historyIndex = -1
		t.draftInput = ""
		t.scrollOffset = 0
	}

	return nil
}

func (t *TengoTerminal) renderTerminalBuffer(assets *Assets) {
	if assets == nil {
		assets = defaultAssets
	}
	if assets == nil || t.termBuffer == nil {
		return
	}

	// 1. Clear terminal buffer with black
	t.termBuffer.Clear()
	for row := 0; row < 45; row++ {
		for col := 0; col < 80; col++ {
			assets.DrawGlyph8x8(t.termBuffer, -1, col, row)
		}
	}

	numInputLines := t.getInputLineCount()
	inputStartRow := 45 - numInputLines
	logRows := inputStartRow

	brightBlue := VGAPalette16[9]
	brightWhite := VGAPalette16[15]

	// 2. Render output history lines
	if t.repl != nil {
		outputLines := t.repl.GetOutputHistory()
		N := len(outputLines)
		if N > 0 {
			bottomLineIdx := (N - 1) - t.scrollOffset
			for r := 0; r < logRows; r++ {
				lineIdx := bottomLineIdx - ((logRows - 1) - r)
				if lineIdx >= 0 && lineIdx < N {
					lineObj := outputLines[lineIdx]
					col := lineObj.Color
					if col == nil {
						col = brightWhite
					}
					assets.DrawString8x8Colored(t.termBuffer, lineObj.Text, 0, r, col)
				}
			}
		}
	}

	// 3. Render input prompt, text, and overflow lines
	// Line 0: "> " + inputRunes[0:min(78, len)]
	firstChunkLen := len(t.inputRunes)
	if firstChunkLen > 78 {
		firstChunkLen = 78
	}
	promptLine0 := "> " + string(t.inputRunes[:firstChunkLen])
	assets.DrawString8x8Colored(t.termBuffer, promptLine0, 0, inputStartRow, brightBlue)

	// Overflow lines
	remaining := len(t.inputRunes) - 78
	offset := 78
	lineIdx := 1
	for remaining > 0 {
		chunkSize := remaining
		if chunkSize > 80 {
			chunkSize = 80
		}
		chunkStr := string(t.inputRunes[offset : offset+chunkSize])
		assets.DrawString8x8Colored(t.termBuffer, chunkStr, 0, inputStartRow+lineIdx, brightBlue)
		offset += chunkSize
		remaining -= chunkSize
		lineIdx++
	}

	// 4. Render animated cursor
	var cursorRow, cursorCol int
	if t.cursorPos <= 78 {
		cursorRow = inputStartRow
		cursorCol = 2 + t.cursorPos
	} else {
		curOffset := t.cursorPos - 78
		cursorRow = inputStartRow + 1 + (curOffset / 80)
		cursorCol = curOffset % 80
	}

	if cursorRow < 45 && cursorCol < 80 {
		cursorGlyph := 5 + (GetAnimFrame() % 4)
		assets.DrawGlyph8x8Colored(t.termBuffer, cursorGlyph, cursorCol, cursorRow, brightBlue)
	}
}

// DrawHalf renders the bottom half of the terminal onto the top half of dst (y: 0..180).
func (t *TengoTerminal) DrawHalf(dst *ebiten.Image, assets *Assets) {
	if dst == nil {
		return
	}

	t.renderTerminalBuffer(assets)
	if t.termBuffer == nil {
		return
	}

	// Extract the bottom half (y: 180 to 360) of the terminal buffer and draw at (0, 0)
	bottomHalf := t.termBuffer.SubImage(image.Rect(0, 180, screenWidth, screenHeight)).(*ebiten.Image)
	op := &ebiten.DrawImageOptions{}
	dst.DrawImage(bottomHalf, op)
}

// DrawFull renders the full terminal filling dst (y: 0..360).
func (t *TengoTerminal) DrawFull(dst *ebiten.Image, assets *Assets) {
	if dst == nil {
		return
	}

	t.renderTerminalBuffer(assets)
	if t.termBuffer == nil {
		return
	}

	op := &ebiten.DrawImageOptions{}
	dst.DrawImage(t.termBuffer, op)
}
