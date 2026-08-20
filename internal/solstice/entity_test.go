package solstice

import "testing"

func TestPreloadSpriteDefs(t *testing.T) {
	defs, err := PreloadSpriteDefs()
	if err != nil {
		t.Fatalf("PreloadSpriteDefs failed: %v", err)
	}

	partyStanding, ok := defs["party_standing"]
	if !ok {
		t.Fatal("Expected 'party_standing' sprite def in loaded map")
	}

	if partyStanding.Tile != 332 {
		t.Errorf("Expected tile 332, got %d", partyStanding.Tile)
	}
	if !partyStanding.Animated {
		t.Error("Expected animated to be true")
	}
	if partyStanding.Frames != 4 {
		t.Errorf("Expected 4 frames, got %d", partyStanding.Frames)
	}
}
