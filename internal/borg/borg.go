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
}

type server struct {
	IP         string `yaml:"ip"`
	Port       int    `yaml:"port"`
	Repository string `yaml:"repository"`
}

type Connector struct {
	Config      *config
	Paths       []string
	Compression string
	// hostname    string
	AccessStr       string
	RepoInitialized bool
}

func NewConnector(cfgPath, compression string) (*Connector, error) {
	borgVer, err := checkBorg()
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

	// hostname, err := os.Hostname()
	// if err != nil {
	// 	return nil, fmt.Errorf("error getting client hostname: %s", err)
	// }
	// conn.hostname = hostname

	conn.Compression = compression
	conn.buildAccessString()
	log.Printf("built SSH access string")

	conn.loadManifest()
	log.Printf("loaded path manifest")

	err = conn.checkRepoInitialized()
	if err != nil {
		log.Fatalf("unexpected error while checking Borg repo: %s/%s: %s",
			conn.Config.Server.IP,
			conn.Config.Server.Repository,
			err,
		)
	}

	if !conn.RepoInitialized {
		log.Printf("borg repo not initalized: %s/%s", conn.Config.Server.IP, conn.Config.Server.Repository)
		err := conn.InitRepo()
		if err != nil {
			log.Fatalf("failed to initialize Borg repo %s/%s: %s",
				conn.Config.Server.IP,
				conn.Config.Server.Repository,
				err,
			)
		}

		log.Printf("successfully initialized new Borg repo: %s/%s", conn.Config.Server.IP, conn.Config.Server.Repository)
	} else {
		log.Printf("borg repo already initalized: '%s:%s'", conn.Config.Server.IP, conn.Config.Server.Repository)
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
	contents, err := os.ReadFile(c.Config.Manifest)
	if err != nil {
		return fmt.Errorf("error reading backups path manifest: %w", err)
	}
	c.Paths = strings.Split(string(contents), "\n")

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

	cmd := exec.Command("borg", args...)
	cmd.Env = append(
		os.Environ(),
		fmt.Sprintf("BORG_REPO=%s", c.AccessStr),
		fmt.Sprintf("BORG_PASSPHRASE=%s", c.Config.Passphrase),
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error: 'borg create' command failed (exit code %s): %s", err, stderr.String())
	}

	for line := range strings.Split(stdout.String(), "\n") {
		log.Println(line)
	}

	return nil
}

func (c *Connector) InitRepo() error {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(
		"borg", "init",
		"--encryption=keyfile",
		c.AccessStr,
	)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	cmd.Env = append(
		os.Environ(),
		fmt.Sprintf("BORG_PASSPHRASE=%s", c.Config.Passphrase),
	)

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error initializing borg repo: %w", err)
	}

	c.RepoInitialized = true
	return nil
}

// TODO: abstract away command running and check the command so that the func can be tested
func (c *Connector) checkRepoInitialized() error {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("borg", "info", c.AccessStr)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	cmd.Env = append(
		os.Environ(),
		fmt.Sprintf("BORG_PASSPHRASE=%s", c.Config.Passphrase),
	)

	err := cmd.Run()
	if err == nil {
		c.RepoInitialized = true
		return nil
	} else if err.Error() == "2" && err.Error() != "Failed to create/acquire the lock" {
		c.RepoInitialized = false
	} else {
		return fmt.Errorf("error: unexpected error while checking Borg repo initialization (code %w): %s", err, stderr.String())
	}

	return nil
}

func checkBorg() (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("borg", "--version")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("error: could not access borg executable (code %w): %s", err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}
