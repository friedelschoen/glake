package widget

import (
	"github.com/friedelschoen/glake/internal/editbuf"
	"github.com/friedelschoen/glake/internal/eventregister"
	"github.com/friedelschoen/glake/internal/historybuf"
	"github.com/friedelschoen/glake/internal/ioutil"
	"github.com/friedelschoen/glake/internal/ui/driver"
)

type TextEdit struct {
	*Text
	UIContext
	rwev    *ioutil.RWEvents
	rwu     *historybuf.RWUndo
	ctx     *editbuf.EditorBuffer   // ctx for rw editing utils (contains cursor)
	RWEvReg *eventregister.Register // the rwundo wraps the rwev, so on a write event callback, the undo data is not commited yet. It is incorrect to try to undo inside a write callback. If a rwev wraps rwundo, undoing will not trigger the outer rwev events, otherwise undoing would register as another undo event (cycle).
}

func NewTextEdit(uiCtx UIContext) *TextEdit {
	t := NewText(uiCtx)
	te := &TextEdit{Text: t, UIContext: uiCtx}

	te.rwev = ioutil.NewRWEvents(te.Text.rw)
	te.RWEvReg = &te.rwev.EvReg
	te.RWEvReg.Add(ioutil.RWEvIdWrite2, te.onWrite2)

	hist := historybuf.NewHistory(200)
	te.rwu = historybuf.NewRWUndo(te.rwev, hist)

	te.ctx = editbuf.NewEditorBuffer()
	te.ctx.RW = te.rwu
	te.ctx.C = editbuf.NewTriggerCursor(te.onCursorChange)
	te.ctx.Fns = te

	return te
}

func (te *TextEdit) CommentLineSym() any { return nil }
func (te *TextEdit) PageUp(up bool)      {}
func (te *TextEdit) ScrollUp(up bool)    {}

func (te *TextEdit) RW() ioutil.ReadWriterAt {
	// TODO: returning rw with undo/events, differs from SetRW(), workaround is to use te.Text.RW() to get underlying rw

	return te.ctx.RW
}

func (te *TextEdit) SetRW(rw ioutil.ReadWriterAt) {
	// TODO: setting basic rw (bytes), differs from RW()

	te.Text.SetRW(rw)
	te.rwev.ReadWriterAt = rw
}

func (te *TextEdit) SetRWFromMaster(m *TextEdit) {
	te.SetRW(m.Text.rw)

	// TODO: should be set at instanciation when known that it will be a duplicate
	te.rwu.History = m.rwu.History
}

// Called when the changes are done on this textedit
func (te *TextEdit) onWrite2(ev any) {
	e := ev.(*ioutil.RWEvWrite2)
	if e.Changed {
		te.contentChanged()
	}
}

// Called when changes were made on another row
func (te *TextEdit) HandleRWWrite2(ev *ioutil.RWEvWrite2) {
	te.stableRuneOffset(&ev.RWEvWrite)
	te.stableCursor(&ev.RWEvWrite)
	if ev.Changed {
		te.contentChanged()
	}
}

func (te *TextEdit) EditCtx() *editbuf.EditorBuffer {
	return te.ctx
}

func (te *TextEdit) onCursorChange() {
	te.Drawer.SetCursorOffset(te.CursorIndex())
	te.MarkNeedsPaint()
}

func (te *TextEdit) Cursor() editbuf.Cursor {
	return te.ctx.C
}

func (te *TextEdit) CursorIndex() int {
	return te.Cursor().Index()
}

func (te *TextEdit) SetCursorIndex(i int) {
	te.Cursor().SetIndex(i)
}

func (te *TextEdit) Undo() error { return te.undoRedo(false) }
func (te *TextEdit) Redo() error { return te.undoRedo(true) }
func (te *TextEdit) undoRedo(redo bool) error {
	c, ok, err := te.rwu.UndoRedo(redo, false)
	if err != nil {
		return err
	}
	if ok {
		te.ctx.C.Set(c) // restore cursor
		te.MakeCursorVisible()
	}
	return nil
}

func (te *TextEdit) ClearUndones() {
	te.rwu.History.ClearUndones()
}

func (te *TextEdit) BeginUndoGroup() {
	c := te.ctx.C.Get()
	te.rwu.History.BeginUndoGroup(c)
}

func (te *TextEdit) EndUndoGroup() {
	c := te.ctx.C.Get()
	te.rwu.History.EndUndoGroup(c)
}

func (te *TextEdit) OnInputEvent(ev driver.Event) bool {
	te.BeginUndoGroup()
	defer te.EndUndoGroup()

	handled, err := editbuf.HandleInput(te.ctx, ev)
	if err != nil {
		te.Error(err)
	}
	return handled
}

func (te *TextEdit) SetBytes(b []byte) error {
	te.BeginUndoGroup()
	defer te.EndUndoGroup()
	defer func() {
		// because after setbytes the possible selection might not be correct (ex: go fmt; variable renames with lsprotorename)
		te.ctx.C.SetSelectionOff()
	}()
	return ioutil.SetBytes(te.ctx.RW, b)
}

func (te *TextEdit) SetBytesClearPos(b []byte) error {
	te.BeginUndoGroup()
	defer te.EndUndoGroup()
	err := ioutil.SetBytes(te.ctx.RW, b)
	te.ClearPos() // keep position in undogroup (history record)
	return err
}

// Keeps position (useful for file save)
func (te *TextEdit) SetBytesClearHistory(b []byte) error {
	te.rwu.History.Clear()
	rw := te.rwu.ReadWriterAt // bypass history
	if err := ioutil.SetBytes(rw, b); err != nil {
		return err
	}
	return nil
}

func (te *TextEdit) AppendBytesClearHistory(b []byte) error {
	te.rwu.History.Clear()
	rw := te.rwu.ReadWriterAt // bypass history
	if err := rw.OverwriteAt(rw.Max(), 0, b); err != nil {
		return err
	}
	return nil
}

func (te *TextEdit) SetStr(str string) error {
	return te.SetBytes([]byte(str))
}

func (te *TextEdit) SetStrClearPos(str string) error {
	return te.SetBytesClearPos([]byte(str))
}

func (te *TextEdit) SetStrClearHistory(str string) error {
	return te.SetBytesClearHistory([]byte(str))
}

func (te *TextEdit) ClearPos() {
	te.ctx.C.SetIndexSelectionOff(0)
	te.MakeIndexVisible(0)
}

func (te *TextEdit) MakeCursorVisible() {
	if a, b, ok := te.ctx.C.SelectionIndexes(); ok {
		te.MakeRangeVisible(a, b-a)
	} else {
		te.MakeIndexVisible(te.ctx.C.Index())
	}
}

func (te *TextEdit) stableRuneOffset(ev *ioutil.RWEvWrite) {
	// keep offset based scrolling stable
	ro := StableOffsetScroll(te.RuneOffset(), ev.Index, ev.Dn, ev.In)
	te.SetRuneOffset(ro)
}

func (te *TextEdit) stableCursor(ev *ioutil.RWEvWrite) {
	c := te.Cursor()
	ci := StableOffsetScroll(c.Index(), ev.Index, ev.Dn, ev.In)
	if c.HaveSelection() {
		si := StableOffsetScroll(c.SelectionIndex(), ev.Index, ev.Dn, ev.In)
		c.SetSelection(si, ci)
	} else {
		te.SetCursorIndex(ci)
	}
}
