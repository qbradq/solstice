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

// NewMainMenuMode creates a new instance of MainMenuMode with default options.
func NewMainMenuMode() *MainMenuMode {
	return &MainMenuMode{
		selectedIndex: 1, // Default to "New Game" since "Continue" is disabled
		options: []MainMenuOption{
			{Label: "Continue", Enabled: false},
			{Label: "New Game", Enabled: true},
			{Label: "Quit", Enabled: true},
		},
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

// InitNewGame initializes a fresh game session by preloading the world map and creating the initial party.
func InitNewGame() error {
	if _, err := PreloadWorldMap(); err != nil {
		return fmt.Errorf("failed to preload world map: %w", err)
	}

	kevinActor, err := NewActorFromDef("kevin-1", "kevin", 0, 0)
	if err != nil {
		return fmt.Errorf("failed to create kevin actor: %w", err)
	}

	party, err := NewParty(0, 0, *kevinActor)
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
	const (
		menuWidth  = 16
		menuHeight = 7
		startX     = (80 - menuWidth) / 2  // 32
		startY     = (45 - menuHeight) / 2 // 19

		glyphTopLeft     = 123
		glyphTopRight    = 124
		glyphBottomLeft  = 125
		glyphBottomRight = 126
		glyphSolidBlock  = 127
		glyphRightArrow  = 2
	)

	frameColor := VGAPalette16[9]  // VGA Bright Blue
	activeTextColor := VGAPalette16[15] // Bright White
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
