package jegan

import (
	"fmt"
	"io"
	"regexp"
)

type JsonPath struct {
	parent *JsonPath
	text   string
	index  int
}

func (j *JsonPath) ChildIndex(i int) *JsonPath {
	return &JsonPath{
		parent: j,
		index:  i,
	}
}

func (j *JsonPath) ChildKey(key string) *JsonPath {
	return &JsonPath{
		parent: j,
		text:   key,
		index:  -1,
	}
}

var rxSymbol = regexp.MustCompile("^[_A-Za-z][_A-Za-z0-9]*$")

func (j *JsonPath) Dump(w io.Writer) {
	if j == nil {
		return
	}
	if j.parent != nil {
		j.parent.Dump(w)
	}
	if j.text != "" {
		if rxSymbol.MatchString(j.text) {
			fmt.Fprintf(w, ".%s", j.text)
		} else {
			fmt.Fprintf(w, ".%q", j.text)
		}
	} else {
		if j.parent == nil {
			w.Write([]byte{'.'})
		}
		fmt.Fprintf(w, "[%d]", j.index)
	}
}
