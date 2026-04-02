#!/bin/bash

param=$1

if [ -z "$param" ]; then
    echo "Usage: $0 [--clear-volume|--only-down|--only-up|--restart]"
    exit 1
fi

current_dir=$(pwd)
util_dir=$(dirname "$(realpath "$0")")
source_dir=$(dirname "$util_dir")

cd "$source_dir/docker"

if [ ! -f "./data/students.txt" ]; then
    echo "Error: 'students.txt' not found in '$(pwd)/docker/data/'"
    exit 1
fi

./generate_compose.sh ./data/students.txt

if [ "$param" == "--clear-volume" ]; then
    sudo docker compose -f ./docker-compose.generated.yml down -v --remove-orphans
    sudo docker compose -f ./docker-compose.generated.yml up -d --build
elif [ "$param" == "--only-down" ]; then
    sudo docker compose -f ./docker-compose.generated.yml down --remove-orphans
elif [ "$param" == "--only-up" ]; then
    sudo docker compose -f ./docker-compose.generated.yml up -d --build
elif [ "$param" == "--restart" ]; then
    sudo docker compose -f ./docker-compose.generated.yml down --remove-orphans
    sudo docker compose -f ./docker-compose.generated.yml up -d --build
fi

cd "$current_dir"