package solstice

import (
	"fmt"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestTerminalWordWrapAndLimit(t *testing.T) {
	term := &Terminal{}

	// Add a long message > 36 chars
	longMsg := "This is a very long line of text that exceeds thirty six characters and should be automatically word wrapped."
	term.AddMessage(longMsg)

	for _, line := range term.lines {
		if len(line) > 36 {
			t.Errorf("Line exceeds 36 characters: %q (len %d)", line, len(line))
		}
	}

	// Add 500 lines to test max history limit (300 lines)
	for i := 0; i < 500; i++ {
		term.AddMessage(fmt.Sprintf("Line number %d", i))
	}

	if len(term.lines) > 300 {
		t.Errorf("Terminal line history exceeds 300 lines: got %d", len(term.lines))
	}
}

func TestNewTerminalAndDraw(t *testing.T) {
	term := NewTerminal()
	term.AddMessage("Test message")

	assets, err := LoadAssets()
	if err != nil {
		t.Fatalf("LoadAssets failed: %v", err)
	}

	screen := ebiten.NewImage(640, 360)
	term.Draw(screen, assets)
}

func TestTerminalScrollBounds(t *testing.T) {
	term := NewTerminal()

	// Initial scroll offset 0
	if term.scrollOffset != 0 {
		t.Errorf("Initial scrollOffset should be 0, got %d", term.scrollOffset)
	}

	// Manual scroll test
	term.scrollOffset = 50
	maxScroll := len(term.lines) - 28
	if maxScroll < 0 {
		maxScroll = 0
	}

	// Clamp test
	term.scrollOffset = 1000
	term.HandleInput() // Should clamp scrollOffset to maxScroll
	if term.scrollOffset > maxScroll {
		t.Errorf("scrollOffset %d exceeds maxScroll %d", term.scrollOffset, maxScroll)
	}

	term.scrollOffset = -50
	term.HandleInput() // Should clamp scrollOffset to 0
	if term.scrollOffset < 0 {
		t.Errorf("scrollOffset should not be negative, got %d", term.scrollOffset)
	}
}
