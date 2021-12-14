package main

import (
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

func metrics(namespace string, subsystem string, probe probeType, result runResult) *prometheus.Registry {
	// Init metrics
	log.Debug("[metrics] Initialize metrics: ", subsystem)

	gaugeUp := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "up",
			Help:      "General availability of this probe",
		},
		probe.labelNames,
	)

	gaugeSuccess := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "success",
			Help:      "Show if the script was executed successfully",
		},
		probe.labelNames,
	)

	gaugeDuration := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "duration_seconds",
			Help:      "Shows the execution time of the script",
		},
		probe.labelNames,
	)

	// Register metrics
	log.Debug("[metrics] Register metrics: ", subsystem)

	registry := prometheus.NewRegistry()
	registry.MustRegister(gaugeUp)
	registry.MustRegister(gaugeSuccess)
	registry.MustRegister(gaugeDuration)

	// Set metrics
	log.Debug("[metrics] Set metrics: ", subsystem)

	gaugeUp.WithLabelValues(probe.labelValues...).Set(1)

	if result.exitCode == 0 {
		gaugeSuccess.WithLabelValues(probe.labelValues...).Set(1)
	} else {
		gaugeSuccess.WithLabelValues(probe.labelValues...).Set(0)
	}

	gaugeDuration.WithLabelValues(probe.labelValues...).Set(result.duration.Seconds())

	return registry
}
