package solstice

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"

	"solstice/data"

	"github.com/d5/tengo/v2"
)

var (
	scriptMu         sync.RWMutex
	moduleMap        *tengo.ModuleMap
	compiledScripts  = make(map[string]*tengo.Compiled)
	rawScriptSources = make(map[string][]byte)
)

// InitScriptSystem initializes the Tengo scripting system by recursively loading and pre-compiling
// all .tengo files in the data/scripts directory from data.FS.
func InitScriptSystem() error {
	scriptMu.Lock()
	defer scriptMu.Unlock()

	moduleMap = tengo.NewModuleMap()

	// Register builtin "game" module
	gameModule := map[string]tengo.Object{
		"log": &tengo.UserFunction{
			Name: "log",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) == 0 {
					return tengo.UndefinedValue, nil
				}
				msg, ok := tengo.ToString(args[0])
				if !ok {
					msg = args[0].String()
				}
				if defaultTerminal != nil {
					defaultTerminal.AddMessage(msg)
				}
				return tengo.UndefinedValue, nil
			},
		},
		"set_map_tile": &tengo.UserFunction{
			Name: "set_map_tile",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 3 {
					return tengo.UndefinedValue, fmt.Errorf("set_map_tile requires 3 arguments: x, y, tileIdx")
				}
				x, ok1 := tengo.ToInt(args[0])
				y, ok2 := tengo.ToInt(args[1])
				tileIdx, ok3 := tengo.ToInt(args[2])
				if !ok1 || !ok2 || !ok3 {
					return tengo.UndefinedValue, fmt.Errorf("set_map_tile arguments must be integers")
				}
				if defaultMap != nil {
					defaultMap.SetTile(x, y, tileIdx)
				}
				return tengo.UndefinedValue, nil
			},
		},
	}
	moduleMap.AddBuiltinModule("game", gameModule)

	// Walk data/scripts directory in embedded data.FS
	root := "scripts"
	entries, err := fs.ReadDir(data.FS, root)
	if err != nil || len(entries) == 0 {
		root = "data/scripts"
	}

	err = fs.WalkDir(data.FS, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".tengo") {
			src, err := data.FS.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read script file %s: %w", path, err)
			}

			normPath := filepath.ToSlash(path)
			rawScriptSources[normPath] = src

			// Pre-compile script
			script := tengo.NewScript(src)
			script.SetImports(moduleMap)
			_ = script.Add("tile", nil)
			compiled, err := script.Compile()
			if err != nil {
				return fmt.Errorf("failed to pre-compile script %s: %w", path, err)
			}

			// Store under multiple keys for flexible lookup (e.g. "data/scripts/main.tengo", "scripts/main.tengo", "main.tengo")
			compiledScripts[normPath] = compiled

			relPath, _ := filepath.Rel(root, path)
			if relPath != "" {
				compiledScripts[filepath.ToSlash(relPath)] = compiled
			}
			if !strings.HasPrefix(normPath, "data/") {
				compiledScripts["data/"+normPath] = compiled
			}

			// Register as a source module for script imports if applicable
			modName := strings.TrimSuffix(filepath.ToSlash(relPath), ".tengo")
			if modName != "" && modName != "main" {
				moduleMap.AddSourceModule(modName, src)
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to load and pre-compile scripts from %s: %w", root, err)
	}

	return nil
}

// RunMainScript executes the script data/scripts/main.tengo.
func RunMainScript() error {
	return ExecuteScript("data/scripts/main.tengo")
}

// ExecuteScript executes a script by path or name (e.g. "data/scripts/main.tengo").
func ExecuteScript(scriptPath string) error {
	scriptMu.RLock()
	cleanKey := filepath.ToSlash(scriptPath)
	compiled, ok := compiledScripts[cleanKey]
	if !ok {
		cleanKey = strings.TrimPrefix(cleanKey, "data/")
		compiled, ok = compiledScripts[cleanKey]
	}
	scriptMu.RUnlock()

	if !ok || compiled == nil {
		return fmt.Errorf("script %s not found or not pre-compiled", scriptPath)
	}

	// Create a fresh VM clone for execution
	vm := compiled.Clone()
	if err := vm.Run(); err != nil {
		return fmt.Errorf("failed to execute script %s: %w", scriptPath, err)
	}

	return nil
}

// ExecuteTileScript executes a tile script with tile coordinates (tileX, tileY) and tile index (tileIdx) passed as the "tile" object.
func ExecuteTileScript(scriptPath string, tileX, tileY, tileIdx int) error {
	scriptMu.RLock()
	cleanKey := filepath.ToSlash(scriptPath)
	compiled, ok := compiledScripts[cleanKey]
	if !ok {
		cleanKey = strings.TrimPrefix(cleanKey, "data/")
		compiled, ok = compiledScripts[cleanKey]
		if !ok {
			cleanKey = strings.TrimPrefix(cleanKey, "scripts/")
			compiled, ok = compiledScripts[cleanKey]
		}
	}
	scriptMu.RUnlock()

	if !ok || compiled == nil {
		return fmt.Errorf("tile script %s not found or not pre-compiled", scriptPath)
	}

	vm := compiled.Clone()
	tileMap := map[string]interface{}{
		"x":    tileX,
		"y":    tileY,
		"tile": tileIdx,
	}

	if err := vm.Set("tile", tileMap); err != nil {
		return fmt.Errorf("failed to set tile context for script %s: %w", scriptPath, err)
	}

	if err := vm.Run(); err != nil {
		return fmt.Errorf("failed to execute tile script %s: %w", scriptPath, err)
	}

	return nil
}
