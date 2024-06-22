package submitter

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/VaiTon/Agamennone/pkg/flag"
)

const (
	ResultOk      = "OK"
	ResultError   = "ERROR"
	ResultSkipped = "SKIPPED"
)

type SubmitterResult struct {
	Flag    string
	Result  string
	Message string
}

type Submitter struct {
	db   *sql.DB
	path string

	submitPeriod int
}

// NewSubmitter creates a new Submitter instance.
func NewSubmitter(path string, submitPeriod int, db *sql.DB) *Submitter {
	return &Submitter{db, path, submitPeriod}
}

// Submit sends a flag to the Agamennone server.
func (s *Submitter) Submit(flags []string) ([]SubmitterResult, error) {
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

	// execute the file, passing flags via stdin, \n is used as a delimiter
	cmd := exec.Command(s.path)
	cmd.Stdin = strings.NewReader(strings.Join(flags, "\n"))

	// capture and parse the output
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error capturing output: %v", err)
	}

	// check the result: for each line in the output, check if it's OK, ERROR or SKIPPED
	results := make([]SubmitterResult, 0, len(flags))

	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, " ")

		if len(parts) < 2 {
			return nil, fmt.Errorf("submitter returned an invalid line: %s", line)
		}

		flag := parts[0]
		if !slices.Contains(flags, flag) {
			return nil, fmt.Errorf("submitter returned an unknown flag: %s", flag)
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
		results = append(results, SubmitterResult{Flag: flag, Result: result, Message: message})
	}

	return results, nil
}

func (s *Submitter) SubmitLoop(ctx context.Context) error {
	log.Print("Starting submit loop")

	for {

		// Get all flags from the database that are in the "queued" state
		rows, err := s.db.Query("SELECT flag FROM flags WHERE status = ?", flag.FlagStatusQueued)
		if err != nil {
			return fmt.Errorf("error querying flags from database: %v", err)
		}

		// Parse the flags
		flags := make([]string, 0)
		for rows.Next() {
			var flag string
			err = rows.Scan(&flag)
			if err != nil {
				return fmt.Errorf("error scanning flags from database: %v", err)
			}
			flags = append(flags, flag)
		}

		// If there are no flags, sleep for a while and try again
		if len(flags) == 0 {
			log.Printf("No flags to submit. Sleeping for %d seconds", s.submitPeriod)
		} else {

			// try to submit the flags
			results, err := s.Submit(flags)
			if err != nil {
				return fmt.Errorf("error submitting flags: %v", err)
			}

			// map result to status
			statuses := make([]int, len(results))
			for i, result := range results {
				switch result.Result {
				case ResultOk:
					statuses[i] = flag.FlagStatusAccepted
				case ResultError:
					statuses[i] = flag.FlagStatusRejected
				case ResultSkipped:
					statuses[i] = flag.FlagStatusSkipped
				default:
					log.Printf("Invalid result: %s", result.Result)
					statuses[i] = flag.FlagStatusQueued
				}
			}

			// Update the status of the flags in the database
			for i, result := range results {
				_, err = s.db.Exec("UPDATE flags SET status = ? WHERE flag = ?", result.Result, flags[i])
				if err != nil {
					return fmt.Errorf("error updating flag status in database: %v", err)
				}
			}

			log.Printf("Submitted %d flags. Sleeping for %d seconds", len(flags), s.submitPeriod)
		}

		// Sleep for the submission period
		select {
		case <-time.After(time.Duration(s.submitPeriod) * time.Second):
		case <-ctx.Done():
			return nil
		}

	}
}
