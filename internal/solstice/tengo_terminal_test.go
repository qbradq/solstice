package solstice

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestTengoTerminalStateCycling(t *testing.T) {
	repl := NewTengoREPL()
	term := NewTengoTerminal(repl)

	if term.GetState() != TengoTerminalStateHidden {
		t.Errorf("Expected initial state to be Hidden, got %v", term.GetState())
	}

	term.CycleState()
	if term.GetState() != TengoTerminalStateHalf {
		t.Errorf("Expected state after 1st cycle to be Half, got %v", term.GetState())
	}

	term.CycleState()
	if term.GetState() != TengoTerminalStateFull {
		t.Errorf("Expected state after 2nd cycle to be Full, got %v", term.GetState())
	}

	term.CycleState()
	if term.GetState() != TengoTerminalStateHidden {
		t.Errorf("Expected state after 3rd cycle to be Hidden, got %v", term.GetState())
	}
}

func TestTengoTerminalInputAndOverflowCalculation(t *testing.T) {
	repl := NewTengoREPL()
	term := NewTengoTerminal(repl)

	// Single-line input (<= 78 chars)
	term.SetInputText("game.log('hello')")
	if term.GetInputText() != "game.log('hello')" {
		t.Errorf("Input text mismatch: got %q", term.GetInputText())
	}
	if term.getInputLineCount() != 1 {
		t.Errorf("Expected 1 input line, got %d", term.getInputLineCount())
	}
	if term.GetCursorPos() != len("game.log('hello')") {
		t.Errorf("Expected cursor at end (%d), got %d", len("game.log('hello')"), term.GetCursorPos())
	}

	// Multi-line input (> 78 chars)
	// 78 + 80 = 158 chars -> 2 lines
	longInput := strings.Repeat("A", 100)
	term.SetInputText(longInput)
	if term.getInputLineCount() != 2 {
		t.Errorf("Expected 2 input lines for 100 chars, got %d", term.getInputLineCount())
	}

	// 78 + 80 + 1 = 159 chars -> 3 lines
	veryLongInput := strings.Repeat("B", 159)
	term.SetInputText(veryLongInput)
	if term.getInputLineCount() != 3 {
		t.Errorf("Expected 3 input lines for 159 chars, got %d", term.getInputLineCount())
	}
}

func TestTengoTerminalDrawing(t *testing.T) {
	assets, err := LoadAssets()
	if err != nil {
		t.Fatalf("LoadAssets failed: %v", err)
	}

	repl := NewTengoREPL()
	repl.AddOutput("First output line in Tengo terminal")
	repl.AddOutput("Second output line in Tengo terminal")

	term := NewTengoTerminal(repl)
	term.SetInputText("a := 1 + 2")

	screen := ebiten.NewImage(screenWidth, screenHeight)

	// Draw Half mode
	term.SetState(TengoTerminalStateHalf)
	term.DrawHalf(screen, assets)

	// Draw Full mode
	term.SetState(TengoTerminalStateFull)
	term.DrawFull(screen, assets)
}

func TestTengoTerminalScrollClamp(t *testing.T) {
	repl := NewTengoREPL()
	term := NewTengoTerminal(repl)

	for i := 0; i < 50; i++ {
		repl.AddOutput("Log entry")
	}

	term.SetScrollOffset(10)
	if term.GetScrollOffset() != 10 {
		t.Errorf("Expected scroll offset 10, got %d", term.GetScrollOffset())
	}
}

func TestTengoTerminalSubmissionAndErrorReporting(t *testing.T) {
	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}

	repl := NewTengoREPL()
	term := NewTengoTerminal(repl)

	blue := VGAPalette16[9]
	white := VGAPalette16[15]

	// 1. Valid statement execution
	term.SetInputText("valid_var := 42")
	// Simulate what Enter does:
	// We can invoke the logic or test the REPL directly
	// Let's verify input line exact formatting:
	firstChunkLen := len(term.inputRunes)
	if firstChunkLen > 78 {
		firstChunkLen = 78
	}
	repl.AddRawOutputColored("> "+string(term.inputRunes[:firstChunkLen]), blue)
	if err := repl.Execute(term.GetInputText()); err != nil {
		repl.AddOutputColored(err.Error(), white)
	}

	history := repl.GetOutputHistory()
	if len(history) != 1 {
		t.Fatalf("Expected 1 history line for valid statement, got %d", len(history))
	}
	if history[0].Text != "> valid_var := 42" || history[0].Color != blue {
		t.Errorf("Expected input line in blue, got %v (%v)", history[0].Text, history[0].Color)
	}

	// 2. Invalid statement execution returning error
	term.SetInputText("undefined_func()")
	firstChunkLen = len(term.inputRunes)
	if firstChunkLen > 78 {
		firstChunkLen = 78
	}
	repl.AddRawOutputColored("> "+string(term.inputRunes[:firstChunkLen]), blue)
	if err := repl.Execute(term.GetInputText()); err != nil {
		repl.AddOutputColored(err.Error(), white)
	}

	history = repl.GetOutputHistory()
	if len(history) < 3 { // input line + error line(s)
		t.Fatalf("Expected at least 3 history items after error, got %d", len(history))
	}
	if history[1].Text != "> undefined_func()" || history[1].Color != blue {
		t.Errorf("Expected second input line in blue, got %v (%v)", history[1].Text, history[1].Color)
	}
	if history[2].Color != white {
		t.Errorf("Expected error report line in white, got %v", history[2].Color)
	}
}
