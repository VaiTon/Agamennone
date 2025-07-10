package agamennone

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/VaiTon/Agamennone/pkg/cachingproxy"
	"github.com/VaiTon/Agamennone/pkg/flag"
	"github.com/VaiTon/Agamennone/pkg/storage"
)

type ServerConfig struct {
	Store            storage.FlagStorage
	SubmissionPeriod time.Duration
	FlagRegex        *regexp.Regexp
	FlagLifetime     int
	Teams            map[string]string
	SubmitterPath    string
	DataSources      []string
	AllowedURLs      []string
}

type Server struct {
	*ServerConfig
	echo         *echo.Echo
	cachingProxy *cachingproxy.Proxy
}

type ClientConfigTeams map[string]string

type Teams map[string]string

type ClientConfig struct {
	FlagFormat   string            `json:"FLAG_FORMAT"`
	FlagLifetime int               `json:"FLAG_LIFETIME"`
	Teams        ClientConfigTeams `json:"TEAMS"`
	SubmitPeriod int               `json:"SUBMIT_PERIOD"`
	DataSources  []string          `json:"DATA_SOURCES"`
}

func NewServer(config *ServerConfig) (*Server, error) {
	e := echo.New()
	e.HideBanner = true
	e.Logger.SetOutput(io.Discard)

	logMiddleware := loggingMiddleware(slog.Default())
	e.Use(logMiddleware)

	cacheDuration := time.Duration(config.SubmissionPeriod) * time.Second
	cachingProxy := cachingproxy.NewCachingProxy(config.AllowedURLs, cacheDuration, http.DefaultClient)

	srv := &Server{
		ServerConfig: config,
		echo:         e,
		cachingProxy: cachingProxy,
	}

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Welcome to Agamennone!")
	})

	apiR := e.Group("/api")
	apiR.GET("/config", srv.getConfig)
	apiR.POST("/flags", srv.postFlags)
	apiR.GET("/flags", srv.getFlags)
	apiR.GET("/stats", srv.getStats)
	apiR.GET("/cache", srv.getCache)

	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	return srv, nil
}

func (s *Server) Start(addr string) error            { return s.echo.Start(addr) }
func (s *Server) Shutdown(ctx context.Context) error { return s.echo.Shutdown(ctx) }

func (s *Server) getConfig(c echo.Context) error {
	return c.JSON(http.StatusOK, ClientConfig{
		FlagFormat:   s.FlagRegex.String(),
		SubmitPeriod: int(s.SubmissionPeriod / time.Second),
		FlagLifetime: s.FlagLifetime,
		Teams:        ClientConfigTeams(s.Teams),
		DataSources:  []string{}, // TODO: remove this now that we have the proxy
	})
}

func (s *Server) getStats(c echo.Context) error {

	stats, err := s.Store.GetStatisticsV1()
	if err != nil {
		return internalError(c, err)
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

func (s *Server) postFlags(c echo.Context) error {
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
		if s.FlagRegex.MatchString(partialFlag.Flag) {
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

	insertedFlags, err := s.Store.InsertFlags(validFlags)
	if err != nil {
		return internalError(c, fmt.Errorf("inserting flags into database: %v", err))
	}

	slog.Info("received flags",
		"unique", insertedFlags, "valid", len(validFlags), "total", len(partialFlags),
		"exploit", validFlags[0].Exploit, "client", c.RealIP())

	return c.NoContent(http.StatusCreated)
}

func (s *Server) getFlags(c echo.Context) error {
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

	flags, err := s.Store.GetLastFlags(limit)
	if err != nil {
		return internalError(c, err)
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

func (r *Server) getCache(c echo.Context) error {
	url := c.QueryParam("url")
	if url == "" {
		return c.String(http.StatusBadRequest, "Missing url parameter")
	}

	err := r.cachingProxy.HandleRequest(url, c.Response().Writer)
	if err != nil {
		slog.Error("error fetching cache", "err", err)
		return internalError(c, err)
	}

	return nil
}
