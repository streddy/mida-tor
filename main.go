package main

import (
	"github.com/pmurley/mida/log"
	_ "net/http/pprof"
)

// Sets up logging and hands off control to command.go, which is responsible
// for parsing args/flags and initiating the appropriate functionality
func main() {
	InitConfig()
	log.InitLogger()

	log.Log.Info("MIDA Starting")

	rootCmd := BuildCommands()
	err := rootCmd.Execute()
	if err != nil {
		log.Log.Debug(err)
	}

	log.Log.Info("MIDA exiting")
}
