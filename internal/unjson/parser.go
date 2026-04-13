package unjson

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/nyaosorg/go-windows-dbg"
)

func debug(s ...any) {
	if false {
		dbg.Println(s...)
	}
}

func read1st(br io.RuneScanner) ([]byte, rune, error) {
	var prefix bytes.Buffer
	for {
		ch, _, err := br.ReadRune()
		if err != nil {
			return prefix.Bytes(), ch, err
		}
		if !unicode.IsSpace(ch) {
			return prefix.Bytes(), ch, nil
		}
		prefix.WriteRune(ch)
	}
}

type UnexpectedTokenError struct {
	expect rune
	got    rune
}

func (e UnexpectedTokenError) Error() string {
	return fmt.Sprintf("expected '%c', but got '%c'", e.expect, e.got)
}

type UnexpectedTokenError2 struct {
	expect1 rune
	expect2 rune
	got     rune
}

func (e UnexpectedTokenError2) Error() string {
	return fmt.Sprintf("expected '%c' or '%c', but got '%c'",
		e.expect1,
		e.expect2,
		e.got)
}

type InvalidLiteralError struct {
	got string
}

func (e InvalidLiteralError) Error() string {
	return fmt.Sprintf("invalid literal: %q", e.got)
}

func expectRune(br io.RuneScanner, expect rune) ([]byte, error) {
	prefix, ch, err := read1st(br)
	if err != nil {
		if err == io.EOF {
			return prefix, io.ErrUnexpectedEOF
		}
		return prefix, err
	}
	if ch != expect {
		return prefix, &UnexpectedTokenError{expect: expect, got: ch}
	}
	return prefix, nil
}

func readString(br io.RuneScanner) (*Literal, error) {
	var buffer bytes.Buffer
	buffer.WriteByte('"')
	backslash := false
	for {
		ch, _, err := br.ReadRune()
		if err != nil {
			if err == io.EOF {
				return nil, io.ErrUnexpectedEOF
			}
			return nil, err
		}
		if !backslash && ch == '"' {
			buffer.WriteByte('"')
			var str string
			bin := buffer.Bytes()
			err := json.Unmarshal(bin, &str)
			return &Literal{
				value: str,
				json:  bin,
			}, err
		}
		if !backslash && ch == '\\' {
			backslash = true
		} else {
			backslash = false
		}
		buffer.WriteRune(ch)
	}
}

func readToken(br io.RuneScanner, first rune) ([]byte, error) {
	var buffer bytes.Buffer
	buffer.WriteRune(first)
	for {
		ch, _, err := br.ReadRune()
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return nil, err
			}
		}
		if unicode.IsSpace(ch) {
			br.UnreadRune()
			return buffer.Bytes(), err
		}
		if ch == ',' || ch == ']' || ch == '}' {
			br.UnreadRune()
			return buffer.Bytes(), err
		}
		buffer.WriteRune(ch)
		if errors.Is(err, io.EOF) {
			return buffer.Bytes(), err
		}
	}
}

func expectToken(br io.RuneScanner, first rune, expect string) error {
	s, err := readToken(br, first)
	result := string(s)
	if result != expect {
		return &InvalidLiteralError{got: result}
	}
	return err
}

func readNumber(br io.RuneScanner, first rune) (*Literal, error) {
	token, err1 := readToken(br, first)
	if err1 != nil {
		return nil, err1
	}
	var number float64
	err2 := json.Unmarshal(token, &number)
	if err2 != nil {
		return nil, err2
	}
	return &Literal{
		value: number,
		json:  token,
	}, nil
}

type Entry struct {
	SpaceValue []byte
	Value      any
}

func (t Entry) GoString() string {
	return fmt.Sprintf("%s%#v", string(t.SpaceValue), t.Value)
}

type KeyValuePair struct {
	Key      string
	Value    *Entry
	SpaceKey []byte
	PreCol   []byte
	Last     []byte
}

func (k KeyValuePair) GoString() string {
	return fmt.Sprintf("%s%q%s:%#v%s",
		string(k.SpaceKey),
		k.Key,
		string(k.PreCol),
		k.Value,
		string(k.Last))
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
	debug("Enter readObject")
	defer debug("Leave readObject")

	firstPrefix, first, err := read1st(br) // check '{'
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
		preKey, err := expectRune(br, '"')
		if err != nil {
			return nil, err
		}
		if len(firstPrefix) > 0 {
			preKey = append(preKey, firstPrefix...)
			firstPrefix = nil
		}
		key, err := readString(br)
		if err != nil {
			return nil, err
		}
		debug("key:", key)
		preVal, err := expectRune(br, ':')
		if err != nil {
			return nil, err
		}
		val, err := readEntry(br)
		if err != nil {
			return nil, err
		}
		last, ch, err := read1st(br)
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, KeyValuePair{
			Key:      key.String(),
			Value:    val,
			SpaceKey: preKey,
			PreCol:   preVal,
			Last:     last,
		})
		if ch == '}' {
			return &Object{Pairs: pairs}, nil
		}
		if ch != ',' {
			return nil, &UnexpectedTokenError2{
				expect1: '}',
				expect2: ',',
				got:     ch}
		}
	}
}

type ArrayElement struct {
	*Entry
	PreComma []byte
}

func (a ArrayElement) GoString() string {
	return fmt.Sprintf("%#v%s", a.Entry, string(a.PreComma))
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
	debug("Enter readArray")
	defer debug("Leave readEntry")

	firstPrefix, first, err := read1st(br) // check '['
	if err != nil {
		return nil, err
	}
	var array1 []ArrayElement
	if first == ']' {
		return &Array{Blank: firstPrefix}, nil
	}
	br.UnreadRune()
	for {
		token, err := readEntry(br)
		if err != nil {
			return nil, err
		}
		if len(firstPrefix) > 0 {
			token.SpaceValue = append(firstPrefix, token.SpaceValue...)
			firstPrefix = nil
		}
		prefix, ch, err := read1st(br)
		if err != nil {
			return nil, err
		}
		array1 = append(array1, ArrayElement{
			Entry:    token,
			PreComma: prefix,
		})
		debug("array:", len(array1))
		if ch == ']' {
			return &Array{Element: array1}, nil
		}
		if ch != ',' {
			return nil, &UnexpectedTokenError2{
				expect1: ']',
				expect2: ',',
				got:     ch,
			}
		}
	}
}

func readEntry(br io.RuneScanner) (*Entry, error) {
	prefix, ch, err := read1st(br)
	if err != nil {
		if len(prefix) <= 0 {
			return nil, err
		}
		return &Entry{Value: &RawBytes{json: prefix}}, err
	}
	if ch == '"' {
		s, err := readString(br)
		debug("readString:", s)
		return &Entry{SpaceValue: prefix, Value: s}, err
	} else if strings.ContainsRune("0123456789-+.", ch) {
		n, err := readNumber(br, ch)
		return &Entry{SpaceValue: prefix, Value: n}, err
	} else if ch == 'n' {
		v := &Literal{value: nil, json: []byte("null")}
		return &Entry{SpaceValue: prefix, Value: v}, expectToken(br, ch, "null")
	} else if ch == 'f' {
		v := &Literal{value: false, json: []byte("false")}
		return &Entry{SpaceValue: prefix, Value: v}, expectToken(br, ch, "false")
	} else if ch == 't' {
		v := &Literal{value: true, json: []byte("true")}
		return &Entry{SpaceValue: prefix, Value: v}, expectToken(br, ch, "true")
	} else if ch == '{' {
		o, err := readObject(br)
		return &Entry{SpaceValue: prefix, Value: o}, err
	} else if ch == '[' {
		a, err := readArray(br)
		return &Entry{SpaceValue: prefix, Value: a}, err
	}
	var b bytes.Buffer
	b.Write(prefix)
	b.WriteRune(ch)
	for {
		ch, siz, err := br.ReadRune()
		if err != nil && !errors.Is(err, io.EOF) {
			bin := b.Bytes()
			rb := &RawBytes{json: bin}
			debug("RawBytes(1):", bin)
			return &Entry{Value: rb}, err
		}
		if siz > 0 {
			b.WriteRune(ch)
		}
		if err != nil {
			bin := b.Bytes()
			debug("RawBytes(2):", bin)
			rb := &RawBytes{json: bin}
			return &Entry{Value: rb}, err
		}
		if ch == '=' {
			bin := b.Bytes()
			debug("JavaScript equation ?:", bin)
			rb := &RawBytes{json: bin}
			return &Entry{Value: rb}, nil
		}
	}
}

func Unmarshal(r io.RuneScanner) (*Entry, error) {
	sc := newScanner(r)
	v, err := readEntry(sc)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return v, err
		}
		return nil, sc.WrapError(err)
	}
	return v, nil
}
