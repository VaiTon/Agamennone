package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/lmittmann/tint"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/VaiTon/Agamennone/pkg/achille"
)

const header = `
	 ░▒▓██████▓▒░ ░▒▓██████▓▒░░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░      ░▒▓████████▓▒░
	░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░      ░▒▓█▓▒░
	░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░      ░▒▓█▓▒░
	░▒▓████████▓▒░▒▓█▓▒░      ░▒▓████████▓▒░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░      ░▒▓██████▓▒░
	░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░      ░▒▓█▓▒░
	░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░      ░▒▓█▓▒░
	░▒▓█▓▒░░▒▓█▓▒░░▒▓██████▓▒░░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░▒▓████████▓▒░▒▓████████▓▒░▒▓████████▓▒░
`

var rootCmd = &cobra.Command{
	Use:   "achille <exploit>",
	Short: "Achille is the client CLI for the Agamennone A/D flag farm",
	Long: `Achille is the client CLI for the Agamennone A/D flag farm.
	It is used to run exploits and send flags to the server.`,
	Example:       `achille exploit1.py`,
	Run:           RootCommand,
	SilenceErrors: true,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("missing exploit path")
		}
		return nil
	},
}

var (
	verbose       bool
	exploitOutput bool
	exploitName   string
	timeout       int
	workers       int
)

func init() {
	viper.AutomaticEnv()
	pFlags := rootCmd.PersistentFlags()

	pFlags.StringP("host", "H", "http://localhost:1234", "Host and port of the Agamennone server")
	viper.BindPFlag("host", pFlags.Lookup("host"))

	pFlags.BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
	pFlags.StringVarP(&exploitName, "exploit", "e", "", "Exploit name")
	pFlags.BoolVarP(&exploitOutput, "output", "o", false, "Print the exploit output")
	pFlags.IntVarP(&timeout, "timeout", "t", 10, "Timeout for the exploit in seconds")
	pFlags.IntVarP(&workers, "workers", "w", 4, "Number of workers to use for the exploit")

	rootCmd.AddCommand(createCmd)
}

func main() {
	println(header)
	setupLogger()
	if err := rootCmd.Execute(); err != nil {
		slog.Error("command failed", "err", err)
		os.Exit(1)
	}
}

func RootCommand(cmd *cobra.Command, args []string) {
	if verbose {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	host := viper.GetString("host")
	if host == "" {
		slog.Error("Host not set. Use -H or --host to set the host and port of the Agamennone server.")
		os.Exit(1)
	}

	slog.Info("Starting Achille...", "host", host)
	api := achille.NewAgamennoneApi(host)

	// resolve exploit path
	exploit := args[0]
	path, err := exec.LookPath(exploit)
	if err != nil {
		slog.Error("executable exploit not found", "path", exploit, "error", err)
		os.Exit(1)
	}

	// if the exploit name is not set, use the last part of the path
	if exploitName == "" {
		exploitName = filepath.Base(path)
	}

	slog.Info("running exploit", "exploit", exploitName, "path", path)

	exploitConfig := &achille.ExploitConfig{
		Path:        path,
		Name:        exploitName,
		PrintOutput: exploitOutput,
		Timeout:     time.Second * time.Duration(timeout),
		Workers:     workers,
		FarmHost:    host,
	}

	// run the exploit
	achille.RunExploit(api, exploitConfig)
}

func setupLogger() {
	handler := tint.NewHandler(os.Stderr, &tint.Options{
		Level:      slog.LevelInfo,
		TimeFormat: time.TimeOnly,
	})
	slog.SetDefault(slog.New(handler))
}
