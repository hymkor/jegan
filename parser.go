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

func read1st(br io.RuneScanner) (rune, error) {
	for {
		ch, _, err := br.ReadRune()
		if err != nil {
			return ch, err
		}
		if !unicode.IsSpace(ch) {
			return ch, nil
		}
	}
}

func expectRune(br io.RuneScanner, expect rune) error {
	ch, err := read1st(br)
	if err != nil {
		if err == io.EOF {
			return io.ErrUnexpectedEOF
		}
		return err
	}
	if ch != expect {
		return fmt.Errorf("expect '%c', but '%c'", expect, ch)
	}
	return nil
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
		return fmt.Errorf("unknown token %s", result)
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

type keyValuePair struct {
	key   string
	value any
}

func readHash(br io.RuneScanner) ([]keyValuePair, error) {
	first, err := read1st(br)
	if err != nil {
		return nil, err
	}
	if first == '}' {
		return []keyValuePair{}, nil
	}
	br.UnreadRune()

	var result []keyValuePair
	for {
		if err := expectRune(br, '"'); err != nil {
			return nil, err
		}
		key, err := readString(br)
		if err != nil {
			return nil, err
		}
		if err := expectRune(br, ':'); err != nil {
			return nil, err
		}
		val, err := readItem(br)
		if err != nil {
			return nil, err
		}
		result = append(result, keyValuePair{key: key, value: val})
		ch, err := read1st(br)
		if err != nil {
			return nil, err
		}
		if ch == '}' {
			return result, err
		}
		if ch != ',' {
			return nil, fmt.Errorf("expect '}' or ',', but '%c'", ch)
		}
	}
}

func readArray(br io.RuneScanner) ([]any, error) {
	var result []any
	first, err := read1st(br)
	if err != nil {
		return nil, err
	}
	if first == ']' {
		return []any{}, nil
	}
	br.UnreadRune()
	for {
		element, err := readItem(br)
		if err != nil {
			return result, err
		}
		result = append(result, element)
		ch, err := read1st(br)
		if ch == ']' {
			return result, nil
		}
		if ch != ',' {
			return nil, fmt.Errorf("expect ']' or ',', but '%c'", ch)
		}
	}
}

func readItem(br io.RuneScanner) (any, error) {
	ch, err := read1st(br)
	if err != nil {
		return nil, err
	}
	if ch == '"' {
		return readString(br)
	} else if strings.ContainsRune("0123456789-+.", ch) {
		return readNumber(br, ch)
	} else if ch == 'n' {
		return nil, expectToken(br, ch, "null")
	} else if ch == 'f' {
		return false, expectToken(br, ch, "false")
	} else if ch == 't' {
		return true, expectToken(br, ch, "true")
	} else if ch == '{' {
		return readHash(br)
	} else if ch == '[' {
		return readArray(br)
	}
	token, err := readToken(br, ch)
	if err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("syntax error %s", string(token))
}

func unmarshal(data []byte) (any, error) {
	br := bytes.NewReader(data)
	return readItem(br)
}
