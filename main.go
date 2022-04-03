package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/etag"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/monitor"
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
	app := fiber.New(fiber.Config{
		// Limits request body (B)
		BodyLimit: ServerBodyLimitBytes,
		// Max. number of concurrent conns
		Concurrency:  ServerConcurrency,
		ReadTimeout:  time.Duration(ServerReadTimeout) * time.Second,
		WriteTimeout: time.Duration(ServerWriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(ServerIdleTimeout) * time.Second,
		// Buffer size for reading each request. This also limits the header size,
		// so increase if big cookies or URIS are included
		ReadBufferSize: ServerReadBufferSize,
		// Buffer size for writing each response
		WriteBufferSize: ServerWriteBufferSize,
		// Adjust when behind a cdn, load balancer, reverse proxy, etc. This will
		// set the ip of ctx.IP() to the value of the header of the given key
		ProxyHeader: ServerProxyHeader,
		// Rejects non-GET requests. The request sizse is limited by ReadBufferSize
		// if enabled. Useful as anti-DoS protection
		GETOnly:           true,
		EnablePrintRoutes: Debug,
	})

	api := app.Group("/api", useSecurity)

	v1 := api.Group("/v1")
	// Bans of `username`
	v1.Get("/ban/user/:username", b.UserEndpoint)
	// Bans of the the channel
	v1.Get("/ban/channel/:channel", b.ChannelEndpoint)

	app.Get("/metrics", monitor.New())

	app.Use(cors.New(cors.Config{
		AllowOrigins: "https://hammertrack.com, http://127.0.0.1:3000",
		AllowHeaders: "Origin, Content-Type, Accept",
		AllowMethods: "GET",
	}))
	app.Use(limiter.New(limiter.Config{
		Max:        30,
		Expiration: 30 * time.Second,
		LimitReached: func(ctx *fiber.Ctx) error {
			return ctx.SendStatus(fiber.StatusTooManyRequests)
		},
		// Adjust when we setting up a reverse proxy, load balancer, cdn, etc
		// KeyGenerator: func (ctx *fiber.Ctx) string {
		//   return ctx.Get("x-forwarded-for")
		// },
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
