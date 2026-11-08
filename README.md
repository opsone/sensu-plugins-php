# sensu-plugins-php

Sensu Go check verifying that PHP-FPM answers: `check-php-fpm` requests the PHP-FPM ping page (`ping.path`) through the web server and compares the answer with `ping.response`.

Static Go binary (no CGO, no Ruby runtime): one asset for every Linux distribution, `amd64` only.

## Behavior

`check-php-fpm` sends `GET http://<host>:<port>/<path>`. Redirects are not followed. The body is compared with `--response` after trimming leading and trailing whitespace.

| Situation | State |
|---|---|
| HTTP 200 with the expected response | OK |
| HTTP 200 with another body | CRITICAL (`<body> instead of pong`) |
| Any other HTTP code (404, 502 when PHP-FPM is down, 301…) | CRITICAL (`Error, http response code: 502`) |
| Connection refused, timeout, DNS error | CRITICAL |
| Missing or invalid argument | UNKNOWN |

## PHP-FPM and web server setup

In the pool configuration (e.g. `/etc/php/8.4/fpm/pool.d/www.conf`):

```ini
ping.path = /fpm-ping
ping.response = pong
```

In nginx, restricted to local requests:

```nginx
location = /fpm-ping {
  allow 127.0.0.1;
  deny all;
  include fastcgi_params;
  fastcgi_param SCRIPT_FILENAME $fastcgi_script_name;
  fastcgi_pass unix:/run/php/php8.4-fpm.sock;
}
```

## Usage

```
check-php-fpm [flags]
  -H, --host string       Web server host serving the PHP-FPM ping page, e.g. 127.0.0.1 (required)
  -p, --port int          Web server port, e.g. 80 (required)
  -q, --path string       Path of the PHP-FPM ping page (ping.path), e.g. fpm-ping (required)
  -r, --response string   Expected response (ping.response) (default "pong")
      --timeout int       HTTP request timeout in seconds (default 10)
```

The path is accepted with or without a leading slash. Every option can also be set with an environment variable:

| Option | Variable | Annotation |
|---|---|---|
| `--host` | `CHECK_PHP_FPM_HOST` | `php_fpm_host` |
| `--port` | `CHECK_PHP_FPM_PORT` | `php_fpm_port` |
| `--path` | `CHECK_PHP_FPM_PATH` | `php_fpm_path` |
| `--response` | `CHECK_PHP_FPM_RESPONSE` | `php_fpm_response` |
| `--timeout` | `CHECK_PHP_FPM_TIMEOUT` | `php_fpm_timeout` |

Example:

```sh
check-php-fpm -H 127.0.0.1 -p 80 -q fpm-ping
```

## Check definition

Every option is passed through `env_vars`, filled per host from entity annotations with token substitution (applied by the agent to the whole check definition, `env_vars` included).

Naming rule: an annotation is named after its environment variable, lowercased, without the `CHECK_` prefix (`CHECK_PHP_FPM_HOST` → `php_fpm_host`).

Every option gets an explicit, non-empty `default` in the check definition. Never use `| default ""`: Sensu rejects an empty environment variable (`FOO=`), and the agent then silently drops the check, without any event.

```yaml
---
type: CheckConfig
api_version: core/v2
metadata:
  name: check-php-fpm
spec:
  command: check-php-fpm
  env_vars:
    - CHECK_PHP_FPM_HOST={{ .annotations.php_fpm_host | default "127.0.0.1" }}
    - CHECK_PHP_FPM_PORT={{ .annotations.php_fpm_port | default "80" }}
    - CHECK_PHP_FPM_PATH={{ .annotations.php_fpm_path | default "fpm-ping" }}
    - CHECK_PHP_FPM_RESPONSE={{ .annotations.php_fpm_response | default "pong" }}
    - CHECK_PHP_FPM_TIMEOUT={{ .annotations.php_fpm_timeout | default "10" }}
  runtime_assets:
    - opsone/sensu-plugins-php
  interval: 60
  timeout: 30
  publish: true
  subscriptions:
    - php-fpm
```

Annotations, only for hosts departing from the defaults, e.g. in the agent configuration (`/etc/sensu/agent.yml`):

```yaml
annotations:
  php_fpm_host: localhost
  php_fpm_port: "8080"
  php_fpm_path: status/ping
  php_fpm_response: alive
  php_fpm_timeout: "5"
```

## Migrating from the Ruby version (≤ 0.2.0)

- The command loses its `.rb` suffix: `check-php-fpm.rb` → `check-php-fpm`.
- The host shorthand is now `-H` (`-h` is `--help`).
- `--host`, `--port` and `--path` are required (they defaulted to `127.0.0.1`, `80` and `fpm-ping`); the defaults now live in the check definition.
- A connection error (refused, timeout) is CRITICAL instead of UNKNOWN.
- The response body is compared after trimming whitespace.
- New `--timeout` option (default 10 seconds).
- The asset no longer depends on the Ruby runtime asset, nor on the Debian release.

## Build and release

```sh
make test          # go vet + unit tests
make integration   # runs the check against a real nginx + PHP-FPM in Docker
```

Release:

```sh
git tag 0.3.0 && git push origin 0.3.0
make release       # version taken from the tag, or forced: make release VERSION=0.3.0
sensuctl create -f dist/opsone-sensu-plugins-php_0.3.0.yml
```

`make release` runs `make dist`, which runs the tests then produces in `dist/`:

- one `linux/amd64` archive in Sensu asset layout (`bin/check-php-fpm`);
- the sha512 checksums file;
- `opsone-sensu-plugins-php_<version>.yml`, the asset definition pointing to the GitHub release of the tag.

It then creates the GitHub release of the tag with `gh`, uploads the archives and checksums, and uses the asset definition as release description.

`check-php-fpm version` prints the version, commit and build date.
