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

	windowWidth  = 1280
	windowHeight = 720
)

var (
	defaultGame     *Game
	defaultMapScale = 2
)

// GetGame returns the global game instance.
func GetGame() *Game {
	return defaultGame
}

// SetGame sets the global game instance.
func SetGame(g *Game) {
	defaultGame = g
}

// GetMapScale returns the current map scale (1 or 2). Default is 2.
func GetMapScale() int {
	if defaultGame != nil && defaultGame.mapScale > 0 {
		return defaultGame.mapScale
	}
	if defaultMapScale > 0 {
		return defaultMapScale
	}
	return 2
}

// SetMapScale sets the current map scale (1 or 2).
func SetMapScale(scale int) {
	if scale <= 0 {
		scale = 2
	}
	defaultMapScale = scale
	if defaultGame != nil {
		defaultGame.mapScale = scale
	}
}

type Game struct {
	assets     *Assets
	terminal   *Terminal
	tengoTerm  *TengoTerminal
	currentMap *Map
	worldMap   *Map
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
	// Top-level check for the tilde key (`/~) to cycle Tengo terminal states
	if inpututil.IsKeyJustPressed(ebiten.KeyGraveAccent) {
		if g.tengoTerm != nil {
			g.tengoTerm.CycleState()
		}
		return nil
	}

	if g.tengoTerm != nil && g.tengoTerm.GetState() != TengoTerminalStateHidden {
		return g.tengoTerm.Update(g)
	}

	if m := g.GetMode(); m != nil {
		return m.Update(g)
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	state := TengoTerminalStateHidden
	if g.tengoTerm != nil {
		state = g.tengoTerm.GetState()
	}

	switch state {
	case TengoTerminalStateHidden:
		if m := g.GetMode(); m != nil {
			m.Draw(g, screen)
		}
	case TengoTerminalStateHalf:
		if m := g.GetMode(); m != nil {
			m.Draw(g, screen)
		}
		if g.tengoTerm != nil {
			g.tengoTerm.DrawHalf(screen, g.assets)
		}
	case TengoTerminalStateFull:
		if g.tengoTerm != nil {
			g.tengoTerm.DrawFull(screen, g.assets)
		}
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

	if _, err := PreloadItemDefs(); err != nil {
		log.Fatalf("failed to preload item defs: %v", err)
	}

	if _, err := PreloadEnemyPacks(); err != nil {
		log.Fatalf("failed to preload enemy packs: %v", err)
	}

	term := NewTerminal()

	// Initialize Tengo script system and pre-compile all scripts from data/scripts
	if err := InitScriptSystem(); err != nil {
		log.Fatalf("failed to initialize script system: %v", err)
	}

	// Initialize Tengo REPL and pull-down terminal
	repl := NewTengoREPL()
	tengoTerm := NewTengoTerminal(repl)

	ebiten.SetWindowSize(windowWidth, windowHeight)
	ebiten.SetWindowTitle("Solstice")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := &Game{
		assets:     assets,
		terminal:   term,
		tengoTerm:  tengoTerm,
		currentMap: GetMap(),
		worldMap:   GetWorldMap(),
		party:      GetParty(),
		spriteDefs: spriteDefs,
		mapScale:   2,
	}
	SetGame(game)
	game.PushMode(NewMainMode())
	game.PushMode(NewMainMenuMode())

	if err := ebiten.RunGame(game); err != nil {
		fmt.Printf("Error running game: %v\n", err)
	}
}
