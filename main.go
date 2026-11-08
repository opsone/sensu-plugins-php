package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	neturl "net/url"
	"strconv"
	"strings"
	"time"

	corev2 "github.com/sensu/core/v2"
	"github.com/sensu/sensu-plugin-sdk/sensu"
)

// Config represents the check plugin config.
type Config struct {
	sensu.PluginConfig
	Host     string
	Port     int
	Path     string
	Response string
	Timeout  int
}

var (
	plugin = Config{
		PluginConfig: sensu.PluginConfig{
			Name:     "check-php-fpm",
			Short:    "Check that PHP-FPM answers its ping page through the web server",
			Keyspace: "sensu.io/plugins/check-php-fpm/config",
		},
	}

	options = []sensu.ConfigOption{
		&sensu.PluginConfigOption[string]{
			Path:      "host",
			Env:       "CHECK_PHP_FPM_HOST",
			Argument:  "host",
			Shorthand: "H",
			Usage:     "Web server host serving the PHP-FPM ping page, e.g. 127.0.0.1 (required)",
			Value:     &plugin.Host,
		},
		&sensu.PluginConfigOption[int]{
			Path:      "port",
			Env:       "CHECK_PHP_FPM_PORT",
			Argument:  "port",
			Shorthand: "p",
			Usage:     "Web server port, e.g. 80 (required)",
			Value:     &plugin.Port,
		},
		&sensu.PluginConfigOption[string]{
			Path:      "path",
			Env:       "CHECK_PHP_FPM_PATH",
			Argument:  "path",
			Shorthand: "q",
			Usage:     "Path of the PHP-FPM ping page (ping.path), e.g. fpm-ping (required)",
			Value:     &plugin.Path,
		},
		&sensu.PluginConfigOption[string]{
			Path:      "response",
			Env:       "CHECK_PHP_FPM_RESPONSE",
			Argument:  "response",
			Shorthand: "r",
			Default:   "pong",
			Usage:     "Expected response (ping.response)",
			Value:     &plugin.Response,
		},
		&sensu.PluginConfigOption[int]{
			Path:     "timeout",
			Env:      "CHECK_PHP_FPM_TIMEOUT",
			Argument: "timeout",
			Default:  10,
			Usage:    "HTTP request timeout in seconds",
			Value:    &plugin.Timeout,
		},
	}
)

// maxBody is the number of bytes of the response body read and compared.
const maxBody = 4096

func main() {
	check := sensu.NewGoCheck(&plugin.PluginConfig, options, checkArgs, executeCheck, false)
	check.Execute()
}

func checkArgs(event *corev2.Event) (int, error) {
	switch {
	case plugin.Host == "":
		return sensu.CheckStateUnknown, errors.New("--host is required")
	case plugin.Port < 1 || plugin.Port > 65535:
		return sensu.CheckStateUnknown, errors.New("--port is required, between 1 and 65535")
	case strings.Trim(plugin.Path, "/") == "":
		return sensu.CheckStateUnknown, errors.New("--path is required")
	case plugin.Response == "":
		return sensu.CheckStateUnknown, errors.New("--response must not be empty")
	case plugin.Timeout <= 0:
		return sensu.CheckStateUnknown, errors.New("--timeout must be greater than 0")
	}
	return sensu.CheckStateOK, nil
}

func executeCheck(event *corev2.Event) (int, error) {
	url := pingURL(plugin.Host, plugin.Port, plugin.Path)
	result := fetch(url, time.Duration(plugin.Timeout)*time.Second, plugin.Response)
	fmt.Printf("%s %s: %s\n", plugin.Name, stateLabel(result.State), result.Message)
	return result.State, nil
}

// pingURL builds the URL of the ping page; the path is accepted with or without a leading slash.
func pingURL(host string, port int, path string) string {
	return "http://" + net.JoinHostPort(host, strconv.Itoa(port)) + "/" + strings.TrimPrefix(path, "/")
}

// fetch requests the ping page and evaluates the answer. Transport errors are critical:
// the service does not answer.
func fetch(url string, timeout time.Duration, expected string) Result {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Result{sensu.CheckStateUnknown, err.Error()}
	}
	client := &http.Client{
		// Like the Ruby version (Net::HTTP), redirects are not followed.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	resp, err := client.Do(req)
	if err != nil {
		// *url.Error repeats the method and URL: keep the underlying cause only.
		var uerr *neturl.Error
		if errors.As(err, &uerr) {
			err = uerr.Err
		}
		return Result{sensu.CheckStateCritical, fmt.Sprintf("%s unreachable: %v", url, err)}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return Result{sensu.CheckStateCritical, fmt.Sprintf("reading %s: %v", url, err)}
	}
	return evaluate(resp.StatusCode, string(body), expected)
}

// Result is the outcome of the evaluation of the ping page.
type Result struct {
	State   int
	Message string
}

// evaluate turns the HTTP status code and body of the ping page into a check result.
func evaluate(code int, body, expected string) Result {
	if code != http.StatusOK {
		return Result{sensu.CheckStateCritical, fmt.Sprintf("Error, http response code: %d", code)}
	}
	body = strings.TrimSpace(body)
	if body != expected {
		return Result{sensu.CheckStateCritical, fmt.Sprintf("%s instead of %s", truncate(body, 100), expected)}
	}
	return Result{sensu.CheckStateOK, expected}
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

func stateLabel(state int) string {
	switch state {
	case sensu.CheckStateOK:
		return "OK"
	case sensu.CheckStateWarning:
		return "WARNING"
	case sensu.CheckStateCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}
