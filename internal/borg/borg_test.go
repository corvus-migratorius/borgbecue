package borg

import (
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
	err := connector.loadConfig("../../configs/config.yaml")
	if err != nil {
		t.Errorf("error loading config: %s", err)
	}

	actual := connector.Config

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("got: %+v, wanted: %+v", expected, actual)
	}
}

func TestBuildAccessString(t *testing.T) {
	expected := "ssh://test@1.2.3.4:22/backups/test"

	conn := Connector{
		Config: &config{
			User:       "test",
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
		t.Fatalf("got: %+v, wanted: %+v", expected, actual)
	}
}
