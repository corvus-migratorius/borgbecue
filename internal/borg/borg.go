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
	User       string  `yaml:"user"`
	Passphrase string  `yaml:"passphrase"`
	Manifest   string  `yaml:"manifest"`
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
	log.Println("creating a new Borg connector")

	log.Println("checking that Borg executable is accessible")
	err := checkBorg()
	if err != nil {
		log.Fatal(err)
	}

	connector := Connector{}

	log.Printf("parsing configuration file: '%s'", cfgPath)
	err = connector.loadConfig(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("error reading configuration file: %w", err)
	}

	// hostname, err := os.Hostname()
	// if err != nil {
	// 	return nil, fmt.Errorf("error getting client hostname: %s", err)
	// }
	// connector.hostname = hostname

	connector.Compression = compression
	log.Printf("building SSH access string")
	connector.buildAccessString()

	log.Printf("loading path manifest")
	connector.loadManifest()

	log.Printf("checking if Borg repo is initialized already")
	connector.checkRepoInitialization()
	if connector.RepoInitialized == false {
		connector.InitRepo()
	}

	return &connector, nil
}

func (c *Connector) loadConfig(path string) error {
	config := config{}
	file, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read the configuration file: %w", err)
	}

	log.Println("unmarshalling YAML")
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
		// fmt.Sprintf("BORG_REPO=%s", c.AccessStr),
		fmt.Sprintf("BORG_PASSPHRASE=%s", c.Config.Passphrase),
	)

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error initializing borg repo: %w", err)
	}

	log.Printf("successfully initialized new Borg repo: %s/%s", c.Config.Server.IP, c.Config.Server.Repository)
	c.RepoInitialized = true
	return nil
}

// TODO: abstract away command running and check the command so that the func can be tested
func (c *Connector) checkRepoInitialization() error {
	var stdout, stderr bytes.Buffer
	log.Printf("init repo: '%s'", c.AccessStr)
	cmd := exec.Command("borg", "info", c.AccessStr)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		c.RepoInitialized = true
		return nil
	} else if err.Error() == "2" && err.Error() != "Failed to create/acquire the lock" {
		log.Printf("borg repo not initalized: %s/%s", c.Config.Server.IP, c.Config.Server.Repository)
		c.RepoInitialized = false
	} else {
		return fmt.Errorf("error: unexpected error while checking Borg repo initialization (code %w): %s", err, stderr.String())
	}

	return nil
}

func checkBorg() error {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("borg", "--version")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error: could not access borg executable (code %w): %s", err, stderr.String())
	}

	return nil
}
