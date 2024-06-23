package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"time"

	"github.com/labstack/echo/v4"
	_ "github.com/mattn/go-sqlite3"

	storage2 "github.com/VaiTon/Agamennone/pkg/storage"
	"github.com/VaiTon/Agamennone/pkg/submitter"

	"github.com/charmbracelet/log"
)

type ClientConfigTeams map[string]string

type ServerConfig struct {
	GameName           string            `json:"gameName"`
	FlagRegexStr       string            `json:"flagRegex"`
	SubmissionProtocol string            `json:"submissionProtocol"`
	SubmissionPeriod   int               `json:"submissionPeriod"`
	FlagLifetime       int               `json:"flagLifetime"`
	ServerHost         string            `json:"serverHost"`
	ServerPort         int               `json:"serverPort"`
	Teams              map[string]string `json:"teams"`
	SubmitterPath      string            `json:"submitterPath"`
	FlagRegex          regexp.Regexp
}

var (
	serverConfig ServerConfig
	configPath   = flag.String("config", "config.json", "path to the configuration file")
	listenAddr   = flag.String("listen", ":1234", "address to listen on")
	debug        = flag.Bool("debug", false, "enable debug logging")
)

var storage storage2.FlagStorage

func main() {
	flag.Parse()

	if *debug {
		log.SetLevel(log.DebugLevel)
	}

	// load configuration
	log.Debug("loading configuration", "path", *configPath)
	err := loadConfig()
	if err != nil {
		log.Fatalf("error loading configuration: %v", err)
	}

	const dbPath = "flags.db"

	log.Print("opening database", "path", dbPath)
	storage, err = storage2.NewSqliteStorage(dbPath)
	if err != nil {
		log.Fatalf("error opening database: %v", err)
	}

	err = storage.Init()
	if err != nil {
		log.Fatalf("Error initializing database: %v", err)
	}

	e := echo.New()
	e.Use(loggingMiddleware)
	setupRouter(e)

	// Handle interrupt signal
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Start server
	go func() {
		log.Printf("Starting server on %s", *listenAddr)
		if err := e.Start(*listenAddr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			e.Logger.Fatal("shutting down the server")
		}
	}()

	s := submitter.NewSubmitter(serverConfig.SubmitterPath, serverConfig.SubmissionPeriod, storage)
	// Start submit loop
	go func() {
		err := s.SubmitLoop(ctx)
		if err != nil {
			log.Printf("Error in submit loop: %v", err)
			stop()
		}
	}()

	// Wait for interrupt signal
	<-ctx.Done()
	log.Printf("Shutting down server...")

	// Gracefully shutdown the server with a timeout of 10 seconds.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}
}

func loadConfig() error {

	file, err := os.Open(*configPath)
	if err != nil {
		return fmt.Errorf("error opening config file: %v", err)
	}

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&serverConfig)
	if err != nil {
		return fmt.Errorf("error decoding config file: %v", err)
	}

	err = file.Close()
	if err != nil {
		return fmt.Errorf("error closing config file: %v", err)
	}

	flagRegex, err := regexp.Compile(serverConfig.FlagRegexStr)
	if err != nil {
		return fmt.Errorf("error compiling flag format regex: %v", err)
	}

	serverConfig.FlagRegex = *flagRegex

	return nil
}

func loggingMiddleware(next echo.HandlerFunc) echo.HandlerFunc {

	l := log.WithPrefix("http")
	return func(c echo.Context) error {
		start := time.Now()
		err := next(c)
		timeTaken := time.Since(start)

		l.Debugf("[%s] %s %s %s %d %s",
			c.RealIP(), c.Request().Method, c.Path(),
			c.Request().Proto, c.Response().Status, timeTaken,
		)

		return err
	}
}
