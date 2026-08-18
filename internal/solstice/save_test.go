package solstice

import (
	"encoding/json"
	"image"
	"os"
	"strings"
	"testing"
)

func setupTestEnvironment(t *testing.T) string {
	tmpDir := t.TempDir()
	SetSaveDirOverride(tmpDir)

	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}
	if _, err := PreloadTileSet(); err != nil {
		t.Fatalf("PreloadTileSet failed: %v", err)
	}
	if _, err := PreloadSpriteDefs(); err != nil {
		t.Fatalf("PreloadSpriteDefs failed: %v", err)
	}
	if _, err := PreloadActorDefs(); err != nil {
		t.Fatalf("PreloadActorDefs failed: %v", err)
	}

	return tmpDir
}

func TestSaveAndLoadGame(t *testing.T) {
	setupTestEnvironment(t)
	ClearLoadedMaps()
	ClearAllFlags()

	// Initialize world map and home map
	wm, err := PreloadWorldMap()
	if err != nil {
		t.Fatalf("PreloadWorldMap failed: %v", err)
	}
	// Modify world map tile and add actor to verify world map save/load
	wm.SetTile(20, 20, 77)
	wmActor := NewActor("world-guard", 21, 21, "guard")
	wm.AddActor(wmActor)

	homeMap, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap(home) failed: %v", err)
	}
	SetMap(homeMap)

	// Modify tile on home map
	homeMap.SetTile(10, 10, 88)

	// Add actor to home map
	testActor := NewActor("test-save-actor", 12, 14, "wizard")
	testActor.DialogScript = "dialog/intro.tengo"
	homeMap.AddActor(testActor)

	// Create party with members
	hero, err := NewActorFromDef("hero-1", "kevin", 10, 15)
	if err != nil {
		t.Fatalf("NewActorFromDef failed: %v", err)
	}
	party, err := NewParty(10, 15, *hero)
	if err != nil {
		t.Fatalf("NewParty failed: %v", err)
	}
	party.WorldX = 42
	party.WorldY = 77
	SetParty(party)

	// Set flags
	SetFlag("rescued_princess")
	SetFlag("opened_gate")

	// Save to slot 1 (compact)
	if err := SaveGame(1, false); err != nil {
		t.Fatalf("SaveGame(1, false) failed: %v", err)
	}

	// Verify save file exists and is not pretty-printed (no newlines except perhaps at EOF)
	slotPath, err := GetSlotFilePath(1)
	if err != nil {
		t.Fatalf("GetSlotFilePath(1) failed: %v", err)
	}
	fileBytes, err := os.ReadFile(slotPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if strings.Contains(string(fileBytes), "\n  ") {
		t.Errorf("Expected compact JSON for SaveGame(1, false)")
	}

	// Reset in-memory game state
	ClearLoadedMaps()
	ClearAllFlags()
	SetParty(nil)
	SetMap(nil)
	SetWorldMap(nil)

	if HasFlag("rescued_princess") {
		t.Fatal("Expected flags to be cleared before load")
	}

	// Load from slot 1
	if err := LoadGame(1); err != nil {
		t.Fatalf("LoadGame(1) failed: %v", err)
	}

	// Verify loaded party
	loadedParty := GetParty()
	if loadedParty == nil {
		t.Fatal("Expected loaded party to be non-nil")
	}
	if loadedParty.X != 10 || loadedParty.Y != 15 {
		t.Errorf("Expected loaded party pos (10, 15), got (%d, %d)", loadedParty.X, loadedParty.Y)
	}
	if loadedParty.WorldX != 42 || loadedParty.WorldY != 77 {
		t.Errorf("Expected loaded party world pos (42, 77), got (%d, %d)", loadedParty.WorldX, loadedParty.WorldY)
	}
	if len(loadedParty.Members) != 1 || loadedParty.Members[0].ID != "hero-1" {
		t.Errorf("Expected 1 member 'hero-1', got %v", loadedParty.Members)
	}

	// Verify loaded flags
	if !HasFlag("rescued_princess") || !HasFlag("opened_gate") {
		t.Errorf("Expected restored flags, got %v", GetAllFlags())
	}

	// Verify loaded active map
	curMap := GetMap()
	if curMap == nil || curMap.Name != "home" {
		t.Fatalf("Expected current map 'home', got %v", curMap)
	}
	if tile := curMap.GetTile(10, 10); tile != 88 {
		t.Errorf("Expected restored tile 88 at (10, 10), got %d", tile)
	}
	restoredActor := curMap.GetActorByID("test-save-actor")
	if restoredActor == nil || restoredActor.X != 12 || restoredActor.Y != 14 {
		t.Errorf("Expected restored actor at (12, 14), got %v", restoredActor)
	}

	// Verify world map was also restored from save data
	restoredWM := GetWorldMap()
	if restoredWM == nil {
		t.Fatal("Expected world map to be restored")
	}
	if restoredWM.GetTile(20, 20) != 77 {
		t.Errorf("Expected restored world map tile at (20, 20) to be 77, got %d", restoredWM.GetTile(20, 20))
	}
	restoredWMActor := restoredWM.GetActorByID("world-guard")
	if restoredWMActor == nil || restoredWMActor.X != 21 || restoredWMActor.Y != 21 {
		t.Errorf("Expected restored world map actor 'world-guard' at (21, 21), got %v", restoredWMActor)
	}
}

func TestSaveGamePretty(t *testing.T) {
	setupTestEnvironment(t)
	ClearLoadedMaps()
	ClearAllFlags()

	homeMap, _ := LoadMap("home")
	SetMap(homeMap)
	party, _ := NewParty(5, 5)
	SetParty(party)

	// Save pretty to slot 2
	if err := SaveGame(2, true); err != nil {
		t.Fatalf("SaveGame(2, true) failed: %v", err)
	}

	slotPath, _ := GetSlotFilePath(2)
	fileBytes, err := os.ReadFile(slotPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if !strings.Contains(string(fileBytes), "\n  \"version\":") {
		t.Errorf("Expected pretty-printed JSON with indentation in save file")
	}

	// Verify valid JSON
	var parsed SaveGameData
	if err := json.Unmarshal(fileBytes, &parsed); err != nil {
		t.Fatalf("Failed to parse pretty JSON: %v", err)
	}
	if parsed.Slot != 2 {
		t.Errorf("Expected slot 2, got %d", parsed.Slot)
	}
}

func TestSaveSlotListingAndInfo(t *testing.T) {
	setupTestEnvironment(t)

	// Initially all slots are empty
	if HasAnySaveFiles() {
		t.Error("Expected HasAnySaveFiles() to be false initially")
	}

	slots, err := GetAllSaveSlotInfos()
	if err != nil {
		t.Fatalf("GetAllSaveSlotInfos failed: %v", err)
	}
	if len(slots) != 3 {
		t.Fatalf("Expected 3 slots, got %d", len(slots))
	}
	for i, s := range slots {
		if s.Exists || s.DisplayTime != "[Empty]" || s.Slot != i+1 {
			t.Errorf("Slot %d unexpected info: %+v", i+1, s)
		}
	}

	// Save to slot 3
	homeMap, _ := LoadMap("home")
	SetMap(homeMap)
	party, _ := NewParty(5, 5)
	SetParty(party)

	if err := SaveGame(3, false); err != nil {
		t.Fatalf("SaveGame(3) failed: %v", err)
	}

	if !HasAnySaveFiles() {
		t.Error("Expected HasAnySaveFiles() to be true after saving to slot 3")
	}

	slots, err = GetAllSaveSlotInfos()
	if err != nil {
		t.Fatalf("GetAllSaveSlotInfos failed: %v", err)
	}
	if !slots[2].Exists {
		t.Error("Expected slot 3 Exists to be true")
	}
	if slots[2].DisplayTime == "[Empty]" {
		t.Errorf("Expected slot 3 DisplayTime to be timestamp, got %s", slots[2].DisplayTime)
	}
}

func TestMapStateRetentionInMemory(t *testing.T) {
	setupTestEnvironment(t)
	ClearLoadedMaps()

	// Load map "home" and edit tile at (5, 5)
	home1, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap(home) failed: %v", err)
	}
	home1.SetTile(5, 5, 99)

	// Load map "kings_shrine"
	_, err = LoadMap("kings_shrine")
	if err != nil {
		t.Fatalf("LoadMap(kings_shrine) failed: %v", err)
	}

	// Load map "home" again - should return cached instance with tile (5, 5) == 99
	home2, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap(home) second time failed: %v", err)
	}
	if home1 != home2 {
		t.Error("Expected LoadMap to return same in-memory pointer")
	}
	if tile := home2.GetTile(5, 5); tile != 99 {
		t.Errorf("Expected in-memory retained tile 99 at (5, 5), got %d", tile)
	}
}

func TestMainMenuModeSaveAndLoadOptions(t *testing.T) {
	setupTestEnvironment(t)
	SetWizardMode(false)

	// 1. Without active game and no save files:
	// Options: Load Game (disabled), New Game (enabled), Quit (enabled)
	SetParty(nil)
	SetMap(nil)
	menu := NewMainMenuMode()

	if len(menu.options) != 3 {
		t.Fatalf("Expected 3 options without game, got %d", len(menu.options))
	}
	if menu.options[0].Label != "Load Game" || menu.options[0].Enabled != false {
		t.Errorf("Expected disabled 'Load Game', got %+v", menu.options[0])
	}

	// 2. With active game (party & map):
	// Options: Load Game (disabled), Save Game (enabled), New Game (enabled), Quit (enabled)
	homeMap, _ := LoadMap("home")
	SetMap(homeMap)
	party, _ := NewParty(5, 5)
	SetParty(party)

	menu.RefreshOptions()
	if len(menu.options) != 4 {
		t.Fatalf("Expected 4 options with active game, got %d: %+v", len(menu.options), menu.options)
	}
	if menu.options[1].Label != "Save Game" || !menu.options[1].Enabled {
		t.Errorf("Expected enabled 'Save Game', got %+v", menu.options[1])
	}

	// 3. With Wizard Mode active:
	// Options: Load Game (disabled), Save Game (enabled), Save Pretty (enabled), New Game (enabled), Quit (enabled)
	SetWizardMode(true)
	menu.RefreshOptions()
	if len(menu.options) != 5 {
		t.Fatalf("Expected 5 options with wizard mode, got %d: %+v", len(menu.options), menu.options)
	}
	if menu.options[2].Label != "Save Pretty" || !menu.options[2].Enabled {
		t.Errorf("Expected enabled 'Save Pretty', got %+v", menu.options[2])
	}

	// 4. Save game to slot 1 and check Load Game becomes enabled
	if err := SaveGame(1, false); err != nil {
		t.Fatalf("SaveGame failed: %v", err)
	}
	menu.RefreshOptions()
	if !menu.options[0].Enabled {
		t.Errorf("Expected 'Load Game' to be enabled after saving, got %+v", menu.options[0])
	}
}

func TestMapStateLoadedEntirelyFromSaveWithoutTMX(t *testing.T) {
	setupTestEnvironment(t)
	ClearLoadedMaps()
	ClearAllFlags()

	// Load "home" map
	homeMap, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap('home') failed: %v", err)
	}
	SetMap(homeMap)

	// Remove any baseline TMX actors and set a custom actor and trigger
	homeMap.Actors = nil
	homeMap.AddActor(NewActor("save-only-actor", 1, 1, "wizard"))

	homeMap.Triggers = nil
	customTrig := &Trigger{
		ID:         99,
		Name:       "save-only-trigger",
		Area:       image.Rect(2, 2, 3, 3),
		ScriptPath: "custom.tengo",
		OnStep:     true,
	}
	homeMap.AddTrigger(customTrig)

	party, err := NewParty(1, 1)
	if err != nil {
		t.Fatalf("NewParty failed: %v", err)
	}
	SetParty(party)

	// Save to slot 1
	if err := SaveGame(1, false); err != nil {
		t.Fatalf("SaveGame failed: %v", err)
	}

	// Reset in-memory state
	ClearLoadedMaps()
	SetParty(nil)
	SetMap(nil)
	SetWorldMap(nil)

	// Load game from slot 1
	if err := LoadGame(1); err != nil {
		t.Fatalf("LoadGame failed: %v", err)
	}

	loadedMap := GetMap()
	if loadedMap == nil {
		t.Fatalf("Expected current map to be loaded")
	}

	// Verify that actors are strictly what was saved (baseline TMX actors are NOT re-added)
	if len(loadedMap.Actors) != 1 {
		t.Fatalf("Expected exactly 1 actor from save file, got %d", len(loadedMap.Actors))
	}
	if loadedMap.Actors[0].ID != "save-only-actor" {
		t.Errorf("Expected actor 'save-only-actor', got %q", loadedMap.Actors[0].ID)
	}

	// Verify triggers are strictly what was saved
	if len(loadedMap.Triggers) != 1 {
		t.Fatalf("Expected exactly 1 trigger from save file, got %d", len(loadedMap.Triggers))
	}
	if loadedMap.Triggers[0].Name != "save-only-trigger" {
		t.Errorf("Expected trigger 'save-only-trigger', got %q", loadedMap.Triggers[0].Name)
	}
}

// Test SlotSelectMode
func TestSlotSelectMode(t *testing.T) {
	setupTestEnvironment(t)
	homeMap, _ := LoadMap("home")
	SetMap(homeMap)
	party, _ := NewParty(5, 5)
	SetParty(party)
	_ = SaveGame(1, false)

	slotMode := NewSlotSelectMode(SlotActionLoad)
	if len(slotMode.slots) != 3 {
		t.Fatalf("Expected 3 slots in SlotSelectMode, got %d", len(slotMode.slots))
	}
	if !slotMode.slots[0].Exists {
		t.Error("Expected slot 1 to exist in SlotSelectMode")
	}
}
