package source

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"unicode"
)

func Read1st(br io.RuneScanner) ([]byte, rune, error) {
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
	Expect1 rune
	Expect2 rune
	Got     rune
}

func (e UnexpectedTokenError2) Error() string {
	return fmt.Sprintf("expected '%c' or '%c', but got '%c'",
		e.Expect1,
		e.Expect2,
		e.Got)
}

type InvalidLiteralError struct {
	got string
}

func (e InvalidLiteralError) Error() string {
	return fmt.Sprintf("invalid literal: %q", e.got)
}

func ExpectRune(br io.RuneScanner, expect rune) ([]byte, error) {
	prefix, ch, err := Read1st(br)
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

func ReadString(br io.RuneScanner) (*Literal, error) {
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
			return NewLiteral(str, bin), err
		}
		if !backslash && ch == '\\' {
			backslash = true
		} else {
			backslash = false
		}
		buffer.WriteRune(ch)
	}
}

func ReadToken(br io.RuneScanner, first rune) ([]byte, error) {
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

func ExpectToken(br io.RuneScanner, first rune, expect string) error {
	s, err := ReadToken(br, first)
	result := string(s)
	if result != expect {
		return &InvalidLiteralError{got: result}
	}
	return err
}

func ReadNumber(br io.RuneScanner, first rune) (*Literal, error) {
	token, err1 := ReadToken(br, first)
	if err1 != nil {
		return nil, err1
	}
	var number float64
	err2 := json.Unmarshal(token, &number)
	if err2 != nil {
		return nil, err2
	}
	return NewLiteral(number, token), nil
}
