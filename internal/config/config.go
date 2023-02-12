package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"linkerd-nodegraph/internal/graph/source/prometheus"
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
	InsecureSkipVerify bool   `yaml:"insecureSkipVerify"`
	CAFile             string `yaml:"caFile"`
	CertFile           string `yaml:"certFile"`
	KeyFile            string `yaml:"keyFile"`
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

func (c *Prometheus) Config() (*prometheus.Config, error) {
	tlsConfig := tls.Config{
		InsecureSkipVerify: c.HTTP.TLSConfig.InsecureSkipVerify,
	}

	if c.HTTP.TLSConfig.CAFile != "" {
		caBytes, err := os.ReadFile(c.HTTP.TLSConfig.CAFile)
		if err != nil {
			return nil, fmt.Errorf("could not open ca file: %w", err)
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caBytes)
		tlsConfig.RootCAs = caCertPool
	}

	if c.HTTP.TLSConfig.CertFile != "" && c.HTTP.TLSConfig.KeyFile != "" {
		certificate, err := tls.LoadX509KeyPair(c.HTTP.TLSConfig.CertFile, c.HTTP.TLSConfig.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("could not load certificate: %w", err)
		}

		tlsConfig.Certificates = []tls.Certificate{certificate}
	}

	return &prometheus.Config{
		Address:   c.HTTP.Addr,
		Labels:    c.Labels,
		Headers:   c.HTTP.Headers,
		TLSConfig: &tlsConfig,
	}, nil
}
