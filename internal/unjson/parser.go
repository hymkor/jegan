package unjson

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/hymkor/jegan/internal/dbg"
	"github.com/hymkor/jegan/internal/source"
)

type Item struct {
	SpaceValue []byte
	Value      any
}

func (t Item) GoString() string {
	return fmt.Sprintf("%s%#v", string(t.SpaceValue), t.Value)
}

type KeyValuePair struct {
	Key               string
	Value             *Item
	SpaceKey          []byte
	SpaceColon        []byte
	SpaceCommaOrClose []byte
}

func (k KeyValuePair) GoString() string {
	return fmt.Sprintf("%s%q%s:%#v%s",
		string(k.SpaceKey),
		k.Key,
		string(k.SpaceColon),
		k.Value,
		string(k.SpaceCommaOrClose))
}

type Object struct {
	Pairs []KeyValuePair
	Blank []byte // when len(pairs) == 0
}

func (o *Object) GoString() string {
	var b strings.Builder
	b.WriteByte('{')
	if len(o.Pairs) == 0 {
		b.Write(o.Blank)
		b.WriteByte('}')
		return b.String()
	}
	for i, p := range o.Pairs {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%#v", p)
	}
	b.WriteByte('}')
	return b.String()
}

func readObject(br io.RuneScanner) (*Object, error) {
	dbg.Println("Enter readObject")
	defer dbg.Println("Leave readObject")

	firstPrefix, first, err := source.Read1st(br) // check '{'
	// first, _, err := br.ReadRune()
	if err != nil {
		return nil, err
	}
	if first == '}' {
		return &Object{
			Blank: firstPrefix,
		}, nil
	}
	br.UnreadRune()

	var pairs []KeyValuePair
	for {
		preKey, err := source.ExpectRune(br, '"')
		if err != nil {
			return nil, err
		}
		if len(firstPrefix) > 0 {
			preKey = append(preKey, firstPrefix...)
			firstPrefix = nil
		}
		key, err := source.ReadString(br)
		if err != nil {
			return nil, err
		}
		dbg.Println("key:", key)
		preVal, err := source.ExpectRune(br, ':')
		if err != nil {
			return nil, err
		}
		val, err := readItem(br)
		if err != nil {
			return nil, err
		}
		last, ch, err := source.Read1st(br)
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, KeyValuePair{
			Key:               key.String(),
			Value:             val,
			SpaceKey:          preKey,
			SpaceColon:        preVal,
			SpaceCommaOrClose: last,
		})
		if ch == '}' {
			return &Object{Pairs: pairs}, nil
		}
		if ch != ',' {
			return nil, &source.UnexpectedTokenError2{
				Expect1: '}',
				Expect2: ',',
				Got:     ch}
		}
	}
}

type ArrayElement struct {
	*Item
	PreComma []byte
}

func (a ArrayElement) GoString() string {
	return fmt.Sprintf("%#v%s", a.Item, string(a.PreComma))
}

type Array struct {
	Element []ArrayElement
	Blank   []byte
}

func (a Array) GoString() string {
	var b strings.Builder
	b.WriteByte('[')
	if len(a.Element) <= 0 {
		b.Write(a.Blank)
		b.WriteByte(']')
		return b.String()
	}
	for i, e := range a.Element {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%#v", e)
	}
	b.WriteByte(']')
	return b.String()
}

func readArray(br io.RuneScanner) (*Array, error) {
	dbg.Println("Enter readArray")
	defer dbg.Println("Leave readArray")

	firstPrefix, first, err := source.Read1st(br) // check '['
	if err != nil {
		return nil, err
	}
	var array1 []ArrayElement
	if first == ']' {
		return &Array{Blank: firstPrefix}, nil
	}
	br.UnreadRune()
	for {
		token, err := readItem(br)
		if err != nil {
			return nil, err
		}
		if len(firstPrefix) > 0 {
			token.SpaceValue = append(firstPrefix, token.SpaceValue...)
			firstPrefix = nil
		}
		prefix, ch, err := source.Read1st(br)
		if err != nil {
			return nil, err
		}
		array1 = append(array1, ArrayElement{
			Item:     token,
			PreComma: prefix,
		})
		dbg.Println("array:", len(array1))
		if ch == ']' {
			return &Array{Element: array1}, nil
		}
		if ch != ',' {
			return nil, &source.UnexpectedTokenError2{
				Expect1: ']',
				Expect2: ',',
				Got:     ch,
			}
		}
	}
}

func readItem(br io.RuneScanner) (*Item, error) {
	prefix, ch, err := source.Read1st(br)
	if err != nil {
		if len(prefix) <= 0 {
			return nil, err
		}
		return &Item{Value: source.NewRawBytes(prefix)}, err
	}
	if ch == '"' {
		s, err := source.ReadString(br)
		dbg.Println("readString:", s)
		return &Item{SpaceValue: prefix, Value: s}, err
	} else if strings.ContainsRune("0123456789-+.", ch) {
		n, err := source.ReadNumber(br, ch)
		return &Item{SpaceValue: prefix, Value: n}, err
	} else if ch == 'n' {
		v := source.NewLiteral(nil, []byte("null"))
		return &Item{SpaceValue: prefix, Value: v}, source.ExpectToken(br, ch, "null")
	} else if ch == 'f' {
		v := source.NewLiteral(false, []byte("false"))
		return &Item{SpaceValue: prefix, Value: v}, source.ExpectToken(br, ch, "false")
	} else if ch == 't' {
		v := source.NewLiteral(true, []byte("true"))
		return &Item{SpaceValue: prefix, Value: v}, source.ExpectToken(br, ch, "true")
	} else if ch == '{' {
		o, err := readObject(br)
		return &Item{SpaceValue: prefix, Value: o}, err
	} else if ch == '[' {
		a, err := readArray(br)
		return &Item{SpaceValue: prefix, Value: a}, err
	}
	var b bytes.Buffer
	b.Write(prefix)
	b.WriteRune(ch)
	for {
		ch, siz, err := br.ReadRune()
		if err != nil && !errors.Is(err, io.EOF) {
			bin := b.Bytes()
			rb := source.NewRawBytes(bin)
			dbg.Println("RawBytes(1):", bin)
			return &Item{Value: rb}, err
		}
		if siz > 0 {
			b.WriteRune(ch)
		}
		if err != nil {
			bin := b.Bytes()
			dbg.Println("RawBytes(2):", bin)
			rb := source.NewRawBytes(bin)
			return &Item{Value: rb}, err
		}
		if ch == '=' {
			bin := b.Bytes()
			dbg.Println("JavaScript equation ?:", bin)
			rb := source.NewRawBytes(bin)
			return &Item{Value: rb}, nil
		}
	}
}

func Unmarshal(r io.RuneScanner) (*Item, error) {
	sc := source.NewScanner(r)
	v, err := readItem(sc)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return v, err
		}
		return nil, sc.WrapError(err)
	}
	return v, nil
}
