#!/bin/bash

set -e

# ============================ #
# Docker Reinstallation Script #
# ============================ #

# 1. Stop Docker and Containerd services
sudo systemctl stop docker || true
sudo systemctl stop containerd || true

# 2. Remove all containers, images, volumes, and networks
sudo docker ps -aq | xargs -r sudo docker rm -f
sudo docker images -aq | xargs -r sudo docker rmi -f
sudo docker volume ls -q | xargs -r sudo docker volume rm
sudo docker network ls -q | xargs -r sudo docker network rm

# 3. Uninstall Docker packages
sudo apt purge -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin docker.io || true
sudo apt autoremove -y

# 4. Remove Docker directories and configurations
sudo rm -rf /var/lib/docker
sudo rm -rf /var/lib/containerd
sudo rm -rf /etc/docker
sudo rm -rf /home/$USER/.docker

# 5. Remove Docker network interfaces
sudo ip link delete docker0 || true

# 6. Reset Docker repository
sudo rm -f /etc/apt/sources.list.d/docker.list

# 7. Add Kubernetes and Docker repositories
sudo mkdir -p /etc/apt/keyrings

curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.28/deb/Release.key \
  | sudo gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg

echo "deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.28/deb/ /" \
  | sudo tee /etc/apt/sources.list.d/kubernetes.list > /dev/null

# 8. Add Docker repository
sudo apt update
sudo apt install -y ca-certificates curl gnupg

sudo install -m 0755 -d /etc/apt/keyrings

curl -fsSL https://download.docker.com/linux/ubuntu/gpg \
  | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg

echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
https://download.docker.com/linux/ubuntu \
$(. /etc/os-release && echo $VERSION_CODENAME) stable" \
| sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

# 9. Reinstall Docker packages
sudo apt update
sudo apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# 10. Start and enable Docker service
sudo systemctl start docker
sudo systemctl enable docker

# 11. Verify Docker installation
sudo docker run hello-world