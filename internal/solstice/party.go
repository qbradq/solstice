package solstice

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// MaxPartyMembers defines the maximum number of members a party can hold.
const MaxPartyMembers = 4

var defaultParty *Party

// GetParty returns the global party instance.
func GetParty() *Party {
	return defaultParty
}

// SetParty sets the global party instance.
func SetParty(p *Party) {
	defaultParty = p
}

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
// It copies the "party-spirit-mode" SpriteDef for the party's Entity.
func NewParty(x, y int, members ...PartyMember) (*Party, error) {
	if len(members) > MaxPartyMembers {
		return nil, fmt.Errorf("cannot create party with %d members (maximum allowed is %d)", len(members), MaxPartyMembers)
	}

	spriteDef, ok := GetSpriteDef("party-spirit-mode")
	if !ok {
		// Fallback if "party-spirit-mode" is not found in loaded sprite defs
		spriteDef = SpriteDef{
			Tile:     372,
			Animated: true,
			Frames:   4,
		}
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

// HandleInput processes movement inputs (WASD, Arrow keys, VI-style HJKL).
// Movement is performed using m.MoveParty, enforcing walkable terrain rules.
func (p *Party) HandleInput(m *Map) {
	if p == nil {
		return
	}

	dx, dy := 0, 0

	// Up: W, ArrowUp, K
	if inpututil.IsKeyJustPressed(ebiten.KeyW) || inpututil.IsKeyJustPressed(ebiten.KeyUp) || inpututil.IsKeyJustPressed(ebiten.KeyK) {
		dy = -1
	}
	// Down: S, ArrowDown, J
	if inpututil.IsKeyJustPressed(ebiten.KeyS) || inpututil.IsKeyJustPressed(ebiten.KeyDown) || inpututil.IsKeyJustPressed(ebiten.KeyJ) {
		dy = 1
	}
	// Left: A, ArrowLeft, H
	if inpututil.IsKeyJustPressed(ebiten.KeyA) || inpututil.IsKeyJustPressed(ebiten.KeyLeft) || inpututil.IsKeyJustPressed(ebiten.KeyH) {
		dx = -1
	}
	// Right: D, ArrowRight, L
	if inpututil.IsKeyJustPressed(ebiten.KeyD) || inpututil.IsKeyJustPressed(ebiten.KeyRight) || inpututil.IsKeyJustPressed(ebiten.KeyL) {
		dx = 1
	}

	if dx != 0 || dy != 0 {
		if m != nil {
			m.MoveParty(p, dx, dy)
		} else {
			p.X += dx
			p.Y += dy
		}
	}
}
