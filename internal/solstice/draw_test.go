package solstice

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestLoadAssetsAndDraw(t *testing.T) {
	assets, err := LoadAssets()
	if err != nil {
		t.Fatalf("LoadAssets failed: %v", err)
	}

	if assets.FontIBM8x8 == nil {
		t.Error("FontIBM8x8 is nil")
	}
	if assets.FontIBM16x12 == nil {
		t.Error("FontIBM16x12 is nil")
	}
	if assets.FontRune8x8 == nil {
		t.Error("FontRune8x8 is nil")
	}
	if assets.FontRune16x12 == nil {
		t.Error("FontRune16x12 is nil")
	}
	if assets.Tiles16 == nil {
		t.Error("Tiles16 is nil")
	}

	screen := ebiten.NewImage(640, 360)

	// Test 8x8 glyph drawing in all ranges
	assets.DrawGlyph8x8(screen, 65, 2, 3)  // IBM.CH ('A')
	assets.DrawGlyph8x8(screen, 130, 3, 3) // RUNES.CH (glyph 2)
	assets.DrawGlyph8x8(screen, -1, 4, 3)  // Out of range (black fill)
	assets.DrawGlyph8x8(screen, 300, 5, 3) // Out of range (black fill)

	// Test string drawing
	assets.DrawString8x8(screen, "Hello World", 0, 0)

	// Test Map Tile drawing at 2x scale (coordinate 2,3 -> 64, 96)
	assets.DrawMapTile(screen, 5, 2, 3, 2)

	// Test Map Tile drawing at 1x scale (coordinate 2,3 -> 24, 40)
	assets.DrawMapTile(screen, 5, 2, 3, 1)

	// Test FillMapScreen at scale 1 and scale 2
	assets.FillMapScreen(screen, 5, 1)
	assets.FillMapScreen(screen, 5, 2)

	// Also test global functions
	DrawGlyph8x8(screen, 65, 2, 3)
	DrawString8x8(screen, "Hello World", 0, 0)
	DrawMapTile(screen, 5, 2, 3, 2)
	FillMapScreen(screen, 5, 1)
}
