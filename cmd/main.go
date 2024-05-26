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

	// // Prune repository
	// cmd2 := exec.Command("borg", "prune",
	// 	"--verbose",
	// 	"--list",
	// 	"--glob-archives", fmt.Sprintf("%s-*", os.Hostname()),
	// 	"--show-rc",
	// 	"--keep-daily", os.Getenv("DAILY"),
	// 	"--keep-weekly", os.Getenv("WEEKLY"),
	// 	"--keep-monthly", os.Getenv("MONTHLY"),
	// )
	// cmd2.Env = append(os.Environ(), fmt.Sprintf("BORG_REPO=%s", borgRepo), fmt.Sprintf("BORG_PASSPHRASE=%s", borgPassphrase))
	// err = cmd2.Run()
	// if err != nil {
	// 	fmt.Println("Error running prune command:", err)
	// }

	// // Compact repository
	// cmd3 := exec.Command("borg", "compact")
	// cmd3.Env = append(os.Environ(), fmt.Sprintf("BORG_REPO=%s", borgRepo), fmt.Sprintf("BORG_PASSPHRASE=%s", borgPassphrase))
	// err = cmd3.Run()
	// if err != nil {
	// 	fmt.Println("Error running compact command:", err)
	// }
}
