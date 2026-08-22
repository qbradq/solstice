package solstice

import (
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestViewCenterStack(t *testing.T) {
	// Reset any state from previous tests
	StopCombat()
	party, err := NewParty(10, 20)
	if err != nil {
		t.Fatalf("NewParty failed: %v", err)
	}
	SetParty(party)
	ResetViewCenterStack()

	center := GetViewCenter()
	if center.X != 10 || center.Y != 20 {
		t.Errorf("Expected view center (10, 20), got (%d, %d)", center.X, center.Y)
	}

	stack := GetViewCenterStack()
	if len(stack) != 1 {
		t.Fatalf("Expected stack length 1, got %d", len(stack))
	}
	if stack[0] != image.Pt(10, 20) {
		t.Errorf("Expected stack[0] (10, 20), got %+v", stack[0])
	}

	// 2. Update party position without pushing/popping (remains at bottom of stack)
	party.X = 12
	party.Y = 22
	SetPartyViewCenter(image.Pt(party.X, party.Y))

	center = GetViewCenter()
	if center.X != 12 || center.Y != 22 {
		t.Errorf("Expected view center (12, 22), got (%d, %d)", center.X, center.Y)
	}
	stack = GetViewCenterStack()
	if len(stack) != 1 || stack[0] != image.Pt(12, 22) {
		t.Errorf("Expected stack of len 1 with (12, 22), got %+v", stack)
	}

	// 3. Popping cannot remove the bottom party location
	popped := PopViewCenter()
	if popped != image.Pt(12, 22) {
		t.Errorf("Expected PopViewCenter to return (12, 22), got %+v", popped)
	}
	stack = GetViewCenterStack()
	if len(stack) != 1 || stack[0] != image.Pt(12, 22) {
		t.Errorf("Expected stack to still have len 1 with (12, 22), got %+v", stack)
	}

	// 4. Test Entering Combat
	m1, _ := NewActorFromDef("hero1", "kevin", 0, 0)
	m2, _ := NewActorFromDef("hero2", "wizard", 0, 0)
	partyWithMembers, err := NewParty(15, 15, *m1, *m2)
	if err != nil {
		t.Fatalf("NewParty failed: %v", err)
	}
	SetParty(partyWithMembers)
	StartCombat()
	if !IsInCombat() {
		t.Fatal("Expected IsInCombat to be true")
	}

	// Move party member 1 to a different position in combat
	partyWithMembers.Members[1].X = 18
	partyWithMembers.Members[1].Y = 19

	// Combat should place active actor on top of stack
	stack = GetViewCenterStack()
	if len(stack) != 2 {
		t.Fatalf("Expected stack length 2 in combat, got %d", len(stack))
	}
	if stack[0] != image.Pt(15, 15) || stack[1] != image.Pt(15, 15) {
		t.Errorf("Expected stack [ (15,15), (15,15) ], got %+v", stack)
	}

	// Change active combat member
	SetCombatMemberIndex(1)
	center = GetViewCenter()
	if center.X != 18 || center.Y != 19 {
		t.Errorf("Expected combat active member view center (18, 19), got (%d, %d)", center.X, center.Y)
	}
	stack = GetViewCenterStack()
	if len(stack) != 2 || stack[1] != image.Pt(18, 19) {
		t.Errorf("Expected stack len 2 with top (18, 19), got %+v", stack)
	}

	// Focus enemy actor during combat AI
	enemy := NewActor("orc1", 25, 30, "orc")
	SetCombatFocusActor(enemy)
	center = GetViewCenter()
	if center.X != 25 || center.Y != 30 {
		t.Errorf("Expected enemy focus view center (25, 30), got (%d, %d)", center.X, center.Y)
	}

	// 5. Test TargetMode inside Combat
	game := &Game{
		party:    partyWithMembers,
		mapScale: 2,
	}
	targetMode := NewTargetMode(18, 19, 3, DistanceDiamond, nil, nil)
	game.PushMode(targetMode)

	stack = GetViewCenterStack()
	if len(stack) != 3 {
		t.Fatalf("Expected stack length 3 in combat + targeting mode, got %d", len(stack))
	}
	if stack[2] != image.Pt(18, 19) {
		t.Errorf("Expected target cursor top (18, 19), got %+v", stack[2])
	}

	// Move cursor
	targetMode.cursorX = 20
	targetMode.cursorY = 21
	UpdateTopViewCenter(image.Pt(targetMode.cursorX, targetMode.cursorY))
	center = GetViewCenter()
	if center.X != 20 || center.Y != 21 {
		t.Errorf("Expected moved cursor view center (20, 21), got (%d, %d)", center.X, center.Y)
	}

	// Pop target mode
	game.PopMode()
	stack = GetViewCenterStack()
	if len(stack) != 2 {
		t.Fatalf("Expected stack length 2 after popping target mode, got %d", len(stack))
	}

	// 6. Test Exiting Combat
	StopCombat()
	if IsInCombat() {
		t.Fatal("Expected IsInCombat to be false")
	}
	stack = GetViewCenterStack()
	if len(stack) != 1 {
		t.Fatalf("Expected stack length 1 after stopping combat, got %d", len(stack))
	}
}

func TestMapDrawCenteredPreservesArbitraryVisibility(t *testing.T) {
	assets, err := LoadAssets()
	if err != nil {
		t.Fatalf("LoadAssets failed: %v", err)
	}
	m, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}

	party, err := NewParty(15, 15)
	if err != nil {
		t.Fatalf("NewParty failed: %v", err)
	}
	SetParty(party)

	screen := ebiten.NewImage(640, 360)

	// DrawCentered uses top of stack for view center
	m.DrawCentered(screen, assets, party, 2)

	// DrawCenteredAt allows arbitrary view center and arbitrary visibility center
	highlights := map[image.Point]bool{{X: 20, Y: 20}: true}
	m.DrawCenteredAt(screen, assets, party, 2, 20, 20, 15, 15, highlights, color.RGBA{R: 0, G: 127, B: 0, A: 15})
}
