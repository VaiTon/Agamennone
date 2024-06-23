package storage

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/VaiTon/Agamennone/pkg/flag"
)

type SqliteStorage struct {
	db *sql.DB
}

func (s *SqliteStorage) Init() error {
	const tableInitQuery = `
		CREATE TABLE IF NOT EXISTS flags (
			flag 					TEXT PRIMARY KEY, 
			sploit 					TEXT, 
			team 					TEXT, 
			received_time 			DATETIME, 
			sent_time 				DATETIME,
			status 					TEXT, 
			checksystem_response 	TEXT
		                                 
	 	)
	`

	_, err := s.db.Exec(tableInitQuery)
	if err != nil {
		return fmt.Errorf("cannot create flags table: %w", err)
	}

	return nil
}

func (s *SqliteStorage) GetStatisticsV1() (StatisticsV1, error) {
	// Get the number of flags in the database
	var flags, queuedFlags, acceptedFlags, rejectedFlags, skippedFlags int
	err := s.db.QueryRow("SELECT COUNT(*) FROM flags").Scan(&flags)
	if err != nil {
		return StatisticsV1{}, fmt.Errorf("cannot query flags from database: %w", err)
	}

	// Get the number of flags in each state
	err = s.db.QueryRow("SELECT COUNT(*) FROM flags WHERE status = ?", flag.StatusQueued).Scan(&queuedFlags)
	if err != nil {
		return StatisticsV1{}, fmt.Errorf("cannot query queued flags from database: %w", err)
	}

	err = s.db.QueryRow("SELECT COUNT(*) FROM flags WHERE status = ?", flag.StatusAccepted).Scan(&acceptedFlags)
	if err != nil {
		return StatisticsV1{}, fmt.Errorf("cannot query accepted flags from database: %w", err)
	}

	err = s.db.QueryRow("SELECT COUNT(*) FROM flags WHERE status = ?", flag.StatusRejected).Scan(&rejectedFlags)
	if err != nil {
		return StatisticsV1{}, fmt.Errorf("cannot query rejected flags from database: %w", err)
	}

	err = s.db.QueryRow("SELECT COUNT(*) FROM flags WHERE status = ?", flag.StatusSkipped).Scan(&skippedFlags)
	if err != nil {
		return StatisticsV1{}, fmt.Errorf("cannot query skipped flags from database: %w", err)
	}

	return StatisticsV1{
		TotalFlags: flags,
		TotalFlagsByStatus: map[string]int{
			flag.StatusQueued:   queuedFlags,
			flag.StatusAccepted: acceptedFlags,
			flag.StatusRejected: rejectedFlags,
		},
	}, nil
}

func rowsToFlags(row *sql.Rows) ([]flag.Flag, error) {
	flags := make([]flag.Flag, 0)
	for row.Next() {
		var f flag.Flag
		err := row.Scan(&f.Flag, &f.Exploit, &f.Team, &f.ReceivedTime, &f.SentTime, &f.Status, &f.CheckSystemResponse)
		if err != nil {
			return nil, fmt.Errorf("cannot scan flag: %w", err)
		}
		flags = append(flags, f)
	}

	return flags, nil
}

func (s *SqliteStorage) GetByStatus(status string, limit int) ([]flag.Flag, error) {
	query := `SELECT * FROM flags WHERE status = ? ORDER BY received_time DESC`

	var (
		rows *sql.Rows
		err  error
	)
	if limit > 0 {
		query += " LIMIT ?"
		rows, err = s.db.Query(query, status, limit)
	} else {
		rows, err = s.db.Query(query, status)
	}

	if err != nil {
		return nil, fmt.Errorf("cannot get flags by status: %w", err)
	}

	flags, err := rowsToFlags(rows)
	if err != nil {
		return nil, fmt.Errorf("cannot convert rows to flags: %w", err)
	}

	err = rows.Close()
	if err != nil {
		return nil, fmt.Errorf("cannot close rows: %w", err)
	}

	return flags, nil
}

func NewSqliteStorage(path string) (*SqliteStorage, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("cannot open sqlite file: %w", err)
	}

	return &SqliteStorage{db}, nil
}

func (s *SqliteStorage) GetLastFlags(limit int) ([]flag.Flag, error) {
	const query = `SELECT * FROM flags ORDER BY received_time DESC LIMIT ?`

	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("cannot get last flags: %w", err)
	}

	flags, err := rowsToFlags(rows)
	if err != nil {
		return nil, fmt.Errorf("cannot convert rows to flags: %w", err)
	}

	err = rows.Close()
	if err != nil {
		return nil, fmt.Errorf("cannot close rows: %w", err)
	}

	return flags, nil
}

func (s *SqliteStorage) Close() error { return s.db.Close() }

func (s *SqliteStorage) UpdateSentFlag(flag, status, checkSystemResponse string, sentTime time.Time) error {
	const query = `UPDATE flags SET status = ?, checksystem_response = ?, sent_time = ? WHERE flag = ?`

	_, err := s.db.Exec(query, status, checkSystemResponse, sentTime, flag)
	if err != nil {
		return fmt.Errorf("cannot update flag status: %w", err)
	}

	return nil
}

func (s *SqliteStorage) InsertFlags(flags []flag.Flag) (int64, error) {
	queryString := `
		INSERT OR IGNORE INTO flags (flag, sploit, team, received_time, sent_time, status, checksystem_response) VALUES 
	`
	var values []interface{}

	// Insert flags into the database
	for i, f := range flags {
		if i > 0 {
			queryString += ", "
		}
		queryString += "(?, ?, ?, ?, '', ?, '')"
		values = append(values, f.Flag, f.Exploit, f.Team, f.ReceivedTime, flag.StatusQueued)
	}

	res, err := s.db.Exec(queryString, values...)
	if err != nil {
		return 0, fmt.Errorf("cannot insert flags into database: %w", err)
	}

	insertedFlags, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("cannot get number of inserted flags: %w", err)
	}

	return insertedFlags, nil
}

func (s *SqliteStorage) GetStatsByExploit() ([]StatsByExploit, error) {
	const query = `
		SELECT sploit, status,
       	strftime('%H:%M', received_time) as hour,
       	COUNT(*)                         as count
		FROM flags
		WHERE status != 'queued'
		GROUP BY hour, sploit, status
		ORDER BY hour;
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("cannot get stats by exploit: %w", err)
	}

	stats := make([]StatsByExploit, 0)
	for rows.Next() {
		var exploit, status, hourStr string
		var count int
		err := rows.Scan(&exploit, &status, &hourStr, &count)
		if err != nil {
			return nil, fmt.Errorf("cannot scan stats by exploit: %w", err)
		}

		hour, err := time.Parse("15:04", hourStr)
		if err != nil {
			return nil, fmt.Errorf("cannot parse hour: %w", err)
		}

		stats = append(stats, StatsByExploit{
			Hour:    hour,
			Exploit: exploit,
			Status:  status,
			Count:   count,
		})
	}
	if len(stats) == 0 {
		return stats, nil
	}

	return stats, nil
}
