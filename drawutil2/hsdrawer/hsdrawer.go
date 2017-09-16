package hsdrawer

import (
	"image"
	"image/draw"

	"github.com/jmigpin/editor/drawutil2/loopers"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// Highlight and Selection drawer.
type HSDrawer struct {
	Face font.Face
	Str  string

	Colors      *Colors
	CursorIndex int // <0 to disable
	HWordIndex  int // <0 to disable
	Selection   *loopers.SelectionIndexes
	OffsetY     fixed.Int26_6

	height fixed.Int26_6

	pdl *loopers.PosDataLooper
	pdk *HSPosDataKeeper
}

func NewHSDrawer(face font.Face) *HSDrawer {
	d := &HSDrawer{Face: face}

	// compute with minimal data
	// allows getpoint to work without a calcrunedata be called
	//d.Measure(&image.Point{})

	return d
}

func (d *HSDrawer) Measure(max *image.Point) *fixed.Point26_6 {
	max2 := fixed.P(max.X, max.Y)

	strl := loopers.NewStringLooper(d.Face, d.Str)
	linel := loopers.NewLineLooper(strl, max2.Y)
	wlinel := loopers.NewWrapLineLooper(strl, linel, max2.X)
	d.pdk = NewHSPosDataKeeper(strl, wlinel)
	d.pdl = loopers.NewPosDataLooper(d.pdk, strl)
	ml := loopers.NewMeasureLooper(strl, &max2)

	// iterator order
	linel.SetOuterLooper(strl)
	wlinel.SetOuterLooper(linel)
	d.pdl.SetOuterLooper(wlinel)
	ml.SetOuterLooper(d.pdl)

	ml.Loop(func() bool { return true })

	d.height = ml.M.Y
	if d.Str == "" {
		d.height = 0
	}

	return ml.M
}
func (d *HSDrawer) Draw(img draw.Image, bounds *image.Rectangle) {
	//min := fixed.P(bounds.Min.X, bounds.Min.Y)
	//max := fixed.P(bounds.Max.X, bounds.Max.Y)
	//max2 := max.Sub(min)

	//strl := loopers.NewStringLooper(d.Face, d.Str)
	//linel := loopers.NewLineLooper(strl, max2.Y)
	//wlinel := loopers.NewWrapLineLooper(strl, linel, max2.X)

	// TODO: check linel and wlinel2 max2.X/Y
	strl := d.pdk.strl
	wlinel := d.pdk.wlinel

	dl := loopers.NewDrawLooper(strl, img, bounds)
	bgl := loopers.NewBgLooper(strl, dl)
	sl := loopers.NewSelectionLooper(strl, bgl, dl)
	cursorl := loopers.NewCursorLooper(strl, dl)
	hwl := loopers.NewHWordLooper(strl, bgl, dl, sl)
	scl := loopers.NewSetColorsLooper(dl, bgl)

	// options
	scl.Fg = d.Colors.Normal.Fg
	scl.Bg = nil // d.Colors.Normal.Bg // default bg filled externallly
	sl.Selection = d.Selection
	sl.Fg = d.Colors.Selection.Fg
	sl.Bg = d.Colors.Selection.Bg
	hwl.WordIndex = d.HWordIndex
	hwl.Fg = d.Colors.Highlight.Fg
	hwl.Bg = d.Colors.Highlight.Bg
	cursorl.CursorIndex = d.CursorIndex

	// iterator order
	scl.SetOuterLooper(wlinel)
	sl.SetOuterLooper(scl)
	hwl.SetOuterLooper(sl)
	bgl.SetOuterLooper(hwl)
	cursorl.SetOuterLooper(bgl)
	dl.SetOuterLooper(cursorl)

	p := &fixed.Point26_6{0, d.OffsetY}
	d.pdl.RestorePosDataCloseToPoint(p)
	d.pdk.strl.Pen.Y -= d.OffsetY

	dl.Loop(func() bool { return true })
}

func (d *HSDrawer) Height() fixed.Int26_6 {
	return d.height
}
func (d *HSDrawer) LineHeight() fixed.Int26_6 {
	// TODO: remove this check
	if d.pdl == nil {
		return 0
	}

	return d.pdk.strl.LineHeight()
}

func (d *HSDrawer) GetPoint(index int) *fixed.Point26_6 {
	// TODO: remove this check
	if d.pdl == nil {
		return &fixed.Point26_6{}
	}

	d.pdl.RestorePosDataCloseToIndex(index)
	return loopers.PosDataGetPoint(index, d.pdk.strl, d.pdk.wlinel)
}
func (d *HSDrawer) GetIndex(p *fixed.Point26_6) int {
	// TODO: remove this check
	if d.pdl == nil {
		return 0
	}

	d.pdl.RestorePosDataCloseToPoint(p)
	return loopers.PosDataGetIndex(p, d.pdk.strl, d.pdk.wlinel)
}

type HSPosDataKeeper struct {
	strl   *loopers.StringLooper
	wlinel *loopers.WrapLineLooper
}

func NewHSPosDataKeeper(strl *loopers.StringLooper, wlinel *loopers.WrapLineLooper) *HSPosDataKeeper {
	return &HSPosDataKeeper{strl: strl, wlinel: wlinel}
}
func (pdk *HSPosDataKeeper) KeepPosData() interface{} {
	return &HSPosData{
		wrapIndent: pdk.wlinel.WrapIndent,
	}
}
func (pdk *HSPosDataKeeper) RestorePosData(data interface{}) {
	u := data.(*HSPosData)
	pdk.wlinel.WrapIndent = u.wrapIndent
}

type HSPosData struct {
	wrapIndent loopers.WrapIndent
}
