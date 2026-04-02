#!/bin/bash

param1=$1

cd docker

./generate_compose.sh ./data/students.txt

sudo docker compose -f ./docker-compose.generated.yml down --remove-orphans $1
sudo docker compose -f ./docker-compose.generated.yml up -d --build

cd ..