package solstice

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"

	"solstice/data"

	"github.com/justinian/dice"
)

// EnemyPack represents an enemy encounter pack configuration from data/json/packs.json.
type EnemyPack struct {
	Level      int      `json:"level"`
	BonusXP    int      `json:"bonus_xp"`
	NumEnemies string   `json:"num_enemies"`
	Enemies    []string `json:"enemies"`
}

var enemyPacks = make(map[string]EnemyPack)

// PreloadEnemyPacks pre-loads enemy pack definitions from data/json/packs.json at program start.
func PreloadEnemyPacks() (map[string]EnemyPack, error) {
	return LoadEnemyPacks("json/packs.json")
}

// LoadEnemyPacks loads enemy pack definitions from a JSON file in data.FS.
func LoadEnemyPacks(path string) (map[string]EnemyPack, error) {
	cleanPath := strings.TrimPrefix(path, "data/")
	if !strings.HasPrefix(cleanPath, "json/") {
		cleanPath = "json/" + cleanPath
	}

	b, err := data.FS.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read enemy packs file %s: %w", path, err)
	}

	var packs map[string]EnemyPack
	if err := json.Unmarshal(b, &packs); err != nil {
		return nil, fmt.Errorf("failed to parse enemy packs JSON %s: %w", path, err)
	}

	for k, v := range packs {
		enemyPacks[k] = v
	}

	return enemyPacks, nil
}

// GetEnemyPack returns the EnemyPack definition for the given name from loaded packs.
func GetEnemyPack(name string) (EnemyPack, bool) {
	p, ok := enemyPacks[name]
	return p, ok
}

// GetAllEnemyPacks returns a copy of all loaded enemy packs.
func GetAllEnemyPacks() map[string]EnemyPack {
	res := make(map[string]EnemyPack, len(enemyPacks))
	for k, v := range enemyPacks {
		res[k] = v
	}
	return res
}

// RollNumEnemies evaluates the pack's NumEnemies dice formula and clamps the result between 1 and 8.
func (p EnemyPack) RollNumEnemies() (int, error) {
	formula := strings.TrimSpace(p.NumEnemies)
	if formula == "" {
		return 1, nil
	}
	res, _, err := dice.Roll(formula)
	if err != nil {
		return 1, fmt.Errorf("failed to roll dice %q: %w", formula, err)
	}
	n := res.Int()
	if n < 1 {
		n = 1
	}
	if n > 8 {
		n = 8
	}
	return n, nil
}

// ChooseEnemy returns a randomly selected enemy actor template name from the pack's Enemies list.
func (p EnemyPack) ChooseEnemy() string {
	if len(p.Enemies) == 0 {
		return ""
	}
	return p.Enemies[rand.Intn(len(p.Enemies))]
}
