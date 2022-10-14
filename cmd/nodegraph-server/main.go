package main

import (
	"context"
	"encoding/json"
	"flag"
	"linkerd-nodegraph/internal/linkerd"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/schema"
)

var decoder = schema.NewDecoder()

func main() {
	prometheusAddr := flag.String("prometheus-addr", "http://prometheus.default", "Address of the Prometheus server")
	listenAddr := flag.String("listen-addr", ":5001", "Host/port to listen on")
	flag.Parse()

	prom, err := linkerd.NewPromGraphSource(*prometheusAddr)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	http.HandleFunc("/api/graph/fields", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(linkerd.GraphSpec)
	})
	http.HandleFunc("/api/graph/data", data(linkerd.Stats{Server: prom}))

	err = http.ListenAndServe(*listenAddr, func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("%s %s %s\n", r.RemoteAddr, r.Method, r.URL)
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

		err := decoder.Decode(&params, r.URL.Query())
		if err != nil {
			log.Printf("%s", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		graph, err := stats.Graph(ctx, params)
		if err != nil {
			log.Printf("%s", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(graph)
	}
}
