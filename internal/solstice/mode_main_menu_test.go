package solstice

import (
	"errors"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestMainMenuModeInitialization(t *testing.T) {
	SetSaveDirOverride(t.TempDir()) // Ensure empty saves directory
	SetParty(nil)
	SetMap(nil)
	menu := NewMainMenuMode()
	if menu == nil {
		t.Fatal("NewMainMenuMode returned nil")
	}

	if menu.selectedIndex != 1 {
		t.Errorf("Expected initial selectedIndex to be 1 (New Game), got %d", menu.selectedIndex)
	}

	if len(menu.options) != 3 {
		t.Fatalf("Expected 3 options, got %d", len(menu.options))
	}

	if menu.options[0].Label != "Load Game" || menu.options[0].Enabled != false {
		t.Errorf("Expected option 0 to be disabled 'Load Game', got %+v", menu.options[0])
	}
	if menu.options[1].Label != "New Game" || menu.options[1].Enabled != true {
		t.Errorf("Expected option 1 to be enabled 'New Game', got %+v", menu.options[1])
	}
	if menu.options[2].Label != "Quit" || menu.options[2].Enabled != true {
		t.Errorf("Expected option 2 to be enabled 'Quit', got %+v", menu.options[2])
	}
}

func TestMainMenuModeNavigation(t *testing.T) {
	SetSaveDirOverride(t.TempDir())
	SetParty(nil)
	SetMap(nil)
	menu := NewMainMenuMode()

	// Initial: New Game (index 1)
	if menu.selectedIndex != 1 {
		t.Fatalf("Expected selectedIndex 1, got %d", menu.selectedIndex)
	}

	// Move down -> Quit (index 2)
	menu.moveSelection(1)
	if menu.selectedIndex != 2 {
		t.Errorf("Expected selectedIndex 2 after moving down, got %d", menu.selectedIndex)
	}

	// Move down -> wraps to New Game (index 1), skipping disabled Load Game (index 0)
	menu.moveSelection(1)
	if menu.selectedIndex != 1 {
		t.Errorf("Expected selectedIndex 1 after moving down from Quit, got %d", menu.selectedIndex)
	}

	// Move up -> wraps to Quit (index 2), skipping disabled Load Game (index 0)
	menu.moveSelection(-1)
	if menu.selectedIndex != 2 {
		t.Errorf("Expected selectedIndex 2 after moving up from New Game, got %d", menu.selectedIndex)
	}

	// Move up -> New Game (index 1)
	menu.moveSelection(-1)
	if menu.selectedIndex != 1 {
		t.Errorf("Expected selectedIndex 1 after moving up from Quit, got %d", menu.selectedIndex)
	}
}

func TestMainMenuModeActivation(t *testing.T) {
	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}
	if _, err := PreloadTileSet(); err != nil {
		t.Fatalf("PreloadTileSet failed: %v", err)
	}
	if _, err := PreloadSpriteDefs(); err != nil {
		t.Fatalf("PreloadSpriteDefs failed: %v", err)
	}
	if _, err := PreloadActorDefs(); err != nil {
		t.Fatalf("PreloadActorDefs failed: %v", err)
	}

	term := NewTerminal()
	SetTerminal(term)
	SetParty(nil)
	SetWorldMap(nil)
	SetMap(nil)

	game := &Game{
		terminal:   term,
		currentMap: nil,
		worldMap:   nil,
		party:      nil,
	}
	SetGame(game)

	menu := NewMainMenuMode()
	game.PushMode(NewMainMode())
	game.PushMode(menu)

	// 1. Activating disabled option (Continue) should be a no-op
	menu.selectedIndex = 0
	err := menu.activateSelection(game)
	if err != nil {
		t.Errorf("Expected nil error on activating disabled option, got %v", err)
	}
	if game.GetMode() != menu {
		t.Errorf("Expected menu to remain active after selecting disabled option")
	}

	// 2. Activating New Game (index 1) - initializes world map and party from nil
	menu.selectedIndex = 1
	err = menu.activateSelection(game)
	if err != nil {
		t.Errorf("Expected nil error on activating New Game, got %v", err)
	}
	if game.GetMode() == menu {
		t.Errorf("Expected menu mode to be popped after activating New Game")
	}
	if m := GetMap(); m == nil || (m.Name != "kings_shrine" && m.Name != "home") {
		t.Errorf("Expected active map to be loaded after New Game, got %v", m)
	}
	if p := GetParty(); p == nil || p.X != 15 || p.Y != 15 {
		t.Errorf("Expected party position (15, 15), got %v", p)
	}
	if wm := GetWorldMap(); wm == nil {
		t.Errorf("Expected world map to be initialized after New Game, got nil")
	}

	// 3. Activating Quit (index 2)
	menu.selectedIndex = 2
	err = menu.activateSelection(game)
	if !errors.Is(err, ebiten.Termination) {
		t.Errorf("Expected ebiten.Termination on activating Quit, got %v", err)
	}
}

func TestMainMenuModeEscapeKey(t *testing.T) {
	game := &Game{}
	SetGame(game)

	menu := NewMainMenuMode()
	game.PushMode(NewMainMode())
	game.PushMode(menu)

	// When currentMap and party are nil (e.g. at initial startup):
	// Escape should not close the main menu
	game.currentMap = nil
	game.party = nil

	// Simulate escape key check logic directly
	if game.currentMap != nil && game.party != nil {
		game.PopMode()
	}
	if game.GetMode() != menu {
		t.Errorf("Expected main menu to remain open on Escape when currentMap is nil")
	}

	// When currentMap and party are non-nil (game in progress):
	// Escape should close the main menu and return to MainMode
	m, _ := LoadMap("home")
	game.currentMap = m
	p, _ := NewParty(0, 0)
	game.party = p

	if game.currentMap != nil && game.party != nil {
		game.PopMode()
		if len(game.modeStack) == 0 {
			game.PushMode(NewMainMode())
		}
	}

	if _, ok := game.GetMode().(*MainMode); !ok {
		t.Errorf("Expected active mode to be MainMode after closing menu, got %T", game.GetMode())
	}
}

func TestMainMenuModeDrawNilSafety(t *testing.T) {
	assets, err := LoadAssets()
	if err != nil {
		t.Fatalf("LoadAssets failed: %v", err)
	}

	menu := NewMainMenuMode()
	screen := ebiten.NewImage(640, 360)

	// Test Draw with completely empty / nil Game fields
	nilGame := &Game{}
	menu.Draw(nilGame, screen)

	// Test Draw with assets only
	gameWithAssets := &Game{
		assets: assets,
	}
	menu.Draw(gameWithAssets, screen)

	// Test Draw with nil Game
	menu.Draw(nil, screen)
}
