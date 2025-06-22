package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type FileConfig struct {
	GameName         string            `json:"gameName"`
	FlagRegexStr     string            `json:"flagRegex"`
	SubmissionPeriod int               `json:"submissionPeriod"`
	FlagLifetime     int               `json:"flagLifetime"`
	ServerHost       string            `json:"serverHost"`
	ServerPort       int               `json:"serverPort"`
	Teams            map[string]string `json:"teams"`
	SubmitterPath    string            `json:"submitterPath"`
	DataSources      []string          `json:"dataSources"`
	AllowedURLs      []string          `json:"allowedURLs"`
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

	return
}
