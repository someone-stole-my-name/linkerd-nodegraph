package main

import (
	"context"
	"encoding/json"
	"flag"
	"linkerd-nodegraph/internal/config"
	"linkerd-nodegraph/internal/graph/source/prometheus"
	"linkerd-nodegraph/internal/linkerd"
	"net/http"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/gorilla/schema"
)

var timeoutDataHandler = 60 * time.Second

func main() {
	var logFormatter log.JSONFormatter

	configFile := flag.String("config-file", "./config.yaml", "Config file")
	flag.Parse()

	config, err := config.FromFile(*configFile)
	if err != nil {
		panic(err)
	}

	timeoutDataHandler = config.Server.Timeout

	log.SetFormatter(&logFormatter)
	log.SetOutput(os.Stderr)
	log.SetLevel(log.InfoLevel)

	promConfig, err := config.Prometheus.Config()
	if err != nil {
		log.Fatal(err)
	}

	prom, err := prometheus.NewClient(*promConfig)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/api/graph/fields", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(linkerd.GraphSpec)
		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}
	})

	http.HandleFunc("/api/graph/data", data(linkerd.Stats{Server: prom}))

	err = http.ListenAndServe(config.Server.Addr, func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.WithFields(log.Fields{
				"address": r.RemoteAddr,
				"method":  r.Method,
				"url":     r.URL.Path,
				"query":   r.URL.Query(),
			}).Info("new request")
			handler.ServeHTTP(w, r)
		})
	}(http.DefaultServeMux))
	if err != nil {
		log.Fatal(err)
	}
}

func data(stats linkerd.Stats) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var params linkerd.Parameters

		decoder := schema.NewDecoder()

		err := decoder.Decode(&params, r.URL.Query())
		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), timeoutDataHandler)
		defer cancel()

		graph, err := stats.Graph(ctx, params)
		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		w.Header().Set("Content-Type", "application/json")

		err = json.NewEncoder(w).Encode(graph)
		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}
	}
}
