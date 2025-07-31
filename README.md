# GlusterFS Sync Utility

A Go-based, plug-and-play GlusterFS cluster management utility for multi-node environments with Docker support. This utility provides shared and replicated storage across nodes, making it easy to mount volumes like `/mnt/shared/tomcat-resources:/opt/tomcat/webapps/zenoptics/resources:rw` with automatic replication and synchronization.

## Features

- 🚀 **Plug-and-Play**: Easy deployment with Docker images from registry
- 🔧 **Environment-Based Configuration**: Manage everything via environment variables
- 🔄 **Automatic Replication**: Built-in data replication across nodes
- 📊 **Cluster Management**: Start, stop, monitor cluster with simple commands
- 🐳 **Docker Native**: Full Docker and Docker Compose support
- ☸️ **Kubernetes Ready**: Kubernetes manifests included
- 📈 **Scalable**: Support for 3+ node clusters
- 🛡️ **Production Ready**: Health checks, logging, and monitoring

## Quick Start

### 1. Pull Images from Registry

```bash
# Pull the latest images
docker pull gluster-cluster/manager:latest
docker pull gluster-cluster/node:latest
docker pull gluster-cluster/client:latest
```

### 2. Set Environment Variables

Create your environment configuration:

```bash
# Copy example configuration
cp examples/environment-variables.txt .env

# Edit the configuration
export GLUSTER_CLUSTER_NAME=my-app-cluster
export GLUSTER_NODE_IPS=172.20.0.10,172.20.0.11,172.20.0.12
export GLUSTER_REPLICA_COUNT=3
export GLUSTER_VOLUMES="shared:replicated:3:/mnt/shared/tomcat-resources:/opt/tomcat/webapps/zenoptics/resources"
```

### 3. Start the Cluster

```bash
# Using Makefile
make start

# Or using Docker Compose directly
docker-compose -f deployments/docker/docker-compose.yml up -d
```

### 4. Use Shared Storage

Your application containers can now use the shared volume:

```yaml
services:
  my-app:
    image: tomcat:9-jre11
    volumes:
      - /mnt/shared/tomcat-resources:/opt/tomcat/webapps/zenoptics/resources:rw
```

## Environment Variables Configuration

### Core Settings

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `GLUSTER_CLUSTER_NAME` | Name of the cluster | `gluster-cluster` | `my-app-cluster` |
| `GLUSTER_NODE_IPS` | Comma-separated node IPs | `172.20.0.10,172.20.0.11,172.20.0.12` | `192.168.1.10,192.168.1.11,192.168.1.12` |
| `GLUSTER_REPLICA_COUNT` | Number of replicas | `3` | `5` |
| `GLUSTER_NETWORK_SUBNET` | Docker network subnet | `172.20.0.0/16` | `192.168.100.0/24` |

### Volume Configuration

Configure volumes using the `GLUSTER_VOLUMES` environment variable:

**Format**: `name:type:replica_count:host_paths:mount_point;volume2:...`

**Examples**:
```bash
# Single shared volume
GLUSTER_VOLUMES="shared:replicated:3:/mnt/shared:/mnt/shared"

# Multiple volumes
GLUSTER_VOLUMES="shared:replicated:3:/mnt/shared:/mnt/shared;data:replicated:3:/mnt/data:/mnt/data"

# For your Tomcat use case
GLUSTER_VOLUMES="shared:replicated:3:/mnt/shared/tomcat-resources:/opt/tomcat/webapps/zenoptics/resources"
```

### Docker Images

| Variable | Description | Default |
|----------|-------------|---------|
| `GLUSTER_NODE_IMAGE` | GlusterFS node image | `gluster-cluster/node:latest` |
| `GLUSTER_MANAGER_IMAGE` | Cluster manager image | `gluster-cluster/manager:latest` |
| `GLUSTER_CLIENT_IMAGE` | GlusterFS client image | `gluster-cluster/client:latest` |

## Usage Examples

### Basic 3-Node Cluster

```bash
# Set environment variables
export GLUSTER_CLUSTER_NAME=basic-cluster
export GLUSTER_NODE_IPS=172.20.0.10,172.20.0.11,172.20.0.12
export GLUSTER_REPLICA_COUNT=3
export GLUSTER_VOLUMES="shared:replicated:3:/mnt/shared:/mnt/shared"

# Start cluster
make start

# Check status
make status
```

### Tomcat Application with Shared Resources

```bash
# Configure for Tomcat
export GLUSTER_CLUSTER_NAME=tomcat-cluster
export GLUSTER_VOLUMES="resources:replicated:3:/mnt/shared/tomcat-resources:/opt/tomcat/webapps/zenoptics/resources"

# Start with Tomcat example
make example-tomcat
```

### 5-Node High Availability Cluster

```bash
export GLUSTER_CLUSTER_NAME=ha-cluster
export GLUSTER_NODE_IPS=172.20.0.10,172.20.0.11,172.20.0.12,172.20.0.13,172.20.0.14
export GLUSTER_REPLICA_COUNT=5
export GLUSTER_VOLUMES="data:replicated:5:/mnt/data:/mnt/data"

# Generate and start 5-node configuration
make generate-5-node
docker-compose -f docker-compose-5node.yml up -d
```

## Management Commands

### Using Makefile

```bash
# Build images
make build

# Start cluster
make start

# Check status
make status

# View logs
make logs

# Stop cluster
make stop

# Clean up
make clean

# Run tests
make test
```

### Using Docker Compose

```bash
# Start cluster
docker-compose -f deployments/docker/docker-compose.yml up -d

# Check status
docker-compose -f deployments/docker/docker-compose.yml ps

# View logs
docker-compose -f deployments/docker/docker-compose.yml logs -f

# Stop cluster
docker-compose -f deployments/docker/docker-compose.yml down
```

### Using the Go CLI

```bash
# Build the CLI tools
make dev-build

# Start cluster
./bin/gluster-manager start

# Check status
./bin/gluster-manager status

# Show configuration
./bin/gluster-manager config

# Stop cluster
./bin/gluster-manager stop
```

## Kubernetes Deployment

Deploy to Kubernetes with environment-based configuration:

```bash
# Deploy to Kubernetes
kubectl apply -f deployments/k8s/

# Check status
kubectl get all -n gluster-system

# Update configuration
kubectl edit configmap gluster-config -n gluster-system

# Delete deployment
kubectl delete -f deployments/k8s/
```

## Custom Configuration

### Using Configuration File

Create `gluster-config.yaml`:

```yaml
cluster_name: "my-cluster"
node_ips:
  - "172.20.0.10"
  - "172.20.0.11"
  - "172.20.0.12"
replica_count: 3
volumes:
  - name: "shared"
    type: "replicated"
    replica_count: 3
    host_paths:
      - "/mnt/shared/tomcat-resources"
    mount_point: "/opt/tomcat/webapps/zenoptics/resources"
    options:
      "performance.cache-size": "512MB"
```

### Using Docker Registry

Push to your own registry:

```bash
# Build and tag for your registry
make build REGISTRY=your-registry.com/gluster-cluster

# Push to registry
make push REGISTRY=your-registry.com/gluster-cluster

# Use in deployment
export GLUSTER_NODE_IMAGE=your-registry.com/gluster-cluster/node:latest
export GLUSTER_MANAGER_IMAGE=your-registry.com/gluster-cluster/manager:latest
```

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   GlusterFS     │    │   GlusterFS     │    │   GlusterFS     │
│     Node 1      │◄──►│     Node 2      │◄──►│     Node 3      │
│ (172.20.0.10)   │    │ (172.20.0.11)   │    │ (172.20.0.12)   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         ▲                       ▲                       ▲
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │     Cluster     │
                    │     Manager     │
                    │ (Go Application)│
                    └─────────────────┘
                                 │
                    ┌─────────────────┐
                    │   Application   │
                    │   Containers    │
                    │ (Tomcat, etc.)  │
                    └─────────────────┘
```

## Monitoring and Troubleshooting

### Health Checks

```bash
# Check cluster health
make status

# View detailed logs
make logs

# Test cluster connectivity
make test

# Check specific node
docker exec gluster-cluster-node1 gluster peer status
docker exec gluster-cluster-node1 gluster volume status
```

### Common Issues

1. **Nodes not connecting**: Check network connectivity and firewall rules
2. **Volume mount fails**: Ensure GlusterFS client is properly configured
3. **Performance issues**: Adjust cache settings and replica count
4. **Split-brain scenarios**: Use healing commands to resolve

### Logging

Logs are available at multiple levels:

```bash
# Container logs
docker logs gluster-cluster-node1

# GlusterFS logs (inside container)
docker exec gluster-cluster-node1 tail -f /var/log/glusterfs/glusterd.log

# Application logs
export GLUSTER_LOG_LEVEL=debug
```

## Building from Source

```bash
# Clone repository
git clone https://github.com/your-org/gluster-cluster
cd gluster-cluster

# Setup development environment
make dev-setup

# Build binaries
make dev-build

# Run tests
make dev-test

# Build Docker images
make build

# Push to registry
make push REGISTRY=your-registry.com/gluster-cluster
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

MIT License - see LICENSE file for details.

## Support

For issues and questions:
- Create an issue on GitHub
- Check the troubleshooting section
- Review logs for error details

---

**Ready to use shared, replicated storage in your multi-node environment? Just set your environment variables and run `make start`!** 🚀