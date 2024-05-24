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
	user       string  `yaml:"user"`
	passphrase string  `yaml:"passphrase"`
	manifest   string  `yaml:"manifest"`
	server     *server `yaml:"server"`
}

type server struct {
	ip         string `yaml:"ip"`
	port       int    `yaml:"port"`
	repository string `yaml:"repository"`
}

type Connector struct {
	config      *config
	paths       []string
	compression string
	// hostname    string
	accessStr string
	repoInitialized bool
}

func NewConnector(cfgPath, compression string) (*Connector, error) {
	err := checkBorg()
	if err != nil {
		log.Fatal(err)
	}

	connector := Connector{}
	err = connector.loadConfig(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("error reading configuration file: %w", err)
	}

	// hostname, err := os.Hostname()
	// if err != nil {
	// 	return nil, fmt.Errorf("error getting client hostname: %s", err)
	// }
	// connector.hostname = hostname

	connector.compression = compression
	connector.buildAccessString()
	connector.loadManifest()

	connector.checkRepoInitialization()

	return &connector, nil
}

func (c *Connector) loadConfig(path string) error {
	config := config{}
	log.Printf("parsing configuration file: '%s'", path)
	file, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read the configuration file: %w", err)
	}

	err = yaml.Unmarshal(file, &config)
	if err != nil {
		return fmt.Errorf("failed to unmarshal the configuration file as YAML: %w", err)
	}

	c.config = &config

	return nil
}

func (c *Connector) loadManifest() error {
	contents, err := os.ReadFile(c.config.manifest)
	if err != nil {
		return fmt.Errorf("error reading backups path manifest: %w", err)
	}
	c.paths = strings.Split(string(contents), "\n")

	return nil
}

func (c *Connector) buildAccessString() {
	c.accessStr = fmt.Sprintf(
		"ssh://%s@%s:%d/%s",
		c.config.user,
		c.config.server.ip,
		c.config.server.port,
		strings.TrimLeft(c.config.server.repository, "/"),
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
		"--compression", c.compression,
		"--exclude-caches",
		"--exclude", "*/.cache/*",
		"::{hostname}-{now}",
	}
	args := append(base, c.paths...)

	cmd := exec.Command("borg", args...)
	cmd.Env = append(
		os.Environ(),
		fmt.Sprintf("BORG_REPO=%s", c.accessStr),
		fmt.Sprintf("BORG_PASSPHRASE=%s", c.config.passphrase),
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

func (c *Connector) checkRepoInitialization() error {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("borg", "info", c.accessStr)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		c.repoInitialized = true
		return nil
	} else if err.Error() == "2" && err.Error() != "Failed to create/acquire the lock" {
		log.Printf("borg repo not initalized: %s/%s", c.config.server.ip, c.config.server.repository)
		c.repoInitialized = false
	} else {
		return fmt.Errorf("error: unexpected error while checking Borg repo initialization (code %w): %s", err, stderr.String())
	}

	return nil
}

func checkBorg () error {
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