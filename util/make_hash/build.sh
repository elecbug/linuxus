#!/bin/bash

set -e

current_dir=$(pwd)
sourcec_dir=$(dirname "$(realpath "$0")")
source="main.go"
output="make_hash.out"

cd "$sourcec_dir"

go build -o "../$output" "$source"

cd "$current_dir"