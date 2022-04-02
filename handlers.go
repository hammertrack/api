package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/hammertrack/tracker/errors"
)

type BanHandler struct {
	sto *Storage
}

func (b *BanHandler) UserEndpoint(ctx *fiber.Ctx) error {
	username := ctx.Params("username")
	if username == "" {
		return ctx.SendStatus(fiber.StatusBadRequest)
	}
	bans, err := b.sto.BansByUser(username)
	if err != nil {
		errors.WrapAndLog(err)
		return ctx.SendStatus(fiber.StatusInternalServerError)
	}
	return ctx.JSON(bans)
}

func (b *BanHandler) ChannelEndpoint(ctx *fiber.Ctx) error {
	username := ctx.Params("channel")
	if username == "" {
		return ctx.SendStatus(fiber.StatusBadRequest)
	}
	bans, err := b.sto.BansByChannel(username)
	if err != nil {
		errors.WrapAndLog(err)
		return ctx.SendStatus(fiber.StatusInternalServerError)
	}
	return ctx.JSON(bans)
}

func NewBanHandler(sto *Storage) *BanHandler {
	return &BanHandler{sto}
}
