package solstice

import (
	"image"
	"testing"
)

func TestActorDefAndMapActorOperations(t *testing.T) {
	if _, err := PreloadSpriteDefs(); err != nil {
		t.Fatalf("PreloadSpriteDefs failed: %v", err)
	}

	defs, err := PreloadActorDefs()
	if err != nil {
		t.Fatalf("PreloadActorDefs failed: %v", err)
	}

	if len(defs) == 0 {
		t.Error("Expected loaded actor defs to be non-empty")
	}

	// 1. Test NewActorFromDef
	guard, err := NewActorFromDef("guard-1", "guard", 17, 11)
	if err != nil {
		t.Fatalf("NewActorFromDef failed: %v", err)
	}
	if guard.X != 17 || guard.Y != 11 {
		t.Errorf("Expected guard position (17, 11), got (%d, %d)", guard.X, guard.Y)
	}
	if guard.ID != "guard-1" {
		t.Errorf("Expected guard ID 'guard-1', got %q", guard.ID)
	}

	// 2. Test Map AddActor & RemoveActor
	homeMap, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}
	homeMap.Actors = nil

	homeMap.AddActor(guard)
	if len(homeMap.Actors) != 1 {
		t.Errorf("Expected 1 map actor, got %d", len(homeMap.Actors))
	}

	wizard := NewActor("wizard-1", 10, 10, "wizard")
	homeMap.AddActor(wizard)
	if len(homeMap.Actors) != 2 {
		t.Errorf("Expected 2 map actors, got %d", len(homeMap.Actors))
	}

	// 3. Test GetActorsInArea
	// Area around (17, 11): (15, 10) to (20, 15)
	area1 := image.Rect(15, 10, 20, 15)
	found1 := homeMap.GetActorsInArea(area1)
	if len(found1) != 1 || found1[0] != guard {
		t.Errorf("Expected to find guard in area %v, got %v", area1, found1)
	}

	// Area covering both (10, 10) and (17, 11)
	area2 := image.Rect(0, 0, 30, 30)
	found2 := homeMap.GetActorsInArea(area2)
	if len(found2) != 2 {
		t.Errorf("Expected to find 2 actors in area %v, got %d", area2, len(found2))
	}

	// 4. Test RemoveActor
	removed := homeMap.RemoveActor(guard)
	if !removed {
		t.Error("Expected RemoveActor(guard) to return true")
	}
	if len(homeMap.Actors) != 1 || homeMap.Actors[0] != wizard {
		t.Errorf("Expected only wizard remaining on map, got actors: %v", homeMap.Actors)
	}
}

func TestPartyBlockedByActor(t *testing.T) {
	homeMap, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}

	// Place actor at (16, 15)
	actor := NewActor("blocking-guard", 16, 15, "guard")
	homeMap.AddActor(actor)

	// Party at (15, 15)
	party, err := NewParty(15, 15)
	if err != nil {
		t.Fatalf("NewParty failed: %v", err)
	}

	// Attempt to move right onto (16, 15) where actor is positioned -> should be blocked!
	moved := homeMap.MoveParty(party, 1, 0)
	if moved {
		t.Error("Expected party movement to be blocked by actor at (16, 15), but MoveParty returned true")
	}

	if party.X != 15 || party.Y != 15 {
		t.Errorf("Expected party to remain at (15, 15), got (%d, %d)", party.X, party.Y)
	}
}
