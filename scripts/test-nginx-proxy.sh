#!/bin/sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
config=${1:-"$root/frontend/nginx.conf"}
image=${NGINX_TEST_IMAGE:-ghcr.io/costuff/doyou-world-scraper-frontend:latest}
prefix="doyou-proxy-test-$$"
network="$prefix-network"
backend="$prefix-backend"
frontend="$prefix-frontend"
parking="$prefix-parking"

cleanup() {
    docker rm -f "$frontend" "$backend" "$parking" >/dev/null 2>&1 || true
    docker network rm "$network" >/dev/null 2>&1 || true
}
trap cleanup EXIT
trap 'exit 1' HUP INT TERM

start_backend() {
    docker run -d --pull=never --name "$backend" --network "$network" \
        --network-alias doyou-api --network-alias doyou-analytics-api \
        --entrypoint sh "$image" -c '
            printf "%s\n" "events {}" \
                "http { server { listen 3001; listen 8080; location / { return 200 \"\$request_uri\"; } } }" \
                > /tmp/proxy-test.conf
            exec nginx -c /tmp/proxy-test.conf -g "daemon off;"
        ' >/dev/null
}

check_route() {
    path=$1
    expected=$2
    attempts=0
    while [ "$attempts" -lt 30 ]; do
        if actual=$(docker exec "$frontend" wget -qO- "http://127.0.0.1$path") &&
            [ "$actual" = "$expected" ]; then
            return 0
        fi
        attempts=$((attempts + 1))
        sleep 1
    done
    printf 'Proxy failed: %s (expected %s)\n' "$path" "$expected" >&2
    docker logs --tail 10 "$frontend" >&2
    return 1
}

check_routes() {
    check_route '/api/v1/health?probe=scraper' '/api/v1/health?probe=scraper'
    check_route '/analytics-api/api/v1/tracks?limit=50&offset=7&search=a%20b' \
        '/api/v1/tracks?limit=50&offset=7&search=a%20b'
}

docker network create "$network" >/dev/null
start_backend
docker run -d --pull=never --name "$frontend" --network "$network" \
    --mount "type=bind,src=$config,dst=/etc/nginx/conf.d/default.conf,readonly" \
    "$image" >/dev/null
check_routes

old_ip=$(docker inspect --format '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "$backend")
docker rm -f "$backend" >/dev/null
docker run -d --pull=never --name "$parking" --network "$network" --ip "$old_ip" \
    --entrypoint sleep "$image" 120 >/dev/null
start_backend
new_ip=$(docker inspect --format '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "$backend")
test "$old_ip" != "$new_ip"
check_routes
printf 'PASS: both proxies recover after backend IP changes; paths and queries are preserved.\n'
