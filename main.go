package main

import (
	"log"
	"os"
	"strings"

	"github.com/vitaminniy/go-cron/cron"
)

func main() {
	log.SetFlags(0)
	log.SetOutput(os.Stderr)

	args := os.Args[1:]
	if len(args) == 0 {
		log.Fatal("no input provided")
	}

	arg := strings.TrimSpace(args[0])
	arg = strings.Trim(arg, "\"")

	expr, err := cron.ParseExpression(arg)
	if err != nil {
		log.Fatalf("could not parse cron expression: %v", err)
	}

	if err := expr.DumpFormatted(os.Stdout); err != nil {
		log.Fatalf("could not dump cron expression: %v", err)
	}
}
