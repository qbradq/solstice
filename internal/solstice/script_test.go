package solstice

import (
	"testing"

	"github.com/d5/tengo/v2"
)

func TestInitScriptSystemAndRunNewGameScript(t *testing.T) {
	term := NewTerminal()
	SetTerminal(term)

	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}

	if err := RunNewGameScript(); err != nil {
		t.Fatalf("RunNewGameScript failed: %v", err)
	}

	// Verify that new_game.tengo logged welcome messages to the terminal and loaded map
	lines := term.GetLineTexts()
	if len(lines) == 0 {
		t.Error("Expected new_game.tengo to log messages to terminal, got 0 lines")
	}

	foundSolstice := false
	for _, l := range lines {
		if l == "Solstice Client v0.1.0" {
			foundSolstice = true
			break
		}
	}

	if !foundSolstice {
		t.Errorf("Expected 'Solstice Client v0.1.0' in terminal lines, got lines: %v", lines)
	}

	if m := GetMap(); m == nil {
		t.Errorf("Expected current map to be loaded, got nil")
	}
}

func TestExecuteTileScriptContext(t *testing.T) {
	term := NewTerminal()
	SetTerminal(term)

	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}

	if err := ExecuteTileScript("tiles/door.tengo", 5, 10, 78); err != nil {
		t.Fatalf("ExecuteTileScript failed: %v", err)
	}
}

func TestAddTimerAndTurnSystem(t *testing.T) {
	term := NewTerminal()
	SetTerminal(term)

	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}

	m, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}
	SetMap(m)

	// Add a 2-turn timer executing tiles/door.tengo with tile_x=5, tile_y=5, tile_idx=78
	m.SetTile(5, 5, 78)
	m.AddTimer(2, "tiles/door.tengo", map[string]interface{}{
		"tile_x":   5,
		"tile_y":   5,
		"tile_idx": 78,
	})

	if len(m.Timers) != 1 {
		t.Fatalf("Expected 1 active timer, got %d", len(m.Timers))
	}

	// Turn 1: timer remaining 1, not expired yet
	m.AdvanceTurn()
	if updatedTile := m.GetTile(5, 5); updatedTile != 78 {
		t.Errorf("Expected tile at (5, 5) to remain 78 on turn 1, got %d", updatedTile)
	}

	// Turn 2: timer expires, runs door.tengo (opens door to 68 and schedules close_door.tengo in 5 turns)
	m.AdvanceTurn()
	if updatedTile := m.GetTile(5, 5); updatedTile != 68 {
		t.Errorf("Expected tile at (5, 5) to change to 68 on turn 2 expiry, got %d", updatedTile)
	}

	// Verify that door.tengo added 1 new timer for close_door.tengo
	if len(m.Timers) != 1 {
		t.Fatalf("Expected 1 active close_door timer, got %d", len(m.Timers))
	}

	// Advance 5 more turns to trigger close_door.tengo
	for i := 0; i < 5; i++ {
		m.AdvanceTurn()
	}

	// Verify that close_door.tengo executed and reset tile to 78
	if updatedTile := m.GetTile(5, 5); updatedTile != 78 {
		t.Errorf("Expected tile at (5, 5) to close back to 78 after close_door timer, got %d", updatedTile)
	}

	if len(m.Timers) != 0 {
		t.Errorf("Expected 0 active timers after all timers expired, got %d", len(m.Timers))
	}
}

func TestGameFlagFunctions(t *testing.T) {
	ClearAllFlags()

	if HasFlag("quest_started") {
		t.Error("Expected quest_started to initially be false")
	}

	SetFlag("quest_started")
	if !HasFlag("quest_started") {
		t.Error("Expected quest_started to be true after SetFlag")
	}

	ToggleFlag("quest_started")
	if HasFlag("quest_started") {
		t.Error("Expected quest_started to be false after ToggleFlag")
	}

	ToggleFlag("quest_started")
	if !HasFlag("quest_started") {
		t.Error("Expected quest_started to be true after second ToggleFlag")
	}

	ClearFlag("quest_started")
	if HasFlag("quest_started") {
		t.Error("Expected quest_started to be false after ClearFlag")
	}

	// Verify Tengo script flag functions
	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}

	gameMod := moduleMap.GetBuiltinModule("game")
	if gameMod == nil {
		t.Fatal("Expected builtin game module")
	}

	// Verify old *_state functions are removed from module
	for _, oldName := range []string{"set_state", "clear_state", "toggle_state", "has_state"} {
		if _, exists := gameMod.Attrs[oldName]; exists {
			t.Errorf("Expected old function %q to be removed from game module", oldName)
		}
	}

	// Verify new *_flag functions exist in module
	for _, newName := range []string{"set_flag", "clear_flag", "toggle_flag", "has_flag"} {
		if _, exists := gameMod.Attrs[newName]; !exists {
			t.Errorf("Expected new function %q to exist in game module", newName)
		}
	}

	// Test executing script with flag functions
	scriptSrc := `
game := import("game")
game.set_flag("hero_awakened")
flag1 := game.has_flag("hero_awakened")
game.toggle_flag("hero_awakened")
flag2 := game.has_flag("hero_awakened")
game.clear_flag("hero_awakened")
flag3 := game.has_flag("hero_awakened")
`
	script := tengo.NewScript([]byte(scriptSrc))
	script.SetImports(moduleMap)
	compiled, err := script.Compile()
	if err != nil {
		t.Fatalf("Failed to compile flag test script: %v", err)
	}
	if err := compiled.Run(); err != nil {
		t.Fatalf("Failed to run flag test script: %v", err)
	}
}

func TestGameRandomFunction(t *testing.T) {
	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}

	gameMod := moduleMap.GetBuiltinModule("game")
	if gameMod == nil {
		t.Fatal("Expected builtin game module")
	}

	randomFuncObj, ok := gameMod.Attrs["random"]
	if !ok || randomFuncObj == nil {
		t.Fatal("Expected 'random' function in game module")
	}

	userFunc, ok := randomFuncObj.(*tengo.UserFunction)
	if !ok {
		t.Fatal("Expected 'random' to be UserFunction")
	}

	// 1. Single argument test
	arg0 := &tengo.String{Value: "hello"}
	res, err := userFunc.Value(arg0)
	if err != nil {
		t.Fatalf("random call failed: %v", err)
	}
	if res.String() != `"hello"` {
		t.Errorf("Expected 'hello', got %s", res.String())
	}

	// 2. Multiple arguments test
	arg1 := &tengo.String{Value: "world"}
	arg2 := &tengo.String{Value: "foo"}
	foundMap := make(map[string]bool)

	for i := 0; i < 100; i++ {
		r, err := userFunc.Value(arg0, arg1, arg2)
		if err != nil {
			t.Fatalf("random call failed: %v", err)
		}
		foundMap[r.String()] = true
	}

	if len(foundMap) < 2 {
		t.Errorf("Expected multiple random choices from 100 iterations, got: %v", foundMap)
	}
}

func TestGameLoadMapAndTeleportParty(t *testing.T) {
	term := NewTerminal()
	SetTerminal(term)

	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}

	p, err := NewParty(0, 0)
	if err != nil {
		t.Fatalf("NewParty failed: %v", err)
	}
	SetParty(p)

	// Run new_game.tengo which executes load_map and teleport_party(15, 15)
	if err := RunNewGameScript(); err != nil {
		t.Fatalf("RunNewGameScript failed: %v", err)
	}

	m := GetMap()
	if m == nil {
		t.Errorf("Expected current map to be loaded after new_game.tengo, got nil")
	}

	party := GetParty()
	if party.X != 15 || party.Y != 15 {
		t.Errorf("Expected party position (15, 15) after teleport_party in new_game.tengo, got (%d, %d)", party.X, party.Y)
	}
	if party.WorldX != 5 || party.WorldY != 84 {
		t.Errorf("Expected party world position (5, 84) after teleport_party_on_world_map in new_game.tengo, got (%d, %d)", party.WorldX, party.WorldY)
	}
}

func TestExecuteTriggerScript(t *testing.T) {
	term := NewTerminal()
	SetTerminal(term)

	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}

	// 1. Triggered by party
	if err := ExecuteTriggerScript("triggers/test.tengo", "home", 10, 15, "party"); err != nil {
		t.Fatalf("ExecuteTriggerScript for party failed: %v", err)
	}

	lines := term.GetLineTexts()
	if len(lines) == 0 || lines[len(lines)-1] != "party trigger" {
		t.Errorf("Expected 'party trigger' in terminal log, got lines: %v", lines)
	}

	// 2. Triggered by actor
	if err := ExecuteTriggerScript("triggers/test.tengo", "home", 10, 15, "guard-1"); err != nil {
		t.Fatalf("ExecuteTriggerScript for actor failed: %v", err)
	}

	lines = term.GetLineTexts()
	if len(lines) == 0 || lines[len(lines)-1] != "actor trigger: guard-1" {
		t.Errorf("Expected 'actor trigger: guard-1' in terminal log, got lines: %v", lines)
	}
}

func TestGameStartDialog(t *testing.T) {
	term := NewTerminal()
	SetTerminal(term)

	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}

	game := &Game{
		terminal: term,
	}
	SetGame(game)
	game.PushMode(NewMainMode())

	// Call start_dialog from Tengo
	scriptSrc := `
game := import("game")
game.start_dialog("dialog/guard.tengo", "guard-1")
`
	script := tengo.NewScript([]byte(scriptSrc))
	script.SetImports(moduleMap)
	compiled, err := script.Compile()
	if err != nil {
		t.Fatalf("Failed to compile script: %v", err)
	}
	if err := compiled.Run(); err != nil {
		t.Fatalf("Failed to run script: %v", err)
	}

	// Verify DialogMode was pushed onto game mode stack
	activeMode := game.GetMode()
	dialogMode, ok := activeMode.(*DialogMode)
	if !ok || dialogMode == nil {
		t.Fatalf("Expected active mode to be *DialogMode, got %T", activeMode)
	}
	if dialogMode.scriptPath != "dialog/guard.tengo" {
		t.Errorf("Expected dialog script 'dialog/guard.tengo', got %s", dialogMode.scriptPath)
	}
	if dialogMode.actor == nil || dialogMode.actor.ID != "guard-1" {
		t.Errorf("Expected actor ID 'guard-1', got %v", dialogMode.actor)
	}
}

func TestGameSpawnAndRemoveActor(t *testing.T) {
	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}
	if _, err := PreloadSpriteDefs(); err != nil {
		t.Fatalf("PreloadSpriteDefs failed: %v", err)
	}
	if _, err := PreloadActorDefs(); err != nil {
		t.Fatalf("PreloadActorDefs failed: %v", err)
	}

	m, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}
	SetMap(m)

	// 1. Spawn actor via Tengo script
	spawnSrc := `
game := import("game")
game.spawn_actor("guard", "new-guard-99", 5, 8)
`
	script := tengo.NewScript([]byte(spawnSrc))
	script.SetImports(moduleMap)
	compiled, err := script.Compile()
	if err != nil {
		t.Fatalf("Failed to compile spawn script: %v", err)
	}
	if err := compiled.Run(); err != nil {
		t.Fatalf("Failed to run spawn script: %v", err)
	}

	// Verify actor was added to map
	actor := m.GetActorByID("new-guard-99")
	if actor == nil {
		t.Fatalf("Expected spawned actor 'new-guard-99' on map, got nil")
	}
	if actor.X != 5 || actor.Y != 8 {
		t.Errorf("Expected actor position (5, 8), got (%d, %d)", actor.X, actor.Y)
	}

	// 2. Remove actor via Tengo script
	removeSrc := `
game := import("game")
game.remove_actor("new-guard-99")
`
	script2 := tengo.NewScript([]byte(removeSrc))
	script2.SetImports(moduleMap)
	compiled2, err := script2.Compile()
	if err != nil {
		t.Fatalf("Failed to compile remove script: %v", err)
	}
	if err := compiled2.Run(); err != nil {
		t.Fatalf("Failed to run remove script: %v", err)
	}

	// Verify actor was removed from map
	if removedActor := m.GetActorByID("new-guard-99"); removedActor != nil {
		t.Errorf("Expected actor 'new-guard-99' to be removed from map, but still found: %v", removedActor)
	}
}

func TestGameExecuteMapScript(t *testing.T) {
	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}
	if _, err := PreloadSpriteDefs(); err != nil {
		t.Fatalf("PreloadSpriteDefs failed: %v", err)
	}
	if _, err := PreloadActorDefs(); err != nil {
		t.Fatalf("PreloadActorDefs failed: %v", err)
	}

	m, err := LoadMap("kings_shrine")
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}
	SetMap(m)

	// Execute intro map script from Tengo
	scriptSrc := `
game := import("game")
game.exec_map_script("intro")
`
	script := tengo.NewScript([]byte(scriptSrc))
	script.SetImports(moduleMap)
	compiled, err := script.Compile()
	if err != nil {
		t.Fatalf("Failed to compile map script runner: %v", err)
	}
	if err := compiled.Run(); err != nil {
		t.Fatalf("Failed to run map script runner: %v", err)
	}

	// Verify actors from intro.tengo were spawned on map
	w1 := m.GetActorByID("wizard-1")
	if w1 == nil || w1.X != 15 || w1.Y != 14 {
		t.Errorf("Expected wizard-1 at (15, 14), got %v", w1)
	}
	duke := m.GetActorByID("duke-lafey")
	if duke == nil || duke.X != 15 || duke.Y != 16 {
		t.Errorf("Expected duke-lafey at (15, 16), got %v", duke)
	}
}

func TestGameReloadMapFunction(t *testing.T) {
	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}

	ClearLoadedMaps()

	// 1. Calling game.reload_map on an unloaded map does nothing
	scriptSrc1 := `
game := import("game")
game.reload_map("home")
`
	s1 := tengo.NewScript([]byte(scriptSrc1))
	s1.SetImports(moduleMap)
	c1, err := s1.Compile()
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}
	if err := c1.Run(); err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if len(GetAllLoadedMaps()) != 0 {
		t.Errorf("Expected no maps loaded, got %d", len(GetAllLoadedMaps()))
	}

	// 2. Load "home" map via game.load_map, modify a tile via game.set_map_tile, then reload via game.reload_map
	scriptSrc2 := `
game := import("game")
game.load_map("home")
game.set_map_tile(2, 2, 999)
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

	curMap := GetMap()
	if curMap == nil {
		t.Fatalf("Expected map to be loaded")
	}
	if curMap.GetTile(2, 2) != 999 {
		t.Fatalf("Expected tile at (2, 2) to be 999, got %d", curMap.GetTile(2, 2))
	}

	scriptSrc3 := `
game := import("game")
game.reload_map("home")
`
	s3 := tengo.NewScript([]byte(scriptSrc3))
	s3.SetImports(moduleMap)
	c3, err := s3.Compile()
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}
	if err := c3.Run(); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	reloadedMap := GetMap()
	if reloadedMap == nil {
		t.Fatalf("Expected reloaded map to be current map")
	}
	if reloadedMap == curMap {
		t.Errorf("Expected new map instance after reload")
	}
	if reloadedMap.GetTile(2, 2) == 999 {
		t.Errorf("Expected tile at (2, 2) to be reset to original, got 999")
	}
}

func TestGameAddToParty(t *testing.T) {
	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}

	term := NewTerminal()
	SetTerminal(term)

	homeMap, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}
	SetMap(homeMap)

	// Create party with 1 member
	hero, _ := NewActorFromDef("hero", "kevin", 0, 0)
	party, _ := NewParty(10, 10, *hero)
	SetParty(party)

	// In homeMap, object 1 is "guard"
	guardActor := homeMap.GetActorByID("guard")
	if guardActor == nil {
		t.Fatalf("Expected actor 'guard' on homeMap")
	}

	// 1. Add "guard" to party
	scriptSrc1 := `
game := import("game")
game.add_to_party("guard")
`
	s1 := tengo.NewScript([]byte(scriptSrc1))
	s1.SetImports(moduleMap)
	c1, err := s1.Compile()
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}
	if err := c1.Run(); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Verify party now has 2 members
	if len(party.Members) != 2 {
		t.Fatalf("Expected 2 party members, got %d", len(party.Members))
	}
	if party.Members[1].ID != "guard" {
		t.Errorf("Expected second member ID 'guard', got %s", party.Members[1].ID)
	}

	// Verify guard was removed from map
	if homeMap.GetActorByID("guard") != nil {
		t.Errorf("Expected guard to be removed from map")
	}

	// Verify log message
	lines := term.GetLineTexts()
	if len(lines) == 0 || lines[len(lines)-1] != "Town Guard joins the party!" {
		t.Errorf("Expected 'Town Guard joins the party!' in terminal log, got lines: %v", lines)
	}

	// 2. Fill party to 4 members
	m3, _ := NewActorFromDef("m3", "wizard", 0, 0)
	m4, _ := NewActorFromDef("m4", "wizard", 0, 0)
	_ = party.AddMember(*m3)
	_ = party.AddMember(*m4)
	if len(party.Members) != 4 {
		t.Fatalf("Expected 4 members, got %d", len(party.Members))
	}

	// 3. Try to add 5th member ("lillian")
	scriptSrc2 := `
game := import("game")
game.add_to_party("lillian")
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

	if len(party.Members) != 4 {
		t.Errorf("Expected party size to remain 4, got %d", len(party.Members))
	}

	lines = term.GetLineTexts()
	if len(lines) == 0 || lines[len(lines)-1] != "Too many party members!" {
		t.Errorf("Expected 'Too many party members!' in terminal log, got lines: %v", lines)
	}
}

