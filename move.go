package jegan

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/nyaosorg/go-readline-ny"

	"github.com/hymkor/jegan/internal/types"
)

var (
	rxIndex  = regexp.MustCompile(`^\s*(?:\[\s*)?([0-9]+)\s*(?:\]\s*)?$`)
	rxSymbol = regexp.MustCompile(`^\s*([_a-zA-Z_][_a-zA-Z0-9]*)\s*`)
)

func (app *Application) keyFuncMoveTo(session *Session) error {
	location, err := app.readLineOpt(session, "JSON Path:", "", func(*readline.Editor) {})
	if err != nil {
		return err
	}

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

	var jsonpath *types.JsonPath
	for _, s := range tokens {
		if s == "" {
			continue
		}
		if m := rxIndex.FindStringSubmatch(s); m != nil {
			n, err := strconv.Atoi(m[1])
			if err != nil {
				panic(err.Error())
			}
			jsonpath = &types.JsonPath{
				Parent: jsonpath,
				Index:  n,
			}
			continue
		}
		if m := rxSymbol.FindStringSubmatch(s); m != nil {
			jsonpath = &types.JsonPath{
				Parent: jsonpath,
				Text:   m[1],
				Index:  -1,
			}
			continue
		}
		var str string
		err := json.Unmarshal([]byte(s), &str)
		if err != nil {
			return fmt.Errorf("%s: %w", s, err)
		}
		jsonpath = &types.JsonPath{
			Parent: jsonpath,
			Text:   str,
			Index:  -1,
		}
	}
	p := app.list.Front()
	n := 0
	for {
		if p == nil {
			var b strings.Builder
			jsonpath.Dump(&b)
			b.WriteString(": not found")
			app.message = b.String()
			return nil
		}
		if jsonpath.Equals(p.Value.Path()) {
			app.setCursor(p)
			app.csrline = n
			session.Window = p
			session.WinPos = n
			return nil
		}
		p = p.Next()
		n++
	}
}
