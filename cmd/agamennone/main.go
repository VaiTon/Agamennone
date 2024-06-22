package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"time"

	"github.com/labstack/echo/v4"
	_ "github.com/mattn/go-sqlite3"

	aflag "github.com/VaiTon/Agamennone/pkg/flag"
	"github.com/VaiTon/Agamennone/pkg/submitter"
)

type ClientConfigTeams map[string]string

type ClientConfig struct {
	FlagFormat   string            `json:"FLAG_FORMAT"`
	SubmitPeriod int               `json:"SUBMIT_PERIOD"`
	FlagLifetime int               `json:"FLAG_LIFETIME"`
	Teams        ClientConfigTeams `json:"TEAMS"`
	AttackInfo   string            `json:"ATTACK_INFO"`
}

type ServerConfig struct {
	GameName           string            `json:"gameName"`
	FlagRegexStr       string            `json:"flagRegex"`
	SubmissionProtocol string            `json:"submissionProtocol"`
	SubmissionHost     string            `json:"submissionHost"`
	SubmissionPort     int               `json:"submissionPort"`
	SubmissionPeriod   int               `json:"submissionPeriod"`
	ServerHost         string            `json:"serverHost"`
	ServerPort         int               `json:"serverPort"`
	Teams              map[string]string `json:"teams"`
	SubmitterPath      string            `json:"submitterPath"`
	FlagRegex          regexp.Regexp
}

func (c ServerConfig) ToClientConfig() ClientConfig {
	return ClientConfig{
		FlagFormat:   c.FlagRegexStr,
		SubmitPeriod: c.SubmissionPeriod,
		FlagLifetime: 0,
		Teams:        c.Teams,
		AttackInfo:   "",
	}
}

var serverConfig ServerConfig
var db *sql.DB

var configPath = flag.String("config", "config.json", "path to the configuration file")
var listenAddr = flag.String("listen", ":1234", "address to listen on")

func loadConfig() error {

	flag.Parse()

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

	regexp, err := regexp.Compile(serverConfig.FlagRegexStr)
	if err != nil {
		return fmt.Errorf("error compiling flag format regex: %v", err)
	}

	serverConfig.FlagRegex = *regexp

	return nil
}

type ClientFlag struct {
	Flag   string
	Sploit string
	Team   string
}

type ServerFlag struct {
	Flag         string `json:"flag"`
	Sploit       string `json:"sploit"`
	Team         string `json:"team"`
	ReceivedTime string `json:"received_time"`
	Status       string `json:"status"`
}

func main() {

	// load configuration
	err := loadConfig()
	if err != nil {
		log.Fatalf("error loading configuration: %v", err)
	}

	db, err = sql.Open("sqlite3", "./flags.db")
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS flags (flag TEXT, sploit TEXT, team TEXT, received_time TEXT, status TEXT)")
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_flags_flag ON flags (flag)")
	if err != nil {
		log.Fatal(err)
	}

	e := echo.New()

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Welcome to Agamennone!")
	})

	e.GET("/api/config", func(c echo.Context) error {
		return c.JSON(http.StatusOK, serverConfig.ToClientConfig())
	})

	e.POST("/api/flags", func(c echo.Context) error {
		recievedTime := time.Now().Format("2006-01-02 15:04:05")

		body := c.Request().Body

		// Parse JSON
		var partialFlags []ClientFlag
		err := json.NewDecoder(body).Decode(&partialFlags)
		if err != nil {
			return c.String(http.StatusBadRequest, "Invalid JSON")
		}

		// Filter out invalid flags
		validFlags := make([]ClientFlag, 0, len(partialFlags))
		for _, flag := range partialFlags {
			if serverConfig.FlagRegex.MatchString(flag.Flag) {
				validFlags = append(validFlags, flag)
			}
		}

		log.Printf("Received %d flags from %s", len(validFlags), c.RealIP())

		if len(validFlags) == 0 {
			return c.String(http.StatusBadRequest, "No valid flags")
		}

		// Insert flags into the database
		queryStr := "INSERT INTO flags (flag, sploit, team, received_time, status) VALUES "
		values := make([]interface{}, 0, len(validFlags)*5)

		for _, flag := range validFlags {
			queryStr += "(?, ?, ?, ?, ?), "
			values = append(values, flag.Flag, flag.Sploit, flag.Team, recievedTime, aflag.FlagStatusQueued)
		}

		queryStr = queryStr[:len(queryStr)-2] // Remove trailing comma and space

		_, err = db.Exec(queryStr, values...)
		if err != nil {
			log.Printf("Error inserting flags into database: %v", err)
			return c.String(http.StatusInternalServerError, "Error inserting flags into database")
		}

		return c.NoContent(http.StatusCreated)
	})

	e.GET("/api/flags", func(c echo.Context) error {
		rows, err := db.Query("SELECT flag, sploit, team, received_time, status FROM flags")
		if err != nil {
			log.Printf("Error querying flags from database: %v", err)
			return c.String(http.StatusInternalServerError, "Oops! Something went wrong")
		}

		flags := make([]ServerFlag, 0)
		for rows.Next() {
			var flag, sploit, team, receivedTime, status string
			err = rows.Scan(&flag, &sploit, &team, &receivedTime, &status)
			if err != nil {
				log.Printf("Error scanning flags from database: %v", err)
				return c.String(http.StatusInternalServerError, "Oops! Something went wrong")
			}

			flags = append(flags, ServerFlag{flag, sploit, team, receivedTime, status})
		}

		return c.JSON(http.StatusOK, flags)
	})

	type Stats struct {
		Flags              int `json:"flags"`
		QueuedFlags        int `json:"queuedFlags"`
		AcceptedFlags      int `json:"acceptedFlags"`
		RejectedFlags      int `json:"rejectedFlags"`
		SkippedFlags       int `json:"skippedFlags"`
		FlagsSentLastCycle int `json:"flagsSentLastCycle"`
		LastCycle          int `json:"lastCycle"`
	}

	e.GET("/api/stats", func(c echo.Context) error {
		// Get the number of flags in the database
		var flags, queuedFlags, acceptedFlags, rejectedFlags, skippedFlags int
		err := db.QueryRow("SELECT COUNT(*) FROM flags").Scan(&flags)
		if err != nil {
			log.Printf("Error querying flags from database: %v", err)
			return c.String(http.StatusInternalServerError, "Oops! Something went wrong")
		}

		// Get the number of flags in each state
		err = db.QueryRow("SELECT COUNT(*) FROM flags WHERE status = ?", aflag.FlagStatusQueued).Scan(&queuedFlags)
		if err != nil {
			log.Printf("Error querying queued flags from database: %v", err)
			return c.String(http.StatusInternalServerError, "Oops! Something went wrong")
		}

		err = db.QueryRow("SELECT COUNT(*) FROM flags WHERE status = ?", aflag.FlagStatusAccepted).Scan(&acceptedFlags)
		if err != nil {
			log.Printf("Error querying accepted flags from database: %v", err)
			return c.String(http.StatusInternalServerError, "Oops! Something went wrong")
		}

		err = db.QueryRow("SELECT COUNT(*) FROM flags WHERE status = ?", aflag.FlagStatusRejected).Scan(&rejectedFlags)
		if err != nil {
			log.Printf("Error querying rejected flags from database: %v", err)
			return c.String(http.StatusInternalServerError, "Oops! Something went wrong")
		}

		err = db.QueryRow("SELECT COUNT(*) FROM flags WHERE status = ?", aflag.FlagStatusSkipped).Scan(&skippedFlags)
		if err != nil {
			log.Printf("Error querying skipped flags from database: %v", err)
			return c.String(http.StatusInternalServerError, "Oops! Something went wrong")
		}

		return c.JSON(http.StatusOK, Stats{
			Flags:         flags,
			QueuedFlags:   queuedFlags,
			AcceptedFlags: acceptedFlags,
			RejectedFlags: rejectedFlags,
			SkippedFlags:  skippedFlags,
		})

	})

	e.Use(LoggingMiddleware)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Start server
	go func() {
		if err := e.Start(*listenAddr); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatal("shutting down the server")
		}
	}()
	s := submitter.NewSubmitter(
		serverConfig.SubmitterPath,
		serverConfig.SubmissionPeriod,
		db,
	)
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
	log.Println("Shutting down server")

	// Gracefully shutdown the server with a timeout of 10 seconds.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}
}

func LoggingMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		start := time.Now()
		err := next(c)
		timeTaken := time.Since(start)
		log.Printf("[%s] %s %s %s %d %s",
			c.RealIP(), c.Request().Method, c.Path(),
			c.Request().Proto, c.Response().Status, timeTaken,
		)
		return err
	}
}
