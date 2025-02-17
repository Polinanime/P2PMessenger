#!/bin/bash

set -e # Exit on error

echo "=== Starting build test ==="

# Clean up any existing containers
echo "Cleaning up existing containers..."
docker-compose -f docker-compose.test.yml down --remove-orphans

# Build the containers
echo "Building containers..."
docker-compose -f docker-compose.test.yml build

echo "=== Build test completed successfully ==="
