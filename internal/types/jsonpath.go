package types

import (
	"fmt"
	"io"
	"regexp"
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
