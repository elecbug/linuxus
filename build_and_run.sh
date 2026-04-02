#!/bin/bash

sudo docker compose -f ./docker/docker-compose.generated.yml down -v
sudo docker compose -f ./docker/docker-compose.generated.yml up -d --build