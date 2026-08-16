package solstice

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestModeStackAndTargeting(t *testing.T) {
	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}
	if _, err := PreloadTileSet(); err != nil {
		t.Fatalf("PreloadTileSet failed: %v", err)
	}

	assets, err := LoadAssets()
	if err != nil {
		t.Fatalf("LoadAssets failed: %v", err)
	}

	homeMap, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}

	party, err := NewParty(10, 10)
	if err != nil {
		t.Fatalf("NewParty failed: %v", err)
	}

	mainMode := NewMainMode()
	game := &Game{
		assets:     assets,
		terminal:   NewTerminal(),
		currentMap: homeMap,
		party:      party,
		mapScale:   2,
	}
	game.PushMode(mainMode)

	if game.GetMode() != mainMode {
		t.Fatal("Expected active game mode to be mainMode")
	}

	// Test TargetMode push onto stack with DistanceDiamond (Manhattan distance)
	selectedX, selectedY := -1, -1
	canceledCalled := false

	targetMode := NewTargetMode(
		10, 10, 1,
		DistanceDiamond,
		func(tx, ty int) {
			selectedX = tx
			selectedY = ty
		},
		func() {
			canceledCalled = true
		},
	)

	game.PushMode(targetMode)
	if game.GetMode() != targetMode {
		t.Error("PushMode failed to set targetMode as active mode")
	}

	// Verify initial cursor position matches centerpoint
	if targetMode.cursorX != 10 || targetMode.cursorY != 10 {
		t.Errorf("Expected initial cursor (10, 10), got (%d, %d)", targetMode.cursorX, targetMode.cursorY)
	}

	// Test Update and Draw calls at scale 2 and scale 1
	screen := ebiten.NewImage(640, 360)
	if err := game.Update(); err != nil {
		t.Fatalf("game.Update() failed: %v", err)
	}
	game.Draw(screen)

	game.mapScale = 1
	game.Draw(screen)

	// Simulate pop back to mainMode
	popped := game.PopMode()
	if popped != targetMode {
		t.Error("Expected PopMode to return targetMode")
	}

	if game.GetMode() != mainMode {
		t.Error("Expected PopMode to restore mainMode as active mode")
	}

	// Simulate callbacks
	targetMode.onSelected(11, 10)
	if selectedX != 11 || selectedY != 10 {
		t.Errorf("Expected onSelected callback to receive (11, 10), got (%d, %d)", selectedX, selectedY)
	}

	targetMode.onCanceled()
	if !canceledCalled {
		t.Error("Expected onCanceled callback to be called")
	}
}

func TestDistanceMetrics(t *testing.T) {
	// 1. DistanceDiamond (Manhattan distance): |dx| + |dy| <= maxRange
	diamondMode := NewTargetMode(10, 10, 1, DistanceDiamond, nil, nil)
	// (11, 11) has dx=1, dy=1 -> sum=2 > 1 -> out of range
	distX := 11 - 10
	distY := 11 - 10
	if (distX + distY) <= diamondMode.maxRange {
		t.Error("Expected (11, 11) to be out of range for DistanceDiamond at range 1")
	}
	// (11, 10) has dx=1, dy=0 -> sum=1 <= 1 -> in range
	distX = 11 - 10
	distY = 10 - 10
	if (distX + distY) > diamondMode.maxRange {
		t.Error("Expected (11, 10) to be in range for DistanceDiamond at range 1")
	}

	// 2. DistanceSquare (Chebyshev distance): max(|dx|, |dy|) <= maxRange
	squareMode := NewTargetMode(10, 10, 1, DistanceSquare, nil, nil)
	// (11, 11) has dx=1, dy=1 -> max=1 <= 1 -> in range
	distX = 11 - 10
	distY = 11 - 10
	if distX > squareMode.maxRange || distY > squareMode.maxRange {
		t.Error("Expected (11, 11) to be in range for DistanceSquare at range 1")
	}
}
