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
    "os"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    log "github.com/sirupsen/logrus"
    "gopkg.in/yaml.v3"
)

var name = "script_exporter"
var version = "v0.0.1"
var commit = "development"
var date = "0001-01-01T00:00:00.000Z"

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
    arguments     map[string]probeArgument
    argumentOrder *[]string
}

type probeArgument struct {
    dynamic      bool
    defaultValue *string
}

var restrictedParams = []string{"module", "debug"}

var config internalConfig
var probes []string

var flags struct {
    verbose   *bool
    port      *int
    host      *string
    configDir *string
    version   *bool
}

func init() {
    flags.verbose = flag.Bool("v", false, "show verbose output")
    flags.port = flag.Int("p", 8501, "port used for listening")
    flags.host = flag.String("h", "", "ip used for listening, leave empty for all available IP addresses")
    flags.configDir = flag.String("c", "/etc/script_exporter", "folder for config yaml files")
    flags.version = flag.Bool("V", false, "show version information")

    config.probes = make(map[string]probeType)
}

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

    files, err := ioutil.ReadDir(*flags.configDir)
    if err != nil {
        log.Fatal("Could not load config files: ", err)
    }

    log.Debug("[readConfig] files found (regardless of .yaml extension): ", len(files))

    for _, file := range files {
        if !file.IsDir() && (strings.HasSuffix(file.Name(), ".yaml") || strings.HasSuffix(file.Name(), ".yml")) {
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

        if _, exists := config.probes[probe.Name]; exists {
            log.Fatalf("Config failure probe with name '%s' already exists (%s)", probe.Name, fileName)
        }

        var argumentOrder []string = make([]string, len(probe.Arguments))

        config.probes[probe.Name] = probeType{
            probe.Cmd,
            make(map[string]probeArgument),
            &argumentOrder,
        }

        i := 0
        for key, argument := range probe.Arguments {
            var argName string = string(key)

            if argument.Param != nil {
                if _, exists := config.probes[probe.Name].arguments[*argument.Param]; exists {
                    log.Fatalf(
                        "Config failure probe argument '%s' already exists for '%s' (%s)",
                        argument.Param,
                        probe.Name,
                        fileName,
                    )
                }

                for _, restricted := range restrictedParams {
                    if restricted == *argument.Param {
                        log.Fatalf(
                            "Config failure restricted probe argument '%s' on '%s' (%s)",
                            argument.Param,
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

    var probesString string
    for probeName := range config.probes {
        probesString = fmt.Sprintf(
            `%s<p><a href="probe?module=%s">Probe %s</a>&nbsp;&nbsp;<a href="probe?module=%s&debug">debug</a></p>`,
            probesString,
            probeName,
            probeName,
            probeName,
        )
    }

    fmt.Fprintf(
        w,
        `<html>
            <head><title>%s</title></head>
            <body>
            <h1>%s</h1>
            <p><a href="metrics">Metrics</a></p>
            %s
            </body>
        </html>`,
        title,
        title,
        probesString,
    )
}

func debugProbe(w http.ResponseWriter, probeName string, result runResult) {
    title := "Debug Probe " + probeName

    fmt.Fprintf(
        w,
        `<html>
            <head><title>%s</title></head>
            <body>
            <h1>%s</h1>
            <table cellspacing="0" cellpadding="5">
                <tr>
                    <td>success</td>
                    <td>%t</td>
                </tr>
                <tr>
                    <td>exit&nbsp;code</td>
                    <td>%d</td>
                </tr>
                <tr>
                    <td>duration</td>
                    <td>%f seconds</td>
                </tr>
                <tr>
                    <td valign="top">stdout</td>
                    <td><textarea rows="20" cols=120>%s</textarea></td>
                </tr>
                <tr>
                    <td valign="top">stderr</td>
                    <td><textarea rows="20" cols=120>%s</textarea></td>
                </tr>
            </table>
            </body>
        </html>`,
        title,
        title,
        result.exitCode == 0,
        result.exitCode,
        result.duration.Seconds(),
        result.stdout,
        result.stderr,
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
        metrics("probe_script", moduleName, result),
        promhttp.HandlerOpts{},
    )
    h.ServeHTTP(w, r)
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
