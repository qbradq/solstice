package solstice

import "github.com/hajimehoshi/ebiten/v2"

// Mode defines the interface for different game execution modes (e.g. MainMode, TargetMode).
type Mode interface {
	Update(g *Game) error
	Draw(g *Game, screen *ebiten.Image)
}
