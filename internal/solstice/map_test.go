package solstice

import (
	"testing"
)

func TestPreloadTileSet(t *testing.T) {
	ts, err := PreloadTileSet()
	if err != nil {
		t.Fatalf("PreloadTileSet failed: %v", err)
	}

	if ts.Name != "tileset" {
		t.Errorf("Expected tileset name 'tileset', got %q", ts.Name)
	}
	if ts.TileCount != 512 {
		t.Errorf("Expected tilecount 512, got %d", ts.TileCount)
	}
	if len(ts.Properties) == 0 {
		t.Error("Expected tile properties to be populated")
	}

	// Tile 1 has deep_water=true, water=true
	prop1 := ts.GetTileProperties(1)
	if !prop1.DeepWater || !prop1.Water {
		t.Errorf("Expected tile 1 to have DeepWater=true and Water=true, got %+v", prop1)
	}

	// Tile 4 has walkable=true
	prop4 := ts.GetTileProperties(4)
	if !prop4.Walkable {
		t.Errorf("Expected tile 4 to have Walkable=true, got %+v", prop4)
	}

	// Tile 13 has blocks_vis=true
	prop13 := ts.GetTileProperties(13)
	if !prop13.BlocksVis {
		t.Errorf("Expected tile 13 to have BlocksVis=true, got %+v", prop13)
	}

	// Tile 45 has actor_half_sprite=true / party_half_sprite=true
	prop45 := ts.GetTileProperties(45)
	if !prop45.ActorHalfSprite {
		t.Errorf("Expected tile 45 to have ActorHalfSprite=true, got %+v", prop45)
	}
}

func TestLoadMap(t *testing.T) {
	m, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}

	if m.Name != "home" {
		t.Errorf("Expected map name 'home', got %q", m.Name)
	}
	if m.Width != 32 || m.Height != 32 {
		t.Errorf("Expected map size 32x32, got %dx%d", m.Width, m.Height)
	}
	if len(m.Tiles) != 1024 {
		t.Errorf("Expected 1024 tiles, got %d", len(m.Tiles))
	}
}

func TestLoadMapObjectLayerActors(t *testing.T) {
	if _, err := PreloadSpriteDefs(); err != nil {
		t.Fatalf("PreloadSpriteDefs failed: %v", err)
	}
	if _, err := PreloadActorDefs(); err != nil {
		t.Fatalf("PreloadActorDefs failed: %v", err)
	}

	homeMap, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}

	if len(homeMap.Actors) < 2 {
		t.Fatalf("Expected at least 2 actors loaded from homeMap object layer, got %d", len(homeMap.Actors))
	}

	// Verify object 1 (guard) at x=271.75, y=192, height=16 (GID tile object, Y bottom-left -> top-left y=176) -> tile (17, 11)
	guardActor := homeMap.GetActorAt(17, 11)
	if guardActor == nil {
		t.Errorf("Expected guard actor at tile (17, 11), got nil")
	} else {
		if guardActor.DialogScript != "dialog/guard.tengo" {
			t.Errorf("Expected guard DialogScript 'dialog/guard.tengo', got %q", guardActor.DialogScript)
		}
	}

	// Verify object 2 (wizard) at x=304, y=240.25, height=16 (GID tile object, Y bottom-left -> top-left y=224.25) -> tile (19, 14)
	wizardActor := homeMap.GetActorAt(19, 14)
	if wizardActor == nil {
		t.Errorf("Expected wizard actor at tile (19, 14), got nil")
	}
}

func TestMapMoveParty(t *testing.T) {
	if _, err := PreloadTileSet(); err != nil {
		t.Fatalf("PreloadTileSet failed: %v", err)
	}

	m, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}

	// Make tile (10, 10) walkable and tile (11, 10) non-walkable
	m.SetTile(10, 10, 4) // Walkable tile
	m.SetTile(11, 10, 0) // Non-walkable tile

	p, err := NewParty(10, 10)
	if err != nil {
		t.Fatalf("NewParty failed: %v", err)
	}

	// Moving to blocked tile should fail
	if m.MoveParty(p, 1, 0) {
		t.Error("Expected MoveParty to (11, 10) to return false (blocked)")
	}

	if p.X != 10 || p.Y != 10 {
		t.Errorf("Expected party position to remain (10, 10), got (%d, %d)", p.X, p.Y)
	}

	// Make tile (11, 10) walkable
	m.SetTile(11, 10, 4)

	// Moving to walkable tile should succeed
	if !m.MoveParty(p, 1, 0) {
		t.Error("Expected MoveParty to (11, 10) to return true")
	}

	if p.X != 11 || p.Y != 10 {
		t.Errorf("Expected party position (11, 10), got (%d, %d)", p.X, p.Y)
	}
}

func TestSpiritModePartyMovement(t *testing.T) {
	if _, err := PreloadTileSet(); err != nil {
		t.Fatalf("PreloadTileSet failed: %v", err)
	}

	m, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}

	// Set tile (10, 10) to walkable and (11, 10) to tile ID with spirit_passable=true (e.g. tile 78 if door)
	m.SetTile(10, 10, 4) // Walkable
	m.SetTile(11, 10, 78) // spirit_passable tile from tileset.tsx

	// Spirit party (0 members)
	spiritParty, err := NewParty(10, 10)
	if err != nil {
		t.Fatalf("NewParty failed: %v", err)
	}

	if !spiritParty.IsSpiritMode() {
		t.Fatal("Expected spiritParty to be in spirit mode")
	}

	// Spirit party SHOULD be able to move onto spirit_passable tile (11, 10)
	if !m.MoveParty(spiritParty, 1, 0) {
		t.Error("Expected spirit mode party to move onto spirit_passable tile")
	}

	// Living party (1 member)
	livingParty, err := NewParty(10, 10, Actor{Entity: Entity{Name: "Hero"}})
	if err != nil {
		t.Fatalf("NewParty failed: %v", err)
	}

	if livingParty.IsSpiritMode() {
		t.Fatal("Expected livingParty NOT to be in spirit mode")
	}

	// Living party SHOULD NOT be able to move onto spirit_passable tile (11, 10)
	if m.MoveParty(livingParty, 1, 0) {
		t.Error("Expected living party to be blocked by spirit_passable (non-walkable) tile")
	}
}
