package solstice

import (
	"fmt"
)

// MaxPartyMembers defines the maximum number of members a party can hold.
const MaxPartyMembers = 4

// PartyMember represents an individual member of the party.
// It embeds Entity for map positioning and individual graphical representation.
type PartyMember struct {
	Entity
}

// Party represents the top-level player entity in the world.
// It embeds Entity for positioning and graphical representation when controlling
// the party as a whole. It contains between 0 and 4 party members.
type Party struct {
	Entity
	Members []PartyMember
}

// NewParty creates a new Party with the specified position and initial members.
// It copies the "party-standing" SpriteDef for the party's Entity.
func NewParty(x, y int, members ...PartyMember) (*Party, error) {
	if len(members) > MaxPartyMembers {
		return nil, fmt.Errorf("cannot create party with %d members (maximum allowed is %d)", len(members), MaxPartyMembers)
	}

	spriteDef, ok := GetSpriteDef("party-standing")
	if !ok {
		// Fallback to empty SpriteDef if "party-standing" is not found
		spriteDef = SpriteDef{}
	}

	memberSlice := make([]PartyMember, len(members))
	copy(memberSlice, members)

	p := &Party{
		Entity: Entity{
			SpriteDef: spriteDef, // Always copied by value
			Name:      "Party",
			X:         x,
			Y:         y,
		},
		Members: memberSlice,
	}

	return p, nil
}

// IsSpiritMode returns true if the party has 0 members.
// In game lore, a party with 0 members is guided by the spirit of a long-lost king.
func (p *Party) IsSpiritMode() bool {
	return p == nil || len(p.Members) == 0
}

// AddMember adds a PartyMember to the party.
// Returns an error if the party already contains the maximum of 4 members.
func (p *Party) AddMember(member PartyMember) error {
	if len(p.Members) >= MaxPartyMembers {
		return fmt.Errorf("cannot add member: party is full (max %d members)", MaxPartyMembers)
	}
	p.Members = append(p.Members, member)
	return nil
}

// RemoveMember removes the party member at the specified index.
func (p *Party) RemoveMember(index int) error {
	if index < 0 || index >= len(p.Members) {
		return fmt.Errorf("member index %d out of bounds (len %d)", index, len(p.Members))
	}
	p.Members = append(p.Members[:index], p.Members[index+1:]...)
	return nil
}
