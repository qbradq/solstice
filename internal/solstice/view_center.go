package solstice

import (
	"image"
)

var (
	viewCenterStack []image.Point
)

// GetViewCenterStack returns a copy of the current view centering stack.
func GetViewCenterStack() []image.Point {
	result := make([]image.Point, len(viewCenterStack))
	copy(result, viewCenterStack)
	return result
}

// ResetViewCenterStack resets the view centering stack to a single location (party's location or fallback).
func ResetViewCenterStack() {
	if p := GetParty(); p != nil {
		viewCenterStack = []image.Point{image.Pt(p.X, p.Y)}
	} else {
		viewCenterStack = []image.Point{image.Pt(16, 16)}
	}
}

// SetPartyViewCenter updates the party's location on the view centering stack at the bottom (index 0).
// It modifies the bottom value directly without pushing or popping.
func SetPartyViewCenter(pt image.Point) {
	if len(viewCenterStack) == 0 {
		viewCenterStack = []image.Point{pt}
	} else {
		viewCenterStack[0] = pt
	}
}

// PushViewCenter pushes a new location to the top of the view centering stack.
func PushViewCenter(pt image.Point) {
	if len(viewCenterStack) == 0 {
		if p := GetParty(); p != nil {
			viewCenterStack = append(viewCenterStack, image.Pt(p.X, p.Y))
		} else {
			viewCenterStack = append(viewCenterStack, image.Pt(16, 16))
		}
	}
	viewCenterStack = append(viewCenterStack, pt)
}

// PopViewCenter pops and returns the top location from the view centering stack.
// The party's location at the bottom of the stack (index 0) is never removed.
func PopViewCenter() image.Point {
	if len(viewCenterStack) == 0 {
		if p := GetParty(); p != nil {
			viewCenterStack = []image.Point{image.Pt(p.X, p.Y)}
			return viewCenterStack[0]
		}
		viewCenterStack = []image.Point{image.Pt(16, 16)}
		return viewCenterStack[0]
	}
	if len(viewCenterStack) == 1 {
		return viewCenterStack[0]
	}
	topIdx := len(viewCenterStack) - 1
	top := viewCenterStack[topIdx]
	viewCenterStack = viewCenterStack[:topIdx]
	return top
}

// UpdateTopViewCenter updates the location at the top of the view centering stack.
func UpdateTopViewCenter(pt image.Point) {
	if len(viewCenterStack) == 0 {
		viewCenterStack = []image.Point{pt}
	} else {
		viewCenterStack[len(viewCenterStack)-1] = pt
	}
}

// GetViewCenter returns the location at the top of the view centering stack.
func GetViewCenter() image.Point {
	if len(viewCenterStack) == 0 {
		if p := GetParty(); p != nil {
			viewCenterStack = []image.Point{image.Pt(p.X, p.Y)}
			return viewCenterStack[0]
		}
		return image.Pt(16, 16)
	}
	return viewCenterStack[len(viewCenterStack)-1]
}
