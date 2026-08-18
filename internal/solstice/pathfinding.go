package solstice

import (
	"container/heap"
	"image"
	"math"
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
