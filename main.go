package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
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
