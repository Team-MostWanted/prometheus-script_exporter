package main

import (
	"bytes"
	"errors"
	"net/http"
	"os/exec"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

type runResult struct {
	exitCode int
	stdout   string
	stderr   string
	duration time.Duration
}

func probe(w http.ResponseWriter, r *http.Request) {
	moduleName := r.URL.Query().Get("module")

	// Retrieve probe
	log.Debug("[probe] Retrieve probe: ", moduleName)

	probe, exists := config.probes[moduleName]
	if !exists {
		http.Error(w, "Invalid Probe", http.StatusNotFound)

		log.Debug("[probe] Invalid probe: ", moduleName)
		return
	}

	// Execute probe cmd
	result := run(buildCmd(probe, r))

	if _, exists := r.URL.Query()["debug"]; exists {
		debugProbe(w, moduleName, result)
		return
	}

	// Serve metrics
	h := promhttp.HandlerFor(
		metrics("probe_script", moduleName, probe, result),
		promhttp.HandlerOpts{},
	)
	h.ServeHTTP(w, r)
}

func run(cmd *exec.Cmd) runResult {
	var result runResult
	var outbuf, errbuf bytes.Buffer
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf

	log.Debug("[Run] Start cmd", cmd)
	start := time.Now()

	err := cmd.Run()

	result.duration = time.Since(start)
	log.Debug("[Run] End cmd duration: ", result.duration)

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

	log.Debug("[Run] end duration: ", result.exitCode)

	return result
}

func buildCmd(probe probeType, r *http.Request) *exec.Cmd {
	var args []string

	log.Debug(probe)
	for _, key := range *probe.argumentOrder {
		argument := probe.arguments[key]
		var value *string

		log.Debug("[buildCmd] argument: ", key)

		if argument.dynamic {
			log.Debug("[buildCmd] retrieve dynamic value for: ", key)

			if _, exists := r.URL.Query()[key]; exists {
				queryParam := r.URL.Query().Get(key)
				value = &queryParam
			}
		}

		if value == nil && argument.defaultValue != nil {
			log.Debug("[buildCmd] use default value for: ", key)

			log.Debug(*argument.defaultValue)

			value = argument.defaultValue
		}

		if value != nil {
			log.Debug("[buildCmd] append argument: ", *value)

			args = append(args, *value)
		}
	}

	return exec.Command(probe.cmd, args...)
}
