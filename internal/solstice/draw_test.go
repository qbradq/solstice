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

	// Test Map Tile drawing at 2x scale
	assets.DrawMapTile(screen, 5, 2, 3, 2)
	assets.DrawMapTile(screen, 5, 2, 3, 1)

	// Test Black Map Tile drawing
	assets.DrawBlackMapTile(screen, 0, 0, 2)
	assets.DrawBlackMapTile(screen, 0, 0, 1)

	// Test SpriteDef drawing with animation ticker
	SetAnimFrame(0)
	sd := SpriteDef{Tile: 372, Animated: true, Frames: 4}
	assets.DrawSpriteDef(screen, sd, 5, 5, 2)

	SetAnimFrame(2)
	assets.DrawSpriteDef(screen, sd, 5, 5, 2)

	// Test FillMapScreen at scale 1 and scale 2
	assets.FillMapScreen(screen, 5, 1)
	assets.FillMapScreen(screen, 5, 2)

	// Also test global functions
	DrawGlyph8x8(screen, 65, 2, 3)
	DrawString8x8(screen, "Hello World", 0, 0)
	DrawMapTile(screen, 5, 2, 3, 2)
	DrawBlackMapTile(screen, 0, 0, 2)
	DrawSpriteDef(screen, sd, 5, 5, 2)
	FillMapScreen(screen, 5, 1)
}

func TestAnimTicker(t *testing.T) {
	SetAnimFrame(0)
	if f := GetAnimFrame(); f != 0 {
		t.Errorf("Expected anim frame 0, got %d", f)
	}

	for i := 0; i < 15; i++ {
		UpdateAnimTicker()
	}

	if f := GetAnimFrame(); f != 1 {
		t.Errorf("Expected anim frame to advance to 1 after 15 ticks, got %d", f)
	}
}

func TestFormatCommasAndDrawPartyRoster(t *testing.T) {
	if got := formatCommas(0); got != "0" {
		t.Errorf("formatCommas(0) = %q, want %q", got, "0")
	}
	if got := formatCommas(100); got != "100" {
		t.Errorf("formatCommas(100) = %q, want %q", got, "100")
	}
	if got := formatCommas(2000); got != "2,000" {
		t.Errorf("formatCommas(2000) = %q, want %q", got, "2,000")
	}
	if got := formatCommas(65000); got != "65,000" {
		t.Errorf("formatCommas(65000) = %q, want %q", got, "65,000")
	}

	assets, err := LoadAssets()
	if err != nil {
		t.Fatalf("LoadAssets failed: %v", err)
	}

	_, err = PreloadSpriteDefs()
	if err != nil {
		t.Fatalf("PreloadSpriteDefs failed: %v", err)
	}

	_, err = PreloadActorDefs()
	if err != nil {
		t.Fatalf("PreloadActorDefs failed: %v", err)
	}

	kevin, err := NewActorFromDef("kevin-1", "kevin", 0, 0)
	if err != nil {
		t.Fatalf("NewActorFromDef failed: %v", err)
	}

	party, err := NewParty(0, 0, *kevin)
	if err != nil {
		t.Fatalf("NewParty failed: %v", err)
	}

	screen := ebiten.NewImage(640, 360)
	assets.DrawPartyRoster(screen, party)

	term := NewTerminal()
	DrawCommonUI(screen, assets, party, term)
}
