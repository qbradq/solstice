package solstice

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestDialogScriptExecution(t *testing.T) {
	term := NewTerminal()
	SetTerminal(term)

	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}

	// 1. Test "look" keyword on guard.tengo
	ended, err := ExecuteDialogScript("dialog/guard.tengo", "look")
	if err != nil {
		t.Fatalf("ExecuteDialogScript(look) failed: %v", err)
	}
	if ended {
		t.Error("Expected look keyword not to end dialog")
	}
	lines := term.GetLineTexts()
	if len(lines) == 0 {
		t.Error("Expected terminal lines after look keyword, got 0")
	}
	allText := strings.Join(lines, " ")
	if !strings.Contains(allText, "You see a tall figure clad in chain") {
		t.Errorf("Unexpected reply for look: %q", allText)
	}

	// 2. Test "name" keyword
	ended, err = ExecuteDialogScript("dialog/guard.tengo", "name")
	if err != nil {
		t.Fatalf("ExecuteDialogScript(name) failed: %v", err)
	}
	if ended {
		t.Error("Expected name keyword not to end dialog")
	}

	// 3. Test uppercase "JOB" keyword -> normalized to "job"
	ended, err = ExecuteDialogScript("dialog/guard.tengo", "JOB")
	if err != nil {
		t.Fatalf("ExecuteDialogScript(JOB) failed: %v", err)
	}
	lines = term.GetLineTexts()
	allText = strings.Join(lines, " ")
	if !strings.Contains(allText, "magistrate of this land") {
		t.Errorf("Unexpected reply for JOB: %q", allText)
	}

	// 4. Test "bye" keyword -> calls game.end_dialog() -> ended == true
	ended, err = ExecuteDialogScript("dialog/guard.tengo", "bye")
	if err != nil {
		t.Fatalf("ExecuteDialogScript(bye) failed: %v", err)
	}
	if !ended {
		t.Error("Expected bye keyword to call end_dialog() and return ended=true")
	}
	lines = term.GetLineTexts()
	allText = strings.Join(lines, " ")
	if !strings.Contains(allText, "Go in peace.") {
		t.Errorf("Unexpected reply for bye: %q", allText)
	}
}

func TestDialogModeStack(t *testing.T) {
	term := NewTerminal()
	SetTerminal(term)

	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}

	actor := NewActor("guard-1", 17, 11, "guard")
	actor.DialogScript = "dialog/guard.tengo"

	game := &Game{
		assets:   nil,
		terminal: term,
	}
	game.PushMode(NewMainMode())

	dialogMode := NewDialogMode(actor, actor.DialogScript)
	game.PushMode(dialogMode)

	if game.GetMode() != dialogMode {
		t.Error("Expected active mode to be dialogMode")
	}

	// First Update initializes DialogMode, sets inputMode=true, and runs "look"
	screen := ebiten.NewImage(640, 360)
	if err := game.Update(); err != nil {
		t.Fatalf("game.Update() failed: %v", err)
	}

	if !term.IsInputMode() {
		t.Error("Expected terminal to be in Input Mode during dialog")
	}

	// Test rendering in DialogMode
	game.Draw(screen)

	// Simulate bye keyword execution
	ended, _ := ExecuteDialogScript(actor.DialogScript, "bye")
	if !ended {
		t.Error("Expected bye to end dialog")
	}
}
