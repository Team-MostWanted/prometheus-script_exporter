# Prometheus Script Exporter

Helper binary for Prometheus to expose basic metrics for a script or executable.

In using `script_exporter` there are 4 things to consider:

1. Web server listening settings for this script exporter
2. Probe settings
3. Usable script to be executed via the probe
4. Scrape configuration in Prometheus

Server settings can be configured via [parameters](#Parameters) or via [configuration](#Configuration). Parameters will take precedence over configuration settings.

Probe settings can only be configured via [configuration](#Configuration). And will be available via http://localhost:8501/probe?module=example.

For the settings in Prometheus look at the [Scrape configuration section](#Scrape%20configuration).

## Usage

To create the binary look at the [Building section](#Building).

Make sure you have a script or executable that returns an exit code 0 for success and any other exit code on failure.

> :warning:
>
> The script or executable should have a fairly quick response, and not be a daemon.

Create an appropriate [configuration](#Configuration).

Start the scrip_exporter:
```
./script_exporter
```

The output given should look like:

```
INFO[0000] Looking for configuration files in: /etc/script_exporter
INFO[0000] Probe initialized: example1
INFO[0000] Probe initialized: example2
INFO[0000] Started on :8501
```

### Parameters

The script exporter has the following command line arguments:

```
Usage of script_exporter
  -V	show version information
  -c string
    	folder for config yaml files (default "/etc/script_exporter")
  -h string
    	ip used for listening, leave empty for all available IP addresses
  -p int
    	port used for listening (default 8501)
  -v	show verbose output
```

The -h and -p setting will overrule any settings made in the config files.

### Configuration
The configuration files are made in yaml format.

Default the config files are searched in the `/etc/script_exporter` folder.

It is possible to have split configuration files, however duplicate settings can result in error on starting the script_exporter.

This will result in an error like this:

> FATA[0000] Config failure 'host' is already set (server.yaml)

#### Server configuration

Settings used for the `server:` section.

> :warning:
>
> Command line flags will overrule these settings

key     | description
--------|----------
host    | ip used for listening, leave empty for all available IP addresses
port    | port used for listening (default 8501)

Example:

```
server:
    host: 127.0.0.1
    port: 80
```

#### Probe configuration

Settings used for the `probes:` section.

key         | description
------------|----------
name        | name of the probe, used for the `module` query parameter in the [probe endpoint](#Endpoints), must match regex ^[a-zA-Z0-9:_]+$
cmd         | command that is executed
subsystem   | optional subsystem for the metric name default empty `probe_script_{up|success|duration_seconds}` with `probe_script_{subsystem}_{up|success|duration_seconds}`
labels      | list of labels used for the metrics
- key       | label name
- value     | label static value
arguments   | list of arguments used for the cmd, in the order as given
- default   | the default value of the argument
- dynamic   | default false, signifies if this could be set via probe query parameters
- param     | name of query parameter, if the argument has `dynamic` true, module and debug are restricted and not usable as param values.

Note make sure a metric is unique by using either labels or subsystem.

With a dynamic argument a probe could be created that could be scraped multiple times with different parameters.

Example:

```
probes:
    - name: example1
      cmd: /usr/local/bin/python3
      labels:
        - key: foo
          value: bar
        - key: baz
          value: qux
        arguments:
        - default: ./test/resources/ok_print_arguments.py
        - dynamic: true
          param: first
          default: hello
        - dynamic: true
          type: dynamic
          param: second
    - name: example2
      cmd: /usr/local/bin/python3
      subsystem: something
      arguments:
        - default: ./test/resources/error_print_arguments.py
        - dynamic: true
          param: foo
          default: hello
        - dynamic: true
          type: dynamic
          param: bar
```

With this configuration the following url's would be valid:

http://localhost:8501/probe?module=example1
```
DEBU[0071] [Run] Start cmd/usr/local/bin/python3 ./test/resources/ok_print_arguments.py hello
```

http://localhost:8501/probe?module=example1&second=lipsum
```
DEBU[0145] [Run] Start cmd/usr/local/bin/python3 ./test/resources/ok_print_arguments.py hello lipsum
```

http://localhost:8501/probe?module=example1&first=bonjour
```
DEBU[0178] [Run] Start cmd/usr/local/bin/python3 ./test/resources/ok_print_arguments.py bonjour
```

http://localhost:8501/probe?second=lipsum&module=example1&first=bonjour
```
DEBU[0244] [Run] Start cmd/usr/local/bin/python3 ./test/resources/ok_print_arguments.py bonjour lipsum
```

http://localhost:8501/probe?module=example2

```
DEBU[0385] [Run] Start cmd/usr/local/bin/python3 ./test/resources/error_print_arguments.py hello
```

http://localhost:8501/probe?module=example2&foo=baz
```
DEBU[0437] [Run] Start cmd/usr/local/bin/python3 ./test/resources/error_print_arguments.py baz
```

### Scrape configuration

In order to use the probe information some relabeling is required.

Down below is an example scrape configuration.

```
scrape_configs:
    - job_name: script_exporter
      static_configs:
        - targets:
            - localhost:8501
    - job_name: amsterdam01_script_exporter
      scrape_timeout: 30s
      metrics_path: /probe
      static_configs:
        - targets:
            - script_1
            - script_2
            - script_3
      relabel_configs:
        - source_labels: [__address__]
          target_label: __param_module
        - target_label: address
          replacement: amsterdam01.example.com:8501
        - source_labels: [__address__]
          target_label: instance
```

### Endpoints

The script exporter exposes 3 endpoints:

endpoint    | description
------------|----------
/           | landingspage with some basic information on possible endpoints
/metrics    | the metrics of the script exporter instance it self
/probe      | the metrics of the configured probes

The probe endpoint has 2 default query parameters:

parameter | description
----------|----------
module    | required, states the name of the specific probe that should be run
debug     | optional, should only be used on a manual run, and shows exit code, stdout and stderr of the probe that is executed.

The probe could also have specific query parameters specified in the [Probe configuration](#Probe%20configuration).

#### Metrics

The output of the probe endpoint will expose the following metrics:

metric              | description
--------------------|----------
up                  | 1 if the probe is up, or not existend
success             | 1 if the exit code of the command = 0, 0 on any other exit code
duration_seconds    | the duration of the command in seconds

Example 1:

```
# HELP probe_script_duration_seconds Shows the execution time of the script
# TYPE probe_script_duration_seconds gauge
probe_script_duration_seconds{baz="qux",foo="bar"} 0.037359001
# HELP probe_script_success Show if the script was executed successfully
# TYPE probe_script_success gauge
probe_script_success{baz="qux",foo="bar"} 1
# HELP probe_script_up General availability of this probe
# TYPE probe_script_up gauge
probe_script_up{baz="qux",foo="bar"} 1
```

Example 2:
```
# HELP probe_script_something_duration_seconds Shows the execution time of the script
# TYPE probe_script_something_duration_seconds gauge
probe_script_something_duration_seconds 0.037780856
# HELP probe_script_something_success Show if the script was executed successfully
# TYPE probe_script_something_success gauge
probe_script_something_success 0
# HELP probe_script_something_up General availability of this probe
# TYPE probe_script_something_up gauge
probe_script_something_up 1
```

## Developing

If you want to contribute to this project please follow these guidelines:

- Script Exporter is build in [Golang](https://golang.org/)
- Use style guides as described in [.editorconfig](.editorconfig)
- Changes in features should be reflected in this [README.md](README.md)
- Changes should be reflected into the [CHANGELOG.md](CHANGELOG.md)

The maintainers
- Version is derived from the [CHANGELOG.md](CHANGELOG.md) using the last non-upcoming header (##)
- Use `make update` to automatically update dependencies and add a line in the [CHANGELOG.md](CHANGELOG.md)

### Building

To create a working binary use:

```
git clone git@github.com:Team-MostWanted/prometheus-script_exporter.git
cd prometheus-script_exporter
make
```

This creates a binary in the `./build` folder. Look at the [Build options section](#Build%20options) for more build options.

#### Make options

The targets used by default are:
- freebsd/amd64
- darwin/amd64
- darwin/arm64
- linux/amd64
- linux/arm64
- windows/amd64

Use `make <command> TARGETS=<os>/<arch>` to execute make with your own OS and Architecture, e.g. `make build TARGETS=freebsd/arm64`.

target      | description
------------|------------
version     | retrieve and display the version from the [CHANGELOG.md](CHANGELOG.md)
test        | use `go test` to execute the unit test and create a coverage report `./build/test-coverage.out`
build       | build the binary for every OS and Arch stated
dist-check  | validations performed to see if the codebase is ready for a release, on default this will skipped unless performed on the acceptance or master branch
dist-create | create tar.gz files in `./dist` for all targets using the binaries from `./build`
dist        | perform a clean, dist-check build and dist-create
update-dependencies | update go version and update dependencies, makes sure everything is tiday, and add a line to the changelog
update      | perform a clean, update-dependencies and a test
clean       | clean the build and dist directories

### TODO

Down below are some features we think about adding in the future.

- replace packages that appear to be abandoned, or are not updated recently:
  - https://github.com/go-yaml/yaml/issues/788
  - https://github.com/stretchr/testify/discussions/1560
  - https://github.com/gorilla/handlers
- improved Probe debug options
- add config option for sensitive data
- add Prometheus style of listener web.listen-address (127.0.0.1:8501)
- add support for long arguments (e.g. --version)
- be able to use stdout as input for configured metric values
- reload configuration via commandline
- reload configuration via endpoint

## Changelog

All notable changes for the Prometheus Script Exporter can be found in [CHANGELOG.md](CHANGELOG.md).
