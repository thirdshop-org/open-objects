#!/bin/bash

# Script to build and run the Open Objects application with Docker

set -e

echo "Building Open Objects Docker image..."
docker-compose build

echo "Starting Open Objects server..."
docker-compose up -d

echo "Application is running on http://localhost:8080"
echo "To stop the application, run: docker-compose down"
echo "To view logs, run: docker-compose logs -f"