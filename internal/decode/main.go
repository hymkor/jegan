package decode

import (
	"bytes"
	"errors"
	"io"
	"strings"

	"github.com/hymkor/go-generics-list"

	"github.com/hymkor/jegan/internal/dbg"
	"github.com/hymkor/jegan/internal/source"
	"github.com/hymkor/jegan/internal/types"
)

type List = list.List[types.Line]

type Element = list.Element[types.Line]

func readObject(br io.RuneScanner, basePath *types.JsonPath, nest int, store func(types.Line)) error {
	space0, first, err := source.Read1st(br) // check '}'
	if err != nil {
		return err
	}
	if first == '}' {
		store(types.NewItem(types.ObjEnd, nest, false, space0))
		return nil
	}
	br.UnreadRune()
	for {
		space1, err := source.ExpectRune(br, '"')
		if err != nil {
			return err
		}
		if len(space0) > 0 {
			space1 = append(space1, space0...)
			space0 = nil
		}
		key, err := source.ReadString(br)
		if err != nil {
			return err
		}
		space2, err := source.ExpectRune(br, ':')
		if err != nil {
			return err
		}
		var last types.Line
		jp := &types.JsonPath{
			Parent: basePath,
			Text:   key.String(),
		}
		err = readItem(br, jp, nest+1, func(line types.Line) {
			if last == nil {
				path := line.Path()
				line = &types.Pair{
					SpaceKey:   space1,
					Key:        key.String(),
					SpaceColon: space2,
					Item:       *types.NewItem(line.Data(), nest+1, false, line.LeadingSpace()),
				}
				line.SetPath(path)
			}
			last = line
			store(line)
		})
		if err != nil {
			return err
		}
		space3, ch, err := source.Read1st(br)
		if err != nil {
			return err
		}
		if ch == '}' {
			store(types.NewItem(types.ObjEnd, nest, false, space3))
			return nil
		}
		if ch != ',' {
			return &source.UnexpectedTokenError2{
				Expect1: '}',
				Expect2: ',',
				Got:     ch}
		}
		last.SetSpaceCommaOrClose(space3)
		last.SetComma(true)
	}
	return nil
}

func readArray(br io.RuneScanner, basePath *types.JsonPath, nest int, store func(types.Line)) error {
	spaces, first, err := source.Read1st(br) // check ']'
	if err != nil {
		return err
	}
	if first == ']' {
		store(types.NewItem(types.ArrayEnd, nest, false, spaces))
		return nil
	}
	br.UnreadRune()
	count := 0
	for {
		var last types.Line
		jp := &types.JsonPath{
			Parent: basePath,
			Index:  count,
		}
		count++
		err := readItem(br, jp, nest+1, func(line types.Line) {
			if len(spaces) > 0 {
				line.SetLeadingSpace(append(spaces, line.LeadingSpace()...))
				spaces = nil
			}
			last = line
			store(line)
		})
		if err != nil {
			return err
		}
		space2, ch, err := source.Read1st(br)
		if err != nil {
			return err
		}
		if ch == ']' {
			store(types.NewItem(types.ArrayEnd, nest, false, space2))
			return nil
		}
		if ch != ',' {
			return &source.UnexpectedTokenError2{
				Expect1: ']',
				Expect2: ',',
				Got:     ch,
			}
		}
		last.SetComma(true)
		last.SetSpaceCommaOrClose(space2)
	}
	return nil
}

func readItem(br io.RuneScanner, basePath *types.JsonPath, nest int, store0 func(types.Line)) error {
	store := func(line types.Line) {
		line.SetPath(basePath)
		store0(line)
	}
	spaces, ch, err := source.Read1st(br)
	if err != nil {
		if len(spaces) <= 0 {
			return err
		}
		data := source.NewRawBytes(spaces)
		store(types.NewItem(data, nest, false, nil))
		return nil
	}
	if ch == '"' {
		s, err := source.ReadString(br)
		if err != nil {
			return err
		}
		store(types.NewItem(s, nest, false, spaces))
		return nil
	}
	if strings.ContainsRune("0123456789-+.", ch) {
		n, err := source.ReadNumber(br, ch)
		if err != nil {
			return err
		}
		store(types.NewItem(n, nest, false, spaces))
		return nil
	}
	if ch == 'n' {
		err := source.ExpectToken(br, ch, "null")
		if err != nil {
			return err
		}
		data := source.NewLiteral(nil, []byte("null"))
		store(types.NewItem(data, nest, false, spaces))
		return nil
	}
	if ch == 'f' {
		err := source.ExpectToken(br, ch, "false")
		if err != nil {
			return err
		}
		data := source.NewLiteral(false, []byte("false"))
		store(types.NewItem(data, nest, false, spaces))
		return nil
	}
	if ch == 't' {
		data := source.NewLiteral(true, []byte("true"))
		err := source.ExpectToken(br, ch, "true")
		if err != nil {
			return err
		}
		store(types.NewItem(data, nest, false, spaces))
		return nil
	}
	if ch == '{' {
		store(types.NewItem(types.ObjStart, nest, false, spaces))
		return readObject(br, basePath, nest, store0)
	}
	if ch == '[' {
		store(types.NewItem(types.ArrayStart, nest, false, spaces))
		return readArray(br, basePath, nest, store0)
	}
	var b bytes.Buffer
	b.Write(spaces)
	b.WriteRune(ch)
	for {
		ch, siz, err := br.ReadRune()
		if err != nil && !errors.Is(err, io.EOF) {
			bin := b.Bytes()
			rb := source.NewRawBytes(bin)
			store(types.NewItem(rb, nest, false, nil))
			return err
		}
		if siz > 0 {
			b.WriteRune(ch)
		}
		if err != nil {
			bin := b.Bytes()
			dbg.Println("RawBytes(2):", bin)
			rb := source.NewRawBytes(bin)
			store(types.NewItem(rb, nest, false, nil))
			return err
		}
		if ch == '=' {
			bin := b.Bytes()
			dbg.Println("JavaScript equation ?:", bin)
			rb := source.NewRawBytes(bin)
			store(types.NewItem(rb, nest, false, nil))
			return nil
		}
	}
	return nil
}

func Unmarshal(r io.RuneScanner, store func(types.Line)) error {
	sc := source.NewScanner(r)
	err := readItem(sc, nil, 0, store)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return err
		}
		return sc.WrapError(err)
	}
	return nil
}
