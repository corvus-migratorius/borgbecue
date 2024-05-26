package borg

import (
	"fmt"
	"os"
	"reflect"
	"testing"
)

func TestConfigLoad(t *testing.T) {
	expected := &config{
		User:       "test",
		Passphrase: "secret",
		Manifest:   "/path/to/versioned-daily.fofn",
		Server: server{
			IP:         "1.2.3.4",
			Port:       22,
			Repository: "/backups/test",
		},
	}

	connector := Connector{}
	err := connector.loadConfig("../../configs/borgbecue.yaml")
	if err != nil {
		t.Errorf("error loading config: %s", err)
	}

	actual := connector.Config

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("got: %+v, wanted: %+v", actual, expected)
	}
}

func TestBuildAccessString(t *testing.T) {
	expected := "ssh://test@1.2.3.4:22/backups/test"

	conn := Connector{
		Config: &config{
			User: "test",
			Server: server{
				IP:         "1.2.3.4",
				Port:       22,
				Repository: "/backups/test",
			},
		},
	}

	conn.buildAccessString()
	actual := conn.AccessStr

	if actual != expected {
		t.Fatalf("got: %+v, wanted: %+v", actual, expected)
	}
}

func TestLoadManifest(t *testing.T) {
	expected := []string{
		"/tmp/123.txt",
		"/home/zeleboba/smth.dat",
	}

	dir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Errorf("failed to create a temporary dir: %s", err)
	}
	defer os.RemoveAll(dir)

	file, err := os.CreateTemp(dir, "")
	if err != nil {
		t.Errorf("failed to create a temporary file: %s", err)
	}
	defer file.Close()

	fmt.Println(file.Name())

	for _, path := range expected {
		_, err := file.WriteString(fmt.Sprintf("%s\n", path))
		if err != nil {
			t.Errorf("failed to write to a temporary file: %s", err)
		}
	}

	conn := Connector{
		Config: &config{
			Manifest: file.Name(),
		},
	}
	conn.loadManifest()

	actual := conn.Paths

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("got: %+v, wanted: %+v", actual, expected)
	}
}
