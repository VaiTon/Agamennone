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
	Name        string        // name of the exploit that is sent to the server
	Path        string        // path to the exploit
	PrintOutput bool          // whether to print the exploit output to the console
	Timeout     time.Duration // timeout for the exploit run
	Workers     int           // how many exploits can run in parallel
}

type exploitServerData struct {
	flagRegex *regexp.Regexp // regex to match flags in the exploit output
	dataPaths []string       // paths to the data sources, which are passed to the exploit
}

type exploitTeam struct{ name, addr string }

type exploitResult struct {
	team  exploitTeam
	flags []flag.Flag
}

type exploitError struct {
	team exploitTeam
	err  error
}

// RunExploit is the main function that runs the exploit.
//
// In a loop, it gets the server config, runs the exploit on all teams,
// and submits the flags to the server.
func RunExploit(api *AgamennoneApi, exploitConfig *ExploitConfig) {

	log.Debug("starting exploit runner", "config", exploitConfig)

	// we initialize this outside the loop to make sure that if
	// the server goes offline we can keep getting flags
	// and submit them later
	flags := make([]flag.Flag, 0)

	// main turn loop
	localTick := 0
	lastTickFailed := false

	log.Debug("creating worker pool", "size", exploitConfig.Workers)
	workersPool, err := ants.NewPool(exploitConfig.Workers)
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
			"tickPeriod", localTickPeriod, "timeout", exploitConfig.Timeout, "workers", exploitConfig.Workers)

		timePerExploit := localTickPeriod / time.Duration(len(config.Teams))
		timePerTimeout := timePerExploit * time.Duration(exploitConfig.Workers)
		if exploitConfig.Timeout > timePerTimeout {
			log.Warnf("╭──────────ATTENTION─────────")
			log.Warnf("│   YOU MAY BE LOSING FLAGS  ")
			log.Warnf("│ exploit timeout is too high!!!")
			log.Warnf("│ %s / %d teams = %s per exploit, for %d workers = %s max timeout (now got %s)",
				localTickPeriod, len(config.Teams), timePerExploit,
				exploitConfig.Workers, timePerTimeout, exploitConfig.Timeout)
			log.Warnf("│ consider reducing the timeout or increasing the number of workers")
			log.Warnf("╰────────────────────────────")
		}

		// submit to the worker pool an attack for each team
		errCh := make(chan exploitError, exploitConfig.Workers)
		resultsCh := make(chan exploitResult, exploitConfig.Workers)
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
			log.Warnf("╭──────────ATTENTION─────────")
			log.Warnf("│ YOU MAY NOT BE ATTACKING ALL TEAMS")
			log.Warnf("│ running exploit took too long!!!")
			log.Warnf("│ attack took %s, but tick period is %s", time.Since(startTime), localTickPeriod)
			log.Warnf("│ consider reducing exploit timeout or optimizing the exploit")
			log.Warnf("╰────────────────────────────")
		}

		localTick++
	}
}

// collectFlags stores a result by collectFlags
type collectFlagsResult struct {
	succeeded int
	errored   int
	timeout   int
	flags     []flag.Flag
}

// collectFlags collects the flags from the channels until all exploits are either
// successful or errored out. It returns the flags and the number of successful,
// errored, and timed out exploits.
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
			if errors.Is(e.err, context.DeadlineExceeded) {
				if localTick == 0 {
					log.Warn("exploit timed out", "team", e.team.name, "error", e.err)
				}
				timeoutExploits++
			} else {
				if localTick == 0 {
					log.Error("error running exploit", "team", e.team.name, "error", e.err)
				}
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

	var err error

	// execute the exploit
	output, err := cmd.CombinedOutput()

	// override error if the context is done
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return nil, context.DeadlineExceeded
	}

	// ignore (for now) any error

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

	return flags, err
}
