package main

import (
	"fmt"
	"log"
	"os"
	"time"
)

type logWriter struct{}

// Write enables us to format a logging prefix for the application. The
// text will appear before the log message output by the caller.
//
// e.g.
//
//	`// 2023-11-27 11:36:57 ERROR :: golang-app:100:main() :: this is an error message, ...some diagnosis`
func (lw *logWriter) Write(logString []byte) (int, error) {
	return fmt.Fprintf(os.Stderr, "%s :: %s :: %s",
		time.Now().UTC().Format(logTimeFormat),
		appname,
		string(logString),
	)
}

func init() {
	// Configure logging to use a custom log writer with Orcfax defaults.
	log.SetFlags(0 | log.Lshortfile | log.LUTC)
	log.SetOutput(new(logWriter))
}
