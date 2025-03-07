package widget

import (
	"image"

	"github.com/friedelschoen/glake/internal/ui/driver"
	"github.com/veandco/go-sdl2/sdl"
)

type ApplyEvent struct {
	drag AEDragState
	cctx CursorContext
}

func NewApplyEvent(cctx CursorContext) *ApplyEvent {
	ae := &ApplyEvent{cctx: cctx}
	return ae
}

func (ae *ApplyEvent) Apply(node Node, ev driver.Event, p image.Point) {
	if !ae.drag.dragging {
		ae.mouseEnterLeave(node, p)
	}

	switch evt := ev.(type) {
	case nil: // allow running the rest of the function without an event
	case *driver.MouseDown:
		ae.depthFirstEv(node, evt, p)
	case *driver.MouseMove:
		ae.depthFirstEv(node, evt, p)
	case *driver.MouseUp:
		ae.depthFirstEv(node, evt, p)
	case *driver.MouseDragStart:
		ae.dragStart(node, evt, p)
		if ae.drag.dragging {
			ae.mouseEnterLeave(node, ae.drag.startEv.Point)
		}
	case *driver.MouseDragMove:
		ae.dragMove(evt, p)
	case *driver.MouseDragEnd:
		ae.dragEnd(evt, p)
		if !ae.drag.dragging {
			ae.mouseEnterLeave(node, p)
		}
	case *driver.KeyDown:
		ae.depthFirstEv(node, evt, p)
	default:
		// ex: driver.KeyUp
		ae.depthFirstEv(node, evt, p)
	}

	ae.setCursor(node, p)
}

func (ae *ApplyEvent) setCursor(node Node, p image.Point) {
	var c sdl.SystemCursor
	if ae.drag.dragging {
		c = ae.drag.node.Embed().Cursor
	} else {
		c = ae.treeCursor(node, p)
	}
	ae.cctx.SetCursor(c)
}

func (ae *ApplyEvent) treeCursor(node Node, p image.Point) sdl.SystemCursor {
	ne := node.Embed()
	if !p.In(ne.Bounds) {
		return 0
	}
	var c sdl.SystemCursor
	ne.IterateWrappersReverse(func(child Node) bool {
		c = ae.treeCursor(child, p)
		return c == 0 // continue while no cursor was set
	})
	if c == 0 {
		c = ne.Cursor
	}
	return c
}

func (ae *ApplyEvent) mouseEnterLeave(node Node, p image.Point) {
	ae.mouseLeave(node, p) // run leave first
	ae.mouseEnter(node, p)
}

func (ae *ApplyEvent) mouseEnter(node Node, p image.Point) bool {
	ne := node.Embed()

	if !p.In(ne.Bounds) {
		return false
	}

	// execute on childs
	h := bool(false)
	// later childs are drawn over previous ones, run loop backwards
	ne.IterateWrappersReverse(func(c Node) bool {
		h = ae.mouseEnter(c, p)
		return !h // continue while not handled
	})

	// execute on node
	if !h {
		if !ne.HasAnyMarks(MarkPointerInside) {
			ne.AddMarks(MarkPointerInside)
			ev2 := &driver.MouseEnter{}
			h = ae.runEv(node, ev2, p)
		}
	}

	if ne.HasAnyMarks(MarkInBoundsHandlesEvent) {
		h = true
	}

	return h
}

func (ae *ApplyEvent) mouseLeave(node Node, p image.Point) bool {
	ne := node.Embed()

	// execute on childs
	h := bool(false)
	// later childs are drawn over previous ones, run loop backwards
	ne.IterateWrappersReverse(func(c Node) bool {
		h = ae.mouseLeave(c, p)
		return !h // continue while not handled
	})

	// execute on node
	if !h {
		if ne.HasAnyMarks(MarkPointerInside) && !p.In(ne.Bounds) {
			ne.RemoveMarks(MarkPointerInside)
			ev2 := &driver.MouseLeave{}
			h = ae.runEv(node, ev2, p)
		}
	}

	return h
}

func (ae *ApplyEvent) dragStart(node Node, ev *driver.MouseDragStart, _ image.Point) {
	if ae.drag.dragging {
		return
	}
	p := ev.Point // use the starting point, not the current point
	ae.findDragNode2(node, ev, p)
}

// Depth first, reverse order.
func (ae *ApplyEvent) findDragNode2(node Node, ev *driver.MouseDragStart, p image.Point) bool {
	if !p.In(node.Embed().Bounds) {
		return false
	}

	// execute on childs
	found := false
	node.Embed().IterateWrappersReverse(func(c Node) bool {
		found = ae.findDragNode2(c, ev, p)
		return !found // continue while not found
	})

	if !found {
		// deepest node
		canDrag := !node.Embed().HasAnyMarks(MarkNotDraggable)
		if canDrag {
			ae.drag.dragging = true
			ae.drag.startEv = ev
			ae.drag.node = node
			ae.runEv(ae.drag.node, ev, p)
			return true
		}
	}

	return found
}

func (ae *ApplyEvent) dragMove(ev *driver.MouseDragMove, p image.Point) {
	if !ae.drag.dragging {
		return
	}
	ae.runEv(ae.drag.node, ev, p)
}

func (ae *ApplyEvent) dragEnd(ev *driver.MouseDragEnd, p image.Point) {
	if !ae.drag.dragging {
		return
	}
	if ev.Key.Mouse != ae.drag.startEv.Key.Mouse {
		return
	}
	ae.runEv(ae.drag.node, ev, p)
	ae.drag = AEDragState{}
}

func (ae *ApplyEvent) depthFirstEv(node Node, ev driver.Event, p image.Point) bool {
	if !p.In(node.Embed().Bounds) {
		return false
	}

	// execute on childs
	h := bool(false)
	// later childs are drawn over previous ones, run loop backwards
	node.Embed().IterateWrappersReverse(func(c Node) bool {
		h = ae.depthFirstEv(c, ev, p)
		return !h // continue while not handled
	})

	// execute on node
	if !h {
		h = ae.runEv(node, ev, p)
	}

	if node.Embed().HasAnyMarks(MarkInBoundsHandlesEvent) {
		h = true
	}

	return h
}

func (ae *ApplyEvent) runEv(node Node, ev driver.Event, p image.Point) bool {
	return node.OnInputEvent(ev, p)
}

type AEDragState struct {
	dragging bool
	startEv  *driver.MouseDragStart
	node     Node
}
