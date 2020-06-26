package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// YamlConfig mapping to config as in Yaml defined
type YamlConfig struct {
	Server *YamlServerConfig
	Probes *YamlProbeConfig
}

// YamlServerConfig mapping to Server part
type YamlServerConfig struct {
	Host    *string `yaml:"host"`
	Port    *int    `yaml:"port"`
	Verbose *bool   `yaml:"verbose"`
}

// YamlProbeConfig mapping to Probes part
type YamlProbeConfig []struct {
	Name      string `yaml:"name"`
	Cmd       string `yaml:"cmd"`
	Arguments []struct {
		Dynamic bool   `yaml:"dynamic"`
		Param   string `yaml:"param"`
		Default string `yaml:"default"`
	}
}

type internalConfig struct {
	server struct {
		host string
		port int
	}

	probes map[string]probeType
}

type probeType struct {
	cmd       string
	arguments map[string]probeArgument
}

type probeArgument struct {
	Dynamic bool
	Default *string
}

var config internalConfig
var probes []string

var flags struct {
	verbose   *bool
	port      *int
	host      *string
	configDir *string
}

func init() {
	flags.verbose = flag.Bool("v", false, "show verbose output")
	flags.port = flag.Int("p", 8501, "port used for listening")
	flags.host = flag.String("h", "", "ip used for listening, leave empty for all available IP addresses")
	flags.configDir = flag.String("c", "/etc/script_exporter", "folder for config yaml files")

	config.probes = make(map[string]probeType)
}

func setup() {
	// retrieve flags since that could contain the config folder
	flag.Parse()

	// Set the verbose logging
	if *flags.verbose {
		log.SetLevel(log.DebugLevel)
	}

	// parse the configuration files
	readConfig()

	// parse the flags
	configFlags()
}

func readConfig() {
	log.Infof("Looking for configuration files in: %s", *flags.configDir)

	files, err := ioutil.ReadDir(*flags.configDir)
	if err != nil {
		log.Fatal("Could not load config files: ", err)
	}

	log.Debug("[readConfig] files found (regardless of extension): ", len(files))

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".yaml") {
			log.Debug("[readConfig] loading config file: ", file.Name())

			yamlFile, err := ioutil.ReadFile(
				path.Join(*flags.configDir, file.Name()),
			)
			if err != nil {
				log.Fatalf("Could not load config file: %v", err)
			}

			log.Debug("[readConfig] parsing config file: ", file.Name())

			var yamlConfig YamlConfig
			err = yaml.Unmarshal(yamlFile, &yamlConfig)
			if err != nil {
				log.Fatalf("Could not parse yaml: %v", err)
			}

			if yamlConfig.Server != nil {
				configServer(*yamlConfig.Server, file.Name())
			}

			if yamlConfig.Probes != nil {
				configProbes(*yamlConfig.Probes, file.Name())
			}
		}
	}
}

func configServer(serverConfig YamlServerConfig, fileName string) {
	log.Debug("[configServer] merging Server block: ", fileName)

	if serverConfig.Host != nil {
		if config.server.host != "" {
			log.Fatalf("Config failure 'host' is already set remove from file: %s", fileName)
		}

		config.server.host = *serverConfig.Host
	}

	if serverConfig.Port != nil {
		if config.server.port != 0 {
			log.Fatalf("Config failure 'port' is already set remove from file: %s", fileName)
		}

		config.server.port = *serverConfig.Port
	}
}

func configProbes(probesConfig YamlProbeConfig, fileName string) {
	log.Debug("[configProbes] merging Probes block: ", fileName)

	for _, probe := range probesConfig {
		log.Debug("[configProbes] found probe: ", probe.Name)

		if _, exists := config.probes[probe.Name]; exists {
			log.Fatalf("Config failure probe with name %s already exists remove from %s", probe.Name, fileName)
		}

		config.probes[probe.Name] = probeType{
			probe.Cmd,
			make(map[string]probeArgument),
		}

		for _, argument := range probe.Arguments {
			if _, exists := config.probes[probe.Name].arguments[argument.Param]; exists {
				log.Fatalf(
					"Config failure probe parameter %s already exists for %s remove from %s",
					argument.Param,
					probe.Name,
					fileName,
				)
			}

			config.probes[probe.Name].arguments[argument.Param] = probeArgument{
				argument.Dynamic,
				&argument.Default,
			}
		}

		log.Info("Probe initialized: ", probe.Name)
	}
}

// command line flags overrule the configuration files
func configFlags() {
	if flags.host != nil {
		config.server.host = *flags.host
	}

	if flags.port != nil || config.server.port == 0 {
		config.server.port = *flags.port
	}
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

	log.Debug("[Run] Start cmd", cmd)
	start := time.Now()

	err := cmd.Run()

	log.Debug("[Run] end cmd", cmd)

	result.duration = time.Since(start)

	log.Debug("[Run] end duration: ", result.duration)

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
