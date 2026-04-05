package main

import (
	"strings"
	"testing"
)

func TestReadObject(t *testing.T) {
	source := `{  "1" : 2  , "3" : 4  , "5":{   "6":  7  } }`
	pairs, err := readObject(strings.NewReader(source[1:]))
	if err != nil {
		t.Fatal(err.Error())
	}
	result := pairs.GoString()
	if source != result {
		t.Fatalf("\nexpect: `%s`\n   got: `%s`", source, result)
	}
}

func TestArray(t *testing.T) {
	source := `[ 1  ,    []    ,{} ,  4 ]`
	array, err := readArray(strings.NewReader(source[1:]))
	if err != nil {
		t.Fatal(err.Error())
	}
	result := array.GoString()
	if source != result {
		t.Fatalf("\nexpect: `%s`\n   got: `%s`", source, result)
	}
}

func TestReadItem(t *testing.T) {
	source := `[ { "1": 2, "3": [ 1,2,  3], "5":6 },7,  8, 9  ]`
	token, err := readItem(strings.NewReader(source))
	if err != nil {
		t.Fatal(err.Error())
	}
	result := token.GoString()
	if source != result {
		t.Fatalf("\nexpect: `%s`\n   got: `%s`", source, result)
	}
}
