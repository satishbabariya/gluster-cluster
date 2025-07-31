package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gluster-cluster/gluster-sync/internal/cluster"
	"github.com/gluster-cluster/gluster-sync/internal/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "gluster-manager",
		Short: "GlusterFS Cluster Manager",
		Long:  `A Go-based manager for GlusterFS clusters with Docker support`,
	}

	var startCmd = &cobra.Command{
		Use:   "start",
		Short: "Start the GlusterFS cluster",
		RunE:  startCluster,
	}

	var stopCmd = &cobra.Command{
		Use:   "stop",
		Short: "Stop the GlusterFS cluster",
		RunE:  stopCluster,
	}

	var statusCmd = &cobra.Command{
		Use:   "status",
		Short: "Show cluster status",
		RunE:  showStatus,
	}

	var configCmd = &cobra.Command{
		Use:   "config",
		Short: "Show configuration",
		RunE:  showConfig,
	}

	rootCmd.AddCommand(startCmd, stopCmd, statusCmd, configCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func setupLogger() *logrus.Logger {
	logger := logrus.New()

	// Set log level from environment
	logLevel := os.Getenv("GLUSTER_LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	return logger
}

func startCluster(cmd *cobra.Command, args []string) error {
	logger := setupLogger()
	logger.Info("Starting GlusterFS cluster manager...")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create cluster manager
	manager, err := cluster.NewManager(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to create cluster manager: %w", err)
	}
	defer manager.Close()

	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Received shutdown signal, stopping cluster...")
		cancel()

		if err := manager.StopCluster(context.Background()); err != nil {
			logger.Errorf("Error stopping cluster: %v", err)
		}
		os.Exit(0)
	}()

	// Start cluster
	if err := manager.StartCluster(ctx); err != nil {
		return fmt.Errorf("failed to start cluster: %w", err)
	}

	logger.Info("Cluster started successfully. Press Ctrl+C to stop.")

	// Keep running until signal
	<-ctx.Done()
	return nil
}

func stopCluster(cmd *cobra.Command, args []string) error {
	logger := setupLogger()
	logger.Info("Stopping GlusterFS cluster...")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create cluster manager
	manager, err := cluster.NewManager(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to create cluster manager: %w", err)
	}
	defer manager.Close()

	// Stop cluster
	ctx := context.Background()
	if err := manager.StopCluster(ctx); err != nil {
		return fmt.Errorf("failed to stop cluster: %w", err)
	}

	logger.Info("Cluster stopped successfully")
	return nil
}

func showStatus(cmd *cobra.Command, args []string) error {
	logger := setupLogger()

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create cluster manager
	manager, err := cluster.NewManager(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to create cluster manager: %w", err)
	}
	defer manager.Close()

	// Get status
	ctx := context.Background()
	status, err := manager.GetClusterStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to get cluster status: %w", err)
	}

	// Print status as JSON
	statusJSON, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal status: %w", err)
	}

	fmt.Println(string(statusJSON))
	return nil
}

func showConfig(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Print configuration as JSON
	configJSON, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	fmt.Println(string(configJSON))
	return nil
}
