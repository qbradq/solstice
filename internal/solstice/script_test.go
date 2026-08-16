package solstice

import "testing"

func TestInitScriptSystemAndRunMainScript(t *testing.T) {
	term := NewTerminal()
	SetTerminal(term)

	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}

	if err := RunMainScript(); err != nil {
		t.Fatalf("RunMainScript failed: %v", err)
	}

	// Verify that main.tengo logged welcome messages to the terminal
	if len(term.lines) == 0 {
		t.Error("Expected main.tengo to log messages to terminal, got 0 lines")
	}

	foundSolstice := false
	for _, l := range term.lines {
		if l == "Solstice Client v0.1.0" {
			foundSolstice = true
			break
		}
	}

	if !foundSolstice {
		t.Errorf("Expected 'Solstice Client v0.1.0' in terminal lines, got lines: %v", term.lines)
	}
}
