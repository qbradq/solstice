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

	// 1. Create a party with 0 members (Spirit Mode)
	party, err := NewParty(10, 20)
	if err != nil {
		t.Fatalf("NewParty(10, 20) failed: %v", err)
	}

	SetParty(party)
	if GetParty() != party {
		t.Error("GetParty() did not return the set global party")
	}

	if !party.IsSpiritMode() {
		t.Error("Expected 0-member party to be in Spirit Mode")
	}

	// Verify party-spirit-mode sprite definition copy (Tile: 372, Animated: true, Frames: 4)
	if party.Tile != 372 {
		t.Errorf("Expected party Tile to be 372, got %d", party.Tile)
	}
	if !party.Animated || party.Frames != 4 {
		t.Errorf("Expected Animated=true and Frames=4, got Animated=%v, Frames=%d", party.Animated, party.Frames)
	}

	// 2. Add up to 4 members
	for i := 1; i <= 4; i++ {
		member := PartyMember{
			Entity: Entity{
				Name: fmt.Sprintf("Hero %d", i),
				X:    10 + i,
				Y:    20 + i,
			},
		}
		if err := party.AddMember(member); err != nil {
			t.Fatalf("AddMember failed on member %d: %v", i, err)
		}
	}

	if party.IsSpiritMode() {
		t.Error("Expected 4-member party not to be in Spirit Mode")
	}

	if len(party.Members) != 4 {
		t.Errorf("Expected 4 members, got %d", len(party.Members))
	}

	// 3. Attempting to add 5th member should fail
	extraMember := PartyMember{
		Entity: Entity{Name: "Hero 5"},
	}
	if err := party.AddMember(extraMember); err == nil {
		t.Error("Expected error adding 5th member, got nil")
	}

	// 4. Test NewParty with > 4 members initially
	fiveMembers := make([]PartyMember, 5)
	if _, err := NewParty(0, 0, fiveMembers...); err == nil {
		t.Error("Expected error creating NewParty with 5 members, got nil")
	}

	// 5. Test RemoveMember
	if err := party.RemoveMember(0); err != nil {
		t.Fatalf("RemoveMember(0) failed: %v", err)
	}
	if len(party.Members) != 3 {
		t.Errorf("Expected 3 members after removal, got %d", len(party.Members))
	}
}
