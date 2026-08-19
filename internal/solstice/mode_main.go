package solstice

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// MainMode encapsulates the primary gameplay controls and rendering (party movement, UI, scale toggle).
type MainMode struct {
	colorIdx int
}

// NewMainMode creates a new instance of MainMode.
func NewMainMode() *MainMode {
	return &MainMode{}
}

func (m *MainMode) Update(g *Game) error {
	// Advance global animation frame ticker
	UpdateAnimTicker()

	// Advance targeting cursor color index
	m.colorIdx = (m.colorIdx + 1) % len(VGAPalette16)

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

			// If the current party member has already taken an action and a move,
			// and the cut scene queue is empty (no animation playing), pass to the next turn.
			if GetCombatMemberMoved() && GetCombatMemberActed() && !IsCutSceneActive() {
				EnqueueCutSceneCommand(CutSceneCommand{
					Type:   CmdDelay,
					Frames: 2,
				})
				AdvanceCombatMember(g)
				return nil
			}

			// Move - M key, allows player to move current party member using A* pathfinding (once per turn)
			if inpututil.IsKeyJustPressed(ebiten.KeyM) && !GetCombatMemberMoved() {
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
							SetCombatMemberMoved(true)
							EnqueueCutSceneCommand(CutSceneCommand{
								Type:   CmdDelay,
								Frames: 1,
							})
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
					targetMode.SetHighlightTiles(reachable, color.RGBA{R: 0, G: 127, B: 0, A: 15})
					g.PushMode(targetMode)
				}
			}

			// Attack - A key, triggers targeting mode with equipped weapon range (diamond) on locations with an actor present (once per turn)
			if inpututil.IsKeyJustPressed(ebiten.KeyA) && !GetCombatMemberActed() {
				curMap := GetMap()
				if curMap != nil {
					weaponRange := curMember.GetWeaponRange()
					if weaponRange <= 0 {
						weaponRange = 1
					}
					attackTiles := make(map[image.Point]bool)
					for dx := -weaponRange; dx <= weaponRange; dx++ {
						for dy := -weaponRange; dy <= weaponRange; dy++ {
							adx := dx
							if adx < 0 {
								adx = -adx
							}
							ady := dy
							if ady < 0 {
								ady = -ady
							}
							if adx+ady <= weaponRange {
								tx := curMember.X + dx
								ty := curMember.Y + dy
								if tx >= 0 && tx < curMap.Width && ty >= 0 && ty < curMap.Height {
									if curMap.GetActorAt(tx, ty) != nil {
										attackTiles[image.Pt(tx, ty)] = true
									}
								}
							}
						}
					}
					targetMode := NewTargetMode(
						curMember.X, curMember.Y,
						weaponRange,
						DistanceDiamond,
						func(tx, ty int) bool {
							targetPt := image.Pt(tx, ty)
							if !attackTiles[targetPt] {
								return false
							}
							act := curMap.GetActorAt(tx, ty)
							if act == nil {
								return false
							}
							SetCombatMemberActed(true)
							_ = ExecuteEffectScript("effects/attack.tengo", tx, ty, act.ID, curMember.ID)
							return true
						},
						nil,
					)
					targetMode.SetHighlightTiles(attackTiles, color.RGBA{R: 127, G: 0, B: 0, A: 31})
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

	// Overlay targeting cursor on top of the active party member in combat mode
	if IsInCombat() && g.party != nil && len(g.party.Members) > 0 {
		DrawTargetCursor(screen, scale, m.colorIdx)
	}

	// 2. Draw common UI (party roster area and terminal UI)
	DrawCommonUI(screen, g.assets, g.party, g.terminal)
}
