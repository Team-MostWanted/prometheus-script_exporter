package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

var name = "script_exporter"
var version = "v0.0.1"
var commit = "development"
var date = "0001-01-01T00:00:00.000Z"

var flags struct {
	verbose   *bool
	port      *int
	host      *string
	configDir *string
	version   *bool
}

// YamlConfig mapping to config as in Yaml defined
type YamlConfig struct {
	Server *YamlServerConfig
	Probes *YamlProbeConfig
}

// YamlServerConfig mapping to Server part
type YamlServerConfig struct {
	Host *string `yaml:"host"`
	Port *int    `yaml:"port"`
}

// YamlProbeConfig mapping to Probes part
type YamlProbeConfig []struct {
	Name      string `yaml:"name"`
	Cmd       string `yaml:"cmd"`
	Subsystem string `yaml:"subsystem"`
	Labels    []struct {
		Key   string `yaml:"key"`
		Value string `yaml:"value"`
	}
	Arguments []struct {
		Dynamic *bool   `yaml:"dynamic"`
		Param   *string `yaml:"param"`
		Default *string `yaml:"default"`
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
	cmd           string
	subsystem     string
	labelNames    []string
	labelValues   []string
	arguments     map[string]probeArgument
	argumentOrder *[]string
}

type probeArgument struct {
	dynamic      bool
	defaultValue *string
}

var restrictedParams = []string{"module", "debug"}

var config internalConfig

func setup() {
	// retrieve flags since that could contain the config folder
	flag.Parse()

	// Set the verbose logging
	if *flags.verbose {
		log.SetLevel(log.DebugLevel)
	}

	if *flags.version {
		log.Infof("%s, %s (%s), build: %s", name, version, date, commit)

		os.Exit(0)
	}

	// parse the configuration files
	readConfig()

	// parse the flags
	configFlags()
}

func readConfig() {
	log.Infof("Looking for configuration files in: %s", *flags.configDir)

	files, err := os.ReadDir(*flags.configDir)
	if err != nil {
		log.Fatal("Could not load config files: ", err)
	}

	log.Debug("[readConfig] files found (regardless of .yaml extension): ", len(files))

	for _, file := range files {
		if !file.IsDir() && (strings.HasSuffix(file.Name(), ".yaml") || strings.HasSuffix(file.Name(), ".yml")) {
			log.Debug("[readConfig] loading config file: ", file.Name())

			yamlFile, err := os.ReadFile(
				path.Join(*flags.configDir, file.Name()),
			)
			if err != nil {
				log.Fatalf("Could not load config file: %v", err)
			}

			log.Debug("[readConfig] parsing config file: ", file.Name())

			var yamlConfig YamlConfig
			err = yaml.Unmarshal(yamlFile, &yamlConfig)
			if err != nil {
				log.Fatalf("Could not parse yaml (%s): %v", file.Name(), err)
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
			log.Fatalf("Config failure 'host' is already set (%s)", fileName)
		}

		config.server.host = *serverConfig.Host
	}

	if serverConfig.Port != nil {
		if config.server.port != 0 {
			log.Fatalf("Config failure 'port' is already set (%s)", fileName)
		}

		config.server.port = *serverConfig.Port
	}
}

func configProbes(probesConfig YamlProbeConfig, fileName string) {
	log.Debug("[configProbes] merging Probes block: ", fileName)

	for _, probe := range probesConfig {
		log.Debug("[configProbes] found probe: ", probe.Name)

		matchName := regexp.MustCompile("^[a-zA-Z0-9:_]+$")
		if match := matchName.MatchString(probe.Name); !match {
			log.Fatalf("Config failure probe with name '%s' name must match ^[a-zA-Z0-9:_]+$ (%s)", probe.Name, fileName)
		}

		if _, exists := config.probes[probe.Name]; exists {
			log.Fatalf("Config failure probe with name '%s' already exists (%s)", probe.Name, fileName)
		}

		var argumentOrder []string = make([]string, len(probe.Arguments))

		var labelNames []string
		var labelValues []string
		for _, label := range probe.Labels {
			if label.Key == "" || label.Value == "" {
				log.Fatalf("Config failure probe with name '%s' invalid labels entry given, key and value are required (%s)", probe.Name, fileName)
			}

			labelNames = append(labelNames, label.Key)
			labelValues = append(labelValues, label.Value)
		}

		config.probes[probe.Name] = probeType{
			probe.Cmd,
			probe.Subsystem,
			labelNames,
			labelValues,
			make(map[string]probeArgument),
			&argumentOrder,
		}

		i := 0
		for key, argument := range probe.Arguments {
			var argName string = fmt.Sprint(key)

			if argument.Param != nil {
				if _, exists := config.probes[probe.Name].arguments[*argument.Param]; exists {
					log.Fatalf(
						"Config failure probe argument '%s' already exists for '%s' (%s)",
						*argument.Param,
						probe.Name,
						fileName,
					)
				}

				for _, restricted := range restrictedParams {
					if restricted == *argument.Param {
						log.Fatalf(
							"Config failure restricted probe argument '%s' on '%s' (%s)",
							*argument.Param,
							probe.Name,
							fileName,
						)
					}
				}

				argName = *argument.Param
			}

			// Keep track of Order of Arguments since golang map is unsorted
			argumentOrder[i] = argName
			i++

			dynamic := false
			if argument.Dynamic != nil {
				dynamic = *argument.Dynamic
			}

			config.probes[probe.Name].arguments[argName] = probeArgument{
				dynamic,
				argument.Default,
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
