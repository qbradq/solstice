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
			g.terminal.AddMessageColored("> LOOK", VGAPalette16[9])
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
		input, submitted, canceled := g.terminal.HandleInputMode()
		if canceled {
			// Log synthetic player input line "> BYE" in bright blue
			g.terminal.AddMessageColored("> BYE", VGAPalette16[9])

			// Player pressed Escape key in dialog:
			// Execute dialog script with "bye" keyword (script may or may not end dialog)
			ended, _ := ExecuteDialogScript(dm.scriptPath, "bye")
			if ended {
				g.terminal.SetInputMode(false)
				g.PopMode()
				return nil
			}
			g.terminal.SetInputMode(true)
			return nil
		}

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

	// 1. Draw the map view area (map tiles, actors, and party sprite)
	if g.currentMap != nil {
		g.currentMap.DrawCentered(screen, g.assets, g.party, scale)
	}

	// 2. Draw common UI (party roster area and terminal UI)
	DrawCommonUI(screen, g.assets, g.party, g.terminal)
}
