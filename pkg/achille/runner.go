package achille

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
	"time"

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
	FarmHost    string        // host and port of the Agamennone server
}

type exploitServerData struct {
	agamennoneHost string         // host and port of the Agamennone server
	flagRegex      *regexp.Regexp // regex to match flags in the exploit output
	dataPaths      []string       // paths to the data sources, which are passed to the exploit
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

	slog.Debug("starting exploit runner", "config", exploitConfig)

	// we initialize this outside the loop to make sure that if
	// the server goes offline we can keep getting flags
	// and submit them later
	flags := make([]flag.Flag, 0)

	// main turn loop
	localTick := 0
	lastTickFailed := false

	slog.Debug("creating worker pool", "size", exploitConfig.Workers)
	workersPool, err := ants.NewPool(exploitConfig.Workers)
	if err != nil {
		slog.Error("error creating worker pool", "err", err)
		os.Exit(1)
	}

turnLoop:
	for {
		if lastTickFailed {
			slog.Warn("waiting 5 seconds before retrying")
			time.Sleep(5 * time.Second)
		}

		lastTickFailed = false
		startTime := time.Now()

		// get server config
		config, err := api.GetConfig()
		if err != nil {
			slog.Error("error getting server config", "err", err)

			slog.Warn("!!! check if the server is online !!!")
			lastTickFailed = true
			continue turnLoop
		}

		// compile the flag regex
		flagRegex, err := regexp.Compile(config.FlagFormat)
		if err != nil {
			slog.Error("error compiling flag regex", "err", err)
			slog.Warn("!!! check the server config !!!")
			lastTickFailed = true
			continue turnLoop
		}

		// write data sources to temp files
		dataPaths := make([]string, 0)
		for _, dataSource := range config.DataSources {
			dataSourceFile, err := os.CreateTemp("", "achille-data-source-")
			if err != nil {
				slog.Error("error creating data source temp file", "err", err)
				lastTickFailed = true
				continue turnLoop
			}

			_, err = dataSourceFile.Write([]byte(dataSource))
			if err != nil {
				slog.Error("error writing data source temp file", "err", err)
				lastTickFailed = true
				continue turnLoop
			}

			err = dataSourceFile.Close()
			if err != nil {
				slog.Error("error closing data source temp file", "err", err)
				lastTickFailed = true
				continue turnLoop
			}

			dataPaths = append(dataPaths, dataSourceFile.Name())
		}

		slog.Debug("Data sources", "files", dataPaths)
		serverData := &exploitServerData{
			agamennoneHost: exploitConfig.FarmHost,
			flagRegex:      flagRegex,
			dataPaths:      dataPaths,
		}

		localTickPeriod := time.Duration(config.SubmitPeriod) * time.Second

		slog.Info("starting attack", "turn", localTick, "teams", len(config.Teams),
			"tickPeriod", localTickPeriod, "timeout", exploitConfig.Timeout, "workers", exploitConfig.Workers)

		warnIfMayLosingFlag(localTickPeriod, exploitConfig.Timeout, len(config.Teams), exploitConfig.Workers)

		// submit to the worker pool an attack for each team
		errCh := make(chan exploitError, exploitConfig.Workers)
		resultsCh := make(chan exploitResult, exploitConfig.Workers)
		go submitAttacks(config.Teams, workersPool, exploitConfig, serverData, errCh, resultsCh)

		// accumulate flags from all coroutines
		result := collectFlags(resultsCh, errCh, localTick, len(config.Teams))
		flags = append(flags, result.flags...)

		// log results to console
		logAttackResults(result, len(config.Teams))

		// check if we got any flags
		if len(flags) != 0 {
			err := api.SubmitFlags(flags)
			if err != nil {
				slog.Debug("error submitting flags", "err", err)
			} else {
				slog.Debug("submitted flags", "count", len(flags))

				// if we successfully submitted the flags, we can clear the slice
				flags = make([]flag.Flag, 0)
			}
		}

		// delete temp files
		for _, dataPath := range dataPaths {
			err := os.Remove(dataPath)
			if err != nil {
				slog.Error("error deleting temp file", "file", dataPath, "err", err)
			}
		}
		slog.Debug("deleted data sources temp files", "files", dataPaths)

		// sleep until the next turn
		remainingTime := time.Until(startTime.Add(localTickPeriod))
		if remainingTime > 0 {
			slog.Info("finished attack. sleeping until next turn", "duration", remainingTime)
			time.Sleep(remainingTime)
		} else {
			slog.Warn("╭────────ATTENTION────────")
			slog.Warn("│ YOU MAY NOT BE ATTACKING ALL TEAMS")
			slog.Warn("│ running exploit took too long!!!")
			slog.Warn("│ attack took", "duration", time.Since(startTime), "tickPeriod", localTickPeriod)
			slog.Warn("│ consider reducing exploit timeout or optimizing the exploit")
			slog.Warn("╰────────────────────────")
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
					slog.Warn("exploit timed out", "team", e.team.name, "err", e.err)
				}
				timeoutExploits++
			} else {
				if localTick == 0 {
					slog.Error("error running exploit", "team", e.team.name, "err", e.err)
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

	notifyPeriod := config.Timeout / 2
	notifyTime := time.Now().Add(notifyPeriod)

	attackedTeams := 0
	for teamName, teamAddr := range teams {
		team := exploitTeam{name: teamName, addr: teamAddr}
		if time.Now().After(notifyTime) {
			slog.Info("still running exploits", "done", attackedTeams, "total", len(teams))
			notifyTime = time.Now().Add(notifyPeriod)
		}

		err := pool.Submit(func() {

			exploitCtx, cancelExploitCtx := context.WithTimeout(context.Background(), config.Timeout)
			exploitFlags, err := runExploitOnTeam(exploitCtx, config, data, team)
			if err != nil {
				errorChan <- exploitError{team, err}
				cancelExploitCtx()
				return
			}
			flagChan <- exploitResult{team, exploitFlags}
			cancelExploitCtx()
		})

		if err != nil {
			slog.Error("error submitting attack to worker pool", "err", err)
			os.Exit(1)
		}

		attackedTeams++
	}
}

func runExploitOnTeam(
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

	// set the environment variables for the exploit
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "PYTHONUNBUFFERED=1")
	cmd.Env = append(cmd.Env, "CACHE_URL="+data.agamennoneHost+"/api/cache?url=")
	cmd.Env = append(cmd.Env, "TARGET="+team.addr)

	slog.Debug("running exploit", "command", cmd.String(), "team", team.name)

	var err error

	// execute the exploit
	output, err := cmd.CombinedOutput()

	// override error if the context is done
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return nil, context.DeadlineExceeded
	}

	// ignore any error. we will return them later

	if exploit.PrintOutput {
		slog.Debug("exploit produced output", "output", string(output))
	}

	flags := extractFlags(data.flagRegex, output, exploit, team)
	return flags, err
}

func extractFlags(regex *regexp.Regexp, output []byte, exploit *ExploitConfig, team exploitTeam) []flag.Flag {
	// extract the flags from the output
	flagStrings := regex.FindAll(output, -1)

	// add info to flags
	flags := make([]flag.Flag, len(flagStrings))
	for i, flagString := range flagStrings {
		flags[i] = flag.Flag{Flag: string(flagString), Exploit: exploit.Name, Team: team.name}
	}

	return flags
}

func warnIfMayLosingFlag(tickLength, exploitTimeout time.Duration, teams, workers int) {
	timePerExploit := tickLength / time.Duration(teams)
	timePerTimeout := timePerExploit * time.Duration(workers)
	if exploitTimeout > timePerTimeout {
		slog.Warn("╭────────ATTENTION────────")
		slog.Warn("│   YOU MAY BE LOSING FLAGS  ")
		slog.Warn("│ exploit timeout is too high!!!")
		slog.Warn("│", "tickLength", tickLength, "teams", teams, "perExploit", timePerExploit, "workers", workers, "maxTimeout", timePerTimeout, "currentTimeout", exploitTimeout)
		slog.Warn("│ consider reducing the timeout or increasing the number of workers")
		slog.Warn("╰────────────────────────")
	}
}

func logAttackResults(result collectFlagsResult, teams int) {
	if result.errored > 0 {
		percentErr := float32(result.errored) / float32(teams) * 100
		slog.Error("some exploits errored", "count", result.errored, "percent", percentErr)
	}

	if result.timeout > 0 {
		percentTout := float32(result.timeout) / float32(teams) * 100
		slog.Warn("some exploits timed out", "count", result.timeout, "percent", percentTout)
	}

	percentAttacked := float32(result.succeeded) / float32(teams) * 100
	flagsPerTeam := float32(len(result.flags)) / float32(result.succeeded)

	slog.Info("attack finished", "attacked", result.succeeded, "percentAttacked", percentAttacked, "flags", len(result.flags), "flagsPerTeam", flagsPerTeam)
}
