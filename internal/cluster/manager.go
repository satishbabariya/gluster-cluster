package cluster

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/gluster-cluster/gluster-sync/internal/config"
	"github.com/sirupsen/logrus"
)

// Manager manages the GlusterFS cluster
type Manager struct {
	config       *config.Config
	dockerClient *client.Client
	logger       *logrus.Logger
}

// NewManager creates a new cluster manager
func NewManager(cfg *config.Config, logger *logrus.Logger) (*Manager, error) {
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	return &Manager{
		config:       cfg,
		dockerClient: dockerClient,
		logger:       logger,
	}, nil
}

// StartCluster starts the GlusterFS cluster
func (m *Manager) StartCluster(ctx context.Context) error {
	m.logger.Info("Starting GlusterFS cluster...")

	// Create network
	if err := m.createNetwork(ctx); err != nil {
		return fmt.Errorf("failed to create network: %w", err)
	}

	// Start nodes
	if err := m.startNodes(ctx); err != nil {
		return fmt.Errorf("failed to start nodes: %w", err)
	}

	// Wait for nodes to be ready
	if err := m.waitForNodes(ctx); err != nil {
		return fmt.Errorf("failed to wait for nodes: %w", err)
	}

	// Initialize cluster
	if err := m.initializeCluster(ctx); err != nil {
		return fmt.Errorf("failed to initialize cluster: %w", err)
	}

	// Create volumes
	if err := m.createVolumes(ctx); err != nil {
		return fmt.Errorf("failed to create volumes: %w", err)
	}

	m.logger.Info("GlusterFS cluster started successfully")
	return nil
}

// StopCluster stops the GlusterFS cluster
func (m *Manager) StopCluster(ctx context.Context) error {
	m.logger.Info("Stopping GlusterFS cluster...")

	// Stop all cluster containers
	containers, err := m.dockerClient.ContainerList(ctx, types.ContainerListOptions{
		All: true,
	})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	for _, c := range containers {
		for _, name := range c.Names {
			if strings.Contains(name, m.config.ClusterName) {
				m.logger.Infof("Stopping container: %s", name)
				if err := m.dockerClient.ContainerStop(ctx, c.ID, container.StopOptions{}); err != nil {
					m.logger.Warnf("Failed to stop container %s: %v", name, err)
				}
				if err := m.dockerClient.ContainerRemove(ctx, c.ID, types.ContainerRemoveOptions{Force: true}); err != nil {
					m.logger.Warnf("Failed to remove container %s: %v", name, err)
				}
			}
		}
	}

	// Remove network
	if err := m.removeNetwork(ctx); err != nil {
		m.logger.Warnf("Failed to remove network: %v", err)
	}

	m.logger.Info("GlusterFS cluster stopped successfully")
	return nil
}

// GetClusterStatus returns the status of the cluster
func (m *Manager) GetClusterStatus(ctx context.Context) (*ClusterStatus, error) {
	status := &ClusterStatus{
		ClusterName: m.config.ClusterName,
		Nodes:       make([]NodeStatus, 0),
		Volumes:     make([]VolumeStatus, 0),
	}

	// Get node statuses
	for i, nodeIP := range m.config.NodeIPs {
		nodeName := m.config.GetNodeName(i)
		nodeStatus := NodeStatus{
			Name: nodeName,
			IP:   nodeIP,
		}

		// Check if container is running
		containers, err := m.dockerClient.ContainerList(ctx, types.ContainerListOptions{})
		if err == nil {
			for _, c := range containers {
				for _, name := range c.Names {
					if strings.Contains(name, nodeName) {
						nodeStatus.Status = "running"
						nodeStatus.ContainerID = c.ID
						break
					}
				}
			}
		}

		if nodeStatus.Status == "" {
			nodeStatus.Status = "stopped"
		}

		status.Nodes = append(status.Nodes, nodeStatus)
	}

	return status, nil
}

// createNetwork creates the Docker network for the cluster
func (m *Manager) createNetwork(ctx context.Context) error {
	networkName := m.config.NetworkName

	// Check if network already exists
	networks, err := m.dockerClient.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		return err
	}

	for _, net := range networks {
		if net.Name == networkName {
			m.logger.Infof("Network %s already exists", networkName)
			return nil
		}
	}

	// Create network
	_, err = m.dockerClient.NetworkCreate(ctx, networkName, types.NetworkCreate{
		Driver: "bridge",
		IPAM: &network.IPAM{
			Config: []network.IPAMConfig{
				{
					Subnet: m.config.NetworkSubnet,
				},
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to create network %s: %w", networkName, err)
	}

	m.logger.Infof("Created network: %s", networkName)
	return nil
}

// removeNetwork removes the Docker network
func (m *Manager) removeNetwork(ctx context.Context) error {
	return m.dockerClient.NetworkRemove(ctx, m.config.NetworkName)
}

// startNodes starts all GlusterFS nodes
func (m *Manager) startNodes(ctx context.Context) error {
	for i, nodeIP := range m.config.NodeIPs {
		nodeName := m.config.GetNodeName(i)
		if err := m.startNode(ctx, nodeName, nodeIP, i); err != nil {
			return fmt.Errorf("failed to start node %s: %w", nodeName, err)
		}
	}
	return nil
}

// startNode starts a single GlusterFS node
func (m *Manager) startNode(ctx context.Context, nodeName, nodeIP string, index int) error {
	m.logger.Infof("Starting node: %s (%s)", nodeName, nodeIP)

	// Container configuration
	containerConfig := &container.Config{
		Image:    m.config.GlusterImage,
		Hostname: nodeName,
		Env: []string{
			fmt.Sprintf("NODE_NAME=%s", nodeName),
			fmt.Sprintf("NODE_IP=%s", nodeIP),
			fmt.Sprintf("CLUSTER_NAME=%s", m.config.ClusterName),
		},
		ExposedPorts: nat.PortSet{
			"24007/tcp": {},
			"49152/tcp": {},
		},
	}

	// Host configuration
	hostConfig := &container.HostConfig{
		Privileged: true,
		Binds: []string{
			fmt.Sprintf("%s-%s:/data/glusterfs", m.config.ClusterName, nodeName),
			"/sys/fs/cgroup:/sys/fs/cgroup:ro",
		},
		PortBindings: nat.PortMap{
			"24007/tcp": []nat.PortBinding{
				{
					HostPort: fmt.Sprintf("%d", 24007+index),
				},
			},
			"49152/tcp": []nat.PortBinding{
				{
					HostPort: fmt.Sprintf("%d", 49152+index),
				},
			},
		},
	}

	// Network configuration
	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			m.config.NetworkName: {
				IPAMConfig: &network.EndpointIPAMConfig{
					IPv4Address: nodeIP,
				},
			},
		},
	}

	// Create container
	resp, err := m.dockerClient.ContainerCreate(ctx, containerConfig, hostConfig, networkConfig, nil, nodeName)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	// Start container
	if err := m.dockerClient.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	m.logger.Infof("Node %s started successfully", nodeName)
	return nil
}

// waitForNodes waits for all nodes to be ready
func (m *Manager) waitForNodes(ctx context.Context) error {
	m.logger.Info("Waiting for nodes to be ready...")

	timeout := time.After(120 * time.Second)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for nodes to be ready")
		case <-ticker.C:
			allReady := true
			for i := range m.config.NodeIPs {
				nodeName := m.config.GetNodeName(i)
				if !m.isNodeReady(ctx, nodeName) {
					allReady = false
					break
				}
			}
			if allReady {
				m.logger.Info("All nodes are ready")
				return nil
			}
		}
	}
}

// isNodeReady checks if a node is ready
func (m *Manager) isNodeReady(ctx context.Context, nodeName string) bool {
	// Execute gluster command to check if glusterd is running
	execConfig := types.ExecConfig{
		Cmd: []string{"systemctl", "is-active", "glusterd"},
	}

	execResp, err := m.dockerClient.ContainerExecCreate(ctx, nodeName, execConfig)
	if err != nil {
		return false
	}

	err = m.dockerClient.ContainerExecStart(ctx, execResp.ID, types.ExecStartCheck{})
	return err == nil
}

// initializeCluster initializes the GlusterFS cluster
func (m *Manager) initializeCluster(ctx context.Context) error {
	m.logger.Info("Initializing GlusterFS cluster...")

	primaryNode := m.config.GetNodeName(0)

	// Probe peers
	for i := 1; i < len(m.config.NodeIPs); i++ {
		peerNode := m.config.GetNodeName(i)
		cmd := fmt.Sprintf("gluster peer probe %s", peerNode)

		if err := m.execCommand(ctx, primaryNode, cmd); err != nil {
			m.logger.Warnf("Failed to probe peer %s: %v", peerNode, err)
		} else {
			m.logger.Infof("Successfully probed peer: %s", peerNode)
		}
	}

	// Wait for peers to connect
	time.Sleep(10 * time.Second)

	m.logger.Info("Cluster initialization completed")
	return nil
}

// createVolumes creates all configured volumes
func (m *Manager) createVolumes(ctx context.Context) error {
	for _, volume := range m.config.Volumes {
		if err := m.createVolume(ctx, volume); err != nil {
			return fmt.Errorf("failed to create volume %s: %w", volume.Name, err)
		}
	}
	return nil
}

// createVolume creates a single volume
func (m *Manager) createVolume(ctx context.Context, volume config.VolumeConfig) error {
	m.logger.Infof("Creating volume: %s", volume.Name)

	primaryNode := m.config.GetNodeName(0)

	// Create volume directories on all nodes
	for i := range m.config.NodeIPs {
		nodeName := m.config.GetNodeName(i)
		cmd := fmt.Sprintf("mkdir -p %s/%s", m.config.DataPath, volume.Name)
		if err := m.execCommand(ctx, nodeName, cmd); err != nil {
			m.logger.Warnf("Failed to create directory on %s: %v", nodeName, err)
		}
	}

	// Build volume create command
	var cmd strings.Builder
	cmd.WriteString(fmt.Sprintf("gluster volume create %s", volume.Name))

	if volume.Type == "replicated" || volume.Type == "" {
		cmd.WriteString(fmt.Sprintf(" replica %d", volume.ReplicaCount))
	}

	// Add brick paths
	for i := 0; i < volume.ReplicaCount && i < len(m.config.NodeIPs); i++ {
		nodeName := m.config.GetNodeName(i)
		cmd.WriteString(fmt.Sprintf(" %s:%s/%s", nodeName, m.config.DataPath, volume.Name))
	}

	// Create volume
	if err := m.execCommand(ctx, primaryNode, cmd.String()); err != nil {
		m.logger.Warnf("Volume %s may already exist: %v", volume.Name, err)
	}

	// Start volume
	startCmd := fmt.Sprintf("gluster volume start %s", volume.Name)
	if err := m.execCommand(ctx, primaryNode, startCmd); err != nil {
		m.logger.Warnf("Failed to start volume %s: %v", volume.Name, err)
	}

	// Configure volume options
	m.configureVolume(ctx, primaryNode, volume)

	m.logger.Infof("Volume %s created successfully", volume.Name)
	return nil
}

// configureVolume configures volume options
func (m *Manager) configureVolume(ctx context.Context, nodeName string, volume config.VolumeConfig) {
	options := map[string]string{
		"auth.allow":                           m.config.NetworkSubnet,
		"server.allow-insecure":                "on",
		"client.allow-insecure":                "on",
		"performance.cache-size":               m.config.CacheSize,
		"performance.write-behind-window-size": m.config.WriteBufferSize,
	}

	// Add custom options
	for key, value := range volume.Options {
		options[key] = value
	}

	// Apply options
	for key, value := range options {
		cmd := fmt.Sprintf("gluster volume set %s %s %s", volume.Name, key, value)
		if err := m.execCommand(ctx, nodeName, cmd); err != nil {
			m.logger.Warnf("Failed to set option %s=%s for volume %s: %v", key, value, volume.Name, err)
		}
	}
}

// execCommand executes a command in a container
func (m *Manager) execCommand(ctx context.Context, containerName, command string) error {
	execConfig := types.ExecConfig{
		Cmd: []string{"sh", "-c", command},
	}

	execResp, err := m.dockerClient.ContainerExecCreate(ctx, containerName, execConfig)
	if err != nil {
		return err
	}

	return m.dockerClient.ContainerExecStart(ctx, execResp.ID, types.ExecStartCheck{})
}

// Close closes the manager and cleans up resources
func (m *Manager) Close() error {
	if m.dockerClient != nil {
		return m.dockerClient.Close()
	}
	return nil
}

// ClusterStatus represents the status of the cluster
type ClusterStatus struct {
	ClusterName string         `json:"cluster_name"`
	Nodes       []NodeStatus   `json:"nodes"`
	Volumes     []VolumeStatus `json:"volumes"`
}

// NodeStatus represents the status of a node
type NodeStatus struct {
	Name        string `json:"name"`
	IP          string `json:"ip"`
	Status      string `json:"status"`
	ContainerID string `json:"container_id,omitempty"`
}

// VolumeStatus represents the status of a volume
type VolumeStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Type   string `json:"type"`
}
