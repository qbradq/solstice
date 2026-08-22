package solstice

import (
	"image"
	"testing"
)

func TestPathfindingAndReachableTiles(t *testing.T) {
	if _, err := PreloadActorDefs(); err != nil {
		t.Fatalf("PreloadActorDefs failed: %v", err)
	}
	if _, err := PreloadTileSet(); err != nil {
		t.Fatalf("PreloadTileSet failed: %v", err)
	}

	// Create a 10x10 walkable map (all grass tiles with ID 4)
	tiles := make([]int, 100)
	for i := range tiles {
		tiles[i] = 4 // grass (walkable)
	}
	m := &Map{
		Name:       "test_map",
		Width:      10,
		Height:     10,
		TileWidth:  16,
		TileHeight: 16,
		Tiles:      tiles,
	}

	// 1. Basic reachability for party member with move limit 3 from (5, 5)
	reachable := FindReachableTiles(m, 5, 5, 3, true)
	if reachable[image.Pt(5, 5)] {
		t.Error("Starting tile should not be in reachable destinations")
	}
	if !reachable[image.Pt(5, 4)] || !reachable[image.Pt(5, 6)] || !reachable[image.Pt(4, 5)] || !reachable[image.Pt(6, 5)] {
		t.Error("Adjacent tiles should be reachable")
	}
	if !reachable[image.Pt(5, 2)] || !reachable[image.Pt(5, 8)] {
		t.Error("Distance 3 straight tiles should be reachable")
	}
	if reachable[image.Pt(5, 1)] || reachable[image.Pt(5, 9)] {
		t.Error("Distance 4 tiles should not be reachable with move limit 3")
	}

	// 2. Pathfinding basic straight path
	path := FindPath(m, 5, 5, 5, 2, true)
	if len(path) != 3 {
		t.Fatalf("Expected 3 steps, got %d: %v", len(path), path)
	}
	for _, dir := range path {
		if dir != "north" {
			t.Errorf("Expected 'north', got %s", dir)
		}
	}

	// 3. Wall obstacle: place wall at (5, 4)
	m.SetTile(5, 4, 13) // wall (non-walkable)
	pathWithWall := FindPath(m, 5, 5, 5, 3, true)
	if len(pathWithWall) != 4 { // Goes around wall: e.g. west, north, north, east
		t.Fatalf("Expected 4 steps around wall, got %d: %v", len(pathWithWall), pathWithWall)
	}

	// 4. Party members passing through other party members but not stopping on them
	kevin, _ := NewActorFromDef("kevin", "kevin", 5, 5)
	lillian, _ := NewActorFromDef("lillian", "lillian", 4, 5)
	party, err := NewParty(5, 5, *kevin, *lillian)
	if err != nil {
		t.Fatalf("NewParty failed: %v", err)
	}
	SetParty(party)

	m.AddActor(&party.Members[0]) // kevin at 5,5
	m.AddActor(&party.Members[1]) // lillian at 4,5

	// Kevin moving west: (4,5) is occupied by Lillian.
	// Lillian's tile (4,5) should NOT be a valid destination.
	reachableKevin := FindReachableTiles(m, 5, 5, 3, true)
	if reachableKevin[image.Pt(4, 5)] {
		t.Error("Tile occupied by party member Lillian (4, 5) should NOT be reachable as a stopping destination")
	}
	// But tile (3, 5) beyond Lillian SHOULD be reachable (Kevin passes through Lillian)
	if !reachableKevin[image.Pt(3, 5)] {
		t.Error("Tile (3, 5) beyond party member Lillian SHOULD be reachable")
	}

	pathThroughLillian := FindPath(m, 5, 5, 3, 5, true)
	if len(pathThroughLillian) != 2 || pathThroughLillian[0] != "west" || pathThroughLillian[1] != "west" {
		t.Errorf("Expected ['west', 'west'] through Lillian, got %v", pathThroughLillian)
	}

	// 5. Enemy blocking movement completely
	rodent, _ := NewActorFromDef("rodent", "rodent", 6, 5)
	m.AddActor(rodent)

	reachableWithEnemy := FindReachableTiles(m, 5, 5, 3, true)
	if reachableWithEnemy[image.Pt(6, 5)] {
		t.Error("Tile with enemy (6, 5) should not be reachable")
	}
	if reachableWithEnemy[image.Pt(7, 5)] {
		// (7, 5) requires moving around the enemy or 2 steps via (5,6)->(6,6)->(7,6)->(7,5) (4 steps, > 3)
		// Or (5,4 is wall) so (5,6)->(6,6)->(7,6)->(7,5) is 4 steps > 3.
		t.Error("Tile (7, 5) should not be reachable in 3 steps when enemy blocks (6, 5)")
	}
}

func TestTargetModeUnlimitedRangeAndCallbacks(t *testing.T) {
	selected := false
	tm := NewTargetMode(5, 5, 0, DistanceDiamond, func(tx, ty int) bool {
		if tx == 10 && ty == 10 {
			selected = true
			return true
		}
		return false
	}, nil)

	if tm.maxRange != 0 {
		t.Errorf("Expected maxRange 0, got %d", tm.maxRange)
	}

	// Test highlight tiles setting
	highlights := map[image.Point]bool{
		{X: 5, Y: 4}: true,
		{X: 5, Y: 6}: true,
	}
	tm.SetHighlightTiles(highlights, nil)
	if len(tm.highlightTiles) != 2 {
		t.Errorf("Expected 2 highlight tiles, got %d", len(tm.highlightTiles))
	}

	// Test callback validation
	if tm.onSelected(1, 1) {
		t.Error("Expected onSelected(1, 1) to return false (invalid target)")
	}
	if !tm.onSelected(10, 10) {
		t.Error("Expected onSelected(10, 10) to return true (valid target)")
	}
	if !selected {
		t.Error("Expected selected flag to be true")
	}
}

func TestFindPathToClosestString(t *testing.T) {
	if _, err := PreloadTileSet(); err != nil {
		t.Fatalf("PreloadTileSet failed: %v", err)
	}

	// 10x10 grass map
	tiles := make([]int, 100)
	for i := range tiles {
		tiles[i] = 4 // walkable
	}
	m := &Map{
		Name:       "test_path_map",
		Width:      10,
		Height:     10,
		TileWidth:  16,
		TileHeight: 16,
		Tiles:      tiles,
	}

	// 1. Direct path within maxMoves
	p1 := FindPathToClosestString(m, 5, 5, 5, 2, 4, false)
	if p1 != "nnn" {
		t.Errorf("Expected 'nnn', got %q", p1)
	}

	// 2. Direct path exceeding maxMoves (truncated)
	p2 := FindPathToClosestString(m, 5, 5, 5, 1, 2, false)
	if p2 != "nn" {
		t.Errorf("Expected 'nn' truncated, got %q", p2)
	}

	// 3. Same tile returns ""
	p3 := FindPathToClosestString(m, 5, 5, 5, 5, 4, false)
	if p3 != "" {
		t.Errorf("Expected empty path for same tile, got %q", p3)
	}

	// 4. Target tile occupied by an actor: should path to closest adjacent tile
	targetActor := &Actor{Entity: Entity{ID: "hero", X: 5, Y: 2}}
	m.AddActor(targetActor)

	p4 := FindPathToClosestString(m, 5, 5, 5, 2, 4, false)
	// (5, 3) is the closest adjacent tile to (5, 2) from (5, 5), distance 2 steps -> "nn"
	if p4 != "nn" {
		t.Errorf("Expected 'nn' towards occupied target, got %q", p4)
	}

	// 5. Target adjacent returns ""
	p5 := FindPathToClosestString(m, 5, 3, 5, 2, 4, false)
	if p5 != "" {
		t.Errorf("Expected empty path when already adjacent to target, got %q", p5)
	}
}
