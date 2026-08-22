package solstice

import (
	"strings"
	"testing"

	"github.com/d5/tengo/v2"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestCombatModeTransitions(t *testing.T) {
	// Create a party with 2 members at (10, 15)
	m1, err := NewActorFromDef("hero1", "kevin", 0, 0)
	if err != nil {
		t.Fatalf("NewActorFromDef hero1 failed: %v", err)
	}
	m2, err := NewActorFromDef("hero2", "wizard", 0, 0)
	if err != nil {
		t.Fatalf("NewActorFromDef hero2 failed: %v", err)
	}

	party, err := NewParty(10, 15, *m1, *m2)
	if err != nil {
		t.Fatalf("NewParty failed: %v", err)
	}
	SetParty(party)

	homeMap, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}
	SetMap(homeMap)

	// Ensure initially not in combat
	SetInCombat(false)
	SetCombatMemberIndex(0)

	// 1. Transition to Combat Mode
	StartCombat()
	if !IsInCombat() {
		t.Fatal("Expected IsInCombat to be true after StartCombat")
	}
	if GetCombatMemberIndex() != 0 {
		t.Errorf("Expected current combat member index 0, got %d", GetCombatMemberIndex())
	}
	if party.Members[0].X != 10 || party.Members[0].Y != 15 {
		t.Errorf("Expected member 0 at (10, 15), got (%d, %d)", party.Members[0].X, party.Members[0].Y)
	}
	if party.Members[1].X != 10 || party.Members[1].Y != 15 {
		t.Errorf("Expected member 1 at (10, 15), got (%d, %d)", party.Members[1].X, party.Members[1].Y)
	}

	// Verify party members were added to the map
	if homeMap.GetActorByID("hero1") == nil {
		t.Errorf("Expected hero1 to be added to map actors during combat")
	}
	if homeMap.GetActorByID("hero2") == nil {
		t.Errorf("Expected hero2 to be added to map actors during combat")
	}

	// Move member 0 to (12, 18) and member 1 to (14, 19)
	party.Members[0].X = 12
	party.Members[0].Y = 18
	party.Members[1].X = 14
	party.Members[1].Y = 19

	// 2. Transition back to Party Mode
	StopCombat()
	if IsInCombat() {
		t.Fatal("Expected IsInCombat to be false after StopCombat")
	}
	// Party position should be set to first party member's location (12, 18)
	if party.X != 12 || party.Y != 18 {
		t.Errorf("Expected party position (12, 18) after StopCombat, got (%d, %d)", party.X, party.Y)
	}

	// Verify party members were removed from the map
	if homeMap.GetActorByID("hero1") != nil {
		t.Errorf("Expected hero1 to be removed from map actors after StopCombat")
	}
	if homeMap.GetActorByID("hero2") != nil {
		t.Errorf("Expected hero2 to be removed from map actors after StopCombat")
	}
}

func TestCombatTurnAdvancement(t *testing.T) {
	homeMap, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}
	SetMap(homeMap)

	m1, _ := NewActorFromDef("hero1", "kevin", 0, 0)
	m2, _ := NewActorFromDef("hero2", "wizard", 0, 0)
	m3, _ := NewActorFromDef("hero3", "guard", 0, 0)

	party, err := NewParty(5, 5, *m1, *m2, *m3)
	if err != nil {
		t.Fatalf("NewParty failed: %v", err)
	}
	SetParty(party)

	game := &Game{
		currentMap: homeMap,
		party:      party,
	}

	StartCombat()
	initialTurn := homeMap.Turn

	// Member 0 takes a move and action
	if GetCombatMemberIndex() != 0 {
		t.Fatalf("Expected member index 0, got %d", GetCombatMemberIndex())
	}
	if GetCombatMemberMoved() {
		t.Errorf("Expected CombatMemberMoved to be false initially")
	}
	if GetCombatMemberActed() {
		t.Errorf("Expected CombatMemberActed to be false initially")
	}
	SetCombatMemberMoved(true)
	SetCombatMemberActed(true)
	if !GetCombatMemberMoved() || !GetCombatMemberActed() {
		t.Errorf("Expected CombatMemberMoved and CombatMemberActed to be true after setting")
	}
	AdvanceCombatMember(game)
	if GetCombatMemberMoved() || GetCombatMemberActed() {
		t.Errorf("Expected CombatMemberMoved and CombatMemberActed to reset to false on AdvanceCombatMember")
	}

	// Member 1 takes a move
	if GetCombatMemberIndex() != 1 {
		t.Fatalf("Expected member index 1, got %d", GetCombatMemberIndex())
	}
	SetCombatMemberMoved(true)
	SetCombatMemberActed(true)
	AdvanceCombatMember(game)
	if GetCombatMemberMoved() || GetCombatMemberActed() {
		t.Errorf("Expected CombatMemberMoved and CombatMemberActed to reset to false on AdvanceCombatMember")
	}

	// Member 2 takes a move
	if GetCombatMemberIndex() != 2 {
		t.Fatalf("Expected member index 2, got %d", GetCombatMemberIndex())
	}
	if homeMap.Turn != initialTurn {
		t.Errorf("Map turn should not advance before all members move")
	}

	// Final member moves -> AI runs, map turn advances, member index resets to 0
	AdvanceCombatMember(game)
	if IsCombatAIPhase() {
		RunCombatAI(game)
	}
	if GetCombatMemberIndex() != 0 {
		t.Errorf("Expected member index to reset to 0, got %d", GetCombatMemberIndex())
	}
	if homeMap.Turn != initialTurn+1 {
		t.Errorf("Expected map turn to advance from %d to %d, got %d", initialTurn, initialTurn+1, homeMap.Turn)
	}
}

func TestCombatDrawing(t *testing.T) {
	assets, err := LoadAssets()
	if err != nil {
		t.Fatalf("LoadAssets failed: %v", err)
	}
	homeMap, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}

	m1, _ := NewActorFromDef("hero1", "kevin", 0, 0)
	m2, _ := NewActorFromDef("hero2", "wizard", 0, 0)
	party, err := NewParty(15, 15, *m1, *m2)
	if err != nil {
		t.Fatalf("NewParty failed: %v", err)
	}
	SetParty(party)

	StartCombat()
	defer StopCombat()

	// Party members at different positions
	party.Members[0].X = 15
	party.Members[0].Y = 15
	party.Members[1].X = 16
	party.Members[1].Y = 15

	screen := ebiten.NewImage(640, 360)
	// Render at scale 2 and scale 1 in combat mode (active member 0 at 15, 15)
	SetCombatMemberIndex(0)
	homeMap.DrawCentered(screen, assets, party, 2)
	homeMap.DrawCentered(screen, assets, party, 1)

	// Advance to member 1 (at 16, 15) and render again
	SetCombatMemberIndex(1)
	homeMap.DrawCentered(screen, assets, party, 2)
	homeMap.DrawCentered(screen, assets, party, 1)

	// Test TargetMode draws centered on cursor position (e.g. cursor at 18, 17)
	targetMode := NewTargetMode(16, 15, 5, DistanceDiamond, nil, nil)
	targetMode.cursorX = 18
	targetMode.cursorY = 17
	game := &Game{
		assets:     assets,
		terminal:   NewTerminal(),
		currentMap: homeMap,
		party:      party,
		mapScale:   2,
	}
	targetMode.Draw(game, screen)
	game.mapScale = 1
	targetMode.Draw(game, screen)

	// Test MainMode.Draw overlays targeting cursor in combat mode
	mainMode := NewMainMode()
	mainMode.Draw(game, screen)
	game.mapScale = 2
	mainMode.Draw(game, screen)
}

func TestCombatTengoScriptFunctions(t *testing.T) {
	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}

	m1, _ := NewActorFromDef("hero1", "kevin", 0, 0)
	party, _ := NewParty(20, 25, *m1)
	SetParty(party)

	StopCombat()

	scriptSrc := `
game := import("game")
game.start_combat()
`
	s := tengo.NewScript([]byte(scriptSrc))
	s.SetImports(moduleMap)
	c, err := s.Compile()
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}
	if err := c.Run(); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if !IsInCombat() {
		t.Fatal("Expected IsInCombat to be true after game.start_combat()")
	}

	// Move member 0 during combat
	party.Members[0].X = 22
	party.Members[0].Y = 28

	scriptSrc2 := `
game := import("game")
game.stop_combat()
`
	s2 := tengo.NewScript([]byte(scriptSrc2))
	s2.SetImports(moduleMap)
	c2, err := s2.Compile()
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}
	if err := c2.Run(); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if IsInCombat() {
		t.Fatal("Expected IsInCombat to be false after game.stop_combat()")
	}
	if party.X != 22 || party.Y != 28 {
		t.Errorf("Expected party position (22, 28), got (%d, %d)", party.X, party.Y)
	}
}

func TestCombatAutoAdvanceWhenMovedAndActed(t *testing.T) {
	homeMap, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}
	SetMap(homeMap)

	m1, _ := NewActorFromDef("hero1", "kevin", 0, 0)
	m2, _ := NewActorFromDef("hero2", "wizard", 0, 0)
	party, err := NewParty(5, 5, *m1, *m2)
	if err != nil {
		t.Fatalf("NewParty failed: %v", err)
	}
	SetParty(party)

	game := &Game{
		currentMap: homeMap,
		party:      party,
	}

	StartCombat()
	defer StopCombat()
	ClearCutScene()

	mainMode := &MainMode{}

	// Initially at member 0, neither moved nor acted
	if GetCombatMemberIndex() != 0 {
		t.Fatalf("Expected member 0, got %d", GetCombatMemberIndex())
	}

	// Moved only -> should not auto-advance
	SetCombatMemberMoved(true)
	_ = mainMode.Update(game)
	if GetCombatMemberIndex() != 0 {
		t.Fatalf("Expected member to remain 0 when only moved, got %d", GetCombatMemberIndex())
	}

	// Acted as well -> enqueues CombatActPass on update, which processes and auto-advances to member 1 with 2 frame delay
	SetCombatMemberActed(true)
	_ = mainMode.Update(game) // Enqueues pass combat action
	_ = mainMode.Update(game) // Processes pass combat action, queues 2 frame delay, advances member
	if GetCombatMemberIndex() != 1 {
		t.Fatalf("Expected member to auto-advance to 1, got %d", GetCombatMemberIndex())
	}
	if GetCombatMemberMoved() || GetCombatMemberActed() {
		t.Errorf("Expected moved and acted to be reset for member 1")
	}
	if !IsCutSceneActive() {
		t.Errorf("Expected 2-frame cutscene delay to be active after auto-pass")
	}
	// Drain the 2-frame delay
	for IsCutSceneActive() {
		SetAnimFrame(GetAnimFrame() + 1)
		_ = mainMode.Update(game)
	}

	// But if another cutscene is active, should NOT process combat action until cutscene is finished
	SetCombatMemberMoved(true)
	SetCombatMemberActed(true)
	EnqueueCutSceneCommand(CutSceneCommand{
		Type:   CmdDelay,
		Frames: 3,
	})

	// Frame 1 of cutscene: cutscene runs first
	_ = mainMode.Update(game)
	if GetCombatMemberIndex() != 1 {
		t.Fatalf("Expected member to remain 1 while cutscene is active, got %d", GetCombatMemberIndex())
	}

	// Increment animation frame so cutscene delay finishes
	SetAnimFrame(GetAnimFrame() + 3)
	_ = mainMode.Update(game) // Cutscene finishes
	_ = mainMode.Update(game) // Auto-pass enqueued
	_ = mainMode.Update(game) // Auto-pass executed (queues 2-frame delay and enters AI phase)

	for IsCutSceneActive() || IsCombatAIPhase() || HasCombatActions() {
		SetAnimFrame(GetAnimFrame() + 1)
		_ = mainMode.Update(game)
	}

	if GetCombatMemberIndex() != 0 {
		t.Fatalf("Expected member to auto-advance back to 0 (after AI turn), got %d", GetCombatMemberIndex())
	}
}

func TestCombatActionQueueSystem(t *testing.T) {
	homeMap, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}
	SetMap(homeMap)

	m1, _ := NewActorFromDef("hero1", "kevin", 0, 0)
	party, err := NewParty(5, 5, *m1)
	if err != nil {
		t.Fatalf("NewParty failed: %v", err)
	}
	SetParty(party)

	game := &Game{
		currentMap: homeMap,
		party:      party,
	}

	StartCombat()
	defer StopCombat()

	if HasCombatActions() {
		t.Error("Expected combat action queue to be empty initially")
	}

	// 1. Test CombatActMove
	EnqueueCombatAction(CombatAction{
		Type:    CombatActMove,
		ActorID: "hero1",
		Path:    []string{"e", "e"},
		TargetX: 7,
		TargetY: 5,
	})

	if !HasCombatActions() {
		t.Error("Expected combat action to be in queue")
	}
	if !GetCombatMemberMoved() {
		t.Error("Expected CombatMemberMoved to be true immediately after enqueueing CombatActMove")
	}
	if !IsCutSceneActive() {
		t.Error("Expected cut scene move commands to be queued immediately by EnqueueCombatAction")
	}

	// Drain cut scenes
	for IsCutSceneActive() {
		SetAnimFrame(GetAnimFrame() + 1)
		UpdateCutScene(game)
	}

	if !UpdateCombatActions(game) {
		t.Error("Expected UpdateCombatActions to return true")
	}

	if party.Members[0].X != 7 || party.Members[0].Y != 5 {
		t.Errorf("Expected hero1 to move to (7, 5), got (%d, %d)", party.Members[0].X, party.Members[0].Y)
	}

	// 2. Test CombatActCustom
	customExecuted := false
	EnqueueCombatAction(CombatAction{
		Type: CombatActCustom,
		Execute: func(g *Game) {
			customExecuted = true
		},
	})
	if !UpdateCombatActions(game) {
		t.Error("Expected UpdateCombatActions to return true for custom action")
	}
	if !customExecuted {
		t.Error("Expected custom combat action to execute")
	}

	// 3. Test that CombatActions do not process while cutscenes are active
	EnqueueCutSceneCommand(CutSceneCommand{
		Type:   CmdDelay,
		Frames: 2,
	})
	actionExecutedWhileCutScene := false
	EnqueueCombatAction(CombatAction{
		Type: CombatActCustom,
		Execute: func(g *Game) {
			actionExecutedWhileCutScene = true
		},
	})

	if UpdateCombatActions(game) {
		t.Error("Expected UpdateCombatActions to return false while cut scene is active")
	}
	if actionExecutedWhileCutScene {
		t.Error("Combat action should not execute while cut scene is active")
	}

	// Finish cutscene
	for IsCutSceneActive() {
		SetAnimFrame(GetAnimFrame() + 1)
		UpdateCutScene(game)
	}

	// Now combat action can process
	if !UpdateCombatActions(game) {
		t.Error("Expected UpdateCombatActions to process after cutscene finished")
	}
	if !actionExecutedWhileCutScene {
		t.Error("Expected combat action to execute after cutscene finished")
	}
}

func TestEnemyCombatAISequentialExecution(t *testing.T) {
	term := NewTerminal()
	SetTerminal(term)

	if _, err := PreloadSpriteDefs(); err != nil {
		t.Fatalf("PreloadSpriteDefs failed: %v", err)
	}
	if _, err := PreloadActorDefs(); err != nil {
		t.Fatalf("PreloadActorDefs failed: %v", err)
	}
	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}

	cbtMap, err := ReloadMap("cbt_grass")
	if err != nil || cbtMap == nil {
		cbtMap, err = LoadMap("cbt_grass")
		if err != nil {
			t.Fatalf("LoadMap failed: %v", err)
		}
	}
	cbtMap.Actors = nil
	SetMap(cbtMap)

	// Create 2 enemy rodents adjacent to party
	mouse1, err := NewActorFromDef("mouse_1", "rodent", 6, 5)
	if err != nil {
		t.Fatalf("NewActorFromDef mouse_1 failed: %v", err)
	}
	cbtMap.AddActor(mouse1)

	mouse2, err := NewActorFromDef("mouse_2", "rodent", 5, 6)
	if err != nil {
		t.Fatalf("NewActorFromDef mouse_2 failed: %v", err)
	}
	cbtMap.AddActor(mouse2)

	// Create party with 2 members
	hero, err := NewActorFromDef("hero", "warrior", 5, 5)
	if err != nil {
		t.Fatalf("NewActorFromDef hero failed: %v", err)
	}
	lillian, err := NewActorFromDef("lillian", "lillian", 5, 5)
	if err != nil {
		t.Fatalf("NewActorFromDef lillian failed: %v", err)
	}

	p, err := NewParty(5, 5, *hero, *lillian)
	if err != nil {
		t.Fatalf("NewParty failed: %v", err)
	}
	SetParty(p)

	game := &Game{
		terminal:   term,
		currentMap: cbtMap,
		party:      p,
	}

	StartCombat()
	initialTurn := cbtMap.Turn

	// Party member 0 turn -> Advance
	AdvanceCombatMember(game)
	if GetCombatMemberIndex() != 1 || IsCombatAIPhase() {
		t.Errorf("Expected member index 1 and IsCombatAIPhase false, got index %d, ai %v", GetCombatMemberIndex(), IsCombatAIPhase())
	}

	// Party member 1 turn -> Advance triggers AI phase
	AdvanceCombatMember(game)
	if !IsCombatAIPhase() {
		t.Fatalf("Expected IsCombatAIPhase true after all party members advance")
	}

	// Step 1: First enemy (mouse1) acts
	if !UpdateCombatAI(game) {
		t.Fatal("Expected UpdateCombatAI to initiate mouse1 turn")
	}
	if GetCombatFocusActor() == nil || GetCombatFocusActor().ID != "mouse_1" {
		t.Errorf("Expected combat focus on mouse_1, got %+v", GetCombatFocusActor())
	}
	if !IsCutSceneActive() {
		t.Errorf("Expected cutscene delay active for mouse_1")
	}

	// While cutscene is active, UpdateCombatAI should return false and not run mouse2 yet
	if UpdateCombatAI(game) {
		t.Errorf("UpdateCombatAI should return false while cutscene is active")
	}

	// Resolve mouse1's delay and actions
	for IsCutSceneActive() || HasCombatActions() {
		if !IsCutSceneActive() && HasCombatActions() {
			UpdateCombatActions(game)
		}
		SetAnimFrame(GetAnimFrame() + 1)
		UpdateCutScene(game)
	}

	// Step 2: Second enemy (mouse2) acts
	if !UpdateCombatAI(game) {
		t.Fatal("Expected UpdateCombatAI to initiate mouse2 turn")
	}
	if GetCombatFocusActor() == nil || GetCombatFocusActor().ID != "mouse_2" {
		t.Errorf("Expected combat focus on mouse_2, got %+v", GetCombatFocusActor())
	}
	if !IsCutSceneActive() {
		t.Errorf("Expected cutscene delay active for mouse_2")
	}

	// Resolve mouse2's delay and actions
	for IsCutSceneActive() || HasCombatActions() {
		if !IsCutSceneActive() && HasCombatActions() {
			UpdateCombatActions(game)
		}
		SetAnimFrame(GetAnimFrame() + 1)
		UpdateCutScene(game)
	}

	// Step 3: Finish AI phase and round
	if !UpdateCombatAI(game) {
		t.Fatal("Expected UpdateCombatAI to finish AI phase and complete round")
	}
	if IsCombatAIPhase() {
		t.Errorf("Expected IsCombatAIPhase false after all enemies acted")
	}
	if GetCombatFocusActor() != nil {
		t.Errorf("Expected combat focus cleared, got %+v", GetCombatFocusActor())
	}
	if GetCombatMemberIndex() != 0 {
		t.Errorf("Expected combat member index reset to 0, got %d", GetCombatMemberIndex())
	}
	if cbtMap.Turn != initialTurn+1 {
		t.Errorf("Expected map turn to advance from %d to %d, got %d", initialTurn, initialTurn+1, cbtMap.Turn)
	}

	// Verify terminal output contains attacking logs from effects/attack.tengo
	lines := term.GetLineTexts()
	foundLog := false
	for _, l := range lines {
		if strings.Contains(l, "Rodent of Unusual Size") {
			foundLog = true
			break
		}
	}
	if !foundLog {
		t.Errorf("Expected 'Rodent of Unusual Size...' in terminal lines, got %v", lines)
	}
}


