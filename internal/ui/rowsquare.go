package ui

import (
	"image"
	"image/draw"

	"github.com/friedelschoen/glake/internal/drawer"
	"github.com/friedelschoen/glake/internal/ui/driver"
	"github.com/friedelschoen/glake/internal/ui/widget"
	"github.com/veandco/go-sdl2/sdl"
)

type RowSquare struct {
	widget.ENode
	Size  image.Point
	row   *Row
	state RowState
}

func NewRowSquare(row *Row) *RowSquare {
	sq := &RowSquare{row: row, Size: image.Point{5, 5}}
	sq.Cursor = sdl.SYSTEM_CURSOR_NO
	return sq
}
func (sq *RowSquare) Measure(hint image.Point) image.Point {
	return drawer.MinPoint(sq.Size, hint)
}

func (sq *RowSquare) Paint() {
	img := sq.row.ui.Image()

	// background
	bg := sq.TreeThemePaletteColor("rowsquare")
	if sq.state.hasAny(RowStateEdited) {
		bg = sq.TreeThemePaletteColor("rs_edited")
	}
	if sq.state.hasAny(RowStateNotExist) {
		bg = sq.TreeThemePaletteColor("rs_not_exist")
	}
	if sq.state.hasAny(RowStateExecuting) {
		bg = sq.TreeThemePaletteColor("rs_executing")
	}
	draw.Draw(img, sq.Bounds, image.NewUniform(bg), image.Point{}, draw.Src)

	// mini-squares
	if sq.state.hasAny(RowStateActive) {
		r := sq.miniSq(0)
		c := sq.TreeThemePaletteColor("rs_active")
		draw.Draw(img, r, image.NewUniform(c), image.Point{}, draw.Src)
	}
	if sq.state.hasAny(RowStateFsDiffer) {
		r := sq.miniSq(1)
		c := sq.TreeThemePaletteColor("rs_disk_changes")
		draw.Draw(img, r, image.NewUniform(c), image.Point{}, draw.Src)
	}
	if sq.state.hasAny(RowStateDuplicate) {
		r := sq.miniSq(2)
		c := sq.TreeThemePaletteColor("rs_duplicate")
		draw.Draw(img, r, image.NewUniform(c), image.Point{}, draw.Src)
	}
	if sq.state.hasAny(RowStateDuplicateHighlight) {
		r := sq.miniSq(2)
		c := sq.TreeThemePaletteColor("rs_duplicate_highlight")
		draw.Draw(img, r, image.NewUniform(c), image.Point{}, draw.Src)
	}
	if sq.state.hasAny(RowStateAnnotations) {
		r := sq.miniSq(3)
		c := sq.TreeThemePaletteColor("rs_annotations")
		draw.Draw(img, r, image.NewUniform(c), image.Point{}, draw.Src)
	}
	if sq.state.hasAny(RowStateAnnotationsEdited) {
		r := sq.miniSq(3)
		c := sq.TreeThemePaletteColor("rs_annotations_edited")
		draw.Draw(img, r, image.NewUniform(c), image.Point{}, draw.Src)
	}
}
func (sq *RowSquare) miniSq(i int) image.Rectangle {
	// mini squares
	// [0,1]
	// [2,3]

	// mini square rectangle
	maxXI, maxYI := 1, 1
	sideX, sideY := sq.Size.X/(maxXI+1), sq.Size.Y/(maxYI+1)
	x, y := i%2, i/2
	r := image.Rect(0, 0, sideX, sideY)
	r = r.Add(image.Point{x * sideX, y * sideY})

	// avoid rounding errors
	if x == maxXI {
		r.Max.X = sq.Size.X
	}
	if y == maxYI {
		r.Max.Y = sq.Size.Y
	}

	// mini square position
	r2 := r.Add(sq.Bounds.Min).Intersect(sq.Bounds)

	return r2
}

func (sq *RowSquare) SetState(s RowState, v bool) {
	u := sq.state.hasAny(s)
	if u != v {
		if v {
			sq.state.add(s)
		} else {
			sq.state.remove(s)
		}
		sq.MarkNeedsPaint()
	}
}
func (sq *RowSquare) HasState(s RowState) bool {
	return sq.state.hasAny(s)
}
func (sq *RowSquare) OnInputEvent(ev driver.Event, p image.Point) bool {
	switch t := ev.(type) {

	// use drag events from row separator (allows dragging using rowsquare)
	case *driver.MouseDragMove:
		sq.Cursor = sdl.SYSTEM_CURSOR_CROSSHAIR
		sq.row.sep.OnInputEvent(ev, p)

	// when exiting a drag, make sure the cursor is back to the default
	case *driver.MouseDragEnd:
		sq.Cursor = sdl.SYSTEM_CURSOR_NO

	// handle scroll events from row separator (allows mouse-wheel ops from rowsquare)
	case *driver.MouseDown:
		sq.row.sep.OnInputEvent(ev, p)

	case *driver.MouseClick:
		switch t.Key.Mouse {
		case driver.ButtonLeft, driver.ButtonMiddle, driver.ButtonRight:
			sq.row.Close()
		}
	}
	return true
}

type RowState uint16

func (m *RowState) add(u RowState)         { *m |= u }
func (m *RowState) remove(u RowState)      { *m &^= u }
func (m *RowState) hasAny(u RowState) bool { return (*m)&u > 0 }

const (
	RowStateActive RowState = 1 << iota
	RowStateExecuting
	RowStateEdited
	RowStateFsDiffer
	RowStateNotExist
	RowStateDuplicate
	RowStateDuplicateHighlight
	RowStateAnnotations
	RowStateAnnotationsEdited
)
