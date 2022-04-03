package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/etag"
	"github.com/hammertrack/tracker/errors"
	"github.com/hammertrack/tracker/logger"
)

func waitSignInt() {
	sigint := make(chan os.Signal, 1)
	signal.Notify(
		sigint,
		os.Interrupt,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGABRT,
		syscall.SIGQUIT,
	)
	<-sigint
	log.Print("Stopping hammertrack api")
}

func main() {
	sto := NewStorage(Cassandra())
	b := NewBanHandler(sto)

	log.Print("spawning server...")
	app := fiber.New()

	api := app.Group("/api", useSecurity)

	v1 := api.Group("/v1")
	// Bans of `username`
	v1.Get("/ban/user/:username", b.UserEndpoint)
	// Bans of the the channel
	v1.Get("/ban/channel/:channel", b.ChannelEndpoint)

	app.Use(cors.New(cors.Config{
		AllowOrigins: "https://hammertrack.com, http://127.0.0.1:3000",
		AllowHeaders: "Origin, Content-Type, Accept",
		AllowMethods: "GET",
	}))
	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))
	app.Use(etag.New(etag.Config{
		Weak: false,
	}))
	app.Use(useSecurity)
	app.Use(use404)

	go func() {
		log.Print("Listening on :" + APIPort)
		if err := app.Listen(":" + APIPort); err != nil {
			errors.WrapFatal(err)
		}
	}()
	waitSignInt()
	if err := sto.Shutdown(); err != nil {
		errors.WrapFatal(err)
	}
	if err := app.Shutdown(); err != nil {
		errors.WrapFatal(err)
	}
}

func init() {
	LoadConfig()
	log.SetFlags(0)
	log.SetOutput(logger.New())
	log.Print("Initializing API server...")
}
