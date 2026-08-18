package solstice

import (
	"image"
	"image/color"

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

	// Update cut scene runner
	if UpdateCutScene(g) {
		// Allow toggling map scale on Z key press during cut scenes
		if inpututil.IsKeyJustPressed(ebiten.KeyZ) {
			if g.mapScale == 2 {
				g.mapScale = 1
			} else {
				g.mapScale = 2
			}
			SetMapScale(g.mapScale)
		}

		// Allow opening main menu on Escape key press
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.PushMode(NewMainMenuMode())
		}
		return nil
	}

	if IsInCombat() {
		// Combat mode: control the current party member
		party := g.party
		if party == nil {
			party = GetParty()
		}

		if party != nil && len(party.Members) > 0 {
			curIdx := GetCombatMemberIndex()
			if curIdx >= len(party.Members) {
				curIdx = 0
				SetCombatMemberIndex(0)
			}
			curMember := &party.Members[curIdx]

			// Move - M key, allows player to move current party member using A* pathfinding
			if inpututil.IsKeyJustPressed(ebiten.KeyM) {
				curMap := GetMap()
				if curMap != nil {
					moveRange := curMember.Move
					if moveRange <= 0 {
						moveRange = 3
					}
					reachable := FindReachableTiles(curMap, curMember.X, curMember.Y, moveRange, true)
					targetMode := NewTargetMode(
						curMember.X, curMember.Y,
						0, // Unlimited range
						DistanceDiamond,
						func(tx, ty int) bool {
							targetPt := image.Pt(tx, ty)
							if !reachable[targetPt] {
								return false
							}
							path := FindPath(curMap, curMember.X, curMember.Y, tx, ty, true)
							if len(path) == 0 {
								return false
							}
							for _, dir := range path {
								EnqueueCutSceneCommand(CutSceneCommand{
									Type:    CmdMove,
									ActorID: curMember.ID,
									Dir:     dir,
								})
								EnqueueCutSceneCommand(CutSceneCommand{
									Type:   CmdDelay,
									Frames: 1,
								})
							}
							return true
						},
						nil,
					)
					targetMode.SetHighlightTiles(reachable, color.RGBA{R: 85, G: 255, B: 85, A: 89})
					g.PushMode(targetMode)
				}
			}

			// Pass - Spacebar, Period, Keypad 5
			if inpututil.IsKeyJustPressed(ebiten.KeySpace) ||
				inpututil.IsKeyJustPressed(ebiten.KeyPeriod) ||
				inpututil.IsKeyJustPressed(ebiten.KeyKP5) ||
				inpututil.IsKeyJustPressed(ebiten.KeyNumpad5) {
				AdvanceCombatMember(g)
			}
		}
	} else {
		// Handle party movement input (WASD, Arrow keys, VI-style HJKL)
		if g.party != nil {
			g.party.HandleInput(g.currentMap)
		}

		// Activate on_enter trigger on E key press
		if inpututil.IsKeyJustPressed(ebiten.KeyE) {
			if g.party != nil && g.currentMap != nil {
				g.currentMap.ActivateTriggersOnEnter(g.party.X, g.party.Y, "party")
			}
		}

		// Enter targeting mode on U key press
		if inpututil.IsKeyJustPressed(ebiten.KeyU) {
			if g.party != nil {
				targetMode := NewTargetMode(
					g.party.X, g.party.Y, // Centerpoint on party location
					1,                   // Maximum range of 1
					DistanceDiamond,     // Manhattan / diamond distance for "use tile/object"
					func(tx, ty int) bool { // On selected callback: execute tile use_script
						if g.currentMap != nil {
							_ = g.currentMap.ExecuteTileUseScript(tx, ty)
							g.currentMap.AdvanceTurn()
						}
						return true
					},
					nil, // On canceled callback: nil
				)
				g.PushMode(targetMode)
			}
		}

		// Enter dialog targeting mode on T key press
		if inpututil.IsKeyJustPressed(ebiten.KeyT) {
			if g.party != nil {
				targetMode := NewTargetMode(
					g.party.X, g.party.Y, // Centerpoint on party location
					5,                   // Maximum range of 5
					DistanceSquare,      // Square distance
					func(tx, ty int) bool { // On selected callback: talk to targeted actor
						if g.currentMap != nil {
							actor := g.currentMap.GetActorAt(tx, ty)
							if actor != nil && actor.DialogScript != "" {
								g.PushMode(NewDialogMode(actor, actor.DialogScript))
							}
						}
						return true
					},
					nil, // On canceled callback: nil
				)
				g.PushMode(targetMode)
			}
		}
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
		SetMapScale(g.mapScale)
	}

	// Toggle wizard mode on F12 key press
	if inpututil.IsKeyJustPressed(ebiten.KeyF12) {
		ToggleWizardMode()
	}

	// Open main menu on Escape key press
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.PushMode(NewMainMenuMode())
	}

	return nil
}

func (m *MainMode) Draw(g *Game, screen *ebiten.Image) {
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
