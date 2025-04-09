package agamennone

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/charmbracelet/log"
	"github.com/labstack/echo/v4"

	"github.com/VaiTon/Agamennone/pkg/flag"
)

func setupRouter(e *echo.Echo) {
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Welcome to Agamennone!")
	})

	apiR := e.Group("/api")
	apiR.GET("/config", getConfig)
	apiR.POST("/flags", postFlags)
	apiR.GET("/flags", getFlags)
	apiR.GET("/stats", getStats)
}

type ClientConfig struct {
	FlagFormat   string            `json:"FLAG_FORMAT"`
	FlagLifetime int               `json:"FLAG_LIFETIME"`
	Teams        ClientConfigTeams `json:"TEAMS"`
	SubmitPeriod int               `json:"SUBMIT_PERIOD"`
	DataSources  []string          `json:"DATA_SOURCES"`
}

func getConfig(c echo.Context) error {

	config := ClientConfig{
		FlagFormat:   serverConfig.FlagRegexStr,
		SubmitPeriod: serverConfig.SubmissionPeriod,
		FlagLifetime: serverConfig.FlagLifetime,
		Teams:        serverConfig.Teams,
	}

	dataSourcesContent := make([]string, 0)

	for _, path := range serverConfig.DataSources {
		res, err := http.Get(path)
		if err != nil {
			log.Errorf("error getting data source %s: %v", path, err)
			return c.String(http.StatusInternalServerError, "Oops! Something went wrong")
		}

		if res.StatusCode != http.StatusOK {
			log.Errorf("error getting data source %s: %v", path, res.Status)
			return c.String(http.StatusInternalServerError, "Oops! Something went wrong")
		}

		body, err := io.ReadAll(res.Body)
		if err != nil {
			log.Errorf("error reading response body: %v", err)
			return c.String(http.StatusInternalServerError, "Oops! Something went wrong")
		}

		dataSourcesContent = append(dataSourcesContent, string(body))
	}

	config.DataSources = dataSourcesContent
	return c.JSON(http.StatusOK, config)
}

func getStats(c echo.Context) error {

	stats, err := store.GetStatisticsV1()
	if err != nil {
		log.Errorf("error getting statistics from database: %v", err)
		return c.String(http.StatusInternalServerError, "Oops! Something went wrong")
	}

	type serverStats struct {
		Flags              int `json:"flags"`
		QueuedFlags        int `json:"queuedFlags"`
		AcceptedFlags      int `json:"acceptedFlags"`
		RejectedFlags      int `json:"rejectedFlags"`
		SkippedFlags       int `json:"skippedFlags"`
		FlagsSentLastCycle int `json:"flagsSentLastCycle"`
		LastCycle          int `json:"lastCycle"`
	}

	return c.JSON(http.StatusOK, serverStats{
		Flags:         stats.TotalFlags,
		QueuedFlags:   stats.TotalFlagsByStatus[flag.StatusQueued],
		AcceptedFlags: stats.TotalFlagsByStatus[flag.StatusAccepted],
		RejectedFlags: stats.TotalFlagsByStatus[flag.StatusRejected],
		SkippedFlags:  stats.TotalFlagsByStatus[flag.StatusSkipped],
	})
}

func postFlags(c echo.Context) error {
	receivedTime := time.Now()

	body := c.Request().Body

	// Parse JSON
	var partialFlags []flag.Flag
	err := json.NewDecoder(body).Decode(&partialFlags)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid JSON")
	}

	// Filter out invalid flags
	validFlags := make([]flag.Flag, 0, len(partialFlags))
	for _, partialFlag := range partialFlags {
		if serverConfig.FlagRegex.MatchString(partialFlag.Flag) {
			validFlags = append(validFlags, partialFlag)
		}
	}

	// add receivedTime to valid flags
	for i := range validFlags {
		validFlags[i].ReceivedTime = receivedTime
	}

	if len(validFlags) == 0 {
		return c.String(http.StatusBadRequest, "No valid flags")
	}

	insertedFlags, err := store.InsertFlags(validFlags)
	if err != nil {
		log.Error("error inserting flags into database", "err", err)
		return c.String(http.StatusInternalServerError, "Oops! Something went wrong")
	}

	log.Info("received flags",
		"unique", insertedFlags,
		"valid", len(validFlags),
		"total", len(partialFlags),
		"exploit", validFlags[0].Exploit,
		"client", c.RealIP())

	return c.NoContent(http.StatusCreated)
}

func getFlags(c echo.Context) error {
	var (
		err   error
		limit = 100
	)

	// Get limit from query params
	if c.QueryParam("limit") != "" {
		limitStr := c.QueryParam("limit")
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			return c.String(http.StatusBadRequest, "Invalid limit")
		}
	}

	flags, err := store.GetLastFlags(limit)
	if err != nil {
		log.Error("error getting flags from database", "err", err)
		return c.String(http.StatusInternalServerError, "Oops! Something went wrong")
	}

	type apiFlag struct {
		Flag                string `json:"flag"`
		Exploit             string `json:"sploit"`
		Team                string `json:"team"`
		ReceivedTime        string `json:"receivedTime"`
		SentTime            string `json:"sentTime,omitempty"`
		Status              string `json:"status"`
		CheckSystemResponse string `json:"checkSystemResponse"`
	}

	apiFlags := make([]apiFlag, len(flags))
	for i, f := range flags {
		apiFlags[i] = apiFlag{
			Flag:                f.Flag,
			Exploit:             f.Exploit,
			Team:                f.Team,
			ReceivedTime:        f.ReceivedTime.Format(time.RFC3339),
			Status:              f.Status,
			CheckSystemResponse: f.CheckSystemResponse,
		}

		if !f.SentTime.IsZero() {
			apiFlags[i].SentTime = f.SentTime.Format(time.RFC3339)
		}
	}

	return c.JSON(http.StatusOK, apiFlags)
}
