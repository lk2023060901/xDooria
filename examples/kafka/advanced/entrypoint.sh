#!/bin/bash
# Custom entrypoint to bypass the buggy /etc/kafka/docker/configure script

set -e

# Create log directory
mkdir -p /tmp/kraft-combined-logs

# Format storage if not already formatted
if [ ! -f /tmp/kraft-combined-logs/meta.properties ]; then
    echo "Formatting KRaft storage..."
    CLUSTER_ID=$(/opt/kafka/bin/kafka-storage.sh random-uuid)
    /opt/kafka/bin/kafka-storage.sh format -t $CLUSTER_ID -c /etc/kafka/server.properties
fi

# Start Kafka with custom server.properties
echo "Starting Kafka..."
exec /opt/kafka/bin/kafka-server-start.sh /etc/kafka/server.properties
