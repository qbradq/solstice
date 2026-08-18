package solstice

import (
	"testing"
)

func TestEnemyPacksLoadingAndMethods(t *testing.T) {
	packs, err := PreloadEnemyPacks()
	if err != nil {
		t.Fatalf("PreloadEnemyPacks failed: %v", err)
	}

	if len(packs) == 0 {
		t.Fatal("Expected at least 1 enemy pack loaded from data/json/packs.json, got 0")
	}

	rodents, ok := GetEnemyPack("rodents")
	if !ok {
		t.Fatal("Expected 'rodents' enemy pack to exist")
	}

	if rodents.Level != 1 {
		t.Errorf("Expected level 1, got %d", rodents.Level)
	}
	if rodents.BonusXP != 200 {
		t.Errorf("Expected bonus_xp 200, got %d", rodents.BonusXP)
	}
	if rodents.NumEnemies != "1d6+1" {
		t.Errorf("Expected num_enemies '1d6+1', got %q", rodents.NumEnemies)
	}
	if len(rodents.Enemies) != 1 || rodents.Enemies[0] != "rodent" {
		t.Errorf("Expected enemies ['rodent'], got %v", rodents.Enemies)
	}

	// Test RollNumEnemies range clamp (1 to 8)
	for i := 0; i < 50; i++ {
		num, err := rodents.RollNumEnemies()
		if err != nil {
			t.Fatalf("RollNumEnemies failed: %v", err)
		}
		if num < 1 || num > 8 {
			t.Errorf("Expected rolled num enemies in range [1, 8], got %d", num)
		}
	}

	// Test clamp with high formula
	highPack := EnemyPack{NumEnemies: "10d10+50"}
	numHigh, err := highPack.RollNumEnemies()
	if err != nil {
		t.Fatalf("RollNumEnemies high failed: %v", err)
	}
	if numHigh != 8 {
		t.Errorf("Expected high roll clamped to 8, got %d", numHigh)
	}

	// Test clamp with low formula
	lowPack := EnemyPack{NumEnemies: "1d1-10"}
	numLow, err := lowPack.RollNumEnemies()
	if err != nil {
		t.Fatalf("RollNumEnemies low failed: %v", err)
	}
	if numLow != 1 {
		t.Errorf("Expected low roll clamped to 1, got %d", numLow)
	}

	// Test ChooseEnemy
	enemy := rodents.ChooseEnemy()
	if enemy != "rodent" {
		t.Errorf("Expected ChooseEnemy to return 'rodent', got %q", enemy)
	}

	multiPack := EnemyPack{
		Enemies: []string{"a", "b"},
	}
	counts := make(map[string]int)
	for i := 0; i < 100; i++ {
		counts[multiPack.ChooseEnemy()]++
	}
	if counts["a"] == 0 || counts["b"] == 0 {
		t.Errorf("Expected both templates to be chosen over 100 iterations, got %v", counts)
	}

	// Test GetAllEnemyPacks
	all := GetAllEnemyPacks()
	if len(all) != len(packs) {
		t.Errorf("Expected GetAllEnemyPacks len %d, got %d", len(packs), len(all))
	}
}
