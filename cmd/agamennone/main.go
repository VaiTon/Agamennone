package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/labstack/echo/v4"
	_ "github.com/mattn/go-sqlite3"

	"github.com/VaiTon/Agamennone/pkg/storage"
	"github.com/VaiTon/Agamennone/pkg/submitter"
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
	dbConnStr    = flag.String("db", "mysql://agamennone:agamennone@tcp(localhost:3306)/agamennone", "mariadb connection string")
)

var store storage.FlagStorage

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

	store, err = createStorage(*dbConnStr)
	if err != nil {
		log.Fatalf("Error creating storage: %v", err)
	}

	err = store.Init()
	if err != nil {
		log.Fatalf("error initializing database: %v", err)
	}

	httpLogger := log.WithPrefix("http")
	e := echo.New()
	e.HideBanner = true
	e.Logger.SetOutput(io.Discard)
	logMiddleware := loggingMiddleware(httpLogger)
	e.Use(logMiddleware)
	setupRouter(e)

	// Handle interrupt signal
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Start server
	go func() {
		httpLogger.Info("starting server", "addr", *listenAddr)
		if err := e.Start(*listenAddr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			httpLogger.Fatalf("shutting down server: %v", err)
		}
	}()

	s := submitter.NewSubmitter(serverConfig.SubmitterPath, serverConfig.SubmissionPeriod, store)
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

func createStorage(storageConnStr string) (store storage.FlagStorage, err error) {
	parts := strings.Split(storageConnStr, "://")
	if len(parts) < 2 {
		err = fmt.Errorf("invalid database connection string: %s", storageConnStr)
		return
	}

	connType := strings.ToLower(parts[0])
	storageConnStr = strings.Join(parts[1:], ":")

	log.Debug("creating storage", "type", connType)
	switch connType {
	case "mysql":
		store, err = storage.NewMariaDBStorage(storageConnStr)
	case "sqlite":
		store, err = storage.NewSQliteStorage(storageConnStr)
	default:
		err = fmt.Errorf("unsupported database type: %s", connType)
	}
	return
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

func loggingMiddleware(logger *log.Logger) func(echo.HandlerFunc) echo.HandlerFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			timeTaken := time.Since(start)

			logger.Debugf("[%s] %s %s %s %d %s",
				c.RealIP(), c.Request().Method, c.Path(),
				c.Request().Proto, c.Response().Status, timeTaken,
			)

			return err
		}
	}
}
