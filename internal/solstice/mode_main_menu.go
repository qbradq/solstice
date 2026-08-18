package solstice

import (
	"fmt"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// MainMenuOption represents an option in the main menu.
type MainMenuOption struct {
	Label   string
	Enabled bool
}

// MainMenuMode manages the main menu screen, navigation, frame rendering, and game start / exit actions.
type MainMenuMode struct {
	selectedIndex int
	options       []MainMenuOption
}

// NewMainMenuMode creates a new instance of MainMenuMode with dynamic options based on game and save state.
func NewMainMenuMode() *MainMenuMode {
	m := &MainMenuMode{}
	m.RefreshOptions()
	return m
}

// RefreshOptions recomputes available menu options based on active game state, save files, and wizard mode.
func (m *MainMenuMode) RefreshOptions() {
	hasSaves := HasAnySaveFiles()
	hasGame := GetParty() != nil && GetMap() != nil

	opts := make([]MainMenuOption, 0, 5)

	// 1. Load Game (enabled only if save files exist)
	opts = append(opts, MainMenuOption{Label: "Load Game", Enabled: hasSaves})

	// 2. Save Game (available if party and map are loaded)
	if hasGame {
		opts = append(opts, MainMenuOption{Label: "Save Game", Enabled: true})
		if IsWizardMode() {
			opts = append(opts, MainMenuOption{Label: "Save Pretty", Enabled: true})
		}
	}

	// 3. New Game
	opts = append(opts, MainMenuOption{Label: "New Game", Enabled: true})

	// 4. Quit
	opts = append(opts, MainMenuOption{Label: "Quit", Enabled: true})

	m.options = opts

	// Ensure selectedIndex points to an enabled option
	if m.selectedIndex >= len(m.options) {
		m.selectedIndex = 0
	}
	if !m.options[m.selectedIndex].Enabled {
		for i, opt := range m.options {
			if opt.Enabled {
				m.selectedIndex = i
				break
			}
		}
	}
}

// moveSelection adjusts the selected option by delta (-1 for up, +1 for down), skipping disabled options.
func (m *MainMenuMode) moveSelection(delta int) {
	numOptions := len(m.options)
	if numOptions == 0 {
		return
	}

	curr := m.selectedIndex
	for i := 0; i < numOptions; i++ {
		curr = (curr + delta + numOptions) % numOptions
		if m.options[curr].Enabled {
			m.selectedIndex = curr
			return
		}
	}
}

func (m *MainMenuMode) Update(g *Game) error {
	UpdateAnimTicker()
	m.RefreshOptions()

	// Up movement keys: W, ArrowUp, K
	if inpututil.IsKeyJustPressed(ebiten.KeyW) || inpututil.IsKeyJustPressed(ebiten.KeyUp) || inpututil.IsKeyJustPressed(ebiten.KeyK) {
		m.moveSelection(-1)
	}

	// Down movement keys: S, ArrowDown, J
	if inpututil.IsKeyJustPressed(ebiten.KeyS) || inpututil.IsKeyJustPressed(ebiten.KeyDown) || inpututil.IsKeyJustPressed(ebiten.KeyJ) {
		m.moveSelection(1)
	}

	// Confirm selection: Enter, KeyKPEnter, Space
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyKPEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		return m.activateSelection(g)
	}

	// Escape key: return to game if a map and party exist
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if g != nil && g.currentMap != nil && g.party != nil {
			g.PopMode()
			if len(g.modeStack) == 0 {
				g.PushMode(NewMainMode())
			}
			return nil
		}
	}

	return nil
}

// InitNewGame initializes a fresh game session by resetting maps and flags, preloading the world map, and creating the initial spirit party.
func InitNewGame() error {
	ClearLoadedMaps()
	ClearAllFlags()
	ResetTengoREPL()
	StopCombat()
	if t := GetTerminal(); t != nil {
		t.Clear()
	}

	if _, err := PreloadWorldMap(); err != nil {
		return fmt.Errorf("failed to preload world map: %w", err)
	}

	party, err := NewParty(0, 0)
	if err != nil {
		return fmt.Errorf("failed to create party: %w", err)
	}
	SetParty(party)

	return nil
}

// activateSelection triggers the action for the currently selected menu option.
func (m *MainMenuMode) activateSelection(g *Game) error {
	if m.selectedIndex < 0 || m.selectedIndex >= len(m.options) {
		return nil
	}

	selected := m.options[m.selectedIndex]
	if !selected.Enabled {
		return nil
	}

	switch selected.Label {
	case "Load Game":
		if g != nil {
			g.PushMode(NewSlotSelectMode(SlotActionLoad))
		}
		return nil

	case "Save Game":
		if g != nil {
			g.PushMode(NewSlotSelectMode(SlotActionSave))
		}
		return nil

	case "Save Pretty":
		if g != nil {
			g.PushMode(NewSlotSelectMode(SlotActionSavePretty))
		}
		return nil

	case "New Game":
		if err := InitNewGame(); err != nil {
			log.Printf("failed to initialize new game: %v", err)
		}
		if g != nil {
			g.PopMode()
			if len(g.modeStack) == 0 {
				g.PushMode(NewMainMode())
			}
		}
		if err := RunNewGameScript(); err != nil {
			log.Printf("failed to run new_game.tengo: %v", err)
		}
		return nil

	case "Quit":
		return ebiten.Termination
	}

	return nil
}

func (m *MainMenuMode) Draw(g *Game, screen *ebiten.Image) {
	if screen == nil {
		return
	}

	// 1. Draw underlying game window content (map view if available, and common elements)
	if g != nil {
		scale := g.mapScale
		if scale == 0 {
			scale = 2
		}
		if g.currentMap != nil {
			g.currentMap.DrawCentered(screen, g.assets, g.party, scale)
		}
		DrawCommonUI(screen, g.assets, g.party, g.terminal)
	}

	// 2. Draw menu frame and options on top
	m.drawMenu(g, screen)
}

// drawMenu renders the frame and menu options centered on the screen.
func (m *MainMenuMode) drawMenu(g *Game, screen *ebiten.Image) {
	numOpts := len(m.options)
	menuWidth := 18
	menuHeight := numOpts + 4
	startX := (80 - menuWidth) / 2
	startY := (45 - menuHeight) / 2

	const (
		glyphTopLeft     = 123
		glyphTopRight    = 124
		glyphBottomLeft  = 125
		glyphBottomRight = 126
		glyphSolidBlock  = 127
		glyphRightArrow  = 2
	)

	frameColor := VGAPalette16[9]       // VGA Bright Blue
	activeTextColor := VGAPalette16[15]  // Bright White
	disabledTextColor := VGAPalette16[8] // Dark Gray

	var assets *Assets
	if g != nil {
		assets = g.assets
	}
	if assets == nil {
		assets = defaultAssets
	}
	if assets == nil {
		return
	}

	// 1. Fill menu box area with black
	for y := 0; y < menuHeight; y++ {
		for x := 0; x < menuWidth; x++ {
			assets.DrawGlyph8x8(screen, -1, startX+x, startY+y)
		}
	}

	// 2. Draw frame border with glyphs in VGA Bright Blue
	// Corners
	assets.DrawGlyph8x8Colored(screen, glyphTopLeft, startX, startY, frameColor)
	assets.DrawGlyph8x8Colored(screen, glyphTopRight, startX+menuWidth-1, startY, frameColor)
	assets.DrawGlyph8x8Colored(screen, glyphBottomLeft, startX, startY+menuHeight-1, frameColor)
	assets.DrawGlyph8x8Colored(screen, glyphBottomRight, startX+menuWidth-1, startY+menuHeight-1, frameColor)

	// Top & Bottom horizontal edges (solid blocks)
	for x := 1; x < menuWidth-1; x++ {
		assets.DrawGlyph8x8Colored(screen, glyphSolidBlock, startX+x, startY, frameColor)
		assets.DrawGlyph8x8Colored(screen, glyphSolidBlock, startX+x, startY+menuHeight-1, frameColor)
	}

	// Left & Right vertical edges (solid blocks)
	for y := 1; y < menuHeight-1; y++ {
		assets.DrawGlyph8x8Colored(screen, glyphSolidBlock, startX, startY+y, frameColor)
		assets.DrawGlyph8x8Colored(screen, glyphSolidBlock, startX+menuWidth-1, startY+y, frameColor)
	}

	// 3. Draw menu options and selection arrow
	for i, opt := range m.options {
		rowY := startY + 2 + i
		arrowX := startX + 2
		textX := startX + 4

		// Draw selection arrow
		if i == m.selectedIndex {
			assets.DrawGlyph8x8Colored(screen, glyphRightArrow, arrowX, rowY, activeTextColor)
		}

		// Draw option text
		col := activeTextColor
		if !opt.Enabled {
			col = disabledTextColor
		}
		assets.DrawString8x8Colored(screen, opt.Label, textX, rowY, col)
	}
}

// SlotAction identifies the action of SlotSelectMode.
type SlotAction int

const (
	SlotActionLoad SlotAction = iota
	SlotActionSave
	SlotActionSavePretty
)

// SlotSelectMode handles the 3-slot save/load selection menu.
type SlotSelectMode struct {
	action        SlotAction
	slots         []SaveSlotInfo
	selectedIndex int
}

// NewSlotSelectMode creates a slot selection modal for the given action.
func NewSlotSelectMode(action SlotAction) *SlotSelectMode {
	mode := &SlotSelectMode{
		action: action,
	}
	mode.RefreshSlots()
	return mode
}

// RefreshSlots reloads slot info from disk.
func (sm *SlotSelectMode) RefreshSlots() {
	infos, _ := GetAllSaveSlotInfos()
	sm.slots = infos

	// In load mode, default selection to first existing slot
	if sm.action == SlotActionLoad {
		for i, info := range sm.slots {
			if info.Exists {
				sm.selectedIndex = i
				break
			}
		}
	}
}

func (sm *SlotSelectMode) moveSelection(delta int) {
	numSlots := len(sm.slots)
	if numSlots == 0 {
		return
	}

	curr := sm.selectedIndex
	for i := 0; i < numSlots; i++ {
		curr = (curr + delta + numSlots) % numSlots
		if sm.action != SlotActionLoad || sm.slots[curr].Exists {
			sm.selectedIndex = curr
			return
		}
	}
}

func (sm *SlotSelectMode) Update(g *Game) error {
	UpdateAnimTicker()

	// Navigation
	if inpututil.IsKeyJustPressed(ebiten.KeyW) || inpututil.IsKeyJustPressed(ebiten.KeyUp) || inpututil.IsKeyJustPressed(ebiten.KeyK) {
		sm.moveSelection(-1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyS) || inpututil.IsKeyJustPressed(ebiten.KeyDown) || inpututil.IsKeyJustPressed(ebiten.KeyJ) {
		sm.moveSelection(1)
	}

	// Confirm
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyKPEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		return sm.activateSelection(g)
	}

	// Cancel / Back to Main Menu
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if g != nil {
			g.PopMode()
		}
		return nil
	}

	return nil
}

func (sm *SlotSelectMode) activateSelection(g *Game) error {
	if sm.selectedIndex < 0 || sm.selectedIndex >= len(sm.slots) {
		return nil
	}
	slotNum := sm.selectedIndex + 1

	switch sm.action {
	case SlotActionLoad:
		if !sm.slots[sm.selectedIndex].Exists {
			return nil
		}
		if err := LoadGame(slotNum); err != nil {
			log.Printf("failed to load game slot %d: %v", slotNum, err)
			return nil
		}
		if g != nil {
			// Clear modal modes and enter MainMode
			g.modeStack = nil
			g.PushMode(NewMainMode())
		}
		return nil

	case SlotActionSave:
		if err := SaveGame(slotNum, false); err != nil {
			log.Printf("failed to save game slot %d: %v", slotNum, err)
		}
		if g != nil {
			g.PopMode() // Return to MainMenuMode
		}
		return nil

	case SlotActionSavePretty:
		if err := SaveGame(slotNum, true); err != nil {
			log.Printf("failed to save pretty game slot %d: %v", slotNum, err)
		}
		if g != nil {
			g.PopMode() // Return to MainMenuMode
		}
		return nil
	}

	return nil
}

func (sm *SlotSelectMode) Draw(g *Game, screen *ebiten.Image) {
	if screen == nil {
		return
	}

	// 1. Draw underlying game view
	if g != nil {
		scale := g.mapScale
		if scale == 0 {
			scale = 2
		}
		if g.currentMap != nil {
			g.currentMap.DrawCentered(screen, g.assets, g.party, scale)
		}
		DrawCommonUI(screen, g.assets, g.party, g.terminal)
	}

	// 2. Draw slot selection modal
	const (
		menuWidth  = 30
		menuHeight = 8
		startX     = (80 - menuWidth) / 2
		startY     = (45 - menuHeight) / 2

		glyphTopLeft     = 123
		glyphTopRight    = 124
		glyphBottomLeft  = 125
		glyphBottomRight = 126
		glyphSolidBlock  = 127
		glyphRightArrow  = 2
	)

	frameColor := VGAPalette16[9]       // VGA Bright Blue
	activeTextColor := VGAPalette16[15]  // Bright White
	disabledTextColor := VGAPalette16[8] // Dark Gray

	var assets *Assets
	if g != nil {
		assets = g.assets
	}
	if assets == nil {
		assets = defaultAssets
	}
	if assets == nil {
		return
	}

	// 1. Fill menu box area with black
	for y := 0; y < menuHeight; y++ {
		for x := 0; x < menuWidth; x++ {
			assets.DrawGlyph8x8(screen, -1, startX+x, startY+y)
		}
	}

	// 2. Draw border
	assets.DrawGlyph8x8Colored(screen, glyphTopLeft, startX, startY, frameColor)
	assets.DrawGlyph8x8Colored(screen, glyphTopRight, startX+menuWidth-1, startY, frameColor)
	assets.DrawGlyph8x8Colored(screen, glyphBottomLeft, startX, startY+menuHeight-1, frameColor)
	assets.DrawGlyph8x8Colored(screen, glyphBottomRight, startX+menuWidth-1, startY+menuHeight-1, frameColor)

	for x := 1; x < menuWidth-1; x++ {
		assets.DrawGlyph8x8Colored(screen, glyphSolidBlock, startX+x, startY, frameColor)
		assets.DrawGlyph8x8Colored(screen, glyphSolidBlock, startX+x, startY+menuHeight-1, frameColor)
	}
	for y := 1; y < menuHeight-1; y++ {
		assets.DrawGlyph8x8Colored(screen, glyphSolidBlock, startX, startY+y, frameColor)
		assets.DrawGlyph8x8Colored(screen, glyphSolidBlock, startX+menuWidth-1, startY+y, frameColor)
	}

	// 3. Draw title
	title := "LOAD GAME"
	if sm.action == SlotActionSave {
		title = "SAVE GAME"
	} else if sm.action == SlotActionSavePretty {
		title = "SAVE PRETTY"
	}
	titleX := startX + (menuWidth-len(title))/2
	assets.DrawString8x8Colored(screen, title, titleX, startY+1, frameColor)

	// 4. Draw slot options
	for i, slot := range sm.slots {
		rowY := startY + 3 + i
		arrowX := startX + 2
		textX := startX + 4

		label := fmt.Sprintf("%d. %s", slot.Slot, slot.DisplayTime)

		if i == sm.selectedIndex {
			assets.DrawGlyph8x8Colored(screen, glyphRightArrow, arrowX, rowY, activeTextColor)
		}

		col := activeTextColor
		if sm.action == SlotActionLoad && !slot.Exists {
			col = disabledTextColor
		}
		assets.DrawString8x8Colored(screen, label, textX, rowY, col)
	}
}
