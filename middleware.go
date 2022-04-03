package main

import "github.com/gofiber/fiber/v2"

func useSecurity(ctx *fiber.Ctx) error {
	ctx.Set("X-Frame-Options", "SAMEORIGIN")
	ctx.Set("X-DNS-Prefetch-Control", "off")
	// TODO - maybe we should request HSTS preload in the future: https://hstspreload.org
	ctx.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
	return ctx.Next()
}

func use404(ctx *fiber.Ctx) error {
	return ctx.Status(fiber.StatusNotFound).
		SendString("404")
}
