/*
 * connector is a module implementing the Connector type which wraps several
 * Borg subcommands like `borg init`, `info`, `create`, `prune` and `compact`
 */

package borg

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"gopkg.in/yaml.v3"
)

type config struct {
	User       string `yaml:"user"`
	Passphrase string `yaml:"passphrase"`
	Manifest   string `yaml:"manifest"`
	Server     server `yaml:"server"`
	Cron       cron   `yaml:"cron"`
}

type server struct {
	IP         string `yaml:"ip"`
	Port       int    `yaml:"port"`
	Repository string `yaml:"repository"`
}

type cron struct {
	Daily   int `yaml:"daily"`
	Weekly  int `yaml:"weekly"`
	Monthly int `yaml:"monthly"`
}

// Connector implements methods wrapping various Borg subcommands
type Connector struct {
	Config          *config
	Paths           []string
	Compression     string
	AccessStr       string
	RepoInitialized bool
	Env             []string
}

// NewConnector is a constructor function returning pointers to new Connector instances
func NewConnector(cfgPath, compression string) (*Connector, error) {
	borgVer, err := checkLocalBorg()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("found local Borg executable (%s)", borgVer)

	conn := Connector{}

	err = conn.loadConfig(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("error reading configuration file: %w", err)
	}
	log.Printf("parsed configuration file: '%s'", cfgPath)

	conn.Compression = compression
	conn.buildAccessString()
	log.Printf("built SSH access string")

	conn.Env = append(
		os.Environ(),
		fmt.Sprintf("BORG_REPO=%s", conn.AccessStr),
		fmt.Sprintf("BORG_PASSPHRASE=%s", conn.Config.Passphrase),
	)

	err = conn.loadManifest()
	if err != nil {
		return nil, fmt.Errorf("error reading manifest file: %w", err)
	}
	log.Printf("loaded path manifest (%d paths): '%s'", len(conn.Paths), conn.Config.Manifest)

	err = conn.checkRepoInitialized()
	if err != nil {
		log.Fatal(err)
	}

	if !conn.RepoInitialized {
		log.Printf("Borg repo not initalized: '%s:%s'", conn.Config.Server.IP, conn.Config.Server.Repository)
		err := conn.InitRepo()
		if err != nil {
			log.Fatalf("failed to initialize Borg repo '%s:%s': %s",
				conn.Config.Server.IP,
				conn.Config.Server.Repository,
				err,
			)
		}

		log.Printf("successfully initialized new Borg repo: '%s:%s'", conn.Config.Server.IP, conn.Config.Server.Repository)
	} else {
		log.Printf("Borg repo already initalized: '%s:%s'", conn.Config.Server.IP, conn.Config.Server.Repository)
	}

	return &conn, nil
}

func (c *Connector) loadConfig(path string) error {
	config := config{}
	file, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read the configuration file: %w", err)
	}

	err = yaml.Unmarshal(file, &config)
	if err != nil {
		return fmt.Errorf("failed to unmarshal the configuration file as YAML: %w", err)
	}

	c.Config = &config

	return nil
}

func (c *Connector) loadManifest() error {
	byteContent, err := os.ReadFile(c.Config.Manifest)
	if err != nil {
		return fmt.Errorf("error reading backups path manifest: %w", err)
	}

	strContent := strings.TrimSpace(string(byteContent))
	c.Paths = strings.Split(strContent, "\n")

	return nil
}

func (c *Connector) buildAccessString() {
	c.AccessStr = fmt.Sprintf(
		"ssh://%s@%s:%d/%s",
		c.Config.User,
		c.Config.Server.IP,
		c.Config.Server.Port,
		strings.TrimLeft(c.Config.Server.Repository, "/"),
	)
}

// BackUp runs `borg create` to create a Borg archive from paths in the manifest
func (c *Connector) BackUp() error {
	base := []string{
		"create",
		"--verbose",
		"--filter", "AMCE",
		"--list",
		"--stats",
		"--show-rc",
		"--compression", c.Compression,
		"--exclude-caches",
		"--exclude", "*/.cache/*",
		"::{hostname}-{now}",
	}
	args := append(base, c.Paths...)

	stderr, err := c.runCommand("borg", args)

	// borg sends all of its outputs to stderr, even non-error messages
	for _, line := range strings.Split(stderr, "\n") {
		if line != "" {
			log.Printf("borg create: %s", line)
		}
	}
	if err != nil {
		return fmt.Errorf("error: 'borg create' command failed (code %s): %s", err, stderr)
	}

	return nil
}

// InitRepo runs `borg init` command to initialize a Borg repo at the target machine
func (c *Connector) InitRepo() error {
	stderr, err := c.runCommand("borg", []string{"init", "--encryption=keyfile"})
	if err != nil {
		return fmt.Errorf("error initializing Borg repo (%w): %s", err, stderr)
	}

	c.RepoInitialized = true
	return nil
}

// Prune runs `borg prune` to identify archives in need of pruning according to the keep rules
func (c *Connector) Prune() error {
	args := []string{
		"prune",
		"--verbose",
		"--list",
		"--glob-archives", "{hostname}-*",
		"--show-rc",
		"--keep-daily", fmt.Sprintf("%d", c.Config.Cron.Daily),
		"--keep-weekly", fmt.Sprintf("%d", c.Config.Cron.Weekly),
		"--keep-monthly", fmt.Sprintf("%d", c.Config.Cron.Monthly),
	}
	stderr, err := c.runCommand("borg", args)

	// borg sends all of its outputs to stderr, even non-error messages
	for _, line := range strings.Split(stderr, "\n") {
		if line != "" {
			log.Printf("borg prune: %s", line)
		}
	}
	if err == nil {
		return nil
	} else if err.Error() == "exit code 1" {
		log.Printf("warnings while runnin `borg prune` (%s)", err)
	} else {
		return fmt.Errorf("unexpected error while pruning Borg repo (%s)", err)
	}

	return nil
}

// Compact runs `borg compact` to apply changes prepared by `borg prune`
func (c *Connector) Compact() error {
	stderr, err := c.runCommand("borg", []string{"compact"})

	// borg sends all of its outputs to stderr, even non-error messages
	for _, line := range strings.Split(stderr, "\n") {
		if line != "" {
			log.Printf("borg compact: %s", line)
		}
	}
	if err == nil {
		return nil
	} else if err.Error() == "exit code 1" {
		log.Printf("warnings while runnin `borg compect` (%s)", err)
	} else {
		return fmt.Errorf("unexpected error while compacting Borg repo (%s)", err)
	}

	return nil
}

// TODO: abstract away command running and check the command so that the func can be tested
func (c *Connector) checkRepoInitialized() error {
	notExistsStr := fmt.Sprintf("Repository %s does not exist.\n", c.AccessStr)
	stderr, err := c.runCommand("borg", []string{"info"})

	if err == nil {
		c.RepoInitialized = true
		return nil
	} else if stderr == notExistsStr {
		c.RepoInitialized = false
	} else {
		return fmt.Errorf("unexpected error while initializing Borg repo (%w): %s", err, stderr)
	}

	return nil
}

func (c *Connector) runCommand(name string, args []string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(name, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	cmd.Env = c.Env

	code := cmd.Run()
	return stderr.String(), code
}

func checkLocalBorg() (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("borg", "--version")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("error: could not access local Borg executable (code %w): %s", err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}
