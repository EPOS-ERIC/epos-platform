package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/EPOS-ERIC/epos-platform/cli/cmd"
	"github.com/EPOS-ERIC/epos-platform/cli/config"
)

func init() {
	logFile, err := os.OpenFile(filepath.Join(config.GetDataPath(), "log.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		log.Fatalln("Failed to open log file:", err)
	}
	log.SetOutput(logFile)
}

func main() {
	cmd.Execute()
}
