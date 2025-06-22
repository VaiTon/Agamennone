package submitter

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/VaiTon/Agamennone/pkg/flag"
	"github.com/VaiTon/Agamennone/pkg/storage"
)

const (
	ResultOk      = "OK"
	ResultError   = "ERROR"
	ResultSkipped = "SKIPPED"
)

type Result struct {
	Flag    string
	Result  string
	Message string
}

type Submitter struct {
	storage storage.FlagStorage
	path    string

	submitPeriod int
}

// NewSubmitter creates a new Submitter instance.
func NewSubmitter(path string, submitPeriod int, storage storage.FlagStorage) *Submitter {
	return &Submitter{storage: storage, path: path, submitPeriod: submitPeriod}
}

// Submit executes the external submitter script with the given flags
// and returns the results
func (s *Submitter) Submit(ctx context.Context, flags []flag.Flag) ([]Result, error) {

	// check if the path exists and is a file
	if _, err := os.Stat(s.path); os.IsNotExist(err) {
		return nil, fmt.Errorf("file %s does not exist", s.path)
	}

	// check if the file is executable
	if info, err := os.Stat(s.path); err == nil {
		if info.Mode()&0111 == 0 {
			return nil, fmt.Errorf("file %s is not executable", s.path)
		}
	}

	// map flags to a string slice
	flagsTxts := make([]string, 0, len(flags))
	for _, f := range flags {
		flagsTxts = append(flagsTxts, f.Flag)
	}

	// execute the file, passing flags via stdin, \n is used as a delimiter
	cmd := exec.CommandContext(ctx, s.path)
	cmd.Stdin = strings.NewReader(strings.Join(flagsTxts, "\n"))

	// capture and parse the output
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error capturing output: %v. output was: %s", err, string(output))
	}

	// check the result: for each line in the output, check if it's OK, ERROR or SKIPPED
	results := make([]Result, 0, len(flags))

	if len(output) == 0 {
		return nil, fmt.Errorf("submitter returned an empty output")
	}

	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, " ")

		if len(parts) < 2 {
			return nil, fmt.Errorf("submitter returned an invalid line: %s", line)
		}

		f := parts[0]
		if !slices.Contains(flagsTxts, f) {
			return nil, fmt.Errorf("submitter returned an unknown flag: %s", f)
		}

		result := parts[1]

		switch result {
		case ResultOk, ResultError, ResultSkipped:
			result = strings.ToUpper(result)

		default:
			return nil, fmt.Errorf("submitter returned an invalid result: %s. Expected one of %v", result,
				[]string{ResultOk, ResultError, ResultSkipped})
		}

		message := strings.Join(parts[2:], " ")
		results = append(results, Result{Flag: f, Result: result, Message: message})
	}

	return results, nil
}

func (s *Submitter) SubmitLoop(ctx context.Context) {

	firstRun := true
	startTime := time.Now()

	for {
		// If this is not the first run, sleep for the submission period
		if !firstRun {
			elapsedTime := time.Since(startTime)
			sleepTime := time.Duration(s.submitPeriod)*time.Second - elapsedTime

			slog.Info("the submitter will now sleep", "duration", sleepTime.String())
			// Sleep for the submission period
			select {
			case <-ctx.Done():
				return
			case <-time.After(sleepTime):
			}

			startTime = time.Now()
		}
		firstRun = false

		// Get all flags from the database that are in the "queued" state
		flags, err := s.storage.GetByStatus(flag.StatusQueued, 0)
		if err != nil {
			slog.Error("error getting flags from database", "err", err)
			continue
		}

		// If there are no flags, sleep for a while and try again
		if len(flags) == 0 {
			slog.Debug("no flags to submit")
			continue
		}

		// try to submit the flags
		submitTime := time.Now()
		slog.Debug("starting external script", "submitter", s.path)
		results, err := s.Submit(ctx, flags)
		if err != nil {
			slog.Error("error submitting flags", "err", err)
			continue
		}

		submitElapsedTime := time.Since(submitTime)
		slog.Debug("script exited", "duration", submitElapsedTime.Seconds())

		// map result to status
		statuses := make([]string, len(results))
		for i, result := range results {
			switch result.Result {
			case ResultOk:
				statuses[i] = flag.StatusAccepted
			case ResultError:
				statuses[i] = flag.StatusRejected
			case ResultSkipped:
				statuses[i] = flag.StatusSkipped
			default:
				slog.Warn("Invalid result", "result", result.Result)
				statuses[i] = flag.StatusQueued
			}
		}

		// Update the status of the flags in the database
		for i := range results {
			err = s.storage.UpdateSentFlag(flags[i].Flag, statuses[i], results[i].Message, submitTime)
			if err != nil {
				slog.Error("error updating flag in database", "err", err)
			}
		}

		slog.Info("submitted flags", "count", len(flags))
	}
}
