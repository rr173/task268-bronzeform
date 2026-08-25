#!/usr/bin/env bash
# 用法: bash build_benzhi_docker.sh <镜像名> <平台>
# 示例: bash build_benzhi_docker.sh bronzeform linux/amd64
set -euo pipefail

IMAGE_NAME="${1:-bronzeform}"
PLATFORM="${2:-linux/amd64}"

docker buildx build --platform "${PLATFORM}" --load -t "${IMAGE_NAME}" .
echo "built ${IMAGE_NAME} for ${PLATFORM}"
