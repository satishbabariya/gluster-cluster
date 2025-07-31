package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

func main() {
	logger := setupLogger()
	logger.Info("Starting GlusterFS node...")

	// Get node configuration from environment
	nodeName := os.Getenv("NODE_NAME")
	nodeIP := os.Getenv("NODE_IP")
	clusterName := os.Getenv("CLUSTER_NAME")

	if nodeName == "" || nodeIP == "" {
		logger.Fatal("NODE_NAME and NODE_IP environment variables are required")
	}

	logger.WithFields(logrus.Fields{
		"node_name":    nodeName,
		"node_ip":      nodeIP,
		"cluster_name": clusterName,
	}).Info("Node configuration loaded")

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Received shutdown signal, stopping glusterd...")
		stopGlusterd(logger)
		os.Exit(0)
	}()

	// Start glusterd
	if err := startGlusterd(logger); err != nil {
		logger.Fatalf("Failed to start glusterd: %v", err)
	}

	// Wait for glusterd to be ready
	if err := waitForGlusterd(logger); err != nil {
		logger.Fatalf("glusterd failed to become ready: %v", err)
	}

	logger.Info("GlusterFS node is ready and running")

	// Keep the process running
	select {}
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

func startGlusterd(logger *logrus.Logger) error {
	logger.Info("Starting glusterd service...")

	// Create data directories
	dataPath := os.Getenv("GLUSTER_DATA_PATH")
	if dataPath == "" {
		dataPath = "/data/glusterfs"
	}

	if err := os.MkdirAll(dataPath, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Start systemd if it's not already running
	if err := exec.Command("systemctl", "start", "glusterd").Run(); err != nil {
		logger.Warnf("Failed to start glusterd via systemctl: %v", err)

		// Try to start glusterd directly
		cmd := exec.Command("glusterd", "--no-daemon")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Start(); err != nil {
			return fmt.Errorf("failed to start glusterd directly: %w", err)
		}

		// Wait a bit for glusterd to start
		time.Sleep(5 * time.Second)
	}

	return nil
}

func waitForGlusterd(logger *logrus.Logger) error {
	logger.Info("Waiting for glusterd to be ready...")

	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		cmd := exec.Command("gluster", "peer", "status")
		if err := cmd.Run(); err == nil {
			logger.Info("glusterd is ready")
			return nil
		}

		logger.Debugf("glusterd not ready yet, attempt %d/%d", i+1, maxRetries)
		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("glusterd failed to become ready after %d attempts", maxRetries)
}

func stopGlusterd(logger *logrus.Logger) {
	logger.Info("Stopping glusterd...")

	if err := exec.Command("systemctl", "stop", "glusterd").Run(); err != nil {
		logger.Warnf("Failed to stop glusterd via systemctl: %v", err)

		// Try to kill glusterd process directly
		if err := exec.Command("pkill", "glusterd").Run(); err != nil {
			logger.Warnf("Failed to kill glusterd process: %v", err)
		}
	}
}
