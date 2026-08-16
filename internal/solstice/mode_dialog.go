package solstice

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// DialogMode encapsulates NPC dialogue state, input loop processing, and rendering.
type DialogMode struct {
	actor       *Actor
	scriptPath  string
	initialized bool
}

// NewDialogMode creates a new DialogMode for interaction with the given actor.
func NewDialogMode(actor *Actor, scriptPath string) *DialogMode {
	return &DialogMode{
		actor:      actor,
		scriptPath: scriptPath,
	}
}

func (dm *DialogMode) Update(g *Game) error {
	UpdateAnimTicker()

	if !dm.initialized {
		dm.initialized = true
		if g.terminal != nil {
			g.terminal.SetInputMode(true)
		}
		// First exchange: execute dialog script with "look" keyword
		ended, _ := ExecuteDialogScript(dm.scriptPath, "look")
		if ended {
			if g.terminal != nil {
				g.terminal.SetInputMode(false)
			}
			g.PopMode()
			return nil
		}
	}

	if g.terminal != nil {
		input, submitted := g.terminal.HandleInputMode()
		if submitted {
			// Log user's input line to terminal history in bright blue
			g.terminal.AddMessageColored("> "+input, VGAPalette16[9])

			// First 4 characters converted to lower case as keyword
			kw := strings.ToLower(strings.TrimSpace(input))
			if len(kw) > 4 {
				kw = kw[:4]
			}

			ended, _ := ExecuteDialogScript(dm.scriptPath, kw)
			if ended {
				g.terminal.SetInputMode(false)
				g.PopMode()
				return nil
			}
		}
	}

	return nil
}

func (dm *DialogMode) Draw(g *Game, screen *ebiten.Image) {
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

	// 1. Draw the map centered on party
	if g.currentMap != nil {
		g.currentMap.DrawCentered(screen, g.assets, partyX, partyY, scale)
	}

	// 2. Draw party sprite
	if g.party != nil && g.assets != nil {
		centerStx := 5
		centerSty := 5
		if scale == 1 {
			centerStx = 11
			centerSty = 11
		}
		g.assets.DrawSpriteDef(screen, g.party.SpriteDef, centerStx, centerSty, scale)
	}

	// 3. Draw terminal UI (renders log history and bottom input line)
	if g.terminal != nil {
		g.terminal.Draw(screen, g.assets)
	}
}
