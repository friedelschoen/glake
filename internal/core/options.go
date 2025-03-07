package core

import (
	"fmt"
	"strings"

	"github.com/friedelschoen/glake/internal/lsproto"
	"github.com/friedelschoen/glake/internal/parser"
)

type Options struct {
	Font        string  `json:"font"`
	FontSize    float64 `json:"font-size"`
	FontHinting string  `json:"font-hinting"`
	DPI         float64 `json:"dpi"`

	TabWidth           int    `json:"tabwidth"`
	WrapLineRune       string `json:"wrapline-rune"`
	CarriageReturnRune string `json:"carriage-return-rune"`

	ColorTheme     string `json:"colortheme"`
	ScrollBarWidth int    `json:"scrollbar-width"`
	ScrollBarLeft  bool   `json:"scrollbar-left"`
	Shadows        bool   `json:"shadows"`

	SessionName string
	Filenames   []string

	UseMultiKey bool

	Plugins string

	LSProtos     RegistrationsOpt
	PreSaveHooks PreSaveHooksOpt

	ZipSessionsFile bool
}

// implements flag.Value interface
type RegistrationsOpt struct {
	regs []*lsproto.Registration
}

func (ro *RegistrationsOpt) Set(s string) error {
	reg, err := lsproto.NewRegistration(s)
	if err != nil {
		return err
	}
	ro.regs = append(ro.regs, reg)
	return nil
}

func (ro *RegistrationsOpt) MustSet(s string) {
	if err := ro.Set(s); err != nil {
		panic(err)
	}
}

func (ro *RegistrationsOpt) String() string {
	u := []string{}
	for _, reg := range ro.regs {
		u = append(u, reg.String())
	}
	return fmt.Sprintf("%v", strings.Join(u, "\n"))
}

// implements flag.Value interface
type PreSaveHooksOpt struct {
	regs []*PreSaveHook
}

func (o *PreSaveHooksOpt) Set(s string) error {
	reg, err := newPreSaveHook(s)
	if err != nil {
		return err
	}
	o.regs = append(o.regs, reg)
	return nil
}

func (o *PreSaveHooksOpt) MustSet(s string) {
	if err := o.Set(s); err != nil {
		panic(err)
	}
}

func (o *PreSaveHooksOpt) String() string {
	u := []string{}
	for _, reg := range o.regs {
		u = append(u, reg.String())
	}
	return fmt.Sprintf("%v", strings.Join(u, "\n"))
}

type PreSaveHook struct {
	Language string
	Exts     []string
	Cmd      string
}

func newPreSaveHook(s string) (*PreSaveHook, error) {
	fields, err := parser.ParseFields(s)
	if err != nil {
		return nil, err
	}
	minFields := 3
	if len(fields) != minFields {
		return nil, fmt.Errorf("expecting %v fields: %v", minFields, len(fields))
	}

	r := &PreSaveHook{}
	r.Language = fields[0]
	if r.Language == "" {
		return nil, fmt.Errorf("empty language")
	}
	r.Exts = strings.Split(fields[1], " ")
	r.Cmd = fields[2]

	return r, nil
}

func (h *PreSaveHook) String() string {
	u := []string{h.Language}

	exts := strings.Join(h.Exts, " ")
	if len(h.Exts) >= 2 {
		exts = fmt.Sprintf("%q", exts)
	}
	u = append(u, exts)

	cmd := h.Cmd
	cmd2 := parser.AddEscapes(cmd, '\\', " ,")
	if cmd != cmd2 {
		cmd = fmt.Sprintf("%q", cmd)
	}
	u = append(u, cmd)

	return strings.Join(u, ",")
}
