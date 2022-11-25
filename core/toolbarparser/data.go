package toolbarparser

import (
	"fmt"
	"strconv"
)

type Data struct {
	Str   string
	Parts []*Part
	//bnd   *lrparser.BuildNodeData
}

func (d *Data) PartAtIndex(i int) (*Part, bool) {
	for _, p := range d.Parts {
		if i >= p.Pos && i <= p.End { // end includes separator and eos
			return p, true
		}
	}
	return nil, false
}
func (d *Data) Part0Arg0() (*Arg, bool) {
	if len(d.Parts) > 0 && len(d.Parts[0].Args) > 0 {
		return d.Parts[0].Args[0], true
	}
	return nil, false
}

//----------

func (d *Data) String() string {
	s := ""
	for i, p := range d.Parts {
		s += fmt.Sprintf("part%v:\n", i)
		for j, arg := range p.Args {
			s += fmt.Sprintf("\targ%v: %q\n", j, arg)
		}
		for j, v := range p.Vars {
			s += fmt.Sprintf("\tvar%v: %q\n", j, v)
		}
	}
	return s
}

//----------
//----------
//----------

type Part struct {
	Node
	Args []*Arg
	Vars []*Var
}

func (p *Part) ArgsUnquoted() []string {
	args := []string{}
	for _, a := range p.Args {
		args = append(args, a.UnquotedStr())
	}
	return args
}

func (p *Part) ArgsStrs() []string {
	args := []string{}
	for _, a := range p.Args {
		args = append(args, a.Str())
	}
	return args
}

func (p *Part) FromArgString(i int) string {
	if i >= len(p.Args) {
		return ""
	}
	a := p.Args[i:]
	n1 := a[0]
	n2 := a[len(a)-1]
	return p.Node.Data.Str[n1.Pos:n2.End]
}

//----------
//----------
//----------

type Arg struct {
	Node
}

//----------
//----------
//----------

type Node struct {
	Pos  int
	End  int   // end pos
	Data *Data // data with full str
}

// TODO: remove
func (node *Node) Str() string {
	return node.Data.Str[node.Pos:node.End]
}

func (node *Node) UnquotedStr() string {
	s := node.Str()
	s2, err := strconv.Unquote(s) // TODO: has issue with single quote, use parseutil.Unquote*?
	if err != nil {
		return s
	}
	return s2
}

func (node *Node) String() string {
	return node.Data.Str[node.Pos:node.End]
}

//----------
//----------
//----------

type Var struct {
	Name, Value string
}

func (v *Var) String() string {
	return fmt.Sprintf("%v=%v", v.Name, v.Value)
}
