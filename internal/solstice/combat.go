package solstice

import "image"

// CombatActionType represents the type of a combat action command.
type CombatActionType int

const (
	CombatActMove CombatActionType = iota
	CombatActAttack
	CombatActPass
	CombatActCustom
)

// CombatAction represents an action in the combat action queue.
type CombatAction struct {
	Type         CombatActionType
	ActorID      string
	TargetID     string
	TargetX      int
	TargetY      int
	EffectScript string
	Path         []string
	Execute      func(g *Game)
}

var (
	inCombat           bool
	combatMemberIndex  int
	combatMemberMoved  bool
	combatMemberActed  bool
	combatActionQueue  []CombatAction
	combatFocusActor   *Actor
	combatAIPhase      bool
	combatAIActorIndex int
)

// GetCombatFocusActor returns the actor currently focused by the camera/action during combat.
func GetCombatFocusActor() *Actor {
	return combatFocusActor
}

// SetCombatFocusActor sets the actor currently focused by the camera/action during combat.
func SetCombatFocusActor(a *Actor) {
	combatFocusActor = a
	if inCombat && a != nil {
		UpdateTopViewCenter(image.Pt(a.X, a.Y))
	}
}

// IsCombatAIPhase returns true if non-party enemy AI turns are currently executing.
func IsCombatAIPhase() bool {
	return combatAIPhase
}

// EnqueueCombatAction adds a combat action to the queue.
func EnqueueCombatAction(action CombatAction) {
	combatActionQueue = append(combatActionQueue, action)
}

// ClearCombatActions clears the combat action queue.
func ClearCombatActions() {
	combatActionQueue = nil
}

// HasCombatActions returns true if there are queued combat actions.
func HasCombatActions() bool {
	return len(combatActionQueue) > 0
}

// UpdateCombatActions processes queued combat actions if cutscenes are not currently active.
// Returns true if a combat action was processed or the queue is active, indicating
// normal input handling should be bypassed.
func UpdateCombatActions(g *Game) bool {
	if IsCutSceneActive() {
		return false
	}

	if len(combatActionQueue) == 0 {
		return false
	}

	action := combatActionQueue[0]
	combatActionQueue = combatActionQueue[1:]

	switch action.Type {
	case CombatActMove:
		SetCombatMemberMoved(true)
		EnqueueCutSceneCommand(CutSceneCommand{
			Type:   CmdDelay,
			Frames: 1,
		})
		for _, dir := range action.Path {
			EnqueueCutSceneCommand(CutSceneCommand{
				Type:    CmdMove,
				ActorID: action.ActorID,
				Dir:     dir,
			})
			EnqueueCutSceneCommand(CutSceneCommand{
				Type:   CmdDelay,
				Frames: 1,
			})
		}

	case CombatActAttack:
		SetCombatMemberActed(true)
		script := action.EffectScript
		if script == "" {
			script = "effects/attack.tengo"
		}
		_ = ExecuteEffectScript(script, action.TargetX, action.TargetY, action.TargetID, action.ActorID)

	case CombatActPass:
		EnqueueCutSceneCommand(CutSceneCommand{
			Type:   CmdDelay,
			Frames: 2,
		})
		AdvanceCombatMember(g)

	case CombatActCustom:
		if action.Execute != nil {
			action.Execute(g)
		}
	}

	return true
}

// IsInCombat returns true if combat mode is currently active.
func IsInCombat() bool {
	return inCombat
}

// SetInCombat sets the combat mode active state.
func SetInCombat(v bool) {
	inCombat = v
}

// GetCombatMemberIndex returns the 0-based index of the currently active party member in combat.
func GetCombatMemberIndex() int {
	return combatMemberIndex
}

// SetCombatMemberIndex sets the 0-based index of the currently active party member in combat.
func SetCombatMemberIndex(idx int) {
	combatMemberIndex = idx
	if inCombat {
		p := GetParty()
		if p != nil && idx >= 0 && idx < len(p.Members) {
			UpdateTopViewCenter(image.Pt(p.Members[idx].X, p.Members[idx].Y))
		}
	}
}

// GetCombatMemberMoved returns true if the active party member has already moved this turn.
func GetCombatMemberMoved() bool {
	return combatMemberMoved
}

// SetCombatMemberMoved sets whether the active party member has moved this turn.
func SetCombatMemberMoved(v bool) {
	combatMemberMoved = v
}

// GetCombatMemberActed returns true if the active party member has already performed a non-move, non-pass action this turn.
func GetCombatMemberActed() bool {
	return combatMemberActed
}

// SetCombatMemberActed sets whether the active party member has performed a non-move, non-pass action this turn.
func SetCombatMemberActed(v bool) {
	combatMemberActed = v
}

// StartCombat transitions the game from party mode to combat mode.
// It sets the starting position of each party member to the party's position (allowing them to overlap),
// adds the party member actors to the current active map, and sets the active party member to the first member (index 0).
func StartCombat() {
	p := GetParty()
	m := GetMap()
	if p != nil {
		for i := range p.Members {
			p.Members[i].X = p.X
			p.Members[i].Y = p.Y
			if m != nil {
				if m.GetActorByID(p.Members[i].ID) == nil {
					m.AddActor(&p.Members[i])
				}
			}
		}
	}
	combatMemberIndex = 0
	combatMemberMoved = false
	combatMemberActed = false
	combatAIPhase = false
	combatAIActorIndex = 0
	SetCombatFocusActor(nil)
	ClearCombatActions()
	ClearCutScene()
	inCombat = true

	if p != nil && len(p.Members) > 0 {
		PushViewCenter(image.Pt(p.Members[0].X, p.Members[0].Y))
	} else if p != nil {
		PushViewCenter(image.Pt(p.X, p.Y))
	} else {
		PushViewCenter(image.Pt(16, 16))
	}
}

// StopCombat transitions the game from combat mode back to party mode.
// It removes party member actors from the active map, sets the party's position to the first
// party member's location, and disables combat mode.
func StopCombat() {
	wasInCombat := inCombat
	p := GetParty()
	m := GetMap()
	if p != nil && len(p.Members) > 0 {
		if m != nil {
			for i := range p.Members {
				m.RemoveActorByID(p.Members[i].ID)
			}
		}
		p.X = p.Members[0].X
		p.Y = p.Members[0].Y
	}
	inCombat = false
	combatMemberIndex = 0
	combatMemberMoved = false
	combatMemberActed = false
	combatAIPhase = false
	combatAIActorIndex = 0
	SetCombatFocusActor(nil)
	ClearCombatActions()
	ClearCutScene()

	if wasInCombat {
		PopViewCenter()
	}
	if p != nil {
		SetPartyViewCenter(image.Pt(p.X, p.Y))
	}
}

// AdvanceCombatMember advances to the next party member's combat move.
// When all party members have completed their combat moves, it initiates the enemy AI phase.
func AdvanceCombatMember(g *Game) {
	p := GetParty()
	if p == nil || len(p.Members) == 0 {
		return
	}

	combatMemberMoved = false
	combatMemberActed = false
	combatMemberIndex++
	if combatMemberIndex >= len(p.Members) {
		combatAIPhase = true
		combatAIActorIndex = 0
	} else {
		UpdateTopViewCenter(image.Pt(p.Members[combatMemberIndex].X, p.Members[combatMemberIndex].Y))
	}
}

// CombatStagingEnd updates the view centering on the currently active actor in combat mode.
// If called outside of combat mode, it does nothing.
func CombatStagingEnd() {
	if !IsInCombat() {
		return
	}
	if focus := GetCombatFocusActor(); focus != nil {
		UpdateTopViewCenter(image.Pt(focus.X, focus.Y))
		return
	}
	p := GetParty()
	if p != nil && len(p.Members) > 0 {
		curIdx := GetCombatMemberIndex()
		if curIdx >= len(p.Members) {
			curIdx = 0
		}
		UpdateTopViewCenter(image.Pt(p.Members[curIdx].X, p.Members[curIdx].Y))
	} else if p != nil {
		UpdateTopViewCenter(image.Pt(p.X, p.Y))
	}
}

// UpdateCombatAI steps through non-party actors on the map one at a time during the combat AI phase.
// Returns true if an action or phase advancement occurred.
func UpdateCombatAI(g *Game) bool {
	if !combatAIPhase {
		return false
	}
	if IsCutSceneActive() || HasCombatActions() {
		return false
	}

	m := GetMap()
	if g != nil && g.currentMap != nil {
		m = g.currentMap
	}
	if m == nil {
		combatAIPhase = false
		combatAIActorIndex = 0
		combatMemberIndex = 0
		SetCombatFocusActor(nil)
		return false
	}

	p := GetParty()
	isPartyMember := func(id string) bool {
		if p == nil {
			return false
		}
		for i := range p.Members {
			if p.Members[i].ID == id {
				return true
			}
		}
		return false
	}

	// Find non-party actors in current map actor order
	var enemyActors []*Actor
	for _, actor := range m.Actors {
		if actor != nil && !isPartyMember(actor.ID) {
			enemyActors = append(enemyActors, actor)
		}
	}

	for combatAIActorIndex < len(enemyActors) {
		actor := enemyActors[combatAIActorIndex]
		combatAIActorIndex++

		// Verify actor is still on the map (not killed by a previous actor)
		if m.GetActorByID(actor.ID) == nil {
			continue
		}

		if actor.CombatScript != "" {
			SetCombatFocusActor(actor)
			EnqueueCutSceneCommand(CutSceneCommand{
				Type:   CmdDelay,
				Frames: 2,
			})
			_ = RunActorAIScript(actor, actor.CombatScript)
			return true
		}
	}

	// All non-party actors have acted -> finish round
	combatAIPhase = false
	combatAIActorIndex = 0
	combatMemberIndex = 0
	SetCombatFocusActor(nil)
	if p != nil && len(p.Members) > 0 {
		UpdateTopViewCenter(image.Pt(p.Members[0].X, p.Members[0].Y))
	}

	m.Turn++

	// Advance map timers
	if len(m.Timers) > 0 {
		activeTimers := make([]*MapTimer, 0, len(m.Timers))
		expiredTimers := make([]*MapTimer, 0)

		for _, timer := range m.Timers {
			timer.RemainingTurns--
			if timer.RemainingTurns <= 0 {
				expiredTimers = append(expiredTimers, timer)
			} else {
				activeTimers = append(activeTimers, timer)
			}
		}

		m.Timers = activeTimers

		for _, timer := range expiredTimers {
			_ = ExecuteScriptWithGlobals(timer.ScriptPath, timer.Globals)
		}
	}

	return true
}

// RunCombatAI runs the combat AI for every actor on the map for one turn.
func RunCombatAI(g *Game) {
	combatAIPhase = true
	combatAIActorIndex = 0
	for combatAIPhase {
		UpdateCombatAI(g)
		for IsCutSceneActive() || HasCombatActions() {
			if IsCutSceneActive() {
				SetAnimFrame(GetAnimFrame() + 1)
				UpdateCutScene(g)
			}
			if !IsCutSceneActive() && HasCombatActions() {
				UpdateCombatActions(g)
			}
		}
	}
}
