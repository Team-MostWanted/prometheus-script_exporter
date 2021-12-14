package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

func init() {
	flags.verbose = flag.Bool("v", false, "show verbose output")
	flags.port = flag.Int("p", 8501, "port used for listening")
	flags.host = flag.String("h", "", "ip used for listening, leave empty for all available IP addresses")
	flags.configDir = flag.String("c", "/etc/script_exporter", "folder for config yaml files")
	flags.version = flag.Bool("V", false, "show version information")

	config.probes = make(map[string]probeType)
}

func main() {
	setup()

	addr := fmt.Sprintf("%s:%d", config.server.host, config.server.port)

	log.Info("Started on ", addr)

	http.HandleFunc("/", landingpage)
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/probe", probe)
	err := http.ListenAndServe(addr, nil)

	if err != nil {
		log.Fatalf("Could not start server: %v", err)
	}
}
