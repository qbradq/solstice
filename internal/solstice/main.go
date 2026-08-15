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
	spriteDefs map[string]SpriteDef
	mapScale   int
}

func (g *Game) Update() error {
	if g.terminal != nil {
		g.terminal.HandleInput()
	}

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
	// Draw the currently loaded map using the active map scale
	if g.currentMap != nil {
		scale := g.mapScale
		if scale == 0 {
			scale = 2
		}
		g.currentMap.Draw(screen, g.assets, scale)
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

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Solstice")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := &Game{
		assets:     assets,
		terminal:   NewTerminal(),
		currentMap: homeMap,
		spriteDefs: spriteDefs,
		mapScale:   2,
	}

	if err := ebiten.RunGame(game); err != nil {
		fmt.Printf("Error running game: %v\n", err)
	}
}
