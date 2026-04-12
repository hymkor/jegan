package jegan

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mattn/go-colorable"
)

func test(t *testing.T, source, operation, expect string) {
	t.Helper()
	resPath := filepath.Join(t.TempDir(), "result.json")

	app := &Application{Name: "TEST"}
	err := app.Load(strings.NewReader(source),"TEST SCRIPT")
	if err != nil {
		t.Fatal(err.Error())
	}
	ttyIn := &autoPilot{
		script: fmt.Sprintf("%s|w|%s|q", operation, resPath),
	}
	ttyOut := io.Discard
	if testing.Verbose() {
		ttyOut = colorable.NewColorableStderr()
	}
	err = app.EventLoop(ttyIn, ttyOut)
	if err != nil {
		t.Fatal(err.Error())
	}
	resultBin, err := os.ReadFile(resPath)
	if err != nil {
		t.Fatal(err.Error())
	}
	result := string(resultBin)
	if expect != result {
		t.Fatalf("\nExpect %#v,\n   but %#v", expect, result)
	}
}

func testLoadSaveOnly(t *testing.T,source string){
	t.Helper()
	test(t, source,`k`, source)
}

func TestLoadSaveOnly(t *testing.T){
	testLoadSaveOnly(t, "[]")
	testLoadSaveOnly(t, "[ ]")
	testLoadSaveOnly(t, "[\t]")
	testLoadSaveOnly(t, "[\n]")
	testLoadSaveOnly(t, "[\r\n]")
	testLoadSaveOnly(t, "[\r\n]\r\n")
	testLoadSaveOnly(t, "[\r\n\t1\r\n]\r\n")
	testLoadSaveOnly(t, "\t[\r\n\t\t1\r\n\t]\r\n\t")
	testLoadSaveOnly(t ," [ [ [ [ [ ] ] ] ] ] ")

	testLoadSaveOnly(t, "{}")
	testLoadSaveOnly(t, "{ }")
	testLoadSaveOnly(t, "{\t}")
	testLoadSaveOnly(t, "{\n}")
	testLoadSaveOnly(t, "{\r\n}")
	testLoadSaveOnly(t, "{\r\n}\r\n")
	testLoadSaveOnly(t, "{\r\n\t\"one\":1\r\n}\r\n")
	testLoadSaveOnly(t, "{\r\n\t\"one\" : 1\r\n}\r\n")
}

func TestInsert(t *testing.T) {
	test(t, `[]`, `o|0`, `[0]`)
	test(t, `[]`, `o|"x"`, `["x"]`)
	test(t, "[\n\t\"y\"\n]", `o|"x"`, "[\n\t\"x\",\n\t\"y\"\n]")
	test(t, "[\n]", `o|"x"`, "[\n\"x\"\n]")
}
