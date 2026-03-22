#!/usr/bin/env bash

set -euo pipefail

binary_path="${1:?usage: scripts/smoke-cli.sh /path/to/binary}"

help_output="$("$binary_path" --help)"
version_output="$("$binary_path" --version)"
completion_output="$("$binary_path" completion bash)"

grep -q '^NAME:' <<<"$help_output"
grep -q 'A Taiga CLI built on Taigo v2' <<<"$help_output"
grep -q '^VERSION:' <<<"$help_output"

grep -q '^name: pine$' <<<"$version_output"
grep -q '^version: ' <<<"$version_output"
grep -q '^commit: ' <<<"$version_output"
grep -q '^go: go' <<<"$version_output"

test -n "$completion_output"
