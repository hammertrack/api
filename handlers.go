package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/hammertrack/tracker/errors"
)

type BanHandler struct {
	sto *Storage
}

func (b *BanHandler) UserEndpoint(ctx *fiber.Ctx) error {
	// see channel endpoint's note
	username, after := ctx.Params("username"), ctx.Query("after")
	var cursor Cursor
	if after != "" {
		cursor = cursorFromString(after)
	}
	if username == "" {
		return ctx.SendStatus(fiber.StatusBadRequest)
	}
	bans, err := b.sto.BansByUser(username, cursor)
	if err != nil {
		errors.WrapAndLog(err)
		return ctx.SendStatus(fiber.StatusInternalServerError)
	}
	return ctx.JSON(bans)
}

func (b *BanHandler) ChannelEndpoint(ctx *fiber.Ctx) error {
	// note: `username` and `after` will be stored in gocql.Query object but they
	// are released after being executed (before this func returns) and reseted,
	// so no refs to the values are stored after then. Also, `After` will be
	// copied by base64.Decode().
	//
	// keep track of every value returned from the context.
	username, after := ctx.Params("channel"), ctx.Query("after")
	var cursor Cursor
	if after != "" {
		cursor = cursorFromString(after)
	}
	if username == "" {
		return ctx.SendStatus(fiber.StatusBadRequest)
	}
	bans, err := b.sto.BansByChannel(username, cursor)
	if err != nil {
		errors.WrapAndLog(err)
		return ctx.SendStatus(fiber.StatusInternalServerError)
	}
	return ctx.JSON(bans)
}

func NewBanHandler(sto *Storage) *BanHandler {
	return &BanHandler{sto}
}
