package solstice

import (
	"fmt"
	"testing"
)

func TestPartySpiritModeAndMembers(t *testing.T) {
	_, err := PreloadSpriteDefs()
	if err != nil {
		t.Fatalf("PreloadSpriteDefs failed: %v", err)
	}

	_, err = PreloadTileSet()
	if err != nil {
		t.Fatalf("PreloadTileSet failed: %v", err)
	}

	m, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}
	SetMap(m)

	// 1. Create a party with 0 members (Spirit Mode)
	party, err := NewParty(15, 15)
	if err != nil {
		t.Fatalf("NewParty(15, 15) failed: %v", err)
	}

	SetParty(party)
	if GetParty() != party {
		t.Error("GetParty() did not return the set global party")
	}

	if !party.IsSpiritMode() {
		t.Error("Expected 0-member party to be in Spirit Mode")
	}

	// Spirit mode sprite: "party-spirit-mode" (Tile: 372)
	sdSpirit := party.GetSpriteDef()
	if sdSpirit.Tile != 372 {
		t.Errorf("Expected spirit mode party Tile to be 372, got %d", sdSpirit.Tile)
	}

	// 2. Add member (non-spirit mode) -> "party-standing" (Tile: 332)
	actor := *NewActor("kevin", 15, 15, "warrior")
	if err := party.AddMember(actor); err != nil {
		t.Fatalf("AddMember failed: %v", err)
	}

	if party.IsSpiritMode() {
		t.Error("Expected party with 1 member NOT to be in Spirit Mode")
	}

	sdStanding := party.GetSpriteDef()
	if sdStanding.Tile != 332 {
		t.Errorf("Expected non-spirit mode party Tile to be 332, got %d", sdStanding.Tile)
	}

	// 3. Test tile with party_sprite property override (e.g. tile 144 has party_sprite = "party-sitting-north", Tile: 304)
	m.SetTile(15, 15, 144)
	sdSitting := party.GetSpriteDef()
	if sdSitting.Tile != 304 {
		t.Errorf("Expected party Tile standing on tile 144 to be 304 (party-sitting-north), got %d", sdSitting.Tile)
	}

	// Reset tile
	m.SetTile(15, 15, 4)

	// 4. Fill up to 4 members
	for i := 2; i <= 4; i++ {
		act := *NewActor(fmt.Sprintf("hero-%d", i), 15+i, 15+i, "warrior")
		if err := party.AddMember(act); err != nil {
			t.Fatalf("Failed to add member %d: %v", i, err)
		}
	}

	// Adding 5th member should fail
	if err := party.AddMember(*NewActor("overflow", 0, 0, "warrior")); err == nil {
		t.Error("Expected error when adding 5th member to full party")
	}

	// Remove member back to 0 members (Spirit Mode)
	for len(party.Members) > 0 {
		if err := party.RemoveMember(0); err != nil {
			t.Fatalf("RemoveMember failed: %v", err)
		}
	}

	if !party.IsSpiritMode() {
		t.Error("Expected party with 0 members to return to Spirit Mode")
	}

	if party.GetSpriteDef().Tile != 372 {
		t.Errorf("Expected party Tile to return to 372 (spirit mode), got %d", party.GetSpriteDef().Tile)
	}
}
