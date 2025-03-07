package widget

import (
	"image"

	"github.com/friedelschoen/glake/internal/ui/driver"
)

// A transparent widget added to a top layer (usually multilayer) to facilitate dragging.
// Calculations are made on top of the reference node (usually a thin separator that otherwise would not be easy to put the pointer over for dragging).
type SeparatorHandle struct {
	ENode
	Top, Bottom, Left, Right int
	DragPad                  image.Point
	ref                      Node // reference node for calc bounds
}

func NewSeparatorHandle(ref Node) *SeparatorHandle {
	sh := &SeparatorHandle{ref: ref}
	sh.AddMarks(MarkNotPaintable)
	return sh
}

func (sh *SeparatorHandle) Measure(hint image.Point) image.Point {
	panic("calling measure on thin separator handle")
}

func (sh *SeparatorHandle) Layout() {
	// calc own bounds based on reference node
	b := sh.ref.Embed().Bounds
	b.Min.X -= sh.Left
	b.Max.X += sh.Right
	b.Min.Y -= sh.Top
	b.Max.Y += sh.Bottom

	// limit with parents bounds (might be wider/thiner)
	pb := sh.Parent.Bounds
	b = b.Intersect(pb)

	// set own bounds
	sh.Bounds = b
}

func DetectMovePad(p, press, ref image.Point) image.Point {
	u := ref.Sub(p)
	v := 3 + 2 // matches value in DetectMove()+2
	if u.X > v || u.X < -v {
		u.X = 0
	}
	if u.Y > v || u.Y < -v {
		u.Y = 0
	}
	return u
}

func (sh *SeparatorHandle) OnInputEvent(ev0 driver.Event, p image.Point) bool {
	switch ev := ev0.(type) {
	case *driver.MouseDragStart:
		u := sh.ref.Embed().Bounds.Min
		sh.DragPad = DetectMovePad(ev.Point2, ev.Point, u)
	}
	return sh.ref.Embed().Wrapper.OnInputEvent(ev0, p)
}
