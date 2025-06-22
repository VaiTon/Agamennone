package main

import (
	"os"
	"path/filepath"

	"log/slog"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new exploit",
	Args:  cobra.ExactArgs(1),
	Run:   runCreate,
}

var (
	force bool
)

func init() {
	createCmd.Flags().BoolVarP(&force, "force", "f", false, "Force overwrite existing exploit")
	rootCmd.AddCommand(createCmd)
}

const template = `#!/usr/bin/env python3
import sys
from pwn import *


def main(team, data):
    # Set up the connection to the server
    if team == "local":
        p = process("./server")
    else:
        p = remote("localhost", 1337)

    # Send the data to the server
    for item in data:
        p.sendline(item)

    # Receive and print the response from the server
    response = p.recvall()
    print(response.decode())

    # Close the connection
    p.close()


if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python3 exploit.py <team> <data>")
        sys.exit(1)

    team = sys.argv[1]
    data = []

    if len(sys.argv) > 2:
        datapaths = sys.argv[2:]

        for datapath in datapaths:
            with open(datapath, "r") as f:
                data.append(f.read())

    main(team, data)
`

func runCreate(cmd *cobra.Command, args []string) {
	name := args[0]
	slog.Info("Creating exploit", "path", name)

	stat, err := os.Stat(name)
	if !force && err == nil && stat.IsDir() || os.IsExist(err) {
		slog.Error("File exists. Run with --force to overwrite", "path", name)
		os.Exit(1)
	}

	if err := os.MkdirAll(name, 0755); err != nil {
		slog.Error("Failed to create directory", "path", name, "err", err)
		os.Exit(1)
	}

	exploitFile := filepath.Join(name, "exploit.py")
	if err := os.WriteFile(exploitFile, []byte(template), 0755); err != nil {
		slog.Error("Failed to create exploit file", "path", exploitFile, "err", err)
		os.Exit(1)
	}

	slog.Info("Exploit created", "path", exploitFile)

}
