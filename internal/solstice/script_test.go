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

func TestExecuteTileScriptContext(t *testing.T) {
	term := NewTerminal()
	SetTerminal(term)

	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}

	if err := ExecuteTileScript("tiles/door.tengo", 5, 10, 78); err != nil {
		t.Fatalf("ExecuteTileScript failed: %v", err)
	}
}

func TestAddTimerAndTurnSystem(t *testing.T) {
	term := NewTerminal()
	SetTerminal(term)

	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}

	m, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}
	SetMap(m)

	// Add a 2-turn timer executing tiles/door.tengo with tile_x=5, tile_y=5, tile_idx=78
	m.SetTile(5, 5, 78)
	m.AddTimer(2, "tiles/door.tengo", map[string]interface{}{
		"tile_x":   5,
		"tile_y":   5,
		"tile_idx": 78,
	})

	if len(m.Timers) != 1 {
		t.Fatalf("Expected 1 active timer, got %d", len(m.Timers))
	}

	// Turn 1: timer remaining 1, not expired yet
	m.AdvanceTurn()
	if updatedTile := m.GetTile(5, 5); updatedTile != 78 {
		t.Errorf("Expected tile at (5, 5) to remain 78 on turn 1, got %d", updatedTile)
	}

	// Turn 2: timer expires, runs door.tengo (opens door to 68 and schedules close_door.tengo in 5 turns)
	m.AdvanceTurn()
	if updatedTile := m.GetTile(5, 5); updatedTile != 68 {
		t.Errorf("Expected tile at (5, 5) to change to 68 on turn 2 expiry, got %d", updatedTile)
	}

	// Verify that door.tengo added 1 new timer for close_door.tengo
	if len(m.Timers) != 1 {
		t.Fatalf("Expected 1 active close_door timer, got %d", len(m.Timers))
	}

	// Advance 5 more turns to trigger close_door.tengo
	for i := 0; i < 5; i++ {
		m.AdvanceTurn()
	}

	// Verify that close_door.tengo executed and reset tile to 184
	if updatedTile := m.GetTile(5, 5); updatedTile != 184 {
		t.Errorf("Expected tile at (5, 5) to close back to 184 after close_door timer, got %d", updatedTile)
	}

	if len(m.Timers) != 0 {
		t.Errorf("Expected 0 active timers after all timers expired, got %d", len(m.Timers))
	}
}
