package solstice

import (
	"fmt"
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	screenWidth  = 640
	screenHeight = 360
)

type Game struct {
	assets    *Assets
	terminal  *Terminal
	startTime time.Time
}

func (g *Game) Update() error {
	if g.terminal != nil {
		g.terminal.HandleInput()
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Swap between scale 1 and scale 2 every 2 seconds
	scale := 1
	if (int(time.Since(g.startTime).Seconds())/2)%2 == 1 {
		scale = 2
	}

	// Fill the map screen area with tile index 5
	g.assets.FillMapScreen(screen, 5, scale)

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

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Solstice")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := &Game{
		assets:    assets,
		terminal:  NewTerminal(),
		startTime: time.Now(),
	}

	if err := ebiten.RunGame(game); err != nil {
		fmt.Printf("Error running game: %v\n", err)
	}
}
