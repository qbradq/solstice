package solstice

import (
	"fmt"
	"strings"
	"testing"

	"github.com/d5/tengo/v2"
)

func TestInitScriptSystemAndRunNewGameScript(t *testing.T) {
	term := NewTerminal()
	SetTerminal(term)

	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}

	_ = RunNewGameScript()

	// Verify that new_game.tengo logged messages or errors to the terminal
	lines := term.GetLineTexts()
	if len(lines) == 0 {
		t.Error("Expected new_game.tengo to log messages to terminal, got 0 lines")
	}
}

func TestExecuteTileScriptContext(t *testing.T) {
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

	// Execute tiles/door.tengo directly with globals tile_x=10, tile_y=12, tile_idx=78
	m.SetTile(10, 12, 78)
	if err := ExecuteTileScript("tiles/door.tengo", 10, 12, 78); err != nil {
		t.Fatalf("ExecuteTileScript failed: %v", err)
	}

	// Verify door tile was changed to 68
	if newTile := m.GetTile(10, 12); newTile != 68 {
		t.Errorf("Expected tile at (10, 12) to be 68, got %d", newTile)
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
	m.Timers = make([]*MapTimer, 0)
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

	// Test load_map, teleport_party, and teleport_party_on_world_map
	scriptSrc := `
game := import("game")
game.load_map("home")
game.teleport_party(15, 15)
game.teleport_party_on_world_map(5, 84)
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

	m := GetMap()
	if m == nil || m.Name != "home" {
		t.Errorf("Expected current map 'home' to be loaded, got %v", m)
	}

	party := GetParty()
	if party.X != 15 || party.Y != 15 {
		t.Errorf("Expected party position (15, 15), got (%d, %d)", party.X, party.Y)
	}
	if party.WorldX != 5 || party.WorldY != 84 {
		t.Errorf("Expected party world position (5, 84), got (%d, %d)", party.WorldX, party.WorldY)
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
game.remove("new-guard-99")
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
	if w1 == nil || w1.X != 7 || w1.Y != 6 {
		t.Errorf("Expected wizard-1 at (7, 6), got %v", w1)
	}
	duke := m.GetActorByID("duke-lafey")
	if duke == nil || duke.X != 7 || duke.Y != 8 {
		t.Errorf("Expected duke-lafey at (7, 8), got %v", duke)
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

func TestGameRoll(t *testing.T) {
	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}

	scriptSrc := `
game := import("game")
roll1 := game.roll("1d4")
roll2 := game.roll("10d1+5")
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

	roll1 := c.Get("roll1").Int()
	if roll1 < 1 || roll1 > 4 {
		t.Errorf("Expected roll1 in [1, 4], got %d", roll1)
	}

	roll2 := c.Get("roll2").Int()
	if roll2 != 15 {
		t.Errorf("Expected roll2 to be 15, got %d", roll2)
	}
}

func TestAIModule(t *testing.T) {
	if _, err := PreloadTileSet(); err != nil {
		t.Fatalf("PreloadTileSet failed: %v", err)
	}
	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}

	homeMap, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}
	SetMap(homeMap)

	m1, _ := NewActorFromDef("hero1", "kevin", 0, 0)
	m2, _ := NewActorFromDef("hero2", "wizard", 0, 0)
	party, err := NewParty(10, 10, *m1, *m2)
	if err != nil {
		t.Fatalf("NewParty failed: %v", err)
	}
	party.Members[0].X = 10
	party.Members[0].Y = 10
	party.Members[1].X = 20
	party.Members[1].Y = 20
	SetParty(party)

	// 1. Test get_nearest_party_member
	scriptSrc1 := `
ai := import("ai")
near1 := ai.get_nearest_party_member(9, 9)
near2 := ai.get_nearest_party_member(21, 20)
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

	if c1.Get("near1").String() != "hero1" {
		t.Errorf("Expected near1 to be 'hero1', got %q", c1.Get("near1").String())
	}
	if c1.Get("near2").String() != "hero2" {
		t.Errorf("Expected near2 to be 'hero2', got %q", c1.Get("near2").String())
	}
}

func TestActorIdleAndCombatScripts(t *testing.T) {
	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}

	// Verify Lillian definition has idle_script set to "ai/idle/wander.tengo"
	lillianDef, ok := GetActorDef("lillian")
	if !ok {
		t.Fatalf("Lillian actor definition not found")
	}
	if lillianDef.IdleScript != "ai/idle/wander.tengo" {
		t.Errorf("Expected Lillian idle_script 'ai/idle/wander.tengo', got %q", lillianDef.IdleScript)
	}

	lillian, err := NewActorFromDef("lillian", "lillian", 15, 15)
	if err != nil {
		t.Fatalf("NewActorFromDef lillian failed: %v", err)
	}
	if lillian.IdleScript != "ai/idle/wander.tengo" {
		t.Errorf("Expected lillian actor instance idle_script 'ai/idle/wander.tengo', got %q", lillian.IdleScript)
	}

	homeMap, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}
	SetMap(homeMap)
	homeMap.AddActor(lillian)

	party, err := NewParty(5, 5)
	if err != nil {
		t.Fatalf("NewParty failed: %v", err)
	}
	SetParty(party)

	// Test game.move_actor directly
	lillian.X = 15
	lillian.Y = 15
	script := []byte(`
		game := import("game")
		res := game.move_actor("lillian", "s")
	`)
	s := tengo.NewScript(script)
	s.SetImports(GetScriptModuleMap())
	compiled, err := s.Compile()
	if err != nil {
		t.Fatalf("Failed to compile script: %v", err)
	}
	if err := compiled.Run(); err != nil {
		t.Fatalf("Failed to run script: %v", err)
	}
	if IsCutSceneActive() {
		t.Error("game.move_actor should not activate cutscenes")
	}

	// Advance turn should execute Lillian's idle script (wander.tengo) without error
	for i := 0; i < 50; i++ {
		homeMap.AdvanceTurn()
	}
}

func TestItemsAndFindItems(t *testing.T) {
	if _, err := PreloadSpriteDefs(); err != nil {
		t.Fatalf("PreloadSpriteDefs failed: %v", err)
	}
	if _, err := PreloadItemDefs(); err != nil {
		t.Fatalf("PreloadItemDefs failed: %v", err)
	}
	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}

	term := NewTerminal()
	SetTerminal(term)

	m, err := LoadMap("cbt_grass")
	if err != nil {
		t.Fatalf("LoadMap cbt_grass failed: %v", err)
	}
	SetMap(m)

	if m.Properties.LoadScript != "map/enter_combat_map.tengo" {
		t.Errorf("Expected map load_script 'map/enter_combat_map.tengo', got %q", m.Properties.LoadScript)
	}

	// 1. Test find_items in Tengo script
	scriptSrc := `
game := import("game")
party_starts := game.find_items("party_start")
enemy_starts := game.find_items("enemy_start")
num_party := len(party_starts)
num_enemy := len(enemy_starts)
first_party_id := party_starts[0].id
first_party_tmpl := party_starts[0].template
first_party_x := party_starts[0].x
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

	if c.Get("num_party").Int() != 4 {
		t.Errorf("Expected 4 party_start items, got %d", c.Get("num_party").Int())
	}
	if c.Get("num_enemy").Int() != 8 {
		t.Errorf("Expected 8 enemy_start items, got %d", c.Get("num_enemy").Int())
	}
	if c.Get("first_party_tmpl").String() != "party_start" {
		t.Errorf("Expected template 'party_start', got %q", c.Get("first_party_tmpl").String())
	}
}

func TestSpawnItemAndRemoveEntity(t *testing.T) {
	if _, err := PreloadItemDefs(); err != nil {
		t.Fatalf("PreloadItemDefs failed: %v", err)
	}
	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}

	m, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap home failed: %v", err)
	}
	SetMap(m)

	scriptSrc := `
game := import("game")
game.spawn_item("enemy_start", 7, 9, "test_spawned_item")
items_before := game.find_items("enemy_start")
num_before := len(items_before)
game.remove("test_spawned_item")
items_after := game.find_items("enemy_start")
num_after := len(items_after)
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
	if c.Get("num_before").Int() != 1 {
		t.Errorf("Expected 1 item before removal, got %d", c.Get("num_before").Int())
	}
	if c.Get("num_after").Int() != 0 {
		t.Errorf("Expected 0 items after removal, got %d", c.Get("num_after").Int())
	}

	// Test spawning item with type and meta (e.g. dagger)
	daggerScript := `
game := import("game")
game.spawn_item("dagger", 3, 4, "test_dagger_1")
daggers := game.find_items("dagger")
dagger_type := daggers[0].type
dagger_range := daggers[0].meta.range
dagger_damage := daggers[0].meta.damage
`
	s2 := tengo.NewScript([]byte(daggerScript))
	s2.SetImports(moduleMap)
	c2, err := s2.Compile()
	if err != nil {
		t.Fatalf("Compile daggerScript failed: %v", err)
	}
	if err := c2.Run(); err != nil {
		t.Fatalf("Run daggerScript failed: %v", err)
	}

	if c2.Get("dagger_type").String() != "weapon" {
		t.Errorf("Expected dagger type 'weapon', got %q", c2.Get("dagger_type").String())
	}
	if c2.Get("dagger_range").Int() != 1 {
		t.Errorf("Expected dagger range 1, got %d", c2.Get("dagger_range").Int())
	}
	if c2.Get("dagger_damage").String() != "2d4+2" {
		t.Errorf("Expected dagger damage '2d4+2', got %q", c2.Get("dagger_damage").String())
	}
}

func TestScriptErrorLoggedToTerminalInBrightRed(t *testing.T) {
	term := NewTerminal()
	SetTerminal(term)

	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}

	// 1. Non-existent script error
	err := ExecuteScript("non_existent_script.tengo")
	if err == nil {
		t.Fatal("Expected error executing non-existent script, got nil")
	}

	lines := term.GetLines()
	if len(lines) == 0 {
		t.Fatal("Expected error message in terminal log, got 0 lines")
	}

	lastLine := lines[len(lines)-1]
	brightRed := VGAPalette16[12]
	if lastLine.Color != brightRed {
		t.Errorf("Expected script error line to be bright red (%v), got %v", brightRed, lastLine.Color)
	}
}

func TestGameStringFunctions(t *testing.T) {
	ClearAllStrings()

	if val := GetString("combat_pack"); val != "" {
		t.Errorf("Expected combat_pack to initially be empty string, got %q", val)
	}

	SetString("combat_pack", "rodents")
	if val := GetString("combat_pack"); val != "rodents" {
		t.Errorf("Expected combat_pack to be 'rodents', got %q", val)
	}

	all := GetAllStrings()
	if all["combat_pack"] != "rodents" {
		t.Errorf("Expected GetAllStrings to contain combat_pack='rodents', got %v", all)
	}

	ClearString("combat_pack")
	if val := GetString("combat_pack"); val != "" {
		t.Errorf("Expected combat_pack to be empty after ClearString, got %q", val)
	}

	// Test in Tengo script
	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}

	scriptSrc := `
game := import("game")
game.set_string("current_pack", "rodents")
pack1 := game.get_string("current_pack")
game.clear_string("current_pack")
pack2 := game.get_string("current_pack")
missing := game.get_string("non_existent")
`
	script := tengo.NewScript([]byte(scriptSrc))
	script.SetImports(moduleMap)
	compiled, err := script.Compile()
	if err != nil {
		t.Fatalf("Failed to compile string test script: %v", err)
	}
	if err := compiled.Run(); err != nil {
		t.Fatalf("Failed to run string test script: %v", err)
	}

	if pack1 := compiled.Get("pack1").String(); pack1 != "rodents" {
		t.Errorf("Expected pack1 'rodents', got %q", pack1)
	}
	if pack2 := compiled.Get("pack2").String(); pack2 != "" {
		t.Errorf("Expected pack2 '', got %q", pack2)
	}
	if missing := compiled.Get("missing").String(); missing != "" {
		t.Errorf("Expected missing '', got %q", missing)
	}
}

func TestSpawnActorUniqueIDs(t *testing.T) {
	if _, err := PreloadActorDefs(); err != nil {
		t.Fatalf("PreloadActorDefs failed: %v", err)
	}
	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}

	m, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap home failed: %v", err)
	}
	SetMap(m)

	scriptSrc := `
game := import("game")
id1 := game.spawn_actor("rodent", 1, 1)
id2 := game.spawn_actor("rodent", "rodent", 1, 2)
id3 := game.spawn_actor("rodent", 1, 3, "rodent")
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

	id1 := c.Get("id1").String()
	id2 := c.Get("id2").String()
	id3 := c.Get("id3").String()

	if id1 == id2 || id1 == id3 || id2 == id3 {
		t.Errorf("Expected distinct unique IDs, got id1=%q, id2=%q, id3=%q", id1, id2, id3)
	}
}

func TestGetEnemiesForPackAndEnterCombatMapScript(t *testing.T) {
	if _, err := PreloadActorDefs(); err != nil {
		t.Fatalf("PreloadActorDefs failed: %v", err)
	}
	if _, err := PreloadItemDefs(); err != nil {
		t.Fatalf("PreloadItemDefs failed: %v", err)
	}
	if _, err := PreloadEnemyPacks(); err != nil {
		t.Fatalf("PreloadEnemyPacks failed: %v", err)
	}
	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}

	// 1. Test get_enemies_for_pack in Tengo
	testScript := `
game := import("game")
enemies := game.get_enemies_for_pack("rodents")
num_enemies := len(enemies)
`
	s := tengo.NewScript([]byte(testScript))
	s.SetImports(moduleMap)
	c, err := s.Compile()
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}
	if err := c.Run(); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	numEnemies := c.Get("num_enemies").Int()
	if numEnemies < 1 || numEnemies > 8 {
		t.Errorf("Expected num_enemies between 1 and 8, got %d", numEnemies)
	}

	// 2. Test full execution of enter_combat_map.tengo
	party, err := NewParty(0, 0)
	if err != nil {
		t.Fatalf("NewParty failed: %v", err)
	}
	kevin, err := NewActorFromDef("kevin", "kevin", 0, 0)
	if err != nil {
		t.Fatalf("NewActorFromDef kevin failed: %v", err)
	}
	lillian, err := NewActorFromDef("lillian", "lillian", 0, 0)
	if err != nil {
		t.Fatalf("NewActorFromDef lillian failed: %v", err)
	}
	_ = party.AddMember(*kevin)
	_ = party.AddMember(*lillian)
	SetParty(party)

	SetString("combat_map_pack", "rodents")

	// Load cbt_grass map which runs load_script: enter_combat_map.tengo
	m, err := loadMapFromTMX("cbt_grass")
	if err != nil {
		t.Fatalf("loadMapFromTMX cbt_grass failed: %v", err)
	}
	SetMap(m)

	if !IsInCombat() {
		t.Errorf("Expected to be in combat mode after enter_combat_map.tengo")
	}

	// Verify party members were placed on party_start positions
	partyStarts := m.FindItemsByTemplate("party_start")
	if len(partyStarts) == 0 {
		partyStarts = m.FindItemsByTemplate("player_start")
	}
	startPositions := make(map[string]bool)
	for _, it := range partyStarts {
		startPositions[fmt.Sprintf("%d,%d", it.X, it.Y)] = true
	}

	curParty := GetParty()
	for _, mem := range curParty.Members {
		key := fmt.Sprintf("%d,%d", mem.X, mem.Y)
		if !startPositions[key] {
			t.Errorf("Expected party member %s at one of start positions %v, got %s", mem.ID, startPositions, key)
		}
	}

	// Verify enemies were spawned with unique IDs on map
	enemyCount := 0
	actorIDs := make(map[string]bool)
	for _, a := range m.Actors {
		if a.ID != "lillian" && a.ID != "kevin" {
			enemyCount++
			if actorIDs[a.ID] {
				t.Errorf("Duplicate enemy actor ID found on map: %s", a.ID)
			}
			actorIDs[a.ID] = true
		}
	}
	if enemyCount < 1 || enemyCount > 8 {
		t.Errorf("Expected 1-8 enemies spawned on cbt_grass, got %d", enemyCount)
	}
}

func TestGameLogFormatting(t *testing.T) {
	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}

	term := NewTerminal()
	SetTerminal(term)

	repl := NewTengoREPL()
	err := repl.Execute(`
game.log("simple string")
game.log("hello %s, you have %d hit points!", "hero", 42)
`)
	if err != nil {
		t.Fatalf("REPL execute game.log failed: %v", err)
	}

	lines := term.GetLineTexts()
	if len(lines) < 2 {
		t.Fatalf("Expected at least 2 terminal lines, got %d: %v", len(lines), lines)
	}
	joined := strings.Join(lines, " ")
	if !strings.Contains(joined, "simple string") {
		t.Errorf("Expected 'simple string' in terminal output %q", joined)
	}
	if !strings.Contains(joined, "hello hero, you have 42 hit points!") {
		t.Errorf("Expected 'hello hero, you have 42 hit points!' in terminal output %q", joined)
	}
}


