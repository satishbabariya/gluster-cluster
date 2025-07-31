#!/bin/bash
set -e

echo "Starting GlusterFS client with official image..."
echo "Cluster: $GLUSTER_CLUSTER_NAME"
echo "Server: $GLUSTER_SERVER"
echo "Volume: $GLUSTER_VOLUME"
echo "Mount Point: $MOUNT_POINT"

# Wait for GlusterFS server to be ready
echo "Waiting for GlusterFS server to be ready..."
for i in $(seq 1 $RETRY_COUNT); do
    if ping -c 1 $GLUSTER_SERVER > /dev/null 2>&1; then
        echo "Server $GLUSTER_SERVER is reachable"
        # Additional check: test gluster port
        if timeout 5 bash -c "</dev/tcp/$GLUSTER_SERVER/24007" 2>/dev/null; then
            echo "GlusterFS port 24007 is accessible"
            break
        fi
    fi
    echo "Waiting for server... (attempt $i/$RETRY_COUNT)"
    sleep $RETRY_DELAY
done

# Wait additional time for GlusterFS to be fully ready
sleep 30

# Create mount point
mkdir -p "$MOUNT_POINT"

# Mount the volume using the latest GlusterFS client
echo "Mounting GlusterFS volume with official client..."
mount.glusterfs "$GLUSTER_SERVER:/$GLUSTER_VOLUME" "$MOUNT_POINT"

if mount | grep -q "$GLUSTER_SERVER:/$GLUSTER_VOLUME"; then
    echo "Successfully mounted GlusterFS volume!"
    echo "Mount point: $MOUNT_POINT"
    ls -la "$MOUNT_POINT/"
else
    echo "ERROR: Failed to mount GlusterFS volume"
    exit 1
fi

# Set permissions
chmod 755 "$MOUNT_POINT"
chown root:root "$MOUNT_POINT"

echo "GlusterFS client setup completed with official image!"

# Keep container running
tail -f /dev/null