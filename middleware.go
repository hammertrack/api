package main

import "github.com/gofiber/fiber/v2"

func useSecurity(ctx *fiber.Ctx) error {
	ctx.Set("X-Frame-Options", "SAMEORIGIN")
	ctx.Set("X-DNS-Prefetch-Control", "off")
	ctx.Set("Strict-Transport-Security", "max-age=5184000")
	return ctx.Next()
}

func use404(ctx *fiber.Ctx) error {
	return ctx.Status(fiber.StatusNotFound).
		SendString("404")
}
