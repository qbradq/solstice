package solstice

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

func TestGameGetActorAndDamageActor(t *testing.T) {
	if _, err := PreloadActorDefs(); err != nil {
		t.Fatalf("PreloadActorDefs failed: %v", err)
	}
	if _, err := PreloadTileSet(); err != nil {
		t.Fatalf("PreloadTileSet failed: %v", err)
	}
	if _, err := PreloadItemDefs(); err != nil {
		t.Fatalf("PreloadItemDefs failed: %v", err)
	}
	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}

	homeMap, err := ReloadMap("home")
	if err != nil || homeMap == nil {
		homeMap, err = LoadMap("home")
		if err != nil {
			t.Fatalf("LoadMap failed: %v", err)
		}
	}
	SetMap(homeMap)

	hero, err := NewActorFromDef("hero1", "kevin", 5, 5)
	if err != nil {
		t.Fatalf("NewActorFromDef hero1 failed: %v", err)
	}
	party, err := NewParty(5, 5, *hero)
	if err != nil {
		t.Fatalf("NewParty failed: %v", err)
	}
	SetParty(party)

	enemy, err := NewActorFromDef("enemy1", "rodent", 6, 5)
	if err != nil {
		t.Fatalf("NewActorFromDef enemy1 failed: %v", err)
	}
	homeMap.AddActor(enemy)

	term := NewTerminal()
	SetTerminal(term)

	repl := NewTengoREPL()

	// Test game.get_actor on map actor (rodent has human: "false")
	err = repl.Execute(`
act := game.get_actor("enemy1")
game.log(act["name"])
game.log(string(act["level"]))
game.log(string(act["strength"]))
game.log(string(act["hit_points"]))
game.log(string(act["human"]))
game.log(string(act["range"]))
game.log(act["damage"])
`)
	if err != nil {
		t.Fatalf("REPL get_actor failed: %v", err)
	}

	lines := term.GetLineTexts()
	if len(lines) < 7 {
		t.Fatalf("Expected at least 7 lines, got %d: %v", len(lines), lines)
	}
	last7 := lines[len(lines)-7:]
	if last7[0] != "Rodent of Unusual Size" {
		t.Errorf("Expected 'Rodent of Unusual Size', got %q", last7[0])
	}
	if last7[1] != "1" {
		t.Errorf("Expected level '1', got %q", last7[1])
	}
	if last7[2] != "8" {
		t.Errorf("Expected strength '8', got %q", last7[2])
	}
	if last7[3] != "15" {
		t.Errorf("Expected hit points '15', got %q", last7[3])
	}
	if last7[4] != "false" {
		t.Errorf("Expected human 'false', got %q", last7[4])
	}
	if last7[5] != "1" {
		t.Errorf("Expected range '1', got %q", last7[5])
	}
	if last7[6] != "1d4+1" {
		t.Errorf("Expected damage '1d4+1', got %q", last7[6])
	}

	// Test hero (kevin) weapon range and damage (dagger: range 1, damage "1d4+2")
	if hero.Weapon == nil || hero.Weapon.Template != "dagger" {
		t.Errorf("Expected hero to equip dagger, got %v", hero.Weapon)
	}
	if hero.GetWeaponRange() != 1 {
		t.Errorf("Expected hero weapon range 1, got %d", hero.GetWeaponRange())
	}
	if hero.GetWeaponDamage() != "1d4+2" {
		t.Errorf("Expected hero weapon damage '1d4+2', got %q", hero.GetWeaponDamage())
	}

	// Test game.damage_actor damaging enemy
	err = repl.Execute(`
game.damage_actor("enemy1", 6)
act2 := game.get_actor("enemy1")
game.log(string(act2["hit_points"]))
`)
	if err != nil {
		t.Fatalf("REPL damage_actor failed: %v", err)
	}

	lines = term.GetLineTexts()
	if lines[len(lines)-1] != "9" {
		t.Errorf("Expected updated hit points '9', got %q", lines[len(lines)-1])
	}

	// Test game.damage_actor killing animal actor (enemy1 has human: false)
	err = repl.Execute(`
game.damage_actor("enemy1", 10)
`)
	if err != nil {
		t.Fatalf("REPL lethal damage_actor failed: %v", err)
	}

	// Verify enemy1 was removed from map
	if homeMap.GetActorByID("enemy1") != nil {
		t.Errorf("Expected enemy1 to be removed from map after death")
	}

	// Verify animal_corpse item spawned at (6, 5)
	animalCorpses := homeMap.FindItemsByTemplate("animal_corpse")
	if len(animalCorpses) == 0 {
		t.Errorf("Expected animal_corpse item to be present on map at (6, 5)")
	} else {
		found := false
		for _, it := range animalCorpses {
			if it != nil && it.X == 6 && it.Y == 5 {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected animal_corpse item at (6, 5), got %v", animalCorpses)
		}
	}

	// Test human actor death spawning human_corpse (guard has human: "true")
	guard, err := NewActorFromDef("guard1", "guard", 7, 5)
	if err != nil {
		t.Fatalf("NewActorFromDef guard1 failed: %v", err)
	}
	homeMap.AddActor(guard)

	err = repl.Execute(`
game.damage_actor("guard1", 100)
`)
	if err != nil {
		t.Fatalf("REPL lethal damage on human failed: %v", err)
	}

	if homeMap.GetActorByID("guard1") != nil {
		t.Errorf("Expected guard1 to be removed from map after death")
	}

	humanCorpses := homeMap.FindItemsByTemplate("human_corpse")
	if len(humanCorpses) == 0 {
		t.Errorf("Expected human_corpse item on map")
	} else {
		found := false
		for _, it := range humanCorpses {
			if it != nil && it.X == 7 && it.Y == 5 {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected human_corpse item at (7, 5), got %v", humanCorpses)
		}
	}

	// Test game.get_actor on non-existent actor returns undefined
	err = repl.Execute(`
none := game.get_actor("nonexistent")
if none == undefined {
    game.log("not_found")
}
`)
	if err != nil {
		t.Fatalf("REPL get_actor nonexistent failed: %v", err)
	}

	lines = term.GetLineTexts()
	if lines[len(lines)-1] != "not_found" {
		t.Errorf("Expected 'not_found', got %q", lines[len(lines)-1])
	}
}

func TestAttackEffectScriptExecution(t *testing.T) {
	if _, err := PreloadActorDefs(); err != nil {
		t.Fatalf("PreloadActorDefs failed: %v", err)
	}
	if _, err := PreloadTileSet(); err != nil {
		t.Fatalf("PreloadTileSet failed: %v", err)
	}
	if _, err := PreloadItemDefs(); err != nil {
		t.Fatalf("PreloadItemDefs failed: %v", err)
	}
	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}

	homeMap, err := ReloadMap("home")
	if err != nil || homeMap == nil {
		homeMap, err = LoadMap("home")
		if err != nil {
			t.Fatalf("LoadMap failed: %v", err)
		}
	}
	SetMap(homeMap)

	hero, err := NewActorFromDef("hero_atk", "kevin", 5, 5)
	if err != nil {
		t.Fatalf("NewActorFromDef hero failed: %v", err)
	}
	party, err := NewParty(5, 5, *hero)
	if err != nil {
		t.Fatalf("NewParty failed: %v", err)
	}
	SetParty(party)

	enemy, err := NewActorFromDef("enemy_target", "rodent", 6, 5)
	if err != nil {
		t.Fatalf("NewActorFromDef enemy failed: %v", err)
	}
	enemy.HitPoints = 50
	enemy.MaxHitPoints = 50
	homeMap.AddActor(enemy)

	term := NewTerminal()
	SetTerminal(term)

	// Execute attack.tengo
	err = ExecuteEffectScript("effects/attack.tengo", 6, 5, "enemy_target", "hero_atk")
	if err != nil {
		t.Fatalf("ExecuteEffectScript failed: %v", err)
	}

	lines := term.GetLineTexts()
	if len(lines) == 0 {
		t.Fatal("Expected attack message logged to terminal")
	}

	joined := strings.Join(lines, " ")
	if !strings.Contains(joined, "Kevin hits Rodent of Unusual Size for") && !strings.Contains(joined, "Kevin misses Rodent of Unusual Size!") {
		t.Errorf("Unexpected attack log message in %q", joined)
	}

	// If it hit, verify HP was reduced
	if strings.Contains(joined, "hits") {
		if enemy.HitPoints >= 50 {
			t.Errorf("Expected enemy HP to be reduced from 50 on hit, got %d", enemy.HitPoints)
		}
	}
}

func TestAttackCombatMoveTargeting(t *testing.T) {
	if _, err := PreloadActorDefs(); err != nil {
		t.Fatalf("PreloadActorDefs failed: %v", err)
	}
	if _, err := PreloadTileSet(); err != nil {
		t.Fatalf("PreloadTileSet failed: %v", err)
	}
	if _, err := PreloadItemDefs(); err != nil {
		t.Fatalf("PreloadItemDefs failed: %v", err)
	}
	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}

	homeMap, err := ReloadMap("home")
	if err != nil || homeMap == nil {
		homeMap, err = LoadMap("home")
		if err != nil {
			t.Fatalf("LoadMap('home') failed: %v", err)
		}
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
	weaponRange := hero.GetWeaponRange()
	attackTiles := make(map[image.Point]bool)
	for dx := -weaponRange; dx <= weaponRange; dx++ {
		for dy := -weaponRange; dy <= weaponRange; dy++ {
			adx := dx
			if adx < 0 {
				adx = -adx
			}
			ady := dy
			if ady < 0 {
				ady = -ady
			}
			if adx+ady <= weaponRange {
				tx := hero.X + dx
				ty := hero.Y + dy
				if tx >= 0 && tx < homeMap.Width && ty >= 0 && ty < homeMap.Height {
					if homeMap.GetActorAt(tx, ty) != nil {
						attackTiles[image.Pt(tx, ty)] = true
					}
				}
			}
		}
	}

	// Only hero tile (5, 5) and enemy tile (6, 5) have actors
	if attackTiles[image.Pt(5, 4)] || attackTiles[image.Pt(5, 6)] || attackTiles[image.Pt(4, 5)] {
		t.Errorf("Expected empty tiles not to be included in attackTiles")
	}
	if !attackTiles[image.Pt(6, 5)] {
		t.Errorf("Expected enemy tile (6, 5) to be in attackTiles")
	}

	tm := NewTargetMode(5, 5, weaponRange, DistanceDiamond, func(tx, ty int) bool {
		if !attackTiles[image.Pt(tx, ty)] {
			return false
		}
		act := homeMap.GetActorAt(tx, ty)
		if act == nil {
			return false
		}
		SetCombatMemberActed(true)
		_ = ExecuteEffectScript("effects/attack.tengo", tx, ty, act.ID, hero.ID)
		attackExecuted = true
		return true
	}, nil)

	tm.SetHighlightTiles(attackTiles, color.RGBA{R: 127, G: 0, B: 0, A: 31})

	if GetCombatMemberActed() {
		t.Errorf("Expected CombatMemberActed to be false before action")
	}

	// Invalid target on unoccupied adjacent tile (5, 4)
	if tm.onSelected(5, 4) {
		t.Errorf("Expected unoccupied target at (5, 4) to be rejected")
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

	// Verify terminal output from attack.tengo (hit or miss message)
	lines := term.GetLineTexts()
	if len(lines) == 0 {
		t.Fatal("Expected at least 1 terminal line from attack.tengo")
	}
	joined := strings.Join(lines, " ")
	if !strings.Contains(joined, "Kevin hits Rodent of Unusual Size for") && !strings.Contains(joined, "Kevin misses Rodent of Unusual Size!") {
		t.Errorf("Unexpected attack output in %q", joined)
	}
}
