package solstice

import (
	"fmt"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	screenWidth  = 640
	screenHeight = 360
)

type Game struct {
	assets     *Assets
	terminal   *Terminal
	currentMap *Map
	party      *Party
	spriteDefs map[string]SpriteDef
	mapScale   int
}

func (g *Game) Update() error {
	// Advance animation frame ticker
	UpdateAnimTicker()

	// Handle party movement input (WASD, Arrow keys, VI-style HJKL)
	if g.party != nil {
		g.party.HandleInput(g.currentMap)
	}

	// Handle terminal input (Page Up, Page Down)
	if g.terminal != nil {
		g.terminal.HandleInput()
	}

	// Toggle map scale on Z key press
	if inpututil.IsKeyJustPressed(ebiten.KeyZ) {
		if g.mapScale == 2 {
			g.mapScale = 1
		} else {
			g.mapScale = 2
		}
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	scale := g.mapScale
	if scale == 0 {
		scale = 2
	}

	partyX := 16
	partyY := 16
	if g.party != nil {
		partyX = g.party.X
		partyY = g.party.Y
	}

	// Draw the currently loaded map centered on the party's position
	if g.currentMap != nil {
		g.currentMap.DrawCentered(screen, g.assets, partyX, partyY, scale)
	}

	// Draw party sprite at center of map view area using global animation ticker
	if g.party != nil && g.assets != nil {
		centerStx := 5
		centerSty := 5
		if scale == 1 {
			centerStx = 11
			centerSty = 11
		}
		g.assets.DrawSpriteDef(screen, g.party.SpriteDef, centerStx, centerSty, scale)
	}

	// Draw terminal UI
	if g.terminal != nil {
		g.terminal.Draw(screen, g.assets)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

// Main is the main entry point of the client.
func Main() {
	assets, err := LoadAssets()
	if err != nil {
		log.Fatalf("failed to load assets: %v", err)
	}

	if _, err := PreloadTileSet(); err != nil {
		log.Fatalf("failed to preload tileset: %v", err)
	}

	homeMap, err := LoadMap("home")
	if err != nil {
		log.Fatalf("failed to load home map: %v", err)
	}

	spriteDefs, err := PreloadSpriteDefs()
	if err != nil {
		log.Fatalf("failed to preload sprite defs: %v", err)
	}

	// Create initial party at position (16, 16) with default "party-spirit-mode" sprite
	party, err := NewParty(16, 16)
	if err != nil {
		log.Fatalf("failed to create party: %v", err)
	}
	SetParty(party)

	term := NewTerminal()

	// Initialize Tengo script system and pre-compile all scripts from data/scripts
	if err := InitScriptSystem(); err != nil {
		log.Fatalf("failed to initialize script system: %v", err)
	}

	// Execute data/scripts/main.tengo after all assets and scripts are loaded
	if err := RunMainScript(); err != nil {
		log.Fatalf("failed to execute main.tengo script: %v", err)
	}

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Solstice")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := &Game{
		assets:     assets,
		terminal:   term,
		currentMap: homeMap,
		party:      party,
		spriteDefs: spriteDefs,
		mapScale:   2,
	}

	if err := ebiten.RunGame(game); err != nil {
		fmt.Printf("Error running game: %v\n", err)
	}
}
