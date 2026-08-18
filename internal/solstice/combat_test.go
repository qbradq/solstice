package solstice

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/d5/tengo/v2"
)

func TestCombatModeTransitions(t *testing.T) {
	// Create a party with 2 members at (10, 15)
	m1, err := NewActorFromDef("hero1", "kevin", 0, 0)
	if err != nil {
		t.Fatalf("NewActorFromDef hero1 failed: %v", err)
	}
	m2, err := NewActorFromDef("hero2", "wizard", 0, 0)
	if err != nil {
		t.Fatalf("NewActorFromDef hero2 failed: %v", err)
	}

	party, err := NewParty(10, 15, *m1, *m2)
	if err != nil {
		t.Fatalf("NewParty failed: %v", err)
	}
	SetParty(party)

	homeMap, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}
	SetMap(homeMap)

	// Ensure initially not in combat
	SetInCombat(false)
	SetCombatMemberIndex(0)

	// 1. Transition to Combat Mode
	StartCombat()
	if !IsInCombat() {
		t.Fatal("Expected IsInCombat to be true after StartCombat")
	}
	if GetCombatMemberIndex() != 0 {
		t.Errorf("Expected current combat member index 0, got %d", GetCombatMemberIndex())
	}
	if party.Members[0].X != 10 || party.Members[0].Y != 15 {
		t.Errorf("Expected member 0 at (10, 15), got (%d, %d)", party.Members[0].X, party.Members[0].Y)
	}
	if party.Members[1].X != 10 || party.Members[1].Y != 15 {
		t.Errorf("Expected member 1 at (10, 15), got (%d, %d)", party.Members[1].X, party.Members[1].Y)
	}

	// Verify party members were added to the map
	if homeMap.GetActorByID("hero1") == nil {
		t.Errorf("Expected hero1 to be added to map actors during combat")
	}
	if homeMap.GetActorByID("hero2") == nil {
		t.Errorf("Expected hero2 to be added to map actors during combat")
	}

	// Move member 0 to (12, 18) and member 1 to (14, 19)
	party.Members[0].X = 12
	party.Members[0].Y = 18
	party.Members[1].X = 14
	party.Members[1].Y = 19

	// 2. Transition back to Party Mode
	StopCombat()
	if IsInCombat() {
		t.Fatal("Expected IsInCombat to be false after StopCombat")
	}
	// Party position should be set to first party member's location (12, 18)
	if party.X != 12 || party.Y != 18 {
		t.Errorf("Expected party position (12, 18) after StopCombat, got (%d, %d)", party.X, party.Y)
	}

	// Verify party members were removed from the map
	if homeMap.GetActorByID("hero1") != nil {
		t.Errorf("Expected hero1 to be removed from map actors after StopCombat")
	}
	if homeMap.GetActorByID("hero2") != nil {
		t.Errorf("Expected hero2 to be removed from map actors after StopCombat")
	}
}

func TestCombatTurnAdvancement(t *testing.T) {
	homeMap, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}
	SetMap(homeMap)

	m1, _ := NewActorFromDef("hero1", "kevin", 0, 0)
	m2, _ := NewActorFromDef("hero2", "wizard", 0, 0)
	m3, _ := NewActorFromDef("hero3", "guard", 0, 0)

	party, err := NewParty(5, 5, *m1, *m2, *m3)
	if err != nil {
		t.Fatalf("NewParty failed: %v", err)
	}
	SetParty(party)

	game := &Game{
		currentMap: homeMap,
		party:      party,
	}

	StartCombat()
	initialTurn := homeMap.Turn

	// Member 0 takes a move
	if GetCombatMemberIndex() != 0 {
		t.Fatalf("Expected member index 0, got %d", GetCombatMemberIndex())
	}
	AdvanceCombatMember(game)

	// Member 1 takes a move
	if GetCombatMemberIndex() != 1 {
		t.Fatalf("Expected member index 1, got %d", GetCombatMemberIndex())
	}
	AdvanceCombatMember(game)

	// Member 2 takes a move
	if GetCombatMemberIndex() != 2 {
		t.Fatalf("Expected member index 2, got %d", GetCombatMemberIndex())
	}
	if homeMap.Turn != initialTurn {
		t.Errorf("Map turn should not advance before all members move")
	}

	// Final member moves -> AI runs, map turn advances, member index resets to 0
	AdvanceCombatMember(game)
	if GetCombatMemberIndex() != 0 {
		t.Errorf("Expected member index to reset to 0, got %d", GetCombatMemberIndex())
	}
	if homeMap.Turn != initialTurn+1 {
		t.Errorf("Expected map turn to advance from %d to %d, got %d", initialTurn, initialTurn+1, homeMap.Turn)
	}
}

func TestCombatDrawing(t *testing.T) {
	assets, err := LoadAssets()
	if err != nil {
		t.Fatalf("LoadAssets failed: %v", err)
	}
	homeMap, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}

	m1, _ := NewActorFromDef("hero1", "kevin", 0, 0)
	m2, _ := NewActorFromDef("hero2", "wizard", 0, 0)
	party, err := NewParty(15, 15, *m1, *m2)
	if err != nil {
		t.Fatalf("NewParty failed: %v", err)
	}
	SetParty(party)

	StartCombat()
	defer StopCombat()

	// Party members at different positions
	party.Members[0].X = 15
	party.Members[0].Y = 15
	party.Members[1].X = 16
	party.Members[1].Y = 15

	screen := ebiten.NewImage(640, 360)
	// Render at scale 2 and scale 1 in combat mode (active member 0 at 15, 15)
	SetCombatMemberIndex(0)
	homeMap.DrawCentered(screen, assets, party, 2)
	homeMap.DrawCentered(screen, assets, party, 1)

	// Advance to member 1 (at 16, 15) and render again
	SetCombatMemberIndex(1)
	homeMap.DrawCentered(screen, assets, party, 2)
	homeMap.DrawCentered(screen, assets, party, 1)

	// Test TargetMode draws centered on cursor position (e.g. cursor at 18, 17)
	targetMode := NewTargetMode(16, 15, 5, DistanceDiamond, nil, nil)
	targetMode.cursorX = 18
	targetMode.cursorY = 17
	game := &Game{
		assets:     assets,
		terminal:   NewTerminal(),
		currentMap: homeMap,
		party:      party,
		mapScale:   2,
	}
	targetMode.Draw(game, screen)
	game.mapScale = 1
	targetMode.Draw(game, screen)
}

func TestCombatTengoScriptFunctions(t *testing.T) {
	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}

	m1, _ := NewActorFromDef("hero1", "kevin", 0, 0)
	party, _ := NewParty(20, 25, *m1)
	SetParty(party)

	StopCombat()

	scriptSrc := `
game := import("game")
game.start_combat()
`
	s := tengo.NewScript([]byte(scriptSrc))
	s.SetImports(moduleMap)
	c, err := s.Compile()
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}
	if err := c.Run(); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if !IsInCombat() {
		t.Fatal("Expected IsInCombat to be true after game.start_combat()")
	}

	// Move member 0 during combat
	party.Members[0].X = 22
	party.Members[0].Y = 28

	scriptSrc2 := `
game := import("game")
game.stop_combat()
`
	s2 := tengo.NewScript([]byte(scriptSrc2))
	s2.SetImports(moduleMap)
	c2, err := s2.Compile()
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}
	if err := c2.Run(); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if IsInCombat() {
		t.Fatal("Expected IsInCombat to be false after game.stop_combat()")
	}
	if party.X != 22 || party.Y != 28 {
		t.Errorf("Expected party position (22, 28), got (%d, %d)", party.X, party.Y)
	}
}
