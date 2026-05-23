#!/bin/sh
DOCKER_GID=$(stat -c '%g' /var/run/docker.sock 2>/dev/null || echo 0)
if grep -q "^DOCKER_GID=" .env 2>/dev/null; then
    sed -i "s/^DOCKER_GID=.*/DOCKER_GID=$DOCKER_GID/" .env
else
    echo "DOCKER_GID=$DOCKER_GID" >> .env
fi
docker-compose "$@"
