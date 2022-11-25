package toolbarparser

import (
	"fmt"
	"log"
	"unicode"

	"github.com/jmigpin/editor/util/osutil"
	"github.com/jmigpin/editor/util/parseutil"
)

// simple parser based on parseutil.pstate
func parse2_basedOnParseutilPState(str string) *Data {
	p := newParser2(str)
	if err := p.start(); err != nil {
		log.Print(err)
	}
	return p.data
}

//----------

type parser2 struct {
	data *Data
	ps   *parseutil.PState
}

func newParser2(str string) *parser2 {
	p := &parser2{}
	p.data = &Data{Str: str}
	p.ps = parseutil.NewPState([]byte(str))
	return p
}
func (p *parser2) start() error {
	parts, err := p.parts()
	if err != nil {
		return err
	}
	p.data.Parts = parts
	return nil
}
func (p *parser2) parts() ([]*Part, error) {
	parts := []*Part{}
	for {
		part, err := p.part()
		if err != nil {
			return nil, err
		}
		parts = append(parts, part)

		// split parts on these runes
		if p.ps.MatchRunesOr([]rune(("|\n"))) == nil {
			continue
		}
		if p.ps.MatchEof() == nil {
			break
		}
	}
	return parts, nil
}
func (p *parser2) part() (*Part, error) {
	part := &Part{}
	part.Data = p.data

	pos0 := p.ps.Pos

	// optional space at start
	_ = p.ps.ConsumeSpacesExcludingNL()

	for {
		arg, err := p.arg()
		if err != nil {
			break // end of part
		}
		part.Args = append(part.Args, arg)

		// need space between args
		if !p.ps.ConsumeSpacesExcludingNL() {
			break
		}
	}

	part.Pos = pos0
	part.End = p.ps.Pos
	return part, nil
}
func (p *parser2) arg() (*Arg, error) {
	arg := &Arg{}
	arg.Data = p.data

	pos0 := p.ps.Pos
	ps2 := p.ps.Copy()
	for {
		if ps2.MatchEof() == nil {
			break
		}
		if ps2.EscapeAny(osutil.EscapeRune) == nil {
			continue
		}
		if ps2.QuotedString() == nil {
			continue
		}

		// split args
		ps3 := ps2.Copy()
		ru, err := ps3.ReadRune()
		if err != nil {
			break
		}
		if ru == '|' || unicode.IsSpace(ru) {
			break
		}
		ps2.Set(ps3) // accept rune into arg
	}
	// empty arg. Ex: parts string with empty args: "|||".
	empty := ps2.Pos == pos0
	if empty {
		return nil, fmt.Errorf("arg")
	}
	p.ps.Set(ps2)

	arg.Pos = pos0
	arg.End = p.ps.Pos
	return arg, nil
}
