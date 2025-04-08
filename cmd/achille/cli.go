package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"time"

	"github.com/VaiTon/Agamennone/pkg/agamennone"
	"github.com/VaiTon/Agamennone/pkg/flag"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	verbose     bool
	exploitName string
)

func init() {
	viper.AutomaticEnv()
	pFlags := rootCmd.PersistentFlags()
	pFlags.StringP("host", "H", "http://localhost:1234", "Host and port of the Agamennone server")
	viper.BindPFlag("host", pFlags.Lookup("host"))
	pFlags.BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
	pFlags.StringVarP(&exploitName, "exploit", "e", "", "Exploit name")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func RootCommand(cmd *cobra.Command, args []string) {
	if verbose {
		log.SetLevel(log.DebugLevel)
	}

	host := viper.GetString("host")
	if host == "" {
		log.Fatal("Host not set. Use -H or --host to set the host and port of the Agamennone server.")
	}

	log.Info("Starting Achille...")
	log.Info("Connecting to Agamennone server...", "host", host)
	api := NewAgamennoneApi(host)
	if err := api.Connect(); err != nil {
		log.Fatal(err)
	}
	log.Debug("Connected to Agamennone server", "host", host)

	// resolve exploit path
	exploit := args[0]
	path, err := exec.LookPath(exploit)
	if err != nil {
		log.Fatal("error finding exploit", "exploit", exploit, "error", err)
	}

	if exploitName == "" {
		// get exploit name as the last part of the path
		exploitName = filepath.Base(path)
	}

	log.Info("running exploit", "exploit", exploitName, "path", path)
	RunExploit(api, path, exploitName)

}

func RunExploit(api *AgamennoneApi, path string, name string) {

	config := api.config
	flagRegex, err := regexp.Compile(config.FlagFormat)
	if err != nil {
		log.Fatal("error compiling flag regex", "error", err)
	}

	// parse attack period
	attackPeriod, err := time.ParseDuration(config.AttackPeriod)
	if err != nil {
		log.Fatal("error parsing attack period. check your server config", "error", err)
	}

	// write data sources to temp files
	dataPaths := make([]string, 0)
	for _, dataSource := range config.DataSources {
		dataSourceFile, err := os.CreateTemp("", "dataSource")
		if err != nil {
			log.Fatal("error creating temp file for data source", "error", err)
		}

		_, err = dataSourceFile.Write([]byte(dataSource))
		if err != nil {
			log.Fatal("error writing data source to temp file", "error", err)
		}

		err = dataSourceFile.Close()
		if err != nil {
			log.Fatal("error closing data source temp file", "error", err)
		}

		dataPaths = append(dataPaths, dataSourceFile.Name())
	}

	log.Debug("Data sources", "files", dataPaths)

	// we initialize this outside the loop to make sure that if
	// the server goes offline we can keep getting flags
	// and submit them later
	flags := make([]flag.Flag, 0)

	// main turn loop
	localTurn := 0
	for {
		log.Info("starting attack", "turn", localTurn)
		startTime := time.Now()
		flagChan := make(chan []flag.Flag)

		turnContext, turnCancel := context.WithTimeout(context.Background(), attackPeriod)

		for teamName, teamAddr := range config.Teams {
			// attack each team in a goroutine
			go func(teamName, teamAddr string) {
				exploitFlags, err := RunExploitOnTeam(turnContext, config, flagRegex,
					path, name, teamAddr, dataPaths, localTurn)
				if err != nil {
					log.Error("error running exploit", "team", teamName, "error", err)
					return
				}
				flagChan <- exploitFlags
			}(teamName, teamAddr)
		}

		// accumulate flags from all teams
		gotFlags := 0

	accumulateLoop:
		for {
			select {
			case f := <-flagChan:
				gotFlags++
				flags = append(flags, f...)
				if gotFlags == len(config.Teams) {
					// all teams have been attacked
					turnCancel()
					break accumulateLoop
				}
			case <-turnContext.Done():
				// timeout reached
				turnCancel()
				break accumulateLoop
			}
		}

		if len(flags) == 0 {
			log.Info("No flags found this round")
		} else {
			log.Info("Found flags", "count", len(flags))
			err := api.SubmitFlags(flags)
			if err != nil {
				log.Error("error submitting flags", "error", err)
			} else {
				log.Debug("submitted flags", "count", len(flags))
				flags = make([]flag.Flag, 0)
			}
		}

		remainingTime := time.Until(startTime.Add(attackPeriod))
		if remainingTime > 0 {
			log.Info("sleeping for", "duration", remainingTime)
			time.Sleep(remainingTime)
		} else {
			log.Debug("round time exceeded, skipping sleep")
		}

		localTurn++
	}

}

func RunExploitOnTeam(
	ctx context.Context,
	config agamennone.ClientConfig, flagRegex *regexp.Regexp,
	path string, name string, team string, dataPaths []string,
	localTurn int,
) ([]flag.Flag, error) {
	commandLine := []string{path, team}
	for _, dataPath := range dataPaths {
		commandLine = append(commandLine, dataPath)
	}

	log.Debug("Running exploit", "command", commandLine)
	cmd := exec.Command(commandLine[0], commandLine[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, string(output))
	}

	if localTurn < 5 {
		log.Debug("Exploit output", "output", string(output))
	}

	flagStrings := flagRegex.FindAllString(string(output), -1)
	flags := make([]flag.Flag, len(flagStrings))
	for i, flagString := range flagStrings {
		flags[i] = flag.Flag{Flag: flagString, Exploit: name, Team: team}
	}

	return flags, nil
}
