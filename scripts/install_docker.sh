#!/bin/bash
# CentOS/RHEL Docker CE Auto Installation Script

set -e

# 1. Ensure running as root
if [ "$EUID" -ne 0 ]; then
  echo "Error: Please run as root."
  exit 1
fi

# 2. Detect package manager
PM="yum"
if command -v dnf >/dev/null 2>&1; then
  PM="dnf"
fi

echo "Installing Docker dependencies..."
$PM install -y yum-utils device-mapper-persistent-data lvm2

echo "Configuring Docker official repository..."
yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo || true

echo "Installing Docker CE..."
$PM install -y docker-ce docker-ce-cli containerd.io

echo "Starting and enabling Docker service..."
systemctl start docker
systemctl enable docker

echo "Docker CE has been successfully installed and started."
