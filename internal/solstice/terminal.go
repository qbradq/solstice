package solstice

import (
	"image"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/mitchellh/go-wordwrap"
)

const (
	terminalWrapWidth   = 36
	terminalMaxHistory  = 300
	terminalVisibleRows = 28 // 224 / 8 = 28 lines
)

// Terminal manages the terminal UI, log message history, input scrolling, and rendering.
type Terminal struct {
	lines        []string
	scrollOffset int
}

// NewTerminal creates a new Terminal initialized with a copyright/license welcome notice and 150+ lines of Lipsum text.
func NewTerminal() *Terminal {
	t := &Terminal{
		lines: make([]string, 0, terminalMaxHistory),
	}

	// Welcome message including copyright and license notice
	t.AddMessage("------------------------------------")
	t.AddMessage("Solstice Client v0.1.0")
	t.AddMessage("Copyright (c) 2026")
	t.AddMessage("Norman B. Lancaster")
	t.AddMessage("Licensed under the MIT License.")
	t.AddMessage("------------------------------------")

	return t
}

// AddMessage adds a message to the terminal log.
// Messages are word-wrapped to 36 characters and up to 300 lines of history are retained.
func (t *Terminal) AddMessage(msg string) {
	wrapped := wordwrap.WrapString(msg, terminalWrapWidth)
	newLines := strings.Split(wrapped, "\n")
	t.lines = append(t.lines, newLines...)

	if len(t.lines) > terminalMaxHistory {
		t.lines = t.lines[len(t.lines)-terminalMaxHistory:]
	}
}

// HandleInput handles keyboard input for terminal UI (Page Up and Page Down scrolling).
func (t *Terminal) HandleInput() {
	if inpututil.IsKeyJustPressed(ebiten.KeyPageUp) {
		t.scrollOffset += 20
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyPageDown) {
		t.scrollOffset -= 20
	}

	maxScroll := len(t.lines) - terminalVisibleRows
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

// Draw renders the terminal log to screen at position (352, 128) with size 288x224 pixels.
func (t *Terminal) Draw(dst *ebiten.Image, assets *Assets) {
	if assets == nil || dst == nil {
		return
	}

	termArea := dst.SubImage(image.Rect(352, 128, 352+288, 128+224)).(*ebiten.Image)

	N := len(t.lines)
	if N == 0 {
		return
	}

	bottomLineIdx := (N - 1) - t.scrollOffset
	for r := 0; r < terminalVisibleRows; r++ {
		lineIdx := bottomLineIdx - ((terminalVisibleRows - 1) - r)
		if lineIdx >= 0 && lineIdx < N {
			// cellX = 352 / 8 = 44, cellY = 128 / 8 + r = 16 + r
			assets.DrawString8x8(termArea, t.lines[lineIdx], 44, 16+r)
		}
	}
}
