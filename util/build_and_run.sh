#!/bin/bash

param1=$1

current_dir=$(pwd)
util_dir=$(dirname "$(realpath "$0")")
source_dir=$(dirname "$util_dir")

cd "$source_dir/docker"

if [ ! -f "./data/students.txt" ]; then
    echo "Error: 'students.txt' not found in '$(pwd)/docker/data/'"
    exit 1
fi

./generate_compose.sh ./data/students.txt

sudo docker compose -f ./docker-compose.generated.yml down --remove-orphans $1
sudo docker compose -f ./docker-compose.generated.yml up -d --build

cd "$current_dir"