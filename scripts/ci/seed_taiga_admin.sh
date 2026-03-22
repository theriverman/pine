#!/usr/bin/env bash

set -euo pipefail

stack_dir="${1:-.}"
attempts="${TAIGA_SUPERUSER_ATTEMPTS:-60}"
sleep_seconds="${TAIGA_SUPERUSER_SLEEP_SECONDS:-10}"

: "${DJANGO_SUPERUSER_EMAIL:?DJANGO_SUPERUSER_EMAIL must be set}"
: "${DJANGO_SUPERUSER_USERNAME:?DJANGO_SUPERUSER_USERNAME must be set}"
: "${DJANGO_SUPERUSER_PASSWORD:?DJANGO_SUPERUSER_PASSWORD must be set}"

do_createsuperuser() {
  docker compose -f docker-compose.yml -f docker-compose-inits.yml run --rm \
    -e DJANGO_SUPERUSER_EMAIL \
    -e DJANGO_SUPERUSER_USERNAME \
    -e DJANGO_SUPERUSER_PASSWORD \
    taiga-manage createsuperuser --noinput
}


cd "$stack_dir"

for attempt in $(seq 1 "$attempts"); do
	if do_createsuperuser; then
		exit 0
	fi
	if [[ "$attempt" -lt "$attempts" ]]; then
		sleep "$sleep_seconds"
	fi
done

echo "Taiga admin bootstrap did not complete in time" >&2
exit 1
