package parser

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"

	"encoding/json"
)

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

func readString(br io.RuneScanner) (string, error) {
	var buffer bytes.Buffer
	buffer.WriteByte('"')
	backslash := false
	for {
		ch, _, err := br.ReadRune()
		if err != nil {
			if err == io.EOF {
				return "", io.ErrUnexpectedEOF
			}
			return "", err
		}
		if !backslash && ch == '"' {
			buffer.WriteByte('"')
			var s string
			err := json.Unmarshal(buffer.Bytes(), &s)
			return s, err
		}
		if ch == '\\' {
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

func readNumber(br io.RuneScanner, first rune) (float64, error) {
	token, err1 := readToken(br, first)
	if err1 != nil {
		return 0, err1
	}
	var number float64
	err2 := json.Unmarshal(token, &number)
	if err2 != nil {
		return number, err2
	}
	return number, err1
}

type Token struct {
	Prefix []byte
	Value  any
}

func (t Token) GoString() string {
	return fmt.Sprintf("%s%#v", string(t.Prefix), t.Value)
}

type KeyValuePair struct {
	Key    string
	Value  *Token
	PreKey []byte
	PreCol []byte
	Last   []byte
}

func (k KeyValuePair) GoString() string {
	return fmt.Sprintf("%s%q%s:%#v%s",
		string(k.PreKey),
		k.Key,
		string(k.PreCol),
		k.Value,
		string(k.Last))
}

type Object []KeyValuePair

func (o Object) GoString() string {
	var b strings.Builder
	b.WriteByte('{')
	for i, p := range o {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%#v", p)
	}
	b.WriteByte('}')
	return b.String()
}

func readObject(br io.RuneScanner) (Object, error) {
	first, _, err := br.ReadRune()
	if err != nil {
		return nil, err
	}
	if first == '}' {
		return []KeyValuePair{}, nil
	}
	br.UnreadRune()

	var pairs []KeyValuePair
	for {
		preKey, err := expectRune(br, '"')
		if err != nil {
			return nil, err
		}
		key, err := readString(br)
		if err != nil {
			return nil, err
		}
		preVal, err := expectRune(br, ':')
		if err != nil {
			return nil, err
		}
		val, err := readItem(br)
		if err != nil {
			return nil, err
		}
		last, ch, err := read1st(br)
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, KeyValuePair{
			Key:    key,
			Value:  val,
			PreKey: preKey,
			PreCol: preVal,
			Last:   last,
		})
		if ch == '}' {
			return Object(pairs), nil
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
	*Token
	PreComma []byte
}

func (a ArrayElement) GoString() string {
	return fmt.Sprintf("%#v%s", a.Token, string(a.PreComma))
}

type Array []ArrayElement

func (a Array) GoString() string {
	var b strings.Builder
	b.WriteByte('[')
	for i, e := range a {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%#v", e)
	}
	b.WriteByte(']')
	return b.String()
}

func readArray(br io.RuneScanner) (Array, error) {
	first, _, err := br.ReadRune() // check '['
	if err != nil {
		return nil, err
	}
	var array1 []ArrayElement
	if first == ']' {
		return array1, nil
	}
	br.UnreadRune()
	for {
		token, err := readItem(br)
		if err != nil {
			return nil, err
		}
		prefix, ch, err := read1st(br)
		if err != nil {
			return nil, err
		}
		array1 = append(array1, ArrayElement{
			Token:    token,
			PreComma: prefix,
		})
		if ch == ']' {
			return Array(array1), nil
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

func readItem(br io.RuneScanner) (*Token, error) {
	prefix, ch, err := read1st(br)
	if err != nil {
		return nil, err
	}
	if ch == '"' {
		s, err := readString(br)
		return &Token{Prefix: prefix, Value: s}, err
	} else if strings.ContainsRune("0123456789-+.", ch) {
		n, err := readNumber(br, ch)
		return &Token{Prefix: prefix, Value: n}, err
	} else if ch == 'n' {
		return &Token{Prefix: prefix, Value: nil}, expectToken(br, ch, "null")
	} else if ch == 'f' {
		return &Token{Prefix: prefix, Value: false}, expectToken(br, ch, "false")
	} else if ch == 't' {
		return &Token{Prefix: prefix, Value: true}, expectToken(br, ch, "true")
	} else if ch == '{' {
		o, err := readObject(br)
		return &Token{Prefix: prefix, Value: o}, err
	} else if ch == '[' {
		a, err := readArray(br)
		return &Token{Prefix: prefix, Value: a}, err
	}
	token, err := readToken(br, ch)
	if err != nil {
		return nil, err
	}
	return nil, &InvalidLiteralError{got: string(token)}
}

func Unmarshal(data []byte) (any, error) {
	sc := newScanner(bytes.NewReader(data))
	v, err := readItem(sc)
	if err != nil {
		err = sc.WrapError(err)
	}
	return v, err
}
