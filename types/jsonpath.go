package types

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

type JsonPath struct {
	Parent *JsonPath
	Text   string
	Index  int
}

func (j *JsonPath) ChildIndex(i int) *JsonPath {
	return &JsonPath{
		Parent: j,
		Index:  i,
	}
}

func (j *JsonPath) ChildKey(key string) *JsonPath {
	return &JsonPath{
		Parent: j,
		Text:   key,
		Index:  -1,
	}
}

func (j *JsonPath) Equals(other *JsonPath) bool {
	if j == nil {
		return other == nil
	}
	if other == nil {
		return false
	}
	if j.Text != other.Text {
		return false
	}
	if j.Text == "" && j.Index != other.Index {
		return false
	}
	return j.Parent.Equals(other.Parent)
}

var rxSymbol = regexp.MustCompile("^[_A-Za-z][_A-Za-z0-9]*$")

func (j *JsonPath) Dump(w io.Writer) {
	if j == nil {
		return
	}
	if j.Parent != nil {
		j.Parent.Dump(w)
	}
	if j.Text != "" {
		if rxSymbol.MatchString(j.Text) {
			fmt.Fprintf(w, ".%s", j.Text)
		} else {
			fmt.Fprintf(w, ".%q", j.Text)
		}
	} else {
		if j.Parent == nil {
			w.Write([]byte{'.'})
		}
		fmt.Fprintf(w, "[%d]", j.Index)
	}
}

func (j *JsonPath) String() string {
	var b strings.Builder
	j.Dump(&b)
	return b.String()
}

var (
	rxIndex   = regexp.MustCompile(`^\s*(?:\[\s*)?([0-9]+)\s*(?:\]\s*)?$`)
	rxSymbol2 = regexp.MustCompile(`^\s*([_a-zA-Z_][_a-zA-Z0-9]*)\s*$`)
)

func ParseJson(location string) (*JsonPath, error) {
	tokens := []string{}
	start := 0
	quote := false
	for i, c := range location {
		if c == '"' {
			quote = !quote
		}
		if !quote && (c == '.' || c == '[') {
			tokens = append(tokens, location[start:i])
			start = i + 1
		}
	}
	tokens = append(tokens, location[start:])

	var jsonpath *JsonPath
	for _, s := range tokens {
		if s == "" {
			continue
		}
		if m := rxIndex.FindStringSubmatch(s); m != nil {
			n, err := strconv.Atoi(m[1])
			if err != nil {
				panic(err.Error())
			}
			jsonpath = jsonpath.ChildIndex(n)
			continue
		}
		if m := rxSymbol2.FindStringSubmatch(s); m != nil {
			jsonpath = jsonpath.ChildKey(m[1])
			continue
		}
		var str string
		err := json.Unmarshal([]byte(s), &str)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", s, err)
		}
		jsonpath = jsonpath.ChildKey(str)
	}
	return jsonpath, nil
}

func (j *JsonPath) Search(p *Element) (*Element, int) {
	count := 0
	for p != nil {
		if j.Equals(p.Value.Path()) {
			return p, count
		}
		p = p.Next()
		count++
	}
	return nil, -1
}
