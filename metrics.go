package main

import (
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

func metrics(namespace string, subsystem string, result runResult) *prometheus.Registry {
	// Init metrics
	log.Debug("[metrics] Initialize metrics: ", subsystem)

	gaugeUp := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "up",
		Help:      "General availability of this probe",
	})

	gaugeSuccess := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "success",
		Help:      "Show if the script was executed successfully",
	})

	gaugeDuration := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "duration_seconds",
		Help:      "Shows the execution time of the script",
	})

	// Register metrics
	log.Debug("[metrics] Register metrics: ", subsystem)

	registry := prometheus.NewRegistry()
	registry.MustRegister(gaugeUp)
	registry.MustRegister(gaugeSuccess)
	registry.MustRegister(gaugeDuration)

	// Set metrics
	log.Debug("[metrics] Set metrics: ", subsystem)

	gaugeUp.Set(1)

	if result.exitCode == 0 {
		gaugeSuccess.Set(1)
	} else {
		gaugeSuccess.Set(0)
	}

	gaugeDuration.Set(result.duration.Seconds())

	return registry
}
