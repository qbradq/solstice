package solstice

import (
	"fmt"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestTerminalWordWrapAndLimit(t *testing.T) {
	term := NewTerminal()

	// Add a long message > 36 chars
	longMsg := "This is a very long line of text that exceeds thirty six characters and should be automatically word wrapped."
	term.AddMessage(longMsg)

	for _, line := range term.GetLines() {
		if len(line.Text) > 36 {
			t.Errorf("Line exceeds 36 characters: %q (len %d)", line.Text, len(line.Text))
		}
	}

	// Add 500 lines to test max history limit (300 lines)
	for i := 0; i < 500; i++ {
		term.AddMessage(fmt.Sprintf("Line number %d", i))
	}

	if len(term.GetLines()) > 300 {
		t.Errorf("Terminal line history exceeds 300 lines: got %d", len(term.GetLines()))
	}
}

func TestNewTerminalAndDraw(t *testing.T) {
	term := NewTerminal()
	term.AddMessage("Test message")
	term.AddMessageColored("> TEST INPUT", VGAPalette16[9])

	assets, err := LoadAssets()
	if err != nil {
		t.Fatalf("LoadAssets failed: %v", err)
	}

	screen := ebiten.NewImage(640, 360)
	term.Draw(screen, assets)

	// Draw input mode as well
	term.SetInputMode(true)
	term.Draw(screen, assets)
}

func TestTerminalInputModeEmptyOnStart(t *testing.T) {
	term := NewTerminal()
	term.SetInputMode(true)

	if term.inputText != "" {
		t.Errorf("Expected initial inputText to be empty string, got %q", term.inputText)
	}

	// First frame after enabling input mode ignores trigger key presses
	_, submitted, _ := term.HandleInputMode()
	if submitted {
		t.Error("Expected no submit on initial frame")
	}

	if term.inputText != "" {
		t.Errorf("Expected inputText to remain empty on initial frame, got %q", term.inputText)
	}
}

func TestTerminalScrollBounds(t *testing.T) {
	term := NewTerminal()

	// Initial scroll offset 0
	if term.scrollOffset != 0 {
		t.Errorf("Initial scrollOffset should be 0, got %d", term.scrollOffset)
	}

	// Manual scroll test
	term.scrollOffset = 50
	maxScroll := len(term.GetLines()) - 28
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

func TestTerminalSaveAndRestore(t *testing.T) {
	term := NewTerminal()
	term.AddMessage("First log line")
	term.AddMessageColored("> COMMAND", VGAPalette16[9])

	saved := term.GetSavedLines()
	if len(saved) != 2 {
		t.Fatalf("Expected 2 saved lines, got %d", len(saved))
	}
	if saved[0].Text != "First log line" || saved[0].Color != nil {
		t.Errorf("Expected line 0 without color, got %+v", saved[0])
	}
	if saved[1].Text != "> COMMAND" || saved[1].Color == nil {
		t.Errorf("Expected line 1 with color, got %+v", saved[1])
	}

	term2 := NewTerminal()
	term2.RestoreSavedLines(saved)

	lines2 := term2.GetLines()
	if len(lines2) != 2 {
		t.Fatalf("Expected 2 restored lines, got %d", len(lines2))
	}
	if lines2[0].Text != "First log line" {
		t.Errorf("Expected text 'First log line', got %q", lines2[0].Text)
	}
	if lines2[1].Text != "> COMMAND" {
		t.Errorf("Expected text '> COMMAND', got %q", lines2[1].Text)
	}
	if lines2[1].Color == nil {
		t.Errorf("Expected non-nil color for line 1")
	}
}
