#!/bin/bash
# Docker CE Auto Installation Script for CentOS/RHEL-compatible systems

set -e

# Ensure running as root
if [ "$EUID" -ne 0 ]; then
  echo "Error: Please run as root."
  exit 1
fi

# Detect package manager
PM="yum"
if command -v dnf >/dev/null 2>&1; then
  PM="dnf"
fi

# Resolve the Docker repo version.
# Docker repo only has: centos/7, centos/8, centos/9.
# Priority 1: use PLATFORM_ID (e.g. platform:el9) - standard for CentOS/RHEL/AlmaLinux/Rocky.
# Priority 2: fallback to VERSION_ID mapping for distros with non-standard PLATFORM_ID
#             (e.g. TencentOS has platform:tl4 instead of platform:el9).
. /etc/os-release 2>/dev/null || true

case "$PLATFORM_ID" in
  *el9*) DOCKER_REPO_VER=9 ;;
  *el8*) DOCKER_REPO_VER=8 ;;
  *el7*) DOCKER_REPO_VER=7 ;;
  *)
    # Fallback: map by distro ID + major VERSION_ID
    MAJOR_VER=$(echo "${VERSION_ID:-0}" | cut -d. -f1)
    if [ "$ID" = "tencentos" ]; then
      # TencentOS 4 -> RHEL9-compatible, 3 -> RHEL8-compatible, 2 -> RHEL7-compatible
      case "$MAJOR_VER" in
        4) DOCKER_REPO_VER=9 ;;
        3) DOCKER_REPO_VER=8 ;;
        *) DOCKER_REPO_VER=7 ;;
      esac
    else
      # Generic fallback: treat major version as CentOS equivalent
      case "$MAJOR_VER" in
        9) DOCKER_REPO_VER=9 ;;
        8) DOCKER_REPO_VER=8 ;;
        *) DOCKER_REPO_VER=7 ;;
      esac
    fi
    ;;
esac
echo "Detected platform: ${PLATFORM_ID:-$ID $VERSION_ID} -> using Docker repo for CentOS $DOCKER_REPO_VER"

echo "Adding Docker CE repository..."
cat > /etc/yum.repos.d/docker-ce.repo << EOF
[docker-ce-stable]
name=Docker CE Stable
baseurl=https://mirrors.cloud.tencent.com/docker-ce/linux/centos/${DOCKER_REPO_VER}/\$basearch/stable
enabled=1
gpgcheck=1
gpgkey=https://mirrors.cloud.tencent.com/docker-ce/linux/centos/gpg
EOF

echo "Installing Docker CE..."
$PM install -y docker-ce docker-ce-cli containerd.io

echo "Starting and enabling Docker service..."
systemctl start docker
systemctl enable docker

echo "Docker CE has been successfully installed and started."
