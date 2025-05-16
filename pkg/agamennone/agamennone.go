package agamennone

import (
	"context"
	"errors"
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
	GameName         string            `json:"gameName"`
	FlagRegexStr     string            `json:"flagRegex"`
	SubmissionPeriod int               `json:"submissionPeriod"`
	FlagLifetime     int               `json:"flagLifetime"`
	ServerHost       string            `json:"serverHost"`
	ServerPort       int               `json:"serverPort"`
	Teams            map[string]string `json:"teams"`
	SubmitterPath    string            `json:"submitterPath"`
	AllowedURLs      []string          `json:"allowedURLs"`
}

var store storage.FlagStorage

type Teams map[string]string

type Config struct {
	ListenAddr       string        // address to listen on
	Debug            bool          // enable verbose logging
	DbConnectionStr  string        // database connection string
	GameName         string        // name of the game
	FlagRegexStr     string        // regex to match flags
	SubmissionPeriod int           // how often to submit flags
	FlagLifetime     int           // how long flags are valid
	ServerHost       string        // game server host to submit flags to
	ServerPort       int           // game server port to submit flags to
	SubmitterPath    string        // path to the submitter executable
	Teams            Teams         // map of team names to addresses
	FlagRegex        regexp.Regexp // compiled regex for matching flags
	AllowedURLs      []string      // URLs to query to get data for exploits
}

var serverConfig *Config

func Start(config *Config) {
	// copy the config to the global variable
	serverConfig = config

	if config.Debug {
		log.SetLevel(log.DebugLevel)
	}

	var err error
	store, err = createStorage(config.DbConnectionStr)
	if err != nil {
		log.Fatalf("error creating storage: %v", err)
	}

	// wait until the database is ready
	for {
		err = store.Init()
		if err == nil {
			break
		}

		log.Errorf("unable to initialize the database: %v", err)
		log.Warnf("is the database running? sleeping for 5 seconds...")
		time.Sleep(5 * time.Second)
		continue
	}

	log.Info("database initialized successfully", "addr", config.DbConnectionStr)

	httpLogger := log.WithPrefix("http")
	e := echo.New()
	e.HideBanner = true
	e.Logger.SetOutput(io.Discard)
	logMiddleware := loggingMiddleware(httpLogger)
	e.Use(logMiddleware)

	router := NewRouter(e, config)
	router.setupRouter()

	// Handle interrupt signal
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Start server
	go func() {
		httpLogger.Info("starting server", "addr", config.ListenAddr)
		if err := e.Start(config.ListenAddr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			httpLogger.Fatalf("shutting down server: %v", err)
		}
	}()

	// Start submit loop
	s := submitter.NewSubmitter(config.SubmitterPath, config.SubmissionPeriod, store)
	go s.SubmitLoop(ctx)

	// Wait for the interrupt signal
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
