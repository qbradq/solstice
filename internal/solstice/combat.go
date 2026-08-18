package solstice

var (
	inCombat          bool
	combatMemberIndex int
)

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
}

// AdvanceCombatMember advances to the next party member's combat move.
// When all party members have completed their combat moves, it runs the combat AI for all actors on the map
// for one turn and loops back to the first party member.
func AdvanceCombatMember(g *Game) {
	p := GetParty()
	if p == nil || len(p.Members) == 0 {
		return
	}

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
