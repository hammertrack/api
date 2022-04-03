package main

import (
	"github.com/hammertrack/tracker/errors"
	"github.com/hammertrack/tracker/utils"
)

type Cursor []byte

func (c Cursor) Obscure() (string, error) {
	b, err := encrypt(c, DBCursorSecret)
	if err != nil {
		return "", err
	}
	return utils.ByteToStr(b), nil
}

func cursorFromString(s string) (Cursor, error) {
	b, err := decrypt(s, DBCursorSecret)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return Cursor(b), nil
}
