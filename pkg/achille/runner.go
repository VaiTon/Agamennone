package achille

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"regexp"
	"time"

	"github.com/charmbracelet/log"
	"github.com/panjf2000/ants/v2"

	"github.com/VaiTon/Agamennone/pkg/agamennone"
	"github.com/VaiTon/Agamennone/pkg/flag"
)

type ExploitConfig struct {
	Name        string
	Path        string
	PrintOutput bool
	Timeout     time.Duration
}

type exploitServerData struct {
	flagRegex *regexp.Regexp
	dataPaths []string
}

type exploitTeam struct {
	name string
	addr string
}

type exploitResult struct {
	team  exploitTeam
	flags []flag.Flag
}

type exploitError struct {
	team exploitTeam
	err  error
}

var errorExploitTimeout = errors.New("exploit timed out")

func RunExploit(api *AgamennoneApi, exploitConfig *ExploitConfig) {

	log.Debug("starting exploit runner", "config", exploitConfig)

	// we initialize this outside the loop to make sure that if
	// the server goes offline we can keep getting flags
	// and submit them later
	flags := make([]flag.Flag, 0)

	// main turn loop
	localTick := 0
	lastTickFailed := false

	workers := 16 // TODO: make this configurable
	log.Debug("creating worker pool", "size", workers)
	workersPool, err := ants.NewPool(workers)
	if err != nil {
		log.Fatal("error creating worker pool", "error", err)
	}

turnLoop:
	for {
		if lastTickFailed {
			log.Warn("waiting 5 seconds before retrying")
			time.Sleep(5 * time.Second)
		}

		lastTickFailed = false
		startTime := time.Now()

		// get server config
		config, err := api.GetConfig()
		if err != nil {
			log.Error("error getting server config", "error", err)
			log.Warn("!!! check if the server is online !!!")
			lastTickFailed = true
			continue turnLoop
		}

		// compile the flag regex
		flagRegex, err := regexp.Compile(config.FlagFormat)
		if err != nil {
			log.Error("error compiling flag regex", "error", err)
			log.Warn("!!! check the server config !!!")
			lastTickFailed = true
			continue turnLoop
		}

		// write data sources to temp files
		dataPaths := make([]string, 0)
		for _, dataSource := range config.DataSources {
			dataSourceFile, err := os.CreateTemp("", "achille-data-source-")
			if err != nil {
				log.Error("error creating data source temp file", "error", err)
				lastTickFailed = true
				continue turnLoop
			}

			_, err = dataSourceFile.Write([]byte(dataSource))
			if err != nil {
				log.Error("error writing data source temp file", "error", err)
				lastTickFailed = true
				continue turnLoop
			}

			err = dataSourceFile.Close()
			if err != nil {
				log.Error("error closing data source temp file", "error", err)
				lastTickFailed = true
				continue turnLoop
			}

			dataPaths = append(dataPaths, dataSourceFile.Name())
		}

		log.Debug("Data sources", "files", dataPaths)
		serverData := &exploitServerData{
			flagRegex: flagRegex,
			dataPaths: dataPaths,
		}

		localTickPeriod := time.Duration(config.SubmitPeriod) * time.Second

		log.Info("starting attack", "turn", localTick, "teams", len(config.Teams),
			"tickPeriod", localTickPeriod, "timeout", exploitConfig.Timeout)

		// submit to the worker pool an attack for each team
		errCh := make(chan exploitError, workers)
		resultsCh := make(chan exploitResult, workers)
		go submitAttacks(config.Teams, workersPool, exploitConfig, serverData, errCh, resultsCh)

		// accumulate flags from all coroutines
		result := collectFlags(resultsCh, errCh, localTick, len(config.Teams))
		flags = append(flags, result.flags...)

		if result.errored > 0 {
			log.Errorf("some exploits errored: %d (%.2f%%)", result.errored,
				float32(result.errored)/float32(len(config.Teams))*100)
		}

		if result.timeout > 0 {
			log.Warnf("some exploits timed out: %d (%.2f%%)", result.timeout,
				float32(result.timeout)/float32(len(config.Teams))*100)
		}

		log.Infof("attack finished. attacked %d teams (%.2f%%) and got %d flags (%.2f flags/team)",
			result.succeeded, float32(result.succeeded)/float32(len(config.Teams))*100,
			len(result.flags), float32(len(result.flags))/float32(result.succeeded),
		)

		// check if we got any flags
		if len(flags) != 0 {
			err := api.SubmitFlags(flags)
			if err != nil {
				log.Debug("error submitting flags", "error", err)
			} else {
				log.Debug("submitted flags", "count", len(flags))

				// if we successfully submitted the flags, we can clear the slice
				flags = make([]flag.Flag, 0)
			}
		}

		// delete temp files
		for _, dataPath := range dataPaths {
			err := os.Remove(dataPath)
			if err != nil {
				log.Error("error deleting temp file", "file", dataPath, "error", err)
			}
		}
		log.Debug("deleted data sources temp files", "files", dataPaths)

		// sleep until the next turn
		remainingTime := time.Until(startTime.Add(localTickPeriod))
		if remainingTime > 0 {
			log.Info("finished attack. sleeping until next turn", "duration", remainingTime)
			time.Sleep(remainingTime)
		} else {
			log.Warnf("running exploit took too long. running %d seconds late", -remainingTime.Seconds())
			log.Warn("consider reducing exploit timeout or optimizing the exploit")
			log.Warn("YOU MAY NOT BE ATTACKING ALL TEAMS")
		}

		localTick++
	}
}

type collectFlagsResult struct {
	succeeded int
	errored   int
	timeout   int
	flags     []flag.Flag
}

func collectFlags(
	flagChan chan exploitResult,
	errorChan chan exploitError,
	localTick int,
	numTeams int,
) collectFlagsResult {
	flags := make([]flag.Flag, 0)
	erroredExploits := 0
	timeoutExploits := 0
	succeededExploits := 0

	for {
		select {
		case newFlags := <-flagChan:
			flags = append(flags, newFlags.flags...)
			succeededExploits++
		case e := <-errorChan:
			if errors.Is(e.err, errorExploitTimeout) {
				if localTick == 0 {
					log.Warn("exploit timed out", "team", e.team.name, "error", e.err)
				}
				timeoutExploits++
			} else {
				log.Error("error running exploit", "team", e.team.name, "error", e.err)
				erroredExploits++
			}
		}

		if succeededExploits+erroredExploits+timeoutExploits == numTeams {
			break
		}
	}
	return collectFlagsResult{
		flags:     flags,
		succeeded: succeededExploits,
		timeout:   timeoutExploits,
		errored:   erroredExploits,
	}
}

func submitAttacks(
	teams agamennone.ClientConfigTeams,

	pool *ants.Pool,
	config *ExploitConfig,
	data *exploitServerData,

	errorChan chan<- exploitError,
	flagChan chan<- exploitResult,
) {
	for teamName, teamAddr := range teams {
		team := exploitTeam{name: teamName, addr: teamAddr}

		err := pool.Submit(func() {
			exploitCtx, cancelExploitCtx := context.WithTimeout(context.Background(), config.Timeout)
			exploitFlags, err := RunExploitOnTeam(exploitCtx, config, data, team)
			if err != nil {
				errorChan <- exploitError{team, err}
				cancelExploitCtx()
				return
			}
			flagChan <- exploitResult{team, exploitFlags}
			cancelExploitCtx()
		})

		if err != nil {
			log.Fatal("error submitting attack to worker pool", "error", err)
		}
	}
}

func RunExploitOnTeam(
	ctx context.Context,
	exploit *ExploitConfig,
	data *exploitServerData,
	team exploitTeam,
) ([]flag.Flag, error) {
	args := []string{team.addr}
	for _, dataPath := range data.dataPaths {
		args = append(args, dataPath)
	}

	cmd := exec.CommandContext(ctx, exploit.Path, args...)
	log.Debug("running exploit", "command", cmd.String())

	// set the environment variables for the exploit
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "PYTHONUNBUFFERED=1")

	// execute the exploit
	output, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, errorExploitTimeout
		}
		return nil, err
	}

	if exploit.PrintOutput {
		log.Debug("exploit produced output", "output", string(output))
	}

	// extract the flags from the output
	flagStrings := data.flagRegex.FindAllString(string(output), -1)

	// add info to flags
	flags := make([]flag.Flag, len(flagStrings))
	for i, flagString := range flagStrings {
		flags[i] = flag.Flag{Flag: flagString, Exploit: exploit.Name, Team: team.name}
	}

	return flags, nil
}
