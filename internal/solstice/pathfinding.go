package solstice

import (
	"container/heap"
	"image"
	"math"
	"sort"
	"strings"
)

// isTileTraversable checks if an actor can traverse through tile (x, y) on map m.
func isTileTraversable(m *Map, x, y int, isPartyMember bool, startPt image.Point) bool {
	if m == nil || x < 0 || x >= m.Width || y < 0 || y >= m.Height {
		return false
	}
	if !m.IsWalkable(x, y) {
		return false
	}
	if image.Pt(x, y) == startPt {
		return true
	}

	act := m.GetActorAt(x, y)
	if act == nil {
		return true
	}

	if isPartyMember {
		party := GetParty()
		if party != nil {
			for _, mem := range party.Members {
				if mem.ID == act.ID {
					// Other party member: can pass through
					return true
				}
			}
		}
		// Non-party member actor blocks traversal
		return false
	}

	// Non-party actors cannot pass through ANY actor
	return false
}

// isTileValidDestination checks if (x, y) can be the stopping point of movement.
func isTileValidDestination(m *Map, x, y int, isPartyMember bool, startPt image.Point) bool {
	if !isTileTraversable(m, x, y, isPartyMember, startPt) {
		return false
	}
	if image.Pt(x, y) == startPt {
		return false
	}
	// Destination cannot be occupied by ANY actor
	if m.GetActorAt(x, y) != nil {
		return false
	}
	if isPartyMember {
		if party := GetParty(); party != nil {
			for _, mem := range party.Members {
				if mem.X == x && mem.Y == y {
					return false
				}
			}
		}
	}
	return true
}

// FindReachableTiles computes all tiles reachable within maxSteps moves using 4-directional pathfinding.
func FindReachableTiles(m *Map, startX, startY, maxSteps int, isPartyMember bool) map[image.Point]bool {
	reachable := make(map[image.Point]bool)
	if m == nil || maxSteps <= 0 {
		return reachable
	}

	startPt := image.Pt(startX, startY)
	distances := make(map[image.Point]int)
	distances[startPt] = 0

	queue := []image.Point{startPt}

	dirs := []image.Point{
		{X: 0, Y: -1}, // North
		{X: 0, Y: 1},  // South
		{X: -1, Y: 0}, // West
		{X: 1, Y: 0},  // East
	}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		currDist := distances[curr]

		if currDist >= maxSteps {
			continue
		}

		for _, d := range dirs {
			next := image.Pt(curr.X+d.X, curr.Y+d.Y)
			if !isTileTraversable(m, next.X, next.Y, isPartyMember, startPt) {
				continue
			}

			newDist := currDist + 1
			if prevDist, exists := distances[next]; !exists || newDist < prevDist {
				distances[next] = newDist
				queue = append(queue, next)

				if isTileValidDestination(m, next.X, next.Y, isPartyMember, startPt) {
					reachable[next] = true
				}
			}
		}
	}

	return reachable
}

type pathNode struct {
	pt    image.Point
	g     int
	f     int
	index int
}

type priorityQueue []*pathNode

func (pq priorityQueue) Len() int           { return len(pq) }
func (pq priorityQueue) Less(i, j int) bool { return pq[i].f < pq[j].f }
func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}
func (pq *priorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*pathNode)
	item.index = n
	*pq = append(*pq, item)
}
func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*pq = old[0 : n-1]
	return item
}

// FindPath finds the shortest path of 4-directional cardinal moves from (startX, startY) to (targetX, targetY) using A*.
func FindPath(m *Map, startX, startY, targetX, targetY int, isPartyMember bool) []string {
	if m == nil {
		return nil
	}
	startPt := image.Pt(startX, startY)
	targetPt := image.Pt(targetX, targetY)
	if startPt == targetPt {
		return nil
	}

	if !isTileValidDestination(m, targetX, targetY, isPartyMember, startPt) {
		return nil
	}

	manhattan := func(a, b image.Point) int {
		return int(math.Abs(float64(a.X-b.X)) + math.Abs(float64(a.Y-b.Y)))
	}

	dirs := []struct {
		d   image.Point
		dir string
	}{
		{d: image.Pt(0, -1), dir: "north"},
		{d: image.Pt(0, 1), dir: "south"},
		{d: image.Pt(-1, 0), dir: "west"},
		{d: image.Pt(1, 0), dir: "east"},
	}

	cameFrom := make(map[image.Point]image.Point)
	cameFromDir := make(map[image.Point]string)
	gScore := make(map[image.Point]int)
	gScore[startPt] = 0

	pq := &priorityQueue{}
	heap.Init(pq)
	heap.Push(pq, &pathNode{
		pt: startPt,
		g:  0,
		f:  manhattan(startPt, targetPt),
	})

	visited := make(map[image.Point]bool)

	for pq.Len() > 0 {
		curr := heap.Pop(pq).(*pathNode)
		if curr.pt == targetPt {
			// Reconstruct path
			var path []string
			step := targetPt
			for step != startPt {
				path = append([]string{cameFromDir[step]}, path...)
				step = cameFrom[step]
			}
			return path
		}

		if visited[curr.pt] {
			continue
		}
		visited[curr.pt] = true

		for _, step := range dirs {
			neighbor := image.Pt(curr.pt.X+step.d.X, curr.pt.Y+step.d.Y)
			if neighbor != targetPt {
				if !isTileTraversable(m, neighbor.X, neighbor.Y, isPartyMember, startPt) {
					continue
				}
			} else {
				if !isTileValidDestination(m, neighbor.X, neighbor.Y, isPartyMember, startPt) {
					continue
				}
			}

			tentativeG := gScore[curr.pt] + 1
			if prevG, exists := gScore[neighbor]; !exists || tentativeG < prevG {
				cameFrom[neighbor] = curr.pt
				cameFromDir[neighbor] = step.dir
				gScore[neighbor] = tentativeG
				heap.Push(pq, &pathNode{
					pt: neighbor,
					g:  tentativeG,
					f:  tentativeG + manhattan(neighbor, targetPt),
				})
			}
		}
	}

	return nil
}

// PathToString converts a slice of direction strings ("north", "east", "south", "west")
// into a compact string of "n", "e", "s", "w" characters.
func PathToString(path []string) string {
	var sb strings.Builder
	for _, dir := range path {
		switch strings.ToLower(dir) {
		case "north", "n":
			sb.WriteByte('n')
		case "east", "e":
			sb.WriteByte('e')
		case "south", "s":
			sb.WriteByte('s')
		case "west", "w":
			sb.WriteByte('w')
		}
	}
	return sb.String()
}

// FindPathToClosestString finds an A* path from (startX, startY) to the point closest to
// (targetX, targetY) that can be pathed to, limited by maxMoves, and returns it as a string of "n/e/s/w".
func FindPathToClosestString(m *Map, startX, startY, targetX, targetY, maxMoves int, isPartyMember bool) string {
	if m == nil || maxMoves <= 0 {
		return ""
	}
	startPt := image.Pt(startX, startY)
	targetPt := image.Pt(targetX, targetY)
	if startPt == targetPt {
		return ""
	}

	manhattan := func(a, b image.Point) int {
		return int(math.Abs(float64(a.X-b.X)) + math.Abs(float64(a.Y-b.Y)))
	}

	// 1. If targetPt itself is a valid destination, try finding a direct path to it.
	if isTileValidDestination(m, targetX, targetY, isPartyMember, startPt) {
		if path := FindPath(m, startX, startY, targetX, targetY, isPartyMember); len(path) > 0 {
			path = truncateAndValidatePath(m, startPt, path, maxMoves, isPartyMember)
			return PathToString(path)
		}
	}

	// 2. If target is not a valid destination (e.g. occupied by target actor), try its unoccupied cardinal neighbors
	cardinalNeighbors := []image.Point{
		{X: targetX, Y: targetY - 1},
		{X: targetX + 1, Y: targetY},
		{X: targetX, Y: targetY + 1},
		{X: targetX - 1, Y: targetY},
	}

	// Sort neighbors by distance to startPt
	sort.Slice(cardinalNeighbors, func(i, j int) bool {
		return manhattan(cardinalNeighbors[i], startPt) < manhattan(cardinalNeighbors[j], startPt)
	})

	var bestNeighborPath []string
	for _, nPt := range cardinalNeighbors {
		if nPt == startPt {
			// Already adjacent to target
			return ""
		}
		if isTileValidDestination(m, nPt.X, nPt.Y, isPartyMember, startPt) {
			if path := FindPath(m, startX, startY, nPt.X, nPt.Y, isPartyMember); len(path) > 0 {
				if len(bestNeighborPath) == 0 || len(path) < len(bestNeighborPath) {
					bestNeighborPath = path
				}
			}
		}
	}

	if len(bestNeighborPath) > 0 {
		bestNeighborPath = truncateAndValidatePath(m, startPt, bestNeighborPath, maxMoves, isPartyMember)
		return PathToString(bestNeighborPath)
	}

	// 3. Fallback: Search all reachable valid destination tiles and pick the one closest to targetPt
	reachable := FindReachableTiles(m, startX, startY, maxMoves, isPartyMember)
	bestDist := -1
	var bestPt image.Point
	found := false

	for pt := range reachable {
		dist := manhattan(pt, targetPt)
		if !found || dist < bestDist || (dist == bestDist && manhattan(pt, startPt) < manhattan(bestPt, startPt)) {
			bestDist = dist
			bestPt = pt
			found = true
		}
	}

	if found && bestPt != startPt {
		if path := FindPath(m, startX, startY, bestPt.X, bestPt.Y, isPartyMember); len(path) > 0 {
			path = truncateAndValidatePath(m, startPt, path, maxMoves, isPartyMember)
			return PathToString(path)
		}
	}

	return ""
}

// truncateAndValidatePath truncates the path to maxMoves and ensures the landing point is a valid destination.
func truncateAndValidatePath(m *Map, startPt image.Point, path []string, maxMoves int, isPartyMember bool) []string {
	if len(path) > maxMoves {
		path = path[:maxMoves]
	}

	for len(path) > 0 {
		endPt := startPt
		for _, dir := range path {
			switch strings.ToLower(dir) {
			case "north", "n":
				endPt.Y--
			case "east", "e":
				endPt.X++
			case "south", "s":
				endPt.Y++
			case "west", "w":
				endPt.X--
			}
		}
		if isTileValidDestination(m, endPt.X, endPt.Y, isPartyMember, startPt) {
			return path
		}
		path = path[:len(path)-1]
	}
	return nil
}
