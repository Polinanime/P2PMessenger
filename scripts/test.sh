#!/bin/bash

set -e # Exit on error

# Initialize peers
./scripts/init-peers.sh

# Clean up any existing containers
echo "Cleaning up existing containers..."
docker-compose -f docker-compose.test.yml down --remove-orphans

# Start the containers with network partition
echo "Starting containers..."
docker-compose -f docker-compose.test.yml up -d

# Function to wait for container to be ready
wait_for_container() {
    local container=$1
    local max_attempts=30
    local attempt=1

    echo "Waiting for $container to be ready..."
    while [ $attempt -le $max_attempts ]; do
        if docker container inspect "$container" >/dev/null 2>&1; then
            if [ "$(docker container inspect -f '{{.State.Running}}' "$container")" = "true" ]; then
                echo "$container is ready!"
                return 0
            fi
        fi
        echo "Attempt $attempt: $container is not ready yet..."
        sleep 1
        attempt=$((attempt + 1))
    done

    echo "Error: $container failed to start after $max_attempts attempts"
    return 1
}

# Wait for all containers to be ready
wait_for_container "messenger1" || exit 1
wait_for_container "messenger2" || exit 1
wait_for_container "messenger3" || exit 1

# Additional delay to ensure services are fully initialized
echo "Waiting for services to initialize..."
sleep 10

# Function to execute command with retry
execute_with_retry() {
    local container=$1
    local cmd=$2
    local max_attempts=3
    local attempt=1

    while [ $attempt -le $max_attempts ]; do
        if docker exec -i "$container" $cmd; then
            return 0
        fi
        echo "Command failed, retrying ($attempt/$max_attempts)..."
        sleep 2
        attempt=$((attempt + 1))
    done

    echo "Error: Command failed after $max_attempts attempts"
    return 1
}

echo "Testing scenario 1: Message from Adam to Eve (should fail initially)"
execute_with_retry messenger1 "./messenger start Adam 1235 send Eve 'Hello Eve!'"

echo "Testing scenario 2: Message from Eve to Bob (should work)"
execute_with_retry messenger2 "./messenger start Eve 1236 send Bob 'Hello Bob!'"

echo "Testing scenario 3: Checking DHT state"
echo "Adam's DHT:"
execute_with_retry messenger1 "./messenger start Adam 1235 list"
echo "Eve's DHT:"
execute_with_retry messenger2 "./messenger start Eve 1236 list"
echo "Bob's DHT:"
execute_with_retry messenger3 "./messenger start Bob 1237 list"

echo "Testing scenario 4: Force network scan"
execute_with_retry messenger1 "./messenger start Adam 1235 scan"
sleep 5

echo "Testing scenario 5: Try sending message again after scan"
execute_with_retry messenger1 "./messenger start Adam 1235 send Eve 'Hello Eve after scan!'"

# Print logs from all containers
echo "Printing container logs..."
docker logs messenger1
docker logs messenger2
docker logs messenger3

# Clean up
echo "Cleaning up..."
docker-compose -f docker-compose.test.yml down
