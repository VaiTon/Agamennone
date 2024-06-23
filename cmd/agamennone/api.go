package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/charmbracelet/log"
	"github.com/labstack/echo/v4"

	aflag "github.com/VaiTon/Agamennone/pkg/flag"
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
	apiR.GET("/stats/exploits", getStatsByExploit)

}

func getStatsByExploit(c echo.Context) error {
	stats, err := storage.GetStatsByExploit()
	if err != nil {
		log.Error("error getting statistics by exploit from database", "err", err)
		return c.String(http.StatusInternalServerError, "Oops! Something went wrong")
	}

	/*
		{
			"exploit1": {
				"rejected": {
					"10:00": 10,
					"11:00": 20
				},
				"accepted": {
					...
				},
				...
			},
			...
		}
	*/
	type exploitStat struct {
		Hour  time.Time `json:"hour"`
		Count int       `json:"count"`
	}
	type exploitsStats map[string]map[string][]exploitStat

	apiStats := make(exploitsStats)
	for _, s := range stats {
		if _, ok := apiStats[s.Exploit]; !ok {
			apiStats[s.Exploit] = make(map[string][]exploitStat)
		}

		apiStats[s.Exploit][s.Status] = append(
			apiStats[s.Exploit][s.Status],

			exploitStat{Hour: s.Hour.In(time.Local), Count: s.Count},
		)
	}

	return c.JSON(http.StatusOK, apiStats)
}

func getConfig(c echo.Context) error {
	type clientConfig struct {
		FlagFormat   string            `json:"FLAG_FORMAT"`
		SubmitPeriod int               `json:"SUBMIT_PERIOD"`
		FlagLifetime int               `json:"FLAG_LIFETIME"`
		Teams        ClientConfigTeams `json:"TEAMS"`
		AttackInfo   string            `json:"ATTACK_INFO"`
	}
	return c.JSON(http.StatusOK, clientConfig{
		FlagFormat:   serverConfig.FlagRegexStr,
		SubmitPeriod: serverConfig.SubmissionPeriod,
		FlagLifetime: serverConfig.FlagLifetime,
		Teams:        serverConfig.Teams,
		AttackInfo:   "",
	})
}

func getStats(c echo.Context) error {

	stats, err := storage.GetStatisticsV1()
	if err != nil {
		log.Error("error getting statistics from database", "err", err)
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
		QueuedFlags:   stats.TotalFlagsByStatus[aflag.StatusQueued],
		AcceptedFlags: stats.TotalFlagsByStatus[aflag.StatusAccepted],
		RejectedFlags: stats.TotalFlagsByStatus[aflag.StatusRejected],
		SkippedFlags:  stats.TotalFlagsByStatus[aflag.StatusSkipped],
	})
}

func postFlags(c echo.Context) error {
	receivedTime := time.Now()

	body := c.Request().Body

	// Parse JSON
	var partialFlags []aflag.Flag
	err := json.NewDecoder(body).Decode(&partialFlags)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid JSON")
	}

	// Filter out invalid flags
	validFlags := make([]aflag.Flag, 0, len(partialFlags))
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

	insertedFlags, err := storage.InsertFlags(validFlags)

	log.Info("received flags",
		"unique_flags", insertedFlags,
		"valid_flags", len(validFlags),
		"total_flags", len(partialFlags),
		"src", c.RealIP())

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

	flags, err := storage.GetLastFlags(limit)
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
