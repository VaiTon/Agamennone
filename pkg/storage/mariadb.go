package storage

import (
	"database/sql"
	"fmt"
	"time"

	charmlog "github.com/charmbracelet/log"
	_ "github.com/go-sql-driver/mysql"

	"github.com/VaiTon/Agamennone/pkg/flag"
)

var log = charmlog.WithPrefix("mariadb")

type MariaDBStorage struct {
	db *sql.DB
}

func NewMariaDBStorage(addr string) (*MariaDBStorage, error) {
	db, err := sql.Open("mysql", addr+"?parseTime=true")
	if err != nil {
		return nil, fmt.Errorf("cannot connect to the database: %w", err)
	}

	return &MariaDBStorage{db}, nil
}

func (s *MariaDBStorage) Init() error {
	const tableInitQuery = `
		CREATE TABLE IF NOT EXISTS flags (
			flag 					varchar(100) PRIMARY KEY,
			sploit 					varchar(100),
			team 					varchar(100),
			received_time 			DATETIME,
			sent_time 				DATETIME,
			status 					varchar(100),
			checksystem_response 	varchar(255)

	 	)
	`

	_, err := s.db.Exec(tableInitQuery)
	if err != nil {
		return fmt.Errorf("cannot create flags table: %w", err)
	}

	return nil
}

func (s *MariaDBStorage) GetStatisticsV1() (StatisticsV1, error) {
	// Get the number of flags in the database
	var flags, queuedFlags, acceptedFlags, rejectedFlags, skippedFlags int
	err := s.db.QueryRow("SELECT COUNT(*) FROM agamennone.flags").Scan(&flags)
	if err != nil {
		return StatisticsV1{}, fmt.Errorf("cannot query flags from database: %w", err)
	}

	// Get the number of flags in each state
	err = s.db.QueryRow("SELECT COUNT(*) FROM agamennone.flags WHERE status = ?", flag.StatusQueued).Scan(&queuedFlags)
	if err != nil {
		return StatisticsV1{}, fmt.Errorf("cannot query queued flags from database: %w", err)
	}

	err = s.db.QueryRow("SELECT COUNT(*) FROM agamennone.flags WHERE status = ?", flag.StatusAccepted).Scan(&acceptedFlags)
	if err != nil {
		return StatisticsV1{}, fmt.Errorf("cannot query accepted flags from database: %w", err)
	}

	err = s.db.QueryRow("SELECT COUNT(*) FROM agamennone.flags WHERE status = ?", flag.StatusRejected).Scan(&rejectedFlags)
	if err != nil {
		return StatisticsV1{}, fmt.Errorf("cannot query rejected flags from database: %w", err)
	}

	err = s.db.QueryRow("SELECT COUNT(*) FROM agamennone.flags WHERE status = ?", flag.StatusSkipped).Scan(&skippedFlags)
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

func (s *MariaDBStorage) GetByStatus(status string, limit int) ([]flag.Flag, error) {
	query := `SELECT * FROM agamennone.flags WHERE status = ? ORDER BY received_time DESC`

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

func (s *MariaDBStorage) GetLastFlags(limit int) ([]flag.Flag, error) {
	const query = `SELECT * FROM agamennone.flags ORDER BY received_time DESC LIMIT ?`

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

func (s *MariaDBStorage) Close() error { return s.db.Close() }

func (s *MariaDBStorage) UpdateSentFlag(flag, status, checkSystemResponse string, sentTime time.Time) error {
	const query = `UPDATE agamennone.flags SET status = ?, checksystem_response = ?, sent_time = ? WHERE flag = ?`

	_, err := s.db.Exec(query, status, checkSystemResponse, sentTime, flag)
	if err != nil {
		return fmt.Errorf("cannot update flag status: %w", err)
	}

	return nil
}

func (s *MariaDBStorage) InsertFlags(flags []flag.Flag) (int64, error) {
	queryString := `
		INSERT IGNORE INTO agamennone.flags (flag, sploit, team, received_time, sent_time, status, checksystem_response) VALUES
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
