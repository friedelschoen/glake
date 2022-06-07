package lsproto

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"unicode/utf16"

	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/osutil"
	"github.com/jmigpin/editor/util/parseutil"
)

//----------

var logger0 = log.New(os.Stdout, "", log.Lshortfile)

func logTestVerbose() bool {
	f := flag.Lookup("test.v")
	return f != nil && f.Value.String() == "true"
}

func logPrintf(f string, args ...interface{}) {
	if !logTestVerbose() {
		return
	}
	logger0.Output(2, fmt.Sprintf(f, args...))
}

func logJson(prefix string, v interface{}) {
	if !logTestVerbose() {
		return
	}
	b, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		panic(err)
	}
	logger0.Output(2, fmt.Sprintf("%v%v", prefix, string(b)))
}

//----------

func encodeJson(a interface{}) ([]byte, error) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	err := enc.Encode(a)
	if err != nil {
		return nil, err
	}
	b := buf.Bytes()
	return b, nil
}

func decodeJson(r io.Reader, a interface{}) error {
	dec := json.NewDecoder(r)
	return dec.Decode(a)
}
func decodeJsonRaw(raw json.RawMessage, a interface{}) error {
	return json.Unmarshal(raw, a)
}

//----------

func Utf16Column(rd iorw.ReaderAt, lineStartOffset, utf8Col int) (int, error) {
	b, err := rd.ReadFastAt(lineStartOffset, utf8Col)
	if err != nil {
		return 0, err
	}
	return len(utf16.Encode([]rune(string(b)))), nil
}

// Input and result is zero based.
func Utf8Column(rd iorw.ReaderAt, lineStartOffset, utf16Col int) (int, error) {
	// ensure good limits
	n := utf16Col * 2
	if lineStartOffset+n > rd.Max() {
		n = rd.Max() - lineStartOffset
	}

	b, err := rd.ReadFastAt(lineStartOffset, n)
	if err != nil {
		return 0, err
	}

	enc := utf16.Encode([]rune(string(b)))
	if len(enc) < utf16Col {
		return 0, fmt.Errorf("encoded string smaller then utf16col")
	}
	nthChar := len(enc[:utf16Col])

	return nthChar, nil
}

//----------

func OffsetToPosition(rd iorw.ReaderAt, offset int) (Position, error) {
	l, c, err := parseutil.IndexLineColumn(rd, offset)
	if err != nil {
		return Position{}, err
	}
	// zero based
	l, c = l-1, c-1

	// character offset in utf16
	c2, err := Utf16Column(rd, offset-c, c)
	if err != nil {
		return Position{}, err
	}

	return Position{Line: l, Character: c2}, nil
}

func RangeToOffsetLen(rd iorw.ReaderAt, rang *Range) (int, int, error) {
	// one-based lines (range is zero based)
	l1 := rang.Start.Line + 1
	l2 := rang.End.Line + 1

	// line start offset
	// TODO: improve getting lso2
	lso1, err := parseutil.LineColumnIndex(rd, l1, 1)
	if err != nil {
		return 0, 0, err
	}
	lso2, err := parseutil.LineColumnIndex(rd, l2, 1)
	if err != nil {
		return 0, 0, err
	}

	// translate utf16 columns to utf8 (input and results are zero based)
	u16c1, err := Utf8Column(rd, lso1, rang.Start.Character)
	if err != nil {
		return 0, 0, err
	}
	u16c2, err := Utf8Column(rd, lso2, rang.End.Character)
	if err != nil {
		return 0, 0, err
	}

	// start/end (range)
	start := lso1 + u16c1
	end := lso2 + u16c2

	offset := start
	length := end - start

	return offset, length, nil
}

//----------

func JsonGetPath(v interface{}, path string) (interface{}, error) {
	args := strings.Split(path, ".")
	return jsonGetPath2(v, args)
}

// TODO: incomplete
func jsonGetPath2(v interface{}, args []string) (interface{}, error) {
	// handle last arg
	if len(args) == 0 {
		switch t := v.(type) {
		case bool, int, float32, float64:
			return t, nil
		}
		return nil, fmt.Errorf("unhandled last type: %T", v)
	}
	// handle args: len(args)>0
	arg, args2 := args[0], args[1:]
	switch t := v.(type) {
	case map[string]interface{}:
		if v, ok := t[arg]; ok {
			return jsonGetPath2(v, args2)
		}
		return nil, fmt.Errorf("not found: %v", arg)
	}
	return nil, fmt.Errorf("unhandled type: %T (arg=%v)", v, arg)
}

//----------

func urlToAbsFilename(url string) (string, error) {
	return parseutil.UrlToAbsFilename(url)
}

func absFilenameToUrl(filename string) (string, error) {
	if runtime.GOOS == "windows" {
		// gopls requires casing to match the OS names in windows (error: case mismatch in path ...)
		if u, err := osutil.FsCaseFilename(filename); err == nil {
			filename = u
		}
	}
	return parseutil.AbsFilenameToUrl(filename)
}
