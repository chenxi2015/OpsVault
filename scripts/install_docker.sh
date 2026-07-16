#!/bin/bash
# Docker CE Auto Installation Script
# Supports: CentOS / RHEL / TencentOS / AlmaLinux / Rocky (yum/dnf)
#           Ubuntu / Debian (apt)

set -e

# Ensure running as root
if [ "$EUID" -ne 0 ]; then
  echo "Error: Please run as root."
  exit 1
fi

# Load OS information
. /etc/os-release 2>/dev/null || true

echo "Detected OS: $NAME ($ID)"

# ─────────────────────────────────────────────
# Branch by OS family
# ─────────────────────────────────────────────

if command -v apt-get >/dev/null 2>&1; then
  # ── Debian / Ubuntu ──────────────────────────
  echo "Using apt-based installation..."

  apt-get update -y
  apt-get install -y ca-certificates curl

  # Add Docker's official GPG key
  install -m 0755 -d /etc/apt/keyrings
  curl -fsSL "https://mirrors.cloud.tencent.com/docker-ce/linux/${ID}/gpg" \
    -o /etc/apt/keyrings/docker.asc
  chmod a+r /etc/apt/keyrings/docker.asc

  # Add Docker repository (uses VERSION_CODENAME, e.g. jammy / bookworm)
  echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] \
https://mirrors.cloud.tencent.com/docker-ce/linux/${ID} ${VERSION_CODENAME} stable" \
    > /etc/apt/sources.list.d/docker.list

  apt-get update -y
  apt-get install -y docker-ce docker-ce-cli containerd.io

elif command -v dnf >/dev/null 2>&1 || command -v yum >/dev/null 2>&1; then
  # ── CentOS / RHEL / TencentOS / AlmaLinux / Rocky ────
  echo "Using dnf/yum-based installation..."

  PM="yum"
  command -v dnf >/dev/null 2>&1 && PM="dnf"

  # Resolve Docker repo version.
  # Docker repo only has: centos/7, centos/8, centos/9.
  # Priority 1: PLATFORM_ID standard (e.g. platform:el9).
  # Priority 2: fallback via distro ID + VERSION_ID for non-standard PLATFORM_ID
  #             (e.g. TencentOS has platform:tl4 instead of platform:el9).
  case "$PLATFORM_ID" in
    *el9*) DOCKER_REPO_VER=9 ;;
    *el8*) DOCKER_REPO_VER=8 ;;
    *el7*) DOCKER_REPO_VER=7 ;;
    *)
      MAJOR_VER=$(echo "${VERSION_ID:-0}" | cut -d. -f1)
      if [ "$ID" = "tencentos" ]; then
        # TencentOS: 4 -> el9, 3 -> el8, 2 -> el7
        case "$MAJOR_VER" in
          4) DOCKER_REPO_VER=9 ;;
          3) DOCKER_REPO_VER=8 ;;
          *) DOCKER_REPO_VER=7 ;;
        esac
      else
        # Generic: treat major VERSION_ID as CentOS equivalent
        case "$MAJOR_VER" in
          9) DOCKER_REPO_VER=9 ;;
          8) DOCKER_REPO_VER=8 ;;
          *) DOCKER_REPO_VER=7 ;;
        esac
      fi
      ;;
  esac
  echo "Mapping to Docker repo: centos/${DOCKER_REPO_VER}"

  # Write Docker CE repo directly, no yum-utils required
  cat > /etc/yum.repos.d/docker-ce.repo << EOF
[docker-ce-stable]
name=Docker CE Stable
baseurl=https://mirrors.cloud.tencent.com/docker-ce/linux/centos/${DOCKER_REPO_VER}/\$basearch/stable
enabled=1
gpgcheck=1
gpgkey=https://mirrors.cloud.tencent.com/docker-ce/linux/centos/gpg
EOF

  $PM install -y docker-ce docker-ce-cli containerd.io

else
  echo "Error: Unsupported OS or package manager not found."
  exit 1
fi

# ─────────────────────────────────────────────
# Common: start and enable Docker
# ─────────────────────────────────────────────
echo "Starting and enabling Docker service..."
systemctl start docker
systemctl enable docker

echo "Docker CE has been successfully installed and started."
