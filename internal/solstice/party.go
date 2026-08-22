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
	if defaultGame != nil {
		defaultGame.party = p
	}
}

// Party represents the top-level player entity in the world.
// It embeds Entity for positioning and graphical representation when controlling
// the party as a whole. It contains between 0 and 4 party member actors.
type Party struct {
	Entity
	WorldX  int
	WorldY  int
	Members []Actor
}

// NewParty creates a new Party with the specified position and initial actor members.
// Sets party sprite to "party_standing" when not in spirit mode, and "party_spirit_mode" when in spirit mode.
// Hard codes the party's starting world map position to (38, 103).
func NewParty(x, y int, members ...Actor) (*Party, error) {
	if len(members) > MaxPartyMembers {
		return nil, fmt.Errorf("cannot create party with %d members (maximum allowed is %d)", len(members), MaxPartyMembers)
	}

	memberSlice := make([]Actor, len(members))
	copy(memberSlice, members)

	p := &Party{
		Entity: Entity{
			Name: "Party",
			X:    x,
			Y:    y,
		},
		WorldX:  38,
		WorldY:  103,
		Members: memberSlice,
	}

	p.UpdateSpriteDef()
	return p, nil
}

// IsSpiritMode returns true if the party has 0 members.
// In game lore, a party with 0 members is guided by the spirit of a long-lost king.
func (p *Party) IsSpiritMode() bool {
	return p == nil || len(p.Members) == 0
}

// GetSpriteDef returns the active SpriteDef for the party:
// - "party_spirit_mode" when in spirit mode (0 members).
// - When not in spirit mode, if the tile the party is standing on defines a non-empty "party_sprite" property, use that sprite.
// - Otherwise, default to "party_standing".
func (p *Party) GetSpriteDef() SpriteDef {
	if p == nil {
		sd, _ := GetSpriteDef("party_spirit_mode")
		return sd
	}

	if p.IsSpiritMode() {
		if sd, ok := GetSpriteDef("party_spirit_mode"); ok {
			return sd
		}
		return p.SpriteDef
	}

	// When not in spirit mode, check if the tile the party is standing on has a party_sprite property
	if m := GetMap(); m != nil {
		tileIdx := m.GetTile(p.X, p.Y)
		props := GetTileProperties(tileIdx)
		if props.PartySprite != "" {
			if sd, ok := GetSpriteDef(props.PartySprite); ok {
				return sd
			}
		}
	}

	if sd, ok := GetSpriteDef("party_standing"); ok {
		return sd
	}
	return p.SpriteDef
}

// UpdateSpriteDef syncs p.SpriteDef with the active spirit mode state.
func (p *Party) UpdateSpriteDef() {
	if p != nil {
		p.SpriteDef = p.GetSpriteDef()
	}
}

// AddMember adds an Actor to the party.
// Returns an error if the party already contains the maximum of 4 members.
func (p *Party) AddMember(member Actor) error {
	if len(p.Members) >= MaxPartyMembers {
		return fmt.Errorf("cannot add member: party is full (max %d members)", MaxPartyMembers)
	}
	p.Members = append(p.Members, member)
	p.UpdateSpriteDef()
	return nil
}

// RemoveMember removes the party member actor at the specified index.
func (p *Party) RemoveMember(index int) error {
	if index < 0 || index >= len(p.Members) {
		return fmt.Errorf("member index %d out of bounds (len %d)", index, len(p.Members))
	}
	p.Members = append(p.Members[:index], p.Members[index+1:]...)
	p.UpdateSpriteDef()
	return nil
}

// GetMember returns a pointer to the party member with the given ID, or nil if not found.
func (p *Party) GetMember(id string) *Actor {
	if p == nil {
		return nil
	}
	for i := range p.Members {
		if p.Members[i].ID == id {
			return &p.Members[i]
		}
	}
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

	// Pass / Wait turn: Space, Period, Keypad 5
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyPeriod) || inpututil.IsKeyJustPressed(ebiten.KeyKP5) {
		if m != nil {
			m.AdvanceTurn()
		}
		return
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
