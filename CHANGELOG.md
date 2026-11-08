# Changelog

## [0.3.0] - 2026-11-08

### Changed
- Rewritten in Go with `github.com/sensu/sensu-plugin-sdk`: static `linux/amd64` binary, no Ruby runtime asset, one asset for every Debian release.
- The command is `check-php-fpm` (no `.rb` suffix).
- The host shorthand is `-H` (`-h` is `--help`).
- `--host`, `--port` and `--path` are required; their former defaults (`127.0.0.1`, `80`, `fpm-ping`) move to the check definition.
- A connection error (refused, timeout) is CRITICAL instead of UNKNOWN.
- The response body is compared after trimming whitespace.

### Added
- Environment variables for every option: `CHECK_PHP_FPM_HOST`, `CHECK_PHP_FPM_PORT`, `CHECK_PHP_FPM_PATH`, `CHECK_PHP_FPM_RESPONSE`, `CHECK_PHP_FPM_TIMEOUT`.
- `--timeout` option (default 10 seconds).
- `--path` accepted with or without a leading slash.
- Integration test against a real nginx + PHP-FPM in Docker.

## [0.2.0]

Last Ruby version.
