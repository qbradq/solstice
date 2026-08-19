package solstice

import (
	"image"
	"image/color"
	"testing"
)

func TestEffectScriptExecution(t *testing.T) {
	if _, err := PreloadActorDefs(); err != nil {
		t.Fatalf("PreloadActorDefs failed: %v", err)
	}
	if _, err := PreloadTileSet(); err != nil {
		t.Fatalf("PreloadTileSet failed: %v", err)
	}
	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}

	term := NewTerminal()
	SetTerminal(term)

	// Execute attack.tengo directly
	err := ExecuteEffectScript("effects/attack.tengo", 10, 15, "rat1", "hero1")
	if err != nil {
		t.Fatalf("ExecuteEffectScript failed: %v", err)
	}

	// Verify terminal output received the 4 game.log lines from attack.tengo
	lines := term.GetLines()
	if len(lines) < 4 {
		t.Fatalf("Expected at least 4 terminal lines, got %d", len(lines))
	}

	last4 := lines[len(lines)-4:]
	if last4[0].Text != "rat1" {
		t.Errorf("Expected target_id 'rat1', got %q", last4[0].Text)
	}
	if last4[1].Text != "10" {
		t.Errorf("Expected target_x '10', got %q", last4[1].Text)
	}
	if last4[2].Text != "15" {
		t.Errorf("Expected target_y '15', got %q", last4[2].Text)
	}
	if last4[3].Text != "hero1" {
		t.Errorf("Expected source_id 'hero1', got %q", last4[3].Text)
	}
}

func TestGameEffectOnTargetAndEffectAt(t *testing.T) {
	if _, err := PreloadActorDefs(); err != nil {
		t.Fatalf("PreloadActorDefs failed: %v", err)
	}
	if _, err := PreloadTileSet(); err != nil {
		t.Fatalf("PreloadTileSet failed: %v", err)
	}
	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}

	homeMap, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap('home') failed: %v", err)
	}
	SetMap(homeMap)

	actor, err := NewActorFromDef("enemy1", "rodent", 7, 8)
	if err != nil {
		t.Fatalf("NewActorFromDef failed: %v", err)
	}
	homeMap.AddActor(actor)

	term := NewTerminal()
	SetTerminal(term)

	// Test game.effect_on_target via REPL (game is pre-imported in autoexec)
	repl := NewTengoREPL()
	err = repl.Execute(`game.effect_on_target("effects/attack.tengo", "enemy1", "player1")`)
	if err != nil {
		t.Fatalf("REPL execute game.effect_on_target failed: %v", err)
	}

	lines := term.GetLines()
	if len(lines) < 4 {
		t.Fatalf("Expected at least 4 terminal lines, got %d", len(lines))
	}
	last4 := lines[len(lines)-4:]
	if last4[0].Text != "enemy1" {
		t.Errorf("Expected target_id 'enemy1', got %q", last4[0].Text)
	}
	if last4[1].Text != "7" {
		t.Errorf("Expected target_x '7', got %q", last4[1].Text)
	}
	if last4[2].Text != "8" {
		t.Errorf("Expected target_y '8', got %q", last4[2].Text)
	}
	if last4[3].Text != "player1" {
		t.Errorf("Expected source_id 'player1', got %q", last4[3].Text)
	}

	// Test game.effect_at via REPL
	err = repl.Execute(`game.effect_at("effects/attack.tengo", 20, 30, "caster1")`)
	if err != nil {
		t.Fatalf("REPL execute game.effect_at failed: %v", err)
	}

	lines = term.GetLines()
	if len(lines) < 8 {
		t.Fatalf("Expected at least 8 terminal lines, got %d", len(lines))
	}
	last4 = lines[len(lines)-4:]
	if last4[0].Text != "" {
		t.Errorf("Expected empty target_id, got %q", last4[0].Text)
	}
	if last4[1].Text != "20" {
		t.Errorf("Expected target_x '20', got %q", last4[1].Text)
	}
	if last4[2].Text != "30" {
		t.Errorf("Expected target_y '30', got %q", last4[2].Text)
	}
	if last4[3].Text != "caster1" {
		t.Errorf("Expected source_id 'caster1', got %q", last4[3].Text)
	}
}

func TestAttackCombatMoveTargeting(t *testing.T) {
	if _, err := PreloadActorDefs(); err != nil {
		t.Fatalf("PreloadActorDefs failed: %v", err)
	}
	if _, err := PreloadTileSet(); err != nil {
		t.Fatalf("PreloadTileSet failed: %v", err)
	}
	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}

	homeMap, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap('home') failed: %v", err)
	}
	SetMap(homeMap)

	hero, err := NewActorFromDef("hero1", "kevin", 5, 5)
	if err != nil {
		t.Fatalf("NewActorFromDef hero1 failed: %v", err)
	}
	enemy, err := NewActorFromDef("target_orc", "rodent", 6, 5)
	if err != nil {
		t.Fatalf("NewActorFromDef target_orc failed: %v", err)
	}
	homeMap.AddActor(enemy)

	party, err := NewParty(5, 5, *hero)
	if err != nil {
		t.Fatalf("NewParty failed: %v", err)
	}
	SetParty(party)

	StartCombat()
	defer StopCombat()

	term := NewTerminal()
	SetTerminal(term)

	attackExecuted := false
	attackTiles := map[image.Point]bool{
		{X: 5, Y: 4}: true,
		{X: 5, Y: 6}: true,
		{X: 4, Y: 5}: true,
		{X: 6, Y: 5}: true,
		{X: 5, Y: 5}: true,
	}

	tm := NewTargetMode(5, 5, 1, DistanceDiamond, func(tx, ty int) bool {
		if !attackTiles[image.Pt(tx, ty)] {
			return false
		}
		SetCombatMemberActed(true)
		targetID := ""
		if act := homeMap.GetActorAt(tx, ty); act != nil {
			targetID = act.ID
		}
		_ = ExecuteEffectScript("effects/attack.tengo", tx, ty, targetID, hero.ID)
		attackExecuted = true
		return true
	}, nil)

	tm.SetHighlightTiles(attackTiles, color.RGBA{R: 127, G: 0, B: 0, A: 31})

	if GetCombatMemberActed() {
		t.Errorf("Expected CombatMemberActed to be false before action")
	}

	// Invalid target beyond range
	if tm.onSelected(8, 8) {
		t.Errorf("Expected target at (8, 8) to be rejected")
	}
	if GetCombatMemberActed() {
		t.Errorf("Expected CombatMemberActed to remain false after rejected target")
	}

	// Valid target at enemy position (6, 5)
	if !tm.onSelected(6, 5) {
		t.Errorf("Expected target at (6, 5) to be accepted")
	}
	if !attackExecuted {
		t.Errorf("Expected attack effect script to execute")
	}
	if !GetCombatMemberActed() {
		t.Errorf("Expected CombatMemberActed to be true after successful attack")
	}

	// Verify terminal output from attack.tengo
	lines := term.GetLines()
	if len(lines) < 4 {
		t.Fatalf("Expected at least 4 terminal lines, got %d", len(lines))
	}
	last4 := lines[len(lines)-4:]
	if last4[0].Text != "target_orc" {
		t.Errorf("Expected target_id 'target_orc', got %q", last4[0].Text)
	}
	if last4[1].Text != "6" {
		t.Errorf("Expected target_x '6', got %q", last4[1].Text)
	}
	if last4[2].Text != "5" {
		t.Errorf("Expected target_y '5', got %q", last4[2].Text)
	}
	if last4[3].Text != "hero1" {
		t.Errorf("Expected source_id 'hero1', got %q", last4[3].Text)
	}
}
