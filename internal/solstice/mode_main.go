package solstice

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// MainMode encapsulates the primary gameplay controls and rendering (party movement, UI, scale toggle).
type MainMode struct{}

// NewMainMode creates a new instance of MainMode.
func NewMainMode() *MainMode {
	return &MainMode{}
}

func (m *MainMode) Update(g *Game) error {
	// Advance global animation frame ticker
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

	// Enter targeting mode on U key press
	if inpututil.IsKeyJustPressed(ebiten.KeyU) {
		if g.party != nil {
			targetMode := NewTargetMode(
				g.party.X, g.party.Y, // Centerpoint on party location
				1,                   // Maximum range of 1
				DistanceDiamond,     // Manhattan / diamond distance for "use tile/object"
				func(tx, ty int) {   // On selected callback: execute tile use_script
					if g.currentMap != nil {
						_ = g.currentMap.ExecuteTileUseScript(tx, ty)
						g.currentMap.AdvanceTurn()
					}
				},
				nil, // On canceled callback: nil
			)
			g.PushMode(targetMode)
		}
	}

	return nil
}

func (m *MainMode) Draw(g *Game, screen *ebiten.Image) {
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
