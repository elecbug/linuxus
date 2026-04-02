#!/bin/bash

param1=$1

cd docker

if [ ! -f "./data/students.txt" ]; then
    echo "Error: 'students.txt' not found in '$(pwd)/docker/data/'"
    exit 1
fi

./generate_compose.sh ./data/students.txt

sudo docker compose -f ./docker-compose.generated.yml down --remove-orphans $1
sudo docker compose -f ./docker-compose.generated.yml up -d --build

cd ..