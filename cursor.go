package main

import (
	"encoding/base64"

	"github.com/hammertrack/tracker/utils"
)

type Cursor []byte

func (c Cursor) Encode() string {
	return base64.URLEncoding.EncodeToString(c)
}

func cursorFromString(s string) Cursor {
	// TODO - maybe we could minimize memory allocations reusing buffers from
	// other cursors by using a pool of buffers
	dbuf := make([]byte, len(s))
	n, err := base64.URLEncoding.Decode(dbuf, utils.StrToByte(s))
	if err != nil {
		panic(err)
	}
	return Cursor(dbuf[:n])
}
