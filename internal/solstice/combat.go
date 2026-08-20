package solstice

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
	inCombat          bool
	combatMemberIndex int
	combatMemberMoved bool
	combatMemberActed bool
	combatActionQueue []CombatAction
)

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
	ClearCombatActions()
	ClearCutScene()
	inCombat = true
}

// StopCombat transitions the game from combat mode back to party mode.
// It removes party member actors from the active map, sets the party's position to the first
// party member's location, and disables combat mode.
func StopCombat() {
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
	ClearCombatActions()
	ClearCutScene()
}

// AdvanceCombatMember advances to the next party member's combat move.
// When all party members have completed their combat moves, it runs the combat AI for all actors on the map
// for one turn and loops back to the first party member.
func AdvanceCombatMember(g *Game) {
	p := GetParty()
	if p == nil || len(p.Members) == 0 {
		return
	}

	combatMemberMoved = false
	combatMemberActed = false
	combatMemberIndex++
	if combatMemberIndex >= len(p.Members) {
		RunCombatAI(g)
		combatMemberIndex = 0
	}
}

// RunCombatAI runs the combat AI for every actor on the map for one turn.
func RunCombatAI(g *Game) {
	m := GetMap()
	if g != nil && g.currentMap != nil {
		m = g.currentMap
	}
	if m == nil {
		return
	}

	m.Turn++

	// Run combat scripts for non-party actors
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

	for _, actor := range m.Actors {
		if actor == nil || isPartyMember(actor.ID) {
			continue
		}
		if actor.CombatScript != "" {
			_ = RunActorAIScript(actor, actor.CombatScript)
		}
	}

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
}
