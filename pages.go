package main

import (
	"fmt"
	"net/http"
)

func landingPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

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

	_, err := fmt.Fprintf(
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
	if err != nil {
		return
	}
}

func debugProbe(w http.ResponseWriter, probeName string, result runResult) {
	title := "Debug Probe " + probeName

	_, err := fmt.Fprintf(
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
	if err != nil {
		return
	}
}
