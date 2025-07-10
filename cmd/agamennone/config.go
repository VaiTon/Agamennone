package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type FileConfig struct {
	GameName     string            `json:"gameName"`
	FlagRegexStr string            `json:"flagRegex"`
	FlagLifetime int               `json:"flagLifetime"`
	ServerHost   string            `json:"serverHost"`
	ServerPort   int               `json:"serverPort"`
	Teams        map[string]string `json:"teams"`
	DataSources  []string          `json:"dataSources"`
	AllowedURLs  []string          `json:"allowedURLs"`

	SubmitterPath       string `json:"submitterPath"`
	SubmissionPeriodStr string `json:"submissionPeriod"`
	SubmissionPeriod    time.Duration
}

func loadConfig(path string) (serverConfig *FileConfig, err error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error opening config file: %v", err)
	}

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&serverConfig)
	if err != nil {
		return nil, fmt.Errorf("error decoding config file: %v", err)
	}

	err = file.Close()
	if err != nil {
		return nil, fmt.Errorf("error closing config file: %v", err)
	}

	if serverConfig.SubmissionPeriodStr != "" {
		submissionPeriod, err := time.ParseDuration(serverConfig.SubmissionPeriodStr)
		if err != nil {
			return nil, fmt.Errorf("error parsing submission period: %v", err)
		}
		serverConfig.SubmissionPeriod = submissionPeriod
	}

	return
}
