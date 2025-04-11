package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"

	"github.com/VaiTon/Agamennone/pkg/agamennone"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

const header = `
	 █████╗  ██████╗  █████╗ ███╗   ███╗███████╗███╗   ██╗███╗   ██╗ ██████╗  ███╗   ██╗███████╗
	██╔══██╗██╔════╝ ██╔══██╗████╗ ████║██╔════╝████╗  ██║████╗  ██║██╔═══██╗ ████╗  ██║██╔════╝
	███████║██║  ███╗███████║██╔████╔██║█████╗  ██╔██╗ ██║██╔██╗ ██║██║   ██║ ██╔██╗ ██║█████╗
	██╔══██║██║   ██║██╔══██║██║╚██╔╝██║██╔══╝  ██║╚██╗██║██║╚██╗██║██║   ██║ ██║╚██╗██║██╔══╝
	██║  ██║╚██████╔╝██║  ██║██║ ╚═╝ ██║███████╗██║ ╚████║██║ ╚████║╚██████╔╝ ██║ ╚████║███████╗
	╚═╝  ╚═╝ ╚═════╝ ╚═╝  ╚═╝╚═╝     ╚═╝╚══════╝╚═╝  ╚═══╝╚═╝  ╚═══╝ ╚═════╝  ╚═╝  ╚═══╝╚══════╝
`

var (
	configPath string
	listenAddr string
	debug      bool
	dbConnStr  string
)

var rootCmd = &cobra.Command{
	Use:   "agamennone",
	Short: "Agamennone is a A/D CTF attack farm",
	Run:   Run,
}

func init() {
	rootCmd.Flags().StringVarP(&configPath, "config", "c", "config.json", "path to the configuration file")
	rootCmd.Flags().StringVarP(&listenAddr, "listen", "l", ":1234", "address to listen on")
	rootCmd.Flags().BoolVarP(&debug, "verbose", "v", false, "enable debug logging")
	rootCmd.Flags().StringVar(&dbConnStr, "db", "mysql://agamennone:agamennone@tcp(localhost:3306)/agamennone", "mariadb connection string")
}

func main() {
	println(header)
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("error executing command: %v", err)
	}
}

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
}

func Run(cmd *cobra.Command, args []string) {
	// load configuration
	log.Debug("loading configuration", "path", configPath)
	fileConfig, err := loadConfig(configPath)
	if err != nil {
		log.Fatalf("error loading configuration: %v", err)
	}

	if fileConfig.SubmitterPath == "" {
		log.Fatalf("error: submission protocol not set")
	} else if fileConfig.FlagLifetime == 0 {
		log.Fatalf("error: flag lifetime not set")
	} else if fileConfig.SubmissionPeriod == 0 {
		log.Fatalf("error: submission period not set")
	}

	flagRegex, err := regexp.Compile(fileConfig.FlagRegexStr)
	if err != nil {
		log.Fatalf("error compiling flag regex. check your config: %v", err)
	}

	config := &agamennone.AgamennoneConfig{
		ListenAddr:       listenAddr,
		Debug:            debug,
		DbConnectionStr:  dbConnStr,
		FlagRegex:        *flagRegex,
		GameName:         fileConfig.GameName,
		FlagRegexStr:     fileConfig.FlagRegexStr,
		SubmissionPeriod: fileConfig.SubmissionPeriod,
		FlagLifetime:     fileConfig.FlagLifetime,
		ServerHost:       fileConfig.ServerHost,
		ServerPort:       fileConfig.ServerPort,
		SubmitterPath:    fileConfig.SubmitterPath,
		Teams:            fileConfig.Teams,
		DataSources:      fileConfig.DataSources,
	}

	agamennone.Start(config)
}

func loadConfig(path string) (serverConfig *agamennone.ServerConfig, err error) {
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
