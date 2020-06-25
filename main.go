package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

var flags struct {
	verbose *bool
	port    *int
	ip      *string
}

func init() {
	flags.verbose = flag.Bool("v", false, "show verbose output")
	flags.port = flag.Int("p", 8501, "port used for listening")
	flags.ip = flag.String("i", "", "ip used for listening, empty for all available IP addresses")
	//config := flag.String("c", "/etc/script_exporter", "folder for config yaml files")
}

func setup() {
	flag.Parse()

	if *flags.verbose {
		log.SetLevel(log.DebugLevel)
	}
}

func main() {
	setup()

	addr := fmt.Sprintf("%s:%d", *flags.ip, *flags.port)

	log.Info("Started on ", addr)

	http.HandleFunc("/", landingpage)
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/probe", probe)
	err := http.ListenAndServe(addr, nil)

	if err != nil {
		log.Error("Could not serve ", err)
		os.Exit(1)
	}
}

func landingpage(w http.ResponseWriter, r *http.Request) {
	title := "Script Exporter"

	fmt.Fprintf(
		w,
		`<html>
			<head><title>%s</title></head>
			<body>
			<h1>%s</h1>
			<p><a href="metrics">Metrics</a></p>
			</body>
		</html>`,
		title,
		title,
	)
}

type runResult struct {
	exitCode int
	stdout   string
	stderr   string
	duration time.Duration
}

func run(cmd *exec.Cmd) runResult {
	var result runResult
	var outbuf, errbuf bytes.Buffer
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf

	log.Debug("Run: Start cmd", cmd)
	start := time.Now()

	err := cmd.Run()

	log.Debug("Run: end cmd", cmd)

	result.duration = time.Since(start)

	log.Debug("Run: end duration: ", result.duration)

	result.stdout = outbuf.String()
	result.stderr = errbuf.String()

	result.exitCode = 0
	if err != nil {
		result.exitCode = 999

		var e *exec.ExitError
		if errors.As(err, &e) {
			result.exitCode = e.ExitCode()
		}
	}

	log.Debug("Run: end duration: ", result.exitCode)

	return result
}

func probe(w http.ResponseWriter, r *http.Request) {
	namespace := "scriptExporter"
	moduleName := r.URL.Query().Get("module")

	if moduleName != "helloworld" {
		http.Error(w, "Invalid Probe", http.StatusNotFound)
		return
	}

	// Init metrics
	gaugeUp := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: moduleName,
		Name:      "up",
		Help:      "Show if the script was executed successfully",
	})

	gaugeDuration := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: moduleName,
		Name:      "duration_seconds",
		Help:      "Shows the execution time of the script",
	})

	gaugeExitCode := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: moduleName,
		Name:      "exitCode",
		Help:      "States the exit code given by the script",
	})

	// Register metrics
	registry := prometheus.NewRegistry()
	registry.MustRegister(gaugeUp)
	registry.MustRegister(gaugeDuration)
	registry.MustRegister(gaugeExitCode)

	// Give metrics values
	result := run(exec.Command("test/resources/helloworld.py"))

	if result.exitCode == 0 {
		gaugeUp.Set(1)
	} else {
		gaugeUp.Set(0)
	}

	gaugeDuration.Set(result.duration.Seconds())
	gaugeExitCode.Set(float64(result.exitCode))

	// Serve metrics
	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
}
