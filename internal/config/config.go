package config

import (
	"fmt"
	"io"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

type (
	GraphSource string
	LogLevel    string
)

const (
	PrometheusGraphSource GraphSource = "prometheus"

	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
)

type TLSConfig struct {
	InsecureSkipVerify bool `yaml:"insecureSkipVerify"`
}

type HTTP struct {
	Addr      string            `yaml:"addr"`
	TLSConfig TLSConfig         `yaml:"tlsConfig"`
	Headers   map[string]string `yaml:"headers"`
}

type Prometheus struct {
	HTTP   HTTP   `yaml:"http"`
	Labels string `yaml:"labels"`
}

type Config struct {
	Server      Server      `yaml:"server"`
	GraphSource GraphSource `yaml:"graphSource"`
	LogLevel    LogLevel    `yaml:"logLevel"`
	Prometheus  Prometheus  `yaml:"prometheus"`
}

type Server struct {
	Timeout time.Duration `yaml:"timeout"`
	Addr    string        `yaml:"addr"`
}

func Default() *Config {
	return &Config{
		LogLevel:    LogLevelInfo,
		GraphSource: PrometheusGraphSource,
		Server: Server{
			Timeout: time.Minute,
			Addr:    ":5001",
		},
		Prometheus: Prometheus{
			HTTP: HTTP{
				Addr:    "http://localhost:9090",
				Headers: map[string]string{},
				TLSConfig: TLSConfig{
					InsecureSkipVerify: false,
				},
			},
			Labels: "",
		},
	}
}

func FromReader(r io.Reader) (*Config, error) {
	config := Default()

	yamlFile, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	err = yaml.Unmarshal(yamlFile, config)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling config file: %w", err)
	}

	return config, nil
}

func FromFile(path string) (*Config, error) {
	reader, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	return FromReader(reader)
}
