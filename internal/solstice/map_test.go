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
