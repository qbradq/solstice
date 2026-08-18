package solstice

// CutSceneCmdType represents the type of a cut scene command.
type CutSceneCmdType int

const (
	CmdDelay CutSceneCmdType = iota
	CmdMove
	CmdSetTile
	CmdRemoveActor
)

// CutSceneCommand represents an action in the cut scene queue.
type CutSceneCommand struct {
	Type    CutSceneCmdType
	ActorID string
	Dir     string
	X       int
	Y       int
	TileID  int
	Frames  int
}

var (
	csQueue                []CutSceneCommand
	csLastAnimFrame        int = -1
	csDelayFramesRemaining int
)

// EnqueueCutSceneCommand adds a command to the cut scene command queue.
func EnqueueCutSceneCommand(cmd CutSceneCommand) {
	csQueue = append(csQueue, cmd)
}

// ClearCutScene clears the cut scene command queue and resets delay state.
func ClearCutScene() {
	csQueue = nil
	csDelayFramesRemaining = 0
	csLastAnimFrame = -1
}

// IsCutSceneActive returns true if there are queued cut scene commands or an active delay.
func IsCutSceneActive() bool {
	return len(csQueue) > 0 || csDelayFramesRemaining > 0
}

// UpdateCutScene advances the cut scene queue in lock-step with animation frames.
// Returns true if a cut scene is currently active.
func UpdateCutScene(g *Game) bool {
	currentFrame := GetAnimFrame()
	if csLastAnimFrame == -1 {
		csLastAnimFrame = currentFrame
	}

	frameDelta := currentFrame - csLastAnimFrame
	if frameDelta > 0 {
		csLastAnimFrame = currentFrame
		if csDelayFramesRemaining > 0 {
			csDelayFramesRemaining -= frameDelta
			if csDelayFramesRemaining < 0 {
				csDelayFramesRemaining = 0
			}
		}
	}

	// Drain queue only if not waiting on a delay
	if csDelayFramesRemaining == 0 {
		for len(csQueue) > 0 {
			cmd := csQueue[0]
			csQueue = csQueue[1:]

			switch cmd.Type {
			case CmdMove:
				if m := GetMap(); m != nil {
					actor := m.GetActorByID(cmd.ActorID)
					if actor != nil {
						dx, dy := 0, 0
						switch cmd.Dir {
						case "n", "N", "north", "North", "NORTH", "up", "Up":
							dy = -1
						case "s", "S", "south", "South", "SOUTH", "down", "Down":
							dy = 1
						case "w", "W", "west", "West", "WEST", "left", "Left":
							dx = -1
						case "e", "E", "east", "East", "EAST", "right", "Right":
							dx = 1
						}
						actor.X += dx
						actor.Y += dy

						if party := GetParty(); party != nil {
							for i := range party.Members {
								if party.Members[i].ID == cmd.ActorID {
									party.Members[i].X = actor.X
									party.Members[i].Y = actor.Y
								}
							}
						}
					}
				}

			case CmdSetTile:
				if m := GetMap(); m != nil {
					m.SetTile(cmd.X, cmd.Y, cmd.TileID)
				}

			case CmdRemoveActor:
				if m := GetMap(); m != nil {
					m.RemoveActorByID(cmd.ActorID)
				}

			case CmdDelay:
				csDelayFramesRemaining = cmd.Frames
				return true
			}
		}
	}

	return len(csQueue) > 0 || csDelayFramesRemaining > 0
}
