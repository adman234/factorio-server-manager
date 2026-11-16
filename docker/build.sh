#!/bin/bash
# Builds the image from the local checkout (frontend and backend are compiled inside Docker).
set -euo pipefail
cd "$(dirname "$0")/.."
docker build -f docker/Dockerfile -t factorio-server-manager:dev "$@" .
