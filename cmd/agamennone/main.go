package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"time"

	"log/slog"

	"github.com/lmittmann/tint"

	"github.com/spf13/cobra"

	"github.com/VaiTon/Agamennone/pkg/agamennone"
	"github.com/VaiTon/Agamennone/pkg/storage"
	"github.com/VaiTon/Agamennone/pkg/submitter"
)

const header = `
	 █████╗  ██████╗  █████╗ ███╗   ███╗███████╗███╗   ██╗███╗   ██╗ ██████╗  ███╗   ██╗███████╗
	██╔══██╗██╔════╝ ██╔══██╗████╗ ████║██╔════╝████╗  ██║████╗  ██║██╔═══██╗ ████╗  ██║██╔════╝
	███████║██║  ███╗███████║██╔████╔██║█████╗  ██╔██╗ ██║██╔██╗ ██║██║   ██║ ██╔██╗ ██║█████╗
	██╔══██║██║   ██║██╔══██║██║╚██╔╝██║██╔══╝  ██║╚██╗██║██║╚██╗██║██║   ██║ ██║╚██╗██║██╔══╝
	██║  ██║╚██████╔╝██║  ██║██║ ╚═╝ ██║███████╗██║ ╚████║██║ ╚████║╚██████╔╝ ██║ ╚████║███████╗
	╚═╝  ╚═╝ ╚═════╝ ╚═╝  ╚═╝╚═╝     ╚═╝╚══════╝╚═╝  ╚═══╝╚═╝  ╚═══╝ ╚═════╝  ╚═╝  ╚═══╝╚══════╝
`

var (
	configPath string
	listenAddr string
	debug      bool
	dbConnStr  string
)

var rootCmd = &cobra.Command{
	Use:   "agamennone",
	Short: "Agamennone is a A/D CTF attack farm",
	Run:   Run,
}

func init() {
	rootCmd.Flags().StringVarP(&configPath, "config", "c", "config.json", "path to the configuration file")
	rootCmd.Flags().StringVarP(&listenAddr, "listen", "l", ":1234", "address to listen on")
	rootCmd.Flags().BoolVarP(&debug, "verbose", "v", false, "enable debug logging")
	rootCmd.Flags().StringVar(&dbConnStr, "db", "mysql://agamennone:agamennone@tcp(localhost:3306)/agamennone", "mariadb connection string")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		slog.Error("error executing command", "err", err)
		os.Exit(1)
	}
}

func checkConfig(config *FileConfig) error {
	if config.SubmitterPath == "" {
		return fmt.Errorf("submission protocol not set")
	} else if config.FlagLifetime == 0 {
		return fmt.Errorf("flag lifetime not set")
	} else if config.SubmissionPeriod == 0 {
		return fmt.Errorf("submission period not set")
	} else if config.FlagRegexStr == "" {
		return fmt.Errorf("flag regex not set")
	}
	return nil
}

func Run(cmd *cobra.Command, args []string) {
	println(header)
	setupLogger()

	// load configuration
	slog.Debug("loading configuration...", "path", configPath)
	config, err := loadConfig(configPath)
	if err != nil {
		slog.Error("error loading configuration", "err", err)
		os.Exit(1)
	}

	slog.Debug("running basic checks on configuration...")
	if checkErr := checkConfig(config); checkErr != nil {
		slog.Error("error in configuration", "err", checkErr)
		os.Exit(1)
	}

	slog.Debug("compiling flag regex...")
	flagRegex, err := regexp.Compile(config.FlagRegexStr)
	if err != nil {
		slog.Error("error compiling flag regex. check your config", "err", err)
		os.Exit(1)
	}

	slog.Info("creating storage...", "dbConnStr", dbConnStr)
	store, err := createStorage(dbConnStr)
	if err != nil {
		slog.Error("error creating storage", "err", err)
		os.Exit(1)
	}

	// wait until the database is ready
	for {
		if err = store.Init(); err == nil {
			break
		}

		slog.Error("unable to initialize the database", "err", err)
		slog.Warn("is the database running? sleeping for 5 seconds...")
		time.Sleep(5 * time.Second)
		continue
	}

	slog.Info("storage initialized successfully")

	// Handle interrupt signal
	globalCtx, cancelGlobalCtx := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancelGlobalCtx()

	// create http server
	slog.Info("creating server...", "address", listenAddr)
	cfg := agamennone.ServerConfig{
		FlagRegex:        flagRegex,
		SubmissionPeriod: config.SubmissionPeriod,
		FlagLifetime:     config.FlagLifetime,
		Store:            store,
		Teams:            config.Teams,
		SubmitterPath:    config.SubmitterPath,
		DataSources:      config.DataSources,
		AllowedURLs:      config.AllowedURLs,
	}
	server, err := agamennone.NewServer(&cfg)
	if err != nil {
		slog.Error("error creating server", "err", err)
		os.Exit(1)
	}
	go server.Start(listenAddr)
	slog.Info("server started successfully")

	// Start submit loop
	slog.Info("starting submitter...", "submitterPath", config.SubmitterPath, "submissionPeriod", config.SubmissionPeriod)
	s := submitter.NewSubmitter(config.SubmitterPath, config.SubmissionPeriod, store)
	go s.Start(globalCtx)
	slog.Info("submit loop started successfully")

	// Wait for the interrupt signal
	<-globalCtx.Done()
	slog.Info("Shutting down server...")

	// Gracefully shutdown the server
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = server.Shutdown(timeoutCtx)
	if err != nil {
		slog.Error("error during server shutdown", "err", err)
		os.Exit(1)
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

func setupLogger() {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}

	handler := tint.NewHandler(os.Stderr, &tint.Options{
		Level:      level,
		TimeFormat: "15:04:05",
	})
	slog.SetDefault(slog.New(handler))
}
