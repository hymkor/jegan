package main

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

type token struct {
	prefix []byte
	value  any
}

func (t token) GoString() string {
	return fmt.Sprintf("%s%#v", string(t.prefix), t.value)
}
func (t *token) Value() any {
	return t.value
}

type keyValuePair struct {
	key    string
	value  *token
	preKey []byte
	preCol []byte
	last   []byte
}

func (k keyValuePair) GoString() string {
	return fmt.Sprintf("%s%q%s:%#v%s",
		string(k.preKey),
		k.key,
		string(k.preCol),
		k.value,
		string(k.last))
}

type object []keyValuePair

func (o object) GoString() string {
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

// .preKey + .key + .preCol + ':' + .value.prefix + .value.value + .last

func readObject(br io.RuneScanner) (object, error) {
	first, _, err := br.ReadRune()
	if err != nil {
		return nil, err
	}
	if first == '}' {
		return []keyValuePair{}, nil
	}
	br.UnreadRune()

	var pairs []keyValuePair
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
		pairs = append(pairs, keyValuePair{
			key:    key,
			value:  val,
			preKey: preKey,
			preCol: preVal,
			last:   last,
		})
		if ch == '}' {
			return object(pairs), nil
		}
		if ch != ',' {
			return nil, &UnexpectedTokenError2{
				expect1: '}',
				expect2: ',',
				got:     ch}
		}
	}
}

type arrayElement struct {
	*token
	preComma []byte
}

func (a arrayElement) GoString() string {
	return fmt.Sprintf("%#v%s", a.token, string(a.preComma))
}

type array []arrayElement

func (a array) GoString() string {
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

func readArray(br io.RuneScanner) (array, error) {
	first, _, err := br.ReadRune() // check '['
	if err != nil {
		return nil, err
	}
	var array1 []arrayElement
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
		array1 = append(array1, arrayElement{
			token:    token,
			preComma: prefix,
		})
		if ch == ']' {
			return array(array1), nil
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

func readItem(br io.RuneScanner) (*token, error) {
	prefix, ch, err := read1st(br)
	if err != nil {
		return nil, err
	}
	if ch == '"' {
		s, err := readString(br)
		return &token{prefix: prefix, value: s}, err
	} else if strings.ContainsRune("0123456789-+.", ch) {
		n, err := readNumber(br, ch)
		return &token{prefix: prefix, value: n}, err
	} else if ch == 'n' {
		return &token{prefix: prefix, value: nil}, expectToken(br, ch, "null")
	} else if ch == 'f' {
		return &token{prefix: prefix, value: false}, expectToken(br, ch, "false")
	} else if ch == 't' {
		return &token{prefix: prefix, value: true}, expectToken(br, ch, "true")
	} else if ch == '{' {
		o, err := readObject(br)
		return &token{prefix: prefix, value: o}, err
	} else if ch == '[' {
		a, err := readArray(br)
		return &token{prefix: prefix, value: a}, err
	}
	token, err := readToken(br, ch)
	if err != nil {
		return nil, err
	}
	return nil, &InvalidLiteralError{got: string(token)}
}

func unmarshal(data []byte) (any, error) {
	sc := newScanner(bytes.NewReader(data))
	v, err := readItem(sc)
	if err != nil {
		err = sc.WrapError(err)
	}
	return v, err
}
