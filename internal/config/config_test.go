package config_test

import (
	"linkerd-nodegraph/internal/config"
	"reflect"
	"strings"
	"testing"
)

func TestConfig(t *testing.T) {
	yamlString := ``

	reader := strings.NewReader(yamlString)

	cnf, err := config.FromReader(reader)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(config.Default(), cnf) {
		t.Fatalf("expected %v, got %v", config.Default(), cnf)
	}
}
