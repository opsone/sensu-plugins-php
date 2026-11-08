#!/usr/bin/env bash
# Runs check-php-fpm against a real nginx + PHP-FPM inside a Docker container.
set -euo pipefail
cd "$(dirname "$0")"

image=check-php-fpm-integration
container=check-php-fpm-integration-$$

docker build -q -t "$image" . >/dev/null
docker run -d --rm --name "$container" "$image" >/dev/null
trap 'docker rm -f "$container" >/dev/null' EXIT

docker exec "$container" bash -c 'php-fpm8.4 && nginx'
for _ in $(seq 30); do
  docker exec "$container" test -S /run/php/php8.4-fpm.sock && break
  sleep 1
done

failures=0
expect() {
  local want=$1 desc=$2; shift 2
  local out rc=0
  # Run as an unprivileged user, like sensu-agent does.
  out=$(docker exec "$container" runuser -u nobody -- "$@" 2>&1) || rc=$?
  out=$(tail -1 <<<"$out")
  if [[ $rc -eq $want ]]; then
    echo "ok   [$rc] $desc -> $out"
  else
    echo "FAIL [$rc, want $want] $desc -> $out"
    failures=$((failures + 1))
  fi
}

args=(check-php-fpm -H 127.0.0.1 -p 80 -q fpm-ping)

expect 3 "without host" check-php-fpm -p 80 -q fpm-ping
expect 3 "without port" check-php-fpm -H 127.0.0.1 -q fpm-ping
expect 3 "without path" check-php-fpm -H 127.0.0.1 -p 80
expect 3 "invalid port" "${args[@]}" -p 70000
expect 3 "invalid timeout" "${args[@]}" --timeout 0
expect 0 "pong" "${args[@]}"
expect 0 "path with leading slash" "${args[@]}" -q /fpm-ping
expect 0 "environment variables" env CHECK_PHP_FPM_HOST=localhost CHECK_PHP_FPM_PORT=80 \
  CHECK_PHP_FPM_PATH=fpm-ping CHECK_PHP_FPM_RESPONSE=pong CHECK_PHP_FPM_TIMEOUT=5 check-php-fpm
expect 2 "wrong response" "${args[@]}" -r alive
expect 2 "page not found" "${args[@]}" -q nope
expect 2 "redirect not followed" "${args[@]}" -q redirect
expect 2 "port closed" "${args[@]}" -p 81

docker exec "$container" bash -c 'kill "$(cat /run/php/php8.4-fpm.pid)"'
for _ in $(seq 30); do
  docker exec "$container" test -S /run/php/php8.4-fpm.sock || break
  sleep 1
done
expect 2 "php-fpm stopped" "${args[@]}"

exit $failures
