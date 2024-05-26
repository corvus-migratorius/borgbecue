package main

import (
	"fmt"
	"log"
	"os"

	"borgbecue/internal/borg"

	"github.com/akamensky/argparse"
)

var version = "development"

type appArgs struct {
	pathConfig  string
	compression string
}

func parseArgs() appArgs {
	parser := argparse.NewParser(fmt.Sprintf("borgbecue %s", version), "A simple tool wrapping calls to Borg")
	pathConfig := parser.String(
		"c", "config",
		&argparse.Options{Required: false, Help: "Path to the YAML configuration file", Default: "borgbecue.yaml"},
	)
	compression := parser.String(
		"z", "compression",
		&argparse.Options{Required: false, Help: "compression type", Default: "lz4"},
	)

	showVersion := parser.Flag(
		"V", "version",
		&argparse.Options{Required: false, Help: "Display program version and exit", Default: false},
	)

	// TODO: add an option for setting the file manifest path

	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
		os.Exit(1)
	}

	if *showVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	return appArgs{
		pathConfig:  *pathConfig,
		compression: *compression,
	}
}

func main() {
	args := parseArgs()

	connector, err := borg.NewConnector(args.pathConfig, args.compression)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("creating new archive")
	err = connector.BackUp()
	if err != nil {
		log.Fatalf("error creating new archive: %s", err)
	}

	log.Println("pruning existing archives")
	err = connector.Prune()
	if err != nil {
		log.Fatalf("error pruning archives: %s", err)
	}

	log.Println("compacting existing archives")
	err = connector.Compact()
	if err != nil {
		log.Fatalf("error compacting archives: %s", err)
	}

	log.Println("job complete")
}
