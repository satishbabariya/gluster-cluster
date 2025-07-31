#!/bin/bash
set -e

echo "Starting GlusterFS node: $NODE_NAME"

# Ensure required directories exist
mkdir -p /var/log/glusterfs
mkdir -p /var/lib/glusterd
mkdir -p /data/glusterfs

# Start glusterd (official image may have its own startup)
if ! pgrep -x "glusterd" > /dev/null; then
    echo "Starting glusterd..."
    /usr/sbin/glusterd --pid-file=/var/run/glusterd.pid --log-file=/var/log/glusterfs/glusterd.log &
    sleep 5
fi

# Start the gluster-node application
exec /usr/local/bin/gluster-node