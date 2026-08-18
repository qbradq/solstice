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
// It sets the starting position of each party member to the party's position (allowing them to overlap)
// and sets the active party member to the first party member (index 0).
func StartCombat() {
	p := GetParty()
	if p != nil {
		for i := range p.Members {
			p.Members[i].X = p.X
			p.Members[i].Y = p.Y
		}
	}
	combatMemberIndex = 0
	inCombat = true
}

// StopCombat transitions the game from combat mode back to party mode.
// It sets the party's position to the first party member's location and disables combat mode.
func StopCombat() {
	p := GetParty()
	if p != nil && len(p.Members) > 0 {
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

// RunCombatAI runs the combat AI for every actor on the map for one turn (stubbed for now).
func RunCombatAI(g *Game) {
	if g != nil && g.currentMap != nil {
		g.currentMap.AdvanceTurn()
	} else if m := GetMap(); m != nil {
		m.AdvanceTurn()
	}
}
