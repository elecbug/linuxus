#!/bin/bash

cd docker

./generate_compose.sh ./data/students.txt

sudo docker compose -f ./docker-compose.generated.yml down -v --remove-orphans
sudo docker compose -f ./docker-compose.generated.yml up -d --build

cd ..