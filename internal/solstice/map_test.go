package solstice

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
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

	// Tile 78 has use_script="tiles/door.tengo"
	prop78 := ts.GetTileProperties(78)
	if prop78.UseScript != "tiles/door.tengo" {
		t.Errorf("Expected tile 78 to have UseScript='tiles/door.tengo', got %q", prop78.UseScript)
	}
}

func TestLoadMapHomeAndGetSetTile(t *testing.T) {
	m, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap('home') failed: %v", err)
	}

	if m.Width != 32 || m.Height != 32 {
		t.Errorf("Expected map dimensions 32x32, got %dx%d", m.Width, m.Height)
	}

	// GID 11 in home.tmx -> tile index 10 (11 - 1)
	initialTile := m.GetTile(0, 0)
	if initialTile != 10 {
		t.Errorf("Expected initial tile at (0,0) to be 10, got %d", initialTile)
	}

	// Test SetTile
	m.SetTile(0, 0, 42)
	if updated := m.GetTile(0, 0); updated != 42 {
		t.Errorf("Expected updated tile at (0,0) to be 42, got %d", updated)
	}

	// Test bounds safety
	if oob := m.GetTile(-1, -1); oob != 0 {
		t.Errorf("Expected out-of-bounds GetTile to return 0, got %d", oob)
	}
	m.SetTile(-1, -1, 99) // Should not panic
}

func TestMovePartyWalkableConstraint(t *testing.T) {
	if _, err := PreloadTileSet(); err != nil {
		t.Fatalf("PreloadTileSet failed: %v", err)
	}

	m, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}

	// Set up map tile at (10, 10) as walkable (tile 4 grass)
	// and tile at (11, 10) as non-walkable (tile 1 deep water)
	m.SetTile(10, 10, 4)
	m.SetTile(11, 10, 1)
	m.SetTile(9, 10, 4)

	party, err := NewParty(10, 10)
	if err != nil {
		t.Fatalf("NewParty failed: %v", err)
	}

	// 1. Moving onto non-walkable tile (11, 10) should fail
	moved := m.MoveParty(party, 1, 0)
	if moved {
		t.Error("Expected movement onto non-walkable tile to fail")
	}
	if party.X != 10 || party.Y != 10 {
		t.Errorf("Expected party position to remain (10, 10), got (%d, %d)", party.X, party.Y)
	}

	// 2. Moving onto walkable tile (9, 10) should succeed
	moved = m.MoveParty(party, -1, 0)
	if !moved {
		t.Error("Expected movement onto walkable tile to succeed")
	}
	if party.X != 9 || party.Y != 10 {
		t.Errorf("Expected party position (9, 10), got (%d, %d)", party.X, party.Y)
	}

	// 3. Moving out of bounds should fail
	party.X = 0
	party.Y = 0
	moved = m.MoveParty(party, -1, 0)
	if moved {
		t.Error("Expected out-of-bounds movement to fail")
	}
}

func TestSpiritModeSpiritPassableMovement(t *testing.T) {
	ts, err := PreloadTileSet()
	if err != nil {
		t.Fatalf("PreloadTileSet failed: %v", err)
	}

	// Manually inject a tile property for testing (tile 99: spirit_passable=true, walkable=false)
	ts.Properties[99] = TileProperties{
		SpiritPassable: true,
		Walkable:       false,
	}

	m, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}

	m.SetTile(10, 10, 4)  // Walkable grass
	m.SetTile(11, 10, 99) // Spirit passable only

	// Spirit mode party (0 members)
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

func TestExecuteTileUseScript(t *testing.T) {
	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}

	if _, err := PreloadTileSet(); err != nil {
		t.Fatalf("PreloadTileSet failed: %v", err)
	}

	m, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}
	SetMap(m)

	// Tile 78 has use_script="tiles/door.tengo"
	m.SetTile(5, 5, 78)

	if err := m.ExecuteTileUseScript(5, 5); err != nil {
		t.Fatalf("ExecuteTileUseScript(5, 5) failed: %v", err)
	}

	// Verify that executing tiles/door.tengo changed the tile at (5, 5) to 68
	if updatedTile := m.GetTile(5, 5); updatedTile != 68 {
		t.Errorf("Expected tile at (5, 5) to be changed to 68 by door.tengo script, got %d", updatedTile)
	}
}

func TestMapDraw(t *testing.T) {
	m, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}

	assets, err := LoadAssets()
	if err != nil {
		t.Fatalf("LoadAssets failed: %v", err)
	}

	screen := ebiten.NewImage(640, 360)
	m.Draw(screen, assets, 1)
	m.Draw(screen, assets, 2)
}
