package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

// Config holds the configuration for the GlusterFS cluster
type Config struct {
	// Cluster configuration
	ClusterName   string   `mapstructure:"cluster_name" yaml:"cluster_name"`
	NodeIPs       []string `mapstructure:"node_ips" yaml:"node_ips"`
	ReplicaCount  int      `mapstructure:"replica_count" yaml:"replica_count"`
	NetworkSubnet string   `mapstructure:"network_subnet" yaml:"network_subnet"`

	// Volume configuration
	Volumes []VolumeConfig `mapstructure:"volumes" yaml:"volumes"`

	// Docker configuration (Manager removed - peer-to-peer architecture)
	GlusterImage string `mapstructure:"gluster_image" yaml:"gluster_image"`
	NetworkName  string `mapstructure:"network_name" yaml:"network_name"`

	// Storage configuration
	DataPath   string `mapstructure:"data_path" yaml:"data_path"`
	BackupPath string `mapstructure:"backup_path" yaml:"backup_path"`

	// Performance settings
	CacheSize       string `mapstructure:"cache_size" yaml:"cache_size"`
	WriteBufferSize string `mapstructure:"write_buffer_size" yaml:"write_buffer_size"`
	AllowInsecure   bool   `mapstructure:"allow_insecure" yaml:"allow_insecure"`

	// Logging
	LogLevel string `mapstructure:"log_level" yaml:"log_level"`
	LogPath  string `mapstructure:"log_path" yaml:"log_path"`
}

// VolumeConfig defines a volume to be created and managed
type VolumeConfig struct {
	Name         string            `mapstructure:"name" yaml:"name"`
	Type         string            `mapstructure:"type" yaml:"type"` // replicated, distributed, etc.
	ReplicaCount int               `mapstructure:"replica_count" yaml:"replica_count"`
	HostPaths    []string          `mapstructure:"host_paths" yaml:"host_paths"`
	MountPoint   string            `mapstructure:"mount_point" yaml:"mount_point"`
	Options      map[string]string `mapstructure:"options" yaml:"options"`
}

// LoadConfig loads configuration from environment variables and config files
func LoadConfig() (*Config, error) {
	// Set default values
	viper.SetDefault("cluster_name", "gluster-cluster")
	viper.SetDefault("replica_count", 3)
	viper.SetDefault("network_subnet", "172.20.0.0/16")
	viper.SetDefault("gluster_image", "gluster/gluster-centos:latest")
	// Manager image removed - peer-to-peer architecture
	viper.SetDefault("network_name", "gluster-net")
	viper.SetDefault("data_path", "/data/glusterfs")
	viper.SetDefault("backup_path", "./backups")
	viper.SetDefault("cache_size", "256MB")
	viper.SetDefault("write_buffer_size", "4MB")
	viper.SetDefault("allow_insecure", true)
	viper.SetDefault("log_level", "info")
	viper.SetDefault("log_path", "/var/log/glusterfs")

	// Environment variable prefix
	viper.SetEnvPrefix("GLUSTER")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Try to read config file if it exists
	viper.SetConfigName("gluster-config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/gluster-sync")

	// Read config file (optional)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Parse node IPs from environment variable if not set in config
	if len(config.NodeIPs) == 0 {
		nodeIPsStr := os.Getenv("GLUSTER_NODE_IPS")
		if nodeIPsStr != "" {
			config.NodeIPs = strings.Split(nodeIPsStr, ",")
			for i, ip := range config.NodeIPs {
				config.NodeIPs[i] = strings.TrimSpace(ip)
			}
		}
	}

	// Parse volumes from environment variables if not set in config
	if len(config.Volumes) == 0 {
		config.Volumes = parseVolumesFromEnv()
	}

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &config, nil
}

// parseVolumesFromEnv parses volume configuration from environment variables
func parseVolumesFromEnv() []VolumeConfig {
	var volumes []VolumeConfig

	// Check for GLUSTER_VOLUMES environment variable
	volumesStr := os.Getenv("GLUSTER_VOLUMES")
	if volumesStr == "" {
		return volumes
	}

	// Expected format: name:type:replica_count:host_paths:mount_point;name2:type2:...
	volumeSpecs := strings.Split(volumesStr, ";")
	for _, spec := range volumeSpecs {
		parts := strings.Split(strings.TrimSpace(spec), ":")
		if len(parts) < 5 {
			continue
		}

		replicaCount, _ := strconv.Atoi(parts[2])
		hostPaths := strings.Split(parts[3], ",")
		for i, path := range hostPaths {
			hostPaths[i] = strings.TrimSpace(path)
		}

		volume := VolumeConfig{
			Name:         strings.TrimSpace(parts[0]),
			Type:         strings.TrimSpace(parts[1]),
			ReplicaCount: replicaCount,
			HostPaths:    hostPaths,
			MountPoint:   strings.TrimSpace(parts[4]),
			Options:      make(map[string]string),
		}

		volumes = append(volumes, volume)
	}

	return volumes
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	if config.ClusterName == "" {
		return fmt.Errorf("cluster_name is required")
	}

	if len(config.NodeIPs) == 0 {
		return fmt.Errorf("at least one node IP is required")
	}

	if config.ReplicaCount < 1 {
		return fmt.Errorf("replica_count must be at least 1")
	}

	if config.ReplicaCount > len(config.NodeIPs) {
		return fmt.Errorf("replica_count (%d) cannot exceed number of nodes (%d)",
			config.ReplicaCount, len(config.NodeIPs))
	}

	// Validate volumes
	for i, volume := range config.Volumes {
		if volume.Name == "" {
			return fmt.Errorf("volume[%d]: name is required", i)
		}
		if volume.Type == "" {
			config.Volumes[i].Type = "replicated"
		}
		if volume.ReplicaCount == 0 {
			config.Volumes[i].ReplicaCount = config.ReplicaCount
		}
		if len(volume.HostPaths) == 0 {
			return fmt.Errorf("volume[%d]: at least one host path is required", i)
		}
	}

	return nil
}

// GetNodeName returns the node name for a given index
func (c *Config) GetNodeName(index int) string {
	return fmt.Sprintf("%s-node%d", c.ClusterName, index+1)
}

// GetVolumeByName returns a volume configuration by name
func (c *Config) GetVolumeByName(name string) (*VolumeConfig, bool) {
	for _, volume := range c.Volumes {
		if volume.Name == name {
			return &volume, true
		}
	}
	return nil, false
}
