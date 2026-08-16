package solstice

import (
	"fmt"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	screenWidth  = 640
	screenHeight = 360

	windowWidth  = 1280
	windowHeight = 720
)

var defaultGame *Game

// GetGame returns the global game instance.
func GetGame() *Game {
	return defaultGame
}

// SetGame sets the global game instance.
func SetGame(g *Game) {
	defaultGame = g
}

type Game struct {
	assets     *Assets
	terminal   *Terminal
	currentMap *Map
	party      *Party
	spriteDefs map[string]SpriteDef
	mapScale   int
	modeStack  []Mode
}

// PushMode pushes a new mode onto the stack, making it the active mode.
func (g *Game) PushMode(m Mode) {
	if m != nil {
		g.modeStack = append(g.modeStack, m)
	}
}

// PopMode pops and returns the current top mode from the stack.
func (g *Game) PopMode() Mode {
	if len(g.modeStack) == 0 {
		return nil
	}
	topIdx := len(g.modeStack) - 1
	top := g.modeStack[topIdx]
	g.modeStack = g.modeStack[:topIdx]
	return top
}

// GetMode returns the current active mode (top of the mode stack).
func (g *Game) GetMode() Mode {
	if len(g.modeStack) == 0 {
		return nil
	}
	return g.modeStack[len(g.modeStack)-1]
}

// SetMode clears the mode stack and pushes m as the active mode.
func (g *Game) SetMode(m Mode) {
	g.modeStack = nil
	if m != nil {
		g.PushMode(m)
	}
}

func (g *Game) Update() error {
	if m := g.GetMode(); m != nil {
		return m.Update(g)
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if m := g.GetMode(); m != nil {
		m.Draw(g, screen)
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

	spriteDefs, err := PreloadSpriteDefs()
	if err != nil {
		log.Fatalf("failed to preload sprite defs: %v", err)
	}

	if _, err := PreloadActorDefs(); err != nil {
		log.Fatalf("failed to preload actor defs: %v", err)
	}

	// Create initial party with default "party-spirit-mode" sprite
	party, err := NewParty(0, 0)
	if err != nil {
		log.Fatalf("failed to create party: %v", err)
	}
	SetParty(party)

	term := NewTerminal()

	// Initialize Tengo script system and pre-compile all scripts from data/scripts
	if err := InitScriptSystem(); err != nil {
		log.Fatalf("failed to initialize script system: %v", err)
	}

	// Execute data/scripts/main.tengo after all assets and scripts are loaded.
	// main.tengo takes care of loading the map (game.load_map) and positioning the party (game.teleport_party).
	if err := RunMainScript(); err != nil {
		log.Fatalf("failed to execute main.tengo script: %v", err)
	}

	ebiten.SetWindowSize(windowWidth, windowHeight)
	ebiten.SetWindowTitle("Solstice")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := &Game{
		assets:     assets,
		terminal:   term,
		currentMap: GetMap(),
		party:      GetParty(),
		spriteDefs: spriteDefs,
		mapScale:   2,
	}
	SetGame(game)
	game.PushMode(NewMainMode())

	if err := ebiten.RunGame(game); err != nil {
		fmt.Printf("Error running game: %v\n", err)
	}
}
