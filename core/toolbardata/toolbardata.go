package toolbardata

import (
	"strconv"
	"strings"
	"unicode"
)

type ToolbarData struct {
	Str   string
	Parts []*Part
	hv    *HomeVars
}

func NewToolbarData(str string, hv *HomeVars) *ToolbarData {
	td := &ToolbarData{Str: str, hv: hv}
	td.Parts = parseParts(str)

	// fill parts toolbardata pointer to have access to root str
	for _, p := range td.Parts {
		p.ToolbarData = td
	}

	return td
}

func (td *ToolbarData) GetPartAtIndex(i int) (*Part, bool) {
	for _, p := range td.Parts {
		if i >= p.S && i <= p.E { // <= E includes separator and eos
			return p, true
		}
	}
	return nil, false
}

func (td *ToolbarData) part0Arg0Token() (*Token, bool) {
	if len(td.Parts) == 0 {
		return nil, false
	}
	if len(td.Parts[0].Args) == 0 {
		return nil, false
	}
	return td.Parts[0].Args[0], true
}

func (td *ToolbarData) DecodePart0Arg0() string {
	tok, ok := td.part0Arg0Token()
	if !ok {
		return ""
	}
	return td.hv.Decode(tok.Str)
}

func (td *ToolbarData) StrWithPart0Arg0Encoded() string {
	tok, ok := td.part0Arg0Token()
	if !ok {
		return td.Str
	}
	s2 := td.hv.Decode(tok.Str)
	s3 := td.hv.Encode(s2)
	return td.Str[:tok.S] + s3 + td.Str[tok.E:]
}
func (td *ToolbarData) StrWithPart0Arg0Decoded() string {
	tok, ok := td.part0Arg0Token()
	if !ok {
		return td.Str
	}
	s2 := td.hv.Decode(tok.Str)
	return td.Str[:tok.S] + s2 + td.Str[tok.E:]
}

func parseParts(str string) []*Part {
	var parts []*Part
	toks := parseTokens(str, 0, len(str), "|\n")
	for _, t := range toks {
		ctoks := parseTokens(str, t.S, t.E, " ")
		ctoks = filterEmptyTokens(ctoks)
		p := &Part{Token: *t, Args: ctoks}
		parts = append(parts, p)
	}
	return parts
}
func parseTokens(str string, a, b int, seps string) []*Token {
	lastQuote := rune(0)
	escape := false
	split := func(ru rune) bool {
		switch {
		case ru == '\\':
			escape = true
			return false
		case escape:
			escape = false
			return false
		case ru == lastQuote:
			lastQuote = 0
			return false
		case lastQuote != 0: // inside a quote
			return false
		case ru == '`', unicode.In(ru, unicode.Quotation_Mark):
			lastQuote = ru
			return false
		default:
			for _, ru2 := range seps {
				if ru2 == ru {
					return true
				}
			}
			return false
		}
	}
	return fieldsFunc(str, a, b, split)
}
func fieldsFunc(str string, a, b int, split func(rune) bool) []*Token {
	var u []*Token
	s := a
	for i, ru := range str[a:b] {
		if split(ru) {
			t := NewToken(str, s, a+i)
			s = a + i + len(string(ru)) // not including separator in tok
			u = append(u, t)
		}
	}
	if s < b {
		t := NewToken(str, s, b)
		u = append(u, t)
	}
	return u
}
func filterEmptyTokens(toks []*Token) []*Token {
	var u []*Token
	for _, t := range toks {
		if t.isEmpty() {
			continue
		}
		u = append(u, t)
	}
	return u
}

type Part struct {
	Token
	Args []*Token

	ToolbarData *ToolbarData // provides access to root string
}

type Token struct {
	Str  string // token string
	S, E int    // start/end str indexes of the root string
}

func NewToken(str string, s, e int) *Token {
	tok := &Token{Str: str[s:e], S: s, E: e}
	return tok
}

func (tok *Token) isEmpty() bool {
	return strings.TrimSpace(tok.Str) == ""
}

func (tok *Token) Unquote() (rune, int, int, string, bool) {
	str, err := strconv.Unquote(tok.Str)
	if err != nil {
		return 0, 0, 0, "", false
	}
	v, _, _, err := strconv.UnquoteChar(tok.Str, 0)
	if err != nil {
		return 0, 0, 0, "", false
	}
	l := len(string(v))
	s := tok.S + l
	e := s + len(str)
	return v, s, e, str, true
}

func (tok *Token) UnquotedStr() string {
	_, _, _, s, ok := tok.Unquote()
	if ok {
		return s
	}
	return tok.Str
}
