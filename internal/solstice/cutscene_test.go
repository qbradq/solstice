package solstice

import (
	"testing"

	"github.com/d5/tengo/v2"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestCutSceneModuleAndExecution(t *testing.T) {
	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}
	if _, err := PreloadSpriteDefs(); err != nil {
		t.Fatalf("PreloadSpriteDefs failed: %v", err)
	}
	if _, err := PreloadTileSet(); err != nil {
		t.Fatalf("PreloadTileSet failed: %v", err)
	}

	m, err := LoadMap("kings_shrine")
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}
	SetMap(m)

	actor := &Actor{
		Entity: Entity{
			ID: "test-actor",
			X:  10,
			Y:  10,
		},
	}
	m.AddActor(actor)

	ClearCutScene()
	if IsCutSceneActive() {
		t.Error("Expected cut scene to be inactive after ClearCutScene()")
	}

	// Execute test_cut_scene.tengo
	if err := ExecuteMapScript("map/test_cut_scene.tengo"); err != nil {
		t.Fatalf("ExecuteMapScript(test_cut_scene.tengo) failed: %v", err)
	}

	if !IsCutSceneActive() {
		t.Fatal("Expected cut scene to be active after running test_cut_scene.tengo")
	}

	game := &Game{}

	// Initial state before frame updates
	// Frame 0: UpdateCutScene executes set_tile(10, 10, 4) and hits cs.next() (CmdDelay 1)
	active := UpdateCutScene(game)
	if !active {
		t.Error("Expected cut scene to be active after first update")
	}
	if tile := m.GetTile(10, 10); tile != 4 {
		t.Errorf("Expected tile (10, 10) to be set to 4, got %d", tile)
	}
	// Actor should not have moved yet
	if actor.X != 10 || actor.Y != 10 {
		t.Errorf("Expected actor position (10, 10), got (%d, %d)", actor.X, actor.Y)
	}

	// Advance global animation frame by 1 (simulating 15 ticks)
	for i := 0; i < 15; i++ {
		UpdateAnimTicker()
	}

	// Next frame: UpdateCutScene drains move 'e' (X=11), move 's' (Y=11), and hits cs.delay(2)
	active = UpdateCutScene(game)
	if !active {
		t.Error("Expected cut scene to remain active")
	}
	if actor.X != 11 || actor.Y != 11 {
		t.Errorf("Expected actor position (11, 11) after moving 'e' and 's', got (%d, %d)", actor.X, actor.Y)
	}

	// Advance 1 animation frame (delay remaining was 2, now 1)
	for i := 0; i < 15; i++ {
		UpdateAnimTicker()
	}
	active = UpdateCutScene(game)
	if !active {
		t.Error("Expected cut scene to remain active during 2-frame delay (1 frame passed)")
	}
	if m.GetActorByID("test-actor") == nil {
		t.Error("Expected actor to still exist during delay")
	}

	// Advance second animation frame (delay remaining was 1, now 0)
	for i := 0; i < 15; i++ {
		UpdateAnimTicker()
	}
	active = UpdateCutScene(game)
	if !active {
		t.Error("Expected cut scene to remain active on cs.next() after remove_actor")
	}
	if m.GetActorByID("test-actor") != nil {
		t.Error("Expected actor to be removed after cs.remove_actor")
	}

	// Advance 1 more animation frame to clear final cs.next()
	for i := 0; i < 15; i++ {
		UpdateAnimTicker()
	}
	active = UpdateCutScene(game)
	if active {
		t.Error("Expected cut scene to be finished (inactive)")
	}
}

func TestIntroCutSceneScriptExecution(t *testing.T) {
	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}
	if _, err := PreloadSpriteDefs(); err != nil {
		t.Fatalf("PreloadSpriteDefs failed: %v", err)
	}
	if _, err := PreloadTileSet(); err != nil {
		t.Fatalf("PreloadTileSet failed: %v", err)
	}

	m, err := LoadMap("kings_shrine")
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}
	SetMap(m)

	ClearCutScene()

	// Spawn actors like intro.tengo does
	m.AddActor(&Actor{Entity: Entity{ID: "wizard-1", X: 15, Y: 14}})
	m.AddActor(&Actor{Entity: Entity{ID: "wizard-2", X: 14, Y: 15}})
	m.AddActor(&Actor{Entity: Entity{ID: "wizard-3", X: 16, Y: 15}})
	m.AddActor(&Actor{Entity: Entity{ID: "duke-lafey", X: 15, Y: 16}})

	// Execute intro_cut_scene.tengo
	if err := ExecuteMapScript("map/intro_cut_scene.tengo"); err != nil {
		t.Fatalf("ExecuteMapScript(intro_cut_scene.tengo) failed: %v", err)
	}

	if !IsCutSceneActive() {
		t.Fatal("Expected cut scene to be active after loading intro_cut_scene.tengo")
	}

	game := &Game{}

	// Run until cutscene completes
	maxFrames := 200
	frames := 0
	for IsCutSceneActive() && frames < maxFrames {
		UpdateCutScene(game)
		for i := 0; i < 15; i++ {
			UpdateAnimTicker()
		}
		frames++
	}

	if IsCutSceneActive() {
		t.Errorf("Cut scene did not complete within %d animation frames", maxFrames)
	}

	// Verify all actors were removed by end of cutscene
	if m.GetActorByID("duke-lafey") != nil {
		t.Error("Expected duke-lafey to be removed")
	}
	if m.GetActorByID("wizard-1") != nil {
		t.Error("Expected wizard-1 to be removed")
	}
	if m.GetActorByID("wizard-2") != nil {
		t.Error("Expected wizard-2 to be removed")
	}
	if m.GetActorByID("wizard-3") != nil {
		t.Error("Expected wizard-3 to be removed")
	}
}

func TestCutSceneMainModeControls(t *testing.T) {
	ClearCutScene()
	EnqueueCutSceneCommand(CutSceneCommand{Type: CmdDelay, Frames: 5})

	mainMode := NewMainMode()
	game := &Game{
		mapScale: 2,
	}
	game.PushMode(mainMode)

	// Update mainMode while cut scene is active
	if err := mainMode.Update(game); err != nil {
		t.Fatalf("MainMode.Update failed: %v", err)
	}

	if !IsCutSceneActive() {
		t.Error("Expected cut scene to be active")
	}
}

func TestCutSceneAnimate(t *testing.T) {
	if err := InitScriptSystem(); err != nil {
		t.Fatalf("InitScriptSystem failed: %v", err)
	}
	if _, err := PreloadSpriteDefs(); err != nil {
		t.Fatalf("PreloadSpriteDefs failed: %v", err)
	}
	if _, err := PreloadTileSet(); err != nil {
		t.Fatalf("PreloadTileSet failed: %v", err)
	}

	m, err := LoadMap("kings_shrine")
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}
	SetMap(m)

	ClearCutScene()
	SetAnimFrame(0)

	scriptSrc := `
cs := import("cut-scene")
cs.animate(12, 12, 3, 300, 3)
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

	if !IsCutSceneActive() {
		t.Fatal("Expected cutscene to be active after cs.animate")
	}

	game := &Game{}

	// Update 1: Starts animation (frame 0)
	UpdateCutScene(game)
	anim := GetActiveTileAnim()
	if anim == nil {
		t.Fatal("Expected active tile anim to be non-nil")
	}
	if anim.X != 12 || anim.Y != 12 || anim.BaseTile != 300 || anim.AnimFrames != 3 || anim.Duration != 3 {
		t.Errorf("Unexpected anim props: %+v", anim)
	}

	// Verify DrawCentered renders without error and centers on the animation tile
	screen := ebiten.NewImage(352, 352)
	assets, err := LoadAssets()
	if err != nil {
		t.Fatalf("LoadAssets failed: %v", err)
	}
	party, _ := NewParty(5, 5)
	m.DrawCentered(screen, assets, party, 2)

	// Advance 1 animation frame (frame 1)
	SetAnimFrame(1)
	UpdateCutScene(game)
	if GetActiveTileAnim() == nil {
		t.Error("Expected active tile anim to persist at frame 1")
	}
	m.DrawCentered(screen, assets, party, 2)

	// Advance 1 animation frame (frame 2)
	SetAnimFrame(2)
	UpdateCutScene(game)
	if GetActiveTileAnim() == nil {
		t.Error("Expected active tile anim to persist at frame 2")
	}

	// Advance 1 animation frame (frame 3 - duration expired)
	SetAnimFrame(3)
	UpdateCutScene(game)
	if GetActiveTileAnim() != nil {
		t.Error("Expected active tile anim to be cleared after duration expired")
	}
	if IsCutSceneActive() {
		t.Error("Expected cutscene to be inactive after animation duration")
	}
}

