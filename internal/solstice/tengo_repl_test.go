package solstice

import (
	"fmt"
	"strings"
	"testing"
)

func TestTengoREPLExecutionAndPersistence(t *testing.T) {
	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}

	repl := NewTengoREPL()
	if repl == nil {
		t.Fatal("Expected NewTengoREPL to return non-nil")
	}

	// 1. Execute variable definition
	if err := repl.Execute("a := 40"); err != nil {
		t.Fatalf("Execute 'a := 40' failed: %v", err)
	}

	// 2. Execute statement referencing previous variable
	if err := repl.Execute("b := a + 2"); err != nil {
		t.Fatalf("Execute 'b := a + 2' failed: %v", err)
	}

	// 3. Execute statement using game module (imported in autoexec.tengo)
	if err := repl.Execute(`game.set_flag("repl_test_flag")`); err != nil {
		t.Fatalf("Execute using game module failed: %v", err)
	}

	if !HasFlag("repl_test_flag") {
		t.Error("Expected repl_test_flag to be set via REPL game.set_flag")
	}
}

func TestTengoREPLAutoexecOnReset(t *testing.T) {
	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}

	repl := NewTengoREPL()

	// Define variable
	if err := repl.Execute("test_var := 123"); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Add output lines
	repl.AddOutput("Some output line")
	if len(repl.GetOutputHistory()) == 0 {
		t.Error("Expected output history to have 1 line")
	}

	// Reset REPL
	repl.Reset()

	// Output history must be cleared
	if len(repl.GetOutputHistory()) != 0 {
		t.Errorf("Expected output history to be cleared, got %d lines", len(repl.GetOutputHistory()))
	}

	// autoexec.tengo should have been re-run, so 'game' and 'cs' are available
	if err := repl.Execute(`game.set_flag("autoexec_reloaded")`); err != nil {
		t.Fatalf("Expected game module to be available after Reset: %v", err)
	}

	if !HasFlag("autoexec_reloaded") {
		t.Error("Expected autoexec_reloaded flag to be set")
	}
}

func TestTengoREPLCommandHistoryLimitAndPreservation(t *testing.T) {
	repl := NewTengoREPL()

	// Add 150 commands to verify 100-line cap
	for i := 0; i < 150; i++ {
		repl.AddCommand(fmt.Sprintf("cmd_%d", i))
	}

	history := repl.GetCommandHistory()
	if len(history) != 100 {
		t.Fatalf("Expected exactly 100 history lines, got %d", len(history))
	}

	if history[0] != "cmd_50" {
		t.Errorf("Expected oldest history line to be cmd_50, got %s", history[0])
	}
	if history[99] != "cmd_149" {
		t.Errorf("Expected newest history line to be cmd_149, got %s", history[99])
	}

	// Verify Reset() does NOT clear command history
	repl.Reset()
	historyAfterReset := repl.GetCommandHistory()
	if len(historyAfterReset) != 100 {
		t.Errorf("Expected command history to be preserved after Reset(), got %d lines", len(historyAfterReset))
	}
}

func TestTengoREPLOutputHistoryLimitAndWordWrap(t *testing.T) {
	repl := NewTengoREPL()

	// Word wrap test: add a line longer than 80 chars
	longText := strings.Repeat("word ", 20) // 100 chars
	repl.AddOutput(longText)

	lines := repl.GetOutputHistory()
	if len(lines) < 2 {
		t.Errorf("Expected long text to wrap into at least 2 lines, got %d", len(lines))
	}
	for _, l := range lines {
		if len(l.Text) > 80 {
			t.Errorf("Output line exceeds 80 columns: %q (len %d)", l.Text, len(l.Text))
		}
	}

	// 1000-line cap test
	repl.ClearOutputHistory()
	for i := 0; i < 1200; i++ {
		repl.AddOutput(fmt.Sprintf("Line %d", i))
	}

	outHistory := repl.GetOutputHistory()
	if len(outHistory) != 1000 {
		t.Fatalf("Expected output history capped at 1000 lines, got %d", len(outHistory))
	}
	if outHistory[0].Text != "Line 200" {
		t.Errorf("Expected oldest output line to be 'Line 200', got %q", outHistory[0].Text)
	}
	if outHistory[999].Text != "Line 1199" {
		t.Errorf("Expected newest output line to be 'Line 1199', got %q", outHistory[999].Text)
	}
}

func TestTengoREPLColoredOutput(t *testing.T) {
	repl := NewTengoREPL()

	blue := VGAPalette16[9]
	white := VGAPalette16[15]

	repl.AddRawOutputColored("> a := 10", blue)
	repl.AddOutputColored("Result is 10", white)

	history := repl.GetOutputHistory()
	if len(history) != 2 {
		t.Fatalf("Expected 2 history items, got %d", len(history))
	}

	if history[0].Text != "> a := 10" || history[0].Color != blue {
		t.Errorf("Expected first line to be '> a := 10' in blue, got %v with color %v", history[0].Text, history[0].Color)
	}

	if history[1].Text != "Result is 10" || history[1].Color != white {
		t.Errorf("Expected second line to be 'Result is 10' in white, got %v with color %v", history[1].Text, history[1].Color)
	}
}

func TestTengoREPLGlobalIdentifierSpecialCase(t *testing.T) {
	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}

	repl := NewTengoREPL()
	repl.ClearOutputHistory()

	// Define global variable
	if err := repl.Execute("my_var := 100"); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Output history should still be empty after variable assignment
	if len(repl.GetOutputHistory()) != 0 {
		t.Errorf("Expected output history to be empty after assignment, got %d lines", len(repl.GetOutputHistory()))
	}

	// Execute just the global name: "my_var"
	if err := repl.Execute("my_var"); err != nil {
		t.Fatalf("Execute global name failed: %v", err)
	}

	history := repl.GetOutputHistory()
	if len(history) != 1 {
		t.Fatalf("Expected 1 history line output, got %d", len(history))
	}
	if history[0].Text != "100" {
		t.Errorf("Expected output '100', got %q", history[0].Text)
	}
	if history[0].Color != VGAPalette16[15] {
		t.Errorf("Expected output color to be bright white, got %v", history[0].Color)
	}

	// Execute another global name (game module from autoexec)
	if err := repl.Execute("game"); err != nil {
		t.Fatalf("Execute 'game' failed: %v", err)
	}

	history = repl.GetOutputHistory()
	if len(history) <= 1 {
		t.Fatalf("Expected history lines after evaluating 'game', got %d", len(history))
	}
	if history[1].Text == "" {
		t.Errorf("Expected non-empty string representation for 'game', got %q", history[1].Text)
	}
}

func TestTengoFmtModuleInREPLAndScripts(t *testing.T) {
	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}

	repl := NewTengoREPL()
	repl.ClearOutputHistory()

	// 1. Test fmt.println in REPL (fmt imported in autoexec)
	if err := repl.Execute(`fmt.println("Hello", "Solstice", 2026)`); err != nil {
		t.Fatalf("Execute fmt.println failed: %v", err)
	}

	history := repl.GetOutputHistory()
	if len(history) != 1 {
		t.Fatalf("Expected 1 history line, got %d", len(history))
	}
	if history[0].Text != "Hello Solstice 2026" {
		t.Errorf("Expected 'Hello Solstice 2026', got %q", history[0].Text)
	}

	// 2. Test fmt.printf
	if err := repl.Execute(`fmt.printf("Value: %04d, Tag: %s", 7, "Alpha")`); err != nil {
		t.Fatalf("Execute fmt.printf failed: %v", err)
	}

	history = repl.GetOutputHistory()
	if len(history) != 2 {
		t.Fatalf("Expected 2 history lines, got %d", len(history))
	}
	if history[1].Text != "Value: 0007, Tag: Alpha" {
		t.Errorf("Expected 'Value: 0007, Tag: Alpha', got %q", history[1].Text)
	}

	// 3. Test fmt.print
	if err := repl.Execute(`fmt.print("A", "B", "C")`); err != nil {
		t.Fatalf("Execute fmt.print failed: %v", err)
	}

	history = repl.GetOutputHistory()
	if len(history) != 3 {
		t.Fatalf("Expected 3 history lines, got %d", len(history))
	}
	if history[2].Text != "ABC" {
		t.Errorf("Expected 'ABC', got %q", history[2].Text)
	}

	// 4. Test fmt.sprintf (does not print to output history, returns string)
	if err := repl.Execute(`formatted := fmt.sprintf("Score: %d", 9999)`); err != nil {
		t.Fatalf("Execute fmt.sprintf failed: %v", err)
	}

	if len(repl.GetOutputHistory()) != 3 {
		t.Errorf("fmt.sprintf should not append to output history, len is %d", len(repl.GetOutputHistory()))
	}

	// Inspect formatted global
	if err := repl.Execute(`formatted`); err != nil {
		t.Fatalf("Execute 'formatted' failed: %v", err)
	}

	history = repl.GetOutputHistory()
	if len(history) != 4 {
		t.Fatalf("Expected 4 history lines, got %d", len(history))
	}
	if history[3].Text != "\"Score: 9999\"" && history[3].Text != "Score: 9999" {
		t.Errorf("Expected 'Score: 9999', got %q", history[3].Text)
	}
}
