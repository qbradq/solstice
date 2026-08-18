package solstice

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestPreloadTileSet(t *testing.T) {
	ts, err := PreloadTileSet()
	if err != nil {
		t.Fatalf("PreloadTileSet failed: %v", err)
	}

	if ts.Name != "tileset" {
		t.Errorf("Expected tileset name 'tileset', got %q", ts.Name)
	}
	if ts.TileCount != 512 {
		t.Errorf("Expected tilecount 512, got %d", ts.TileCount)
	}
	if len(ts.Properties) == 0 {
		t.Error("Expected tile properties to be populated")
	}

	// Tile 1 has deep_water=true, water=true
	prop1 := ts.GetTileProperties(1)
	if !prop1.DeepWater || !prop1.Water {
		t.Errorf("Expected tile 1 to have DeepWater=true and Water=true, got %+v", prop1)
	}

	// Tile 4 has walkable=true
	prop4 := ts.GetTileProperties(4)
	if !prop4.Walkable {
		t.Errorf("Expected tile 4 to have Walkable=true, got %+v", prop4)
	}

	// Tile 13 has blocks_vis=true
	prop13 := ts.GetTileProperties(13)
	if !prop13.BlocksVis {
		t.Errorf("Expected tile 13 to have BlocksVis=true, got %+v", prop13)
	}

	// Tile 45 has actor_half_sprite=true / party_half_sprite=true
	prop45 := ts.GetTileProperties(45)
	if !prop45.ActorHalfSprite {
		t.Errorf("Expected tile 45 to have ActorHalfSprite=true, got %+v", prop45)
	}
}

func TestLoadMap(t *testing.T) {
	m, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}

	if m.Name != "home" {
		t.Errorf("Expected map name 'home', got %q", m.Name)
	}
	if m.Width != 32 || m.Height != 32 {
		t.Errorf("Expected map size 32x32, got %dx%d", m.Width, m.Height)
	}
	if len(m.Tiles) != 1024 {
		t.Errorf("Expected 1024 tiles, got %d", len(m.Tiles))
	}
}

func TestPreloadWorldMap(t *testing.T) {
	wm, err := PreloadWorldMap()
	if err != nil {
		t.Fatalf("PreloadWorldMap failed: %v", err)
	}

	if wm == nil {
		t.Fatal("Expected non-nil world map")
	}

	if wm.Name != "world" {
		t.Errorf("Expected world map name 'world', got %q", wm.Name)
	}

	if wm.Width != 128 || wm.Height != 128 {
		t.Errorf("Expected world map size 128x128, got %dx%d", wm.Width, wm.Height)
	}

	if GetWorldMap() != wm {
		t.Errorf("GetWorldMap() does not match preloaded world map")
	}

	// Verify trigger on world map: (38, 103)
	if len(wm.Triggers) == 0 {
		t.Fatalf("Expected triggers on world map, got 0")
	}
	trig := wm.Triggers[0]
	if trig.Area != image.Rect(38, 103, 39, 104) {
		t.Errorf("Expected world trigger area (38, 103, 39, 104), got %v", trig.Area)
	}
	if !trig.OnEnter {
		t.Errorf("Expected world trigger OnEnter=true, got %v", trig.OnEnter)
	}
	if trig.ScriptPath != "triggers/enter_home.tengo" {
		t.Errorf("Expected world trigger script 'triggers/enter_home.tengo', got %q", trig.ScriptPath)
	}
}

func TestCalculateVisibility(t *testing.T) {
	if _, err := PreloadTileSet(); err != nil {
		t.Fatalf("PreloadTileSet failed: %v", err)
	}

	m, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}

	// 1. Open space test with radius 4: 9x9 grid
	// Fill a 9x9 area around (15, 15) with open floor (tile 4, blocks_vis=false)
	for y := 11; y <= 19; y++ {
		for x := 11; x <= 19; x++ {
			m.SetTile(x, y, 4)
		}
	}

	vis := m.CalculateVisibility(15, 15, 4)
	if vis == nil {
		t.Fatal("Expected non-nil visibility bitset")
	}
	if vis.Len() != 81 {
		t.Fatalf("Expected 81 bits for radius 4, got %d", vis.Len())
	}
	if vis.Count() != 81 {
		t.Errorf("Expected all 81 tiles visible in open space, got %d", vis.Count())
	}

	// 2. Wall blocking test with radius 4:
	// Place a wall line at x=17 (dx=+2 from center x=15) from y=11 to y=19 (tile 13, blocks_vis=true)
	for y := 11; y <= 19; y++ {
		m.SetTile(17, y, 13)
		// Tiles behind the wall: x=18, 19
		m.SetTile(18, y, 4)
		m.SetTile(19, y, 4)
	}

	visWall := m.CalculateVisibility(15, 15, 4)
	// Local grid coords for center (15, 15) is (lx=4, ly=4)
	// Local coord for wall at x=17 is lx = 4 + (17 - 15) = 6
	// The wall at lx=6 should be visible
	centerWallIdx := uint(4*9 + 6)
	if !visWall.Test(centerWallIdx) {
		t.Errorf("Expected wall tile at lx=6, ly=4 to be visible")
	}

	// Behind the wall at lx=7 (x=18, ly=4): should NOT be visible
	behindWallIdx := uint(4*9 + 7)
	if visWall.Test(behindWallIdx) {
		t.Errorf("Expected tile behind wall at lx=7, ly=4 to NOT be visible")
	}
	behindWallIdx2 := uint(4*9 + 8)
	if visWall.Test(behindWallIdx2) {
		t.Errorf("Expected tile behind wall at lx=8, ly=4 to NOT be visible")
	}

	// 3. Second generation visibility test:
	// A blocking wall tile not reached in gen 1 requires at least 2 adjacent visible locations in vis1 to be marked visible in vis2.
	// Place wall at x=18, y=15 (adjacent only to lx=6 in vis1) -> only 1 adjacent visible tile -> NOT marked in vis2.
	m.SetTile(18, 15, 13)
	visSingleAdj := m.CalculateVisibility(15, 15, 4)
	if visSingleAdj.Test(uint(4*9 + 7)) {
		t.Errorf("Expected wall at lx=7 with only 1 adjacent visible tile to NOT be visible in second generation")
	}

	// Now create a corner wall at (x=17, y=14) which is not in vis1 (e.g. if we block flood fill to it),
	// or create a wall tile with 2 adjacent open visible floor tiles.
	// For example, if center is (15, 15), floor at (15, 14) is visible in vis1, and floor at (16, 15) is visible in vis1.
	// If (16, 14) is a wall (blocks_vis=true), but (16, 14) was reached in gen 1... wait, in gen 1 all direct neighbors of floor are reached.
	// But suppose we set up a wall at (17, 16) that is adjacent to (17, 15) [vis1=true] and (16, 16) [vis1=true].
	// (17, 15) was a wall in gen 1. (16, 16) is open floor in gen 1.
	// (17, 16) is adjacent to both (17, 15) and (16, 16), which are BOTH in vis1!
	// Therefore (17, 16) has 2 adjacent locations in vis1, so gen 2 marks it visible!
	m.SetTile(17, 16, 13) // Wall at (17, 16), local (lx=6, ly=5)
	// (16, 16) is open floor (local lx=5, ly=5) in vis1
	// (17, 15) is wall (local lx=6, ly=4) in vis1
	// (17, 16) has 2 adjacent visible tiles in vis1 -> marked visible in vis2
	visDoubleAdj := m.CalculateVisibility(15, 15, 4)
	if !visDoubleAdj.Test(uint(5*9 + 6)) {
		t.Errorf("Expected wall tile at lx=6, ly=5 with 2 adjacent visible tiles in vis1 to be visible in second generation")
	}

	// 4. Test standing on a blocking tile (e.g. center tile at (15, 15) has blocks_vis=true):
	// Even when standing on a blocking tile, the center and all 4 adjacent locations are checked
	// and propagate into surrounding open floor.
	m.SetTile(15, 15, 13) // center is blocking
	visCenterBlocking := m.CalculateVisibility(15, 15, 4)
	if !visCenterBlocking.Test(uint(4*9 + 4)) {
		t.Errorf("Expected center tile to be visible even when blocking")
	}
	if !visCenterBlocking.Test(uint(3*9 + 4)) || !visCenterBlocking.Test(uint(5*9 + 4)) ||
		!visCenterBlocking.Test(uint(4*9 + 3)) || !visCenterBlocking.Test(uint(4*9 + 5)) {
		t.Errorf("Expected all 4 adjacent tiles to be visible when standing on blocking tile")
	}
	// And open floor at dx=-2 (lx=2, ly=4) should also be visible because flood fill propagated from adjacent floor (lx=3, ly=4)
	if !visCenterBlocking.Test(uint(4*9 + 2)) {
		t.Errorf("Expected flood fill to propagate from adjacent open tiles when standing on blocking center")
	}

	// 5. Test wall at edge of map:
	// Place wall at x=1 extending from y=0 to y=10. Player is at (2, 2).
	// Tile (0, 2) is behind the wall and should NOT be visible (must not leak around y < 0 edge).
	for y := 0; y <= 10; y++ {
		m.SetTile(1, y, 13)
		m.SetTile(0, y, 4)
		m.SetTile(2, y, 4)
	}
	visEdge := m.CalculateVisibility(2, 2, 4)
	// (0, 2) is dx=-2, dy=0 -> lx = 4 + (-2) = 2, ly = 4
	if visEdge.Test(uint(4*9 + 2)) {
		t.Errorf("Expected tile (0, 2) behind edge wall to NOT be visible")
	}

	// 6. Negative radius returns empty bitset
	visNeg := m.CalculateVisibility(15, 15, -1)
	if visNeg == nil || visNeg.Len() != 0 {
		t.Errorf("Expected empty bitset for negative radius")
	}
}

func TestLoadMapObjectLayerActors(t *testing.T) {
	ClearLoadedMaps()
	if _, err := PreloadSpriteDefs(); err != nil {
		t.Fatalf("PreloadSpriteDefs failed: %v", err)
	}
	if _, err := PreloadActorDefs(); err != nil {
		t.Fatalf("PreloadActorDefs failed: %v", err)
	}

	homeMap, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}

	if len(homeMap.Actors) < 2 {
		t.Fatalf("Expected at least 2 actors loaded from homeMap object layer, got %d", len(homeMap.Actors))
	}

	// Verify object 1 (guard) at x=271.75, y=192, height=16 (GID tile object, Y bottom-left -> top-left y=176) -> tile (17, 11)
	guardActor := homeMap.GetActorAt(17, 11)
	if guardActor == nil {
		t.Errorf("Expected guard actor at tile (17, 11), got nil")
	} else {
		if guardActor.DialogScript != "dialog/guard.tengo" {
			t.Errorf("Expected guard DialogScript 'dialog/guard.tengo', got %q", guardActor.DialogScript)
		}
	}

	// Verify object 2 (wizard) at x=304, y=240.25, height=16 (GID tile object, Y bottom-left -> top-left y=224.25) -> tile (19, 14)
	wizardActor := homeMap.GetActorAt(19, 14)
	if wizardActor == nil {
		t.Errorf("Expected wizard actor at tile (19, 14), got nil")
	}
}

func TestMapMoveParty(t *testing.T) {
	if _, err := PreloadTileSet(); err != nil {
		t.Fatalf("PreloadTileSet failed: %v", err)
	}

	m, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}

	// Make tile (10, 10) walkable and tile (11, 10) non-walkable
	m.SetTile(10, 10, 4) // Walkable tile
	m.SetTile(11, 10, 0) // Non-walkable tile

	p, err := NewParty(10, 10)
	if err != nil {
		t.Fatalf("NewParty failed: %v", err)
	}

	// Moving to blocked tile should fail
	if m.MoveParty(p, 1, 0) {
		t.Error("Expected MoveParty to (11, 10) to return false (blocked)")
	}

	if p.X != 10 || p.Y != 10 {
		t.Errorf("Expected party position to remain (10, 10), got (%d, %d)", p.X, p.Y)
	}

	// Make tile (11, 10) walkable
	m.SetTile(11, 10, 4)

	// Moving to walkable tile should succeed
	if !m.MoveParty(p, 1, 0) {
		t.Error("Expected MoveParty to (11, 10) to return true")
	}

	if p.X != 11 || p.Y != 10 {
		t.Errorf("Expected party position (11, 10), got (%d, %d)", p.X, p.Y)
	}
}

func TestSpiritModePartyMovement(t *testing.T) {
	if _, err := PreloadTileSet(); err != nil {
		t.Fatalf("PreloadTileSet failed: %v", err)
	}

	m, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}

	// Set tile (10, 10) to walkable and (11, 10) to tile ID with spirit_passable=true (e.g. tile 78 if door)
	m.SetTile(10, 10, 4) // Walkable
	m.SetTile(11, 10, 78) // spirit_passable tile from tileset.tsx

	// Spirit party (0 members)
	spiritParty, err := NewParty(10, 10)
	if err != nil {
		t.Fatalf("NewParty failed: %v", err)
	}

	if !spiritParty.IsSpiritMode() {
		t.Fatal("Expected spiritParty to be in spirit mode")
	}

	// Spirit party SHOULD be able to move onto spirit_passable tile (11, 10)
	if !m.MoveParty(spiritParty, 1, 0) {
		t.Error("Expected spirit mode party to move onto spirit_passable tile")
	}

	// Living party (1 member)
	livingParty, err := NewParty(10, 10, Actor{Entity: Entity{Name: "Hero"}})
	if err != nil {
		t.Fatalf("NewParty failed: %v", err)
	}

	if livingParty.IsSpiritMode() {
		t.Fatal("Expected livingParty NOT to be in spirit mode")
	}

	// Living party SHOULD NOT be able to move onto spirit_passable (non-walkable) tile (11, 10)
	if m.MoveParty(livingParty, 1, 0) {
		t.Error("Expected living party to be blocked by spirit_passable (non-walkable) tile")
	}

	// Test actor collision: place an actor on walkable tile (12, 10)
	m.SetTile(12, 10, 4) // Walkable tile
	m.AddActor(&Actor{Entity: Entity{X: 12, Y: 10}, ID: "test-npc"})

	// Position parties at (11, 10) with tile (11, 10) set to walkable
	m.SetTile(11, 10, 4)
	spiritParty.X = 11
	spiritParty.Y = 10
	livingParty.X = 11
	livingParty.Y = 10

	// Spirit party SHOULD be able to move onto tile with actor
	if !m.MoveParty(spiritParty, 1, 0) {
		t.Error("Expected spirit mode party to be able to move onto tile occupied by an actor")
	}

	// Living party SHOULD NOT be able to move onto tile with actor
	if m.MoveParty(livingParty, 1, 0) {
		t.Error("Expected living party to be blocked by an actor")
	}
}

func TestMapDrawCenteredVisibility(t *testing.T) {
	assets, err := LoadAssets()
	if err != nil {
		t.Fatalf("LoadAssets failed: %v", err)
	}
	if _, err := PreloadSpriteDefs(); err != nil {
		t.Fatalf("PreloadSpriteDefs failed: %v", err)
	}

	m, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}

	party, err := NewParty(15, 15)
	if err != nil {
		t.Fatalf("NewParty failed: %v", err)
	}

	// Add an actor behind a blocking wall
	for y := 0; y < 32; y++ {
		m.SetTile(17, y, 13) // Wall line at x=17
	}
	hiddenActor := NewActor("hidden", 18, 15, "guard")
	m.AddActor(hiddenActor)

	visibleActor := NewActor("visible", 14, 15, "guard")
	m.AddActor(visibleActor)

	screen := ebiten.NewImage(640, 360)

	// Draw at scale 2 (11x11) and scale 1 (23x23)
	m.DrawCentered(screen, assets, party, 2)
	m.DrawCentered(screen, assets, party, 1)
	m.Draw(screen, assets, 2)
	m.Draw(screen, assets, 1)
}

func TestMapPropertiesAndExitToWorld(t *testing.T) {
	if _, err := PreloadTileSet(); err != nil {
		t.Fatalf("PreloadTileSet failed: %v", err)
	}

	worldMap, err := PreloadWorldMap()
	if err != nil {
		t.Fatalf("PreloadWorldMap failed: %v", err)
	}

	homeMap, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}

	// Verify home map has ExitToWorld = true
	if !homeMap.Properties.ExitToWorld {
		t.Errorf("Expected homeMap.Properties.ExitToWorld to be true")
	}

	// Place party at boundary of home map: (0, 15)
	party, err := NewParty(0, 15)
	if err != nil {
		t.Fatalf("NewParty failed: %v", err)
	}
	party.WorldX = 38
	party.WorldY = 103

	SetMap(homeMap)
	SetParty(party)

	// Move left (dx=-1) stepping outside the bounds of home map (targetX = -1 < 0)
	moved := homeMap.MoveParty(party, -1, 0)
	if !moved {
		t.Fatalf("Expected MoveParty outside bounds of map with ExitToWorld=true to return true")
	}

	// Verify active map is now worldMap
	if GetMap() != worldMap {
		t.Errorf("Expected current map to be worldMap after exit_to_world, got %v", GetMap().Name)
	}

	// Verify party position is restored to world position (38, 103)
	if party.X != 38 || party.Y != 103 {
		t.Errorf("Expected party position on world map to be (38, 103), got (%d, %d)", party.X, party.Y)
	}

	// Test moving outside bounds on a map with ExitToWorld = false
	worldMap.Properties.ExitToWorld = false
	party.X = 0
	party.Y = 0
	blocked := worldMap.MoveParty(party, -1, 0)
	if blocked {
		t.Errorf("Expected MoveParty outside bounds with ExitToWorld=false to be blocked")
	}
}

func TestTriggersAndWizardMode(t *testing.T) {
	term := NewTerminal()
	SetTerminal(term)

	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}
	if _, err := PreloadSpriteDefs(); err != nil {
		t.Fatalf("PreloadSpriteDefs failed: %v", err)
	}

	homeMap, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}

	// 1. Add trigger to homeMap: (12, 12) to (17, 14), on enter, triggers/test.tengo
	debugTrig := &Trigger{
		ID:         9999,
		Name:       "debug_home_trigger",
		Area:       image.Rect(12, 12, 17, 14),
		ScriptPath: "triggers/test.tengo",
		OnEnter:    true,
	}
	homeMap.AddTrigger(debugTrig)

	if debugTrig.Area != image.Rect(12, 12, 17, 14) {
		t.Errorf("Expected debug trigger area (12, 12, 17, 14), got %v", debugTrig.Area)
	}
	if !debugTrig.OnEnter || debugTrig.OnStep {
		t.Errorf("Expected debug trigger OnEnter=true, OnStep=false, got OnEnter=%v, OnStep=%v", debugTrig.OnEnter, debugTrig.OnStep)
	}
	if debugTrig.ScriptPath != "triggers/test.tengo" {
		t.Errorf("Expected debug trigger script 'triggers/test.tengo', got %q", debugTrig.ScriptPath)
	}

	// 2. Test OnEnter trigger activation
	term.Clear()
	// Inside trigger area (13, 12)
	homeMap.ActivateTriggersOnEnter(13, 12, "party")
	lines := term.GetLineTexts()
	if len(lines) == 0 || lines[len(lines)-1] != "party trigger" {
		t.Errorf("Expected 'party trigger' when activating OnEnter inside trigger area, got lines: %v", lines)
	}

	// Outside trigger area (11, 12)
	term.Clear()
	homeMap.ActivateTriggersOnEnter(11, 12, "party")
	if len(term.GetLineTexts()) != 0 {
		t.Errorf("Expected no trigger activation outside trigger area, got: %v", term.GetLineTexts())
	}

	// 3. Test OnStep trigger activation on party move
	stepTrig := &Trigger{
		ID:         1001,
		Name:       "step_trigger",
		Area:       image.Rect(14, 15, 15, 16), // (14, 15)
		ScriptPath: "triggers/test.tengo",
		OnStep:     true,
	}
	homeMap.AddTrigger(stepTrig)
	homeMap.SetTile(14, 15, 4) // Walkable tile

	party, err := NewParty(13, 15)
	if err != nil {
		t.Fatalf("NewParty failed: %v", err)
	}
	SetParty(party)

	term.Clear()
	moved := homeMap.MoveParty(party, 1, 0) // Moves onto (14, 15)
	if !moved {
		t.Fatalf("Expected party to move to (14, 15)")
	}
	lines = term.GetLineTexts()
	if len(lines) == 0 || lines[len(lines)-1] != "party trigger" {
		t.Errorf("Expected 'party trigger' on step into trigger area, got: %v", lines)
	}

	// 4. Test Wizard Mode rendering
	assets, err := LoadAssets()
	if err != nil {
		t.Fatalf("LoadAssets failed: %v", err)
	}
	screen := ebiten.NewImage(640, 360)

	SetWizardMode(false)
	if IsWizardMode() {
		t.Errorf("Expected WizardMode to be false")
	}
	homeMap.DrawCentered(screen, assets, party, 2)
	homeMap.DrawCentered(screen, assets, party, 1)

	SetWizardMode(true)
	if !IsWizardMode() {
		t.Errorf("Expected WizardMode to be true")
	}
	homeMap.DrawCentered(screen, assets, party, 2)
	homeMap.DrawCentered(screen, assets, party, 1)
	SetWizardMode(false)
}

func TestReloadMap(t *testing.T) {
	ClearLoadedMaps()

	// 1. If named map is not in loaded maps set, don't load it
	reloaded, err := ReloadMap("home")
	if err != nil {
		t.Fatalf("ReloadMap on unloaded map returned error: %v", err)
	}
	if reloaded != nil {
		t.Errorf("Expected nil when reloading unloaded map, got %v", reloaded)
	}
	allLoaded := GetAllLoadedMaps()
	if len(allLoaded) != 0 {
		t.Errorf("Expected loaded maps set to remain empty, got %d maps", len(allLoaded))
	}

	// 2. Load "home" map and modify it
	homeMap, err := LoadMap("home")
	if err != nil {
		t.Fatalf("LoadMap('home') failed: %v", err)
	}
	SetMap(homeMap)

	originalTile := homeMap.GetTile(5, 5)
	homeMap.SetTile(5, 5, 999)
	if homeMap.GetTile(5, 5) != 999 {
		t.Fatalf("Failed to modify tile for test")
	}

	// Reload "home" map
	newHomeMap, err := ReloadMap("home")
	if err != nil {
		t.Fatalf("ReloadMap('home') failed: %v", err)
	}
	if newHomeMap == nil {
		t.Fatalf("Expected non-nil reloaded map")
	}
	if newHomeMap == homeMap {
		t.Errorf("Expected new map instance, got same instance")
	}
	if newHomeMap.GetTile(5, 5) != originalTile {
		t.Errorf("Expected tile at (5, 5) to be restored to %d, got %d", originalTile, newHomeMap.GetTile(5, 5))
	}
	if GetMap() != newHomeMap {
		t.Errorf("Expected current map pointer to be updated to newHomeMap")
	}

	// 3. World map reload updates world map pointer
	worldMap, err := LoadMap("world")
	if err != nil {
		t.Fatalf("LoadMap('world') failed: %v", err)
	}
	SetWorldMap(worldMap)

	newWorldMap, err := ReloadMap("world")
	if err != nil {
		t.Fatalf("ReloadMap('world') failed: %v", err)
	}
	if newWorldMap == nil {
		t.Fatalf("Expected non-nil reloaded world map")
	}
	if GetWorldMap() != newWorldMap {
		t.Errorf("Expected world map pointer to be updated to newWorldMap")
	}
}


