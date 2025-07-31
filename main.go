package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

const (
	GlusterImage  = "gluster/gluster-fedora:latest"
	ContainerName = "glusterfs-node"
	GlusterPort   = "24007"
	BrickPort     = "49152"
	VolumeName    = "shared"
)

type GlusterCLI struct{}

func main() {
	cli := &GlusterCLI{}

	// Check if Docker is available
	if !cli.isDockerAvailable() {
		log.Fatal("Docker is not available. Please install Docker and ensure it's running.")
	}

	rootCmd := &cobra.Command{
		Use:   "gluster-sync",
		Short: "Simple GlusterFS cluster setup tool",
		Long:  `A simple CLI tool to set up GlusterFS clusters using Docker`,
	}

	rootCmd.AddCommand(
		cli.setupCommand(),
		cli.statusCommand(),
		cli.removeCommand(),
	)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func (c *GlusterCLI) setupCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Set up a new GlusterFS node",
		Long:  `Interactive setup of a GlusterFS node that can join or create a cluster`,
		RunE:  c.runSetup,
	}
}

func (c *GlusterCLI) statusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show cluster status",
		Long:  `Display the current status of the GlusterFS cluster`,
		RunE:  c.runStatus,
	}
}

func (c *GlusterCLI) removeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "remove",
		Short: "Remove GlusterFS node",
		Long:  `Stop and remove the GlusterFS container`,
		RunE:  c.runRemove,
	}
}

func (c *GlusterCLI) runSetup(cmd *cobra.Command, args []string) error {
	fmt.Println("🚀 GlusterFS Cluster Setup")
	fmt.Println("==========================")

	// Check if already running
	if c.isNodeRunning() {
		fmt.Println("⚠️  GlusterFS node is already running!")
		fmt.Println("Use 'gluster-sync status' to check status or 'gluster-sync remove' to remove")
		return nil
	}

	// Get local folder to sync
	localPath, err := c.promptLocalPath()
	if err != nil {
		return err
	}

	// Get this node's IP
	nodeIP, err := c.getLocalIP()
	if err != nil {
		return err
	}
	fmt.Printf("📍 Detected local IP: %s\n", nodeIP)

	// Start GlusterFS container
	fmt.Println("🐳 Starting GlusterFS container...")
	if err := c.startGlusterContainer(localPath, nodeIP); err != nil {
		return err
	}

	// Wait for container to be ready
	fmt.Println("⏳ Waiting for GlusterFS to start...")
	time.Sleep(10 * time.Second)

	// Get peer IPs
	peerIPs, err := c.promptPeerIPs()
	if err != nil {
		return err
	}

	// Set up cluster
	if len(peerIPs) > 0 {
		fmt.Println("🔗 Joining existing cluster...")
		if err := c.joinCluster(peerIPs); err != nil {
			return err
		}
	} else {
		fmt.Println("🆕 Creating new cluster...")
		if err := c.createCluster(localPath); err != nil {
			return err
		}
	}

	fmt.Println("✅ GlusterFS setup complete!")
	fmt.Printf("📁 Local folder '%s' is now synced with the cluster\n", localPath)
	fmt.Println("\n🎯 Next steps:")
	fmt.Println("• Use 'gluster-sync status' to check cluster status")
	fmt.Println("• Run this tool on other machines to join the cluster")
	fmt.Printf("• Files in '%s' will automatically sync across all nodes\n", localPath)

	return nil
}

func (c *GlusterCLI) promptLocalPath() (string, error) {
	prompt := promptui.Prompt{
		Label:   "Enter local folder path to sync",
		Default: "/tmp/gluster-shared",
		Validate: func(input string) error {
			if input == "" {
				return fmt.Errorf("path cannot be empty")
			}
			absPath, err := filepath.Abs(input)
			if err != nil {
				return err
			}
			// Create directory if it doesn't exist
			if err := os.MkdirAll(absPath, 0755); err != nil {
				return fmt.Errorf("cannot create directory: %v", err)
			}
			return nil
		},
	}

	result, err := prompt.Run()
	if err != nil {
		return "", err
	}

	return filepath.Abs(result)
}

func (c *GlusterCLI) promptPeerIPs() ([]string, error) {
	fmt.Println("\n🌐 Cluster Configuration")
	fmt.Println("------------------------")
	fmt.Println("Enter ALL machine IPs in your cluster (including this machine's peers)")
	fmt.Println("This ensures every node knows about all other nodes")
	fmt.Println("(Leave empty and press Enter if this is the first/only node)")

	var peers []string
	scanner := bufio.NewScanner(os.Stdin)

	// Option 1: Bulk entry (comma-separated)
	fmt.Println("\n💡 You can enter multiple IPs separated by commas:")
	fmt.Print("All peer IPs (comma-separated) or Enter for individual entry: ")
	scanner.Scan()
	bulkInput := strings.TrimSpace(scanner.Text())

	if bulkInput != "" {
		// Parse comma-separated IPs
		ipList := strings.Split(bulkInput, ",")
		for _, ip := range ipList {
			ip = strings.TrimSpace(ip)
			if ip != "" {
				if !c.isValidIP(ip) {
					fmt.Printf("❌ Invalid IP address: %s\n", ip)
					continue
				}
				peers = append(peers, ip)
				fmt.Printf("✅ Added node: %s\n", ip)
			}
		}
	} else {
		// Individual entry mode
		fmt.Println("\nEnter IPs one by one:")
		for {
			fmt.Printf("Node IP [%d] (or Enter to finish): ", len(peers)+1)
			scanner.Scan()
			ip := strings.TrimSpace(scanner.Text())

			if ip == "" {
				break
			}

			if !c.isValidIP(ip) {
				fmt.Println("❌ Invalid IP address format")
				continue
			}

			peers = append(peers, ip)
			fmt.Printf("✅ Added node: %s\n", ip)
		}
	}

	if len(peers) > 0 {
		fmt.Printf("\n📋 Cluster nodes: %s\n", strings.Join(peers, ", "))
	}

	return peers, nil
}

func (c *GlusterCLI) isValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

func (c *GlusterCLI) getLocalIP() (string, error) {
	// Get the local IP address by connecting to a remote address
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}

func (c *GlusterCLI) startGlusterContainer(localPath, nodeIP string) error {
	// Pull image if not present
	fmt.Println("📥 Pulling GlusterFS image...")
	pullCmd := exec.Command("docker", "pull", GlusterImage)
	if err := pullCmd.Run(); err != nil {
		return fmt.Errorf("failed to pull image: %v", err)
	}

	// Create brick directory
	brickPath := filepath.Join(localPath, ".gluster-brick")
	if err := os.MkdirAll(brickPath, 0755); err != nil {
		return fmt.Errorf("failed to create brick directory: %v", err)
	}

	// Remove existing container if it exists
	exec.Command("docker", "rm", "-f", ContainerName).Run()

	// Start container
	dockerCmd := exec.Command("docker", "run", "-d",
		"--name", ContainerName,
		"--privileged",
		"--net", "host",
		"-p", fmt.Sprintf("%s:%s", GlusterPort, GlusterPort),
		"-p", fmt.Sprintf("%s:%s", BrickPort, BrickPort),
		"-v", fmt.Sprintf("%s:/data/glusterfs:rw", brickPath),
		"-v", fmt.Sprintf("%s:/mnt/shared:rw", localPath),
		"-e", fmt.Sprintf("NODE_IP=%s", nodeIP),
		"--hostname", "gluster-node",
		GlusterImage,
		"sh", "-c", "glusterd --no-daemon --log-level=INFO")

	if err := dockerCmd.Run(); err != nil {
		return fmt.Errorf("failed to start container: %v", err)
	}

	return nil
}

func (c *GlusterCLI) joinCluster(peerIPs []string) error {
	myIP, _ := c.getLocalIP()
	fmt.Printf("🏠 This node IP: %s\n", myIP)

	// Probe all peers (excluding self)
	connectedPeers := 0
	for _, peerIP := range peerIPs {
		if peerIP == myIP {
			fmt.Printf("⏭️  Skipping self IP: %s\n", peerIP)
			continue
		}

		fmt.Printf("🔗 Connecting to peer %s...\n", peerIP)
		if err := c.execInContainer(fmt.Sprintf("gluster peer probe %s", peerIP)); err != nil {
			fmt.Printf("⚠️  Failed to probe %s: %v\n", peerIP, err)
		} else {
			fmt.Printf("✅ Connected to %s\n", peerIP)
			connectedPeers++
		}
	}

	if connectedPeers == 0 {
		fmt.Println("⚠️  No peers connected. Creating new cluster...")
		return c.createCluster("")
	}

	// Try to mount existing volume or join volume creation
	fmt.Println("🔍 Checking for existing volumes...")
	if err := c.execInContainer("sleep 5 && gluster volume status"); err != nil {
		fmt.Println("📦 No existing volume found. Checking if we need to participate in volume creation...")

		// Wait a bit for other nodes to potentially create volume
		time.Sleep(10 * time.Second)

		if err := c.execInContainer("gluster volume status"); err != nil {
			fmt.Println("🆕 Creating new replicated volume...")
			return c.createClusterVolume(peerIPs)
		}
	}

	// Mount existing volume
	fmt.Println("📁 Mounting shared volume...")
	mountCmd := fmt.Sprintf("mount -t glusterfs localhost:/%s /mnt/shared", VolumeName)
	if err := c.execInContainer(mountCmd); err != nil {
		fmt.Printf("⚠️  Failed to mount volume: %v\n", err)
	} else {
		fmt.Println("✅ Shared volume mounted")
	}

	return nil
}

func (c *GlusterCLI) createCluster(localPath string) error {
	fmt.Println("📦 Creating shared volume...")

	// Create volume
	brickPath := "/data/glusterfs"
	volumeCmd := fmt.Sprintf("gluster volume create %s %s force", VolumeName, brickPath)
	if err := c.execInContainer(volumeCmd); err != nil {
		return fmt.Errorf("failed to create volume: %v", err)
	}

	// Start volume
	startCmd := fmt.Sprintf("gluster volume start %s", VolumeName)
	if err := c.execInContainer(startCmd); err != nil {
		return fmt.Errorf("failed to start volume: %v", err)
	}

	// Mount volume
	fmt.Println("📁 Mounting shared volume...")
	mountCmd := fmt.Sprintf("mount -t glusterfs localhost:/%s /mnt/shared", VolumeName)
	if err := c.execInContainer(mountCmd); err != nil {
		fmt.Printf("⚠️  Failed to mount volume: %v\n", err)
	} else {
		fmt.Println("✅ Shared volume mounted")
	}

	return nil
}

func (c *GlusterCLI) createClusterVolume(allPeerIPs []string) error {
	fmt.Println("📦 Creating replicated volume across all nodes...")

	// Build brick list for all nodes
	var bricks []string
	myIP, _ := c.getLocalIP()

	// Add this node's brick
	bricks = append(bricks, fmt.Sprintf("%s:/data/glusterfs", myIP))

	// Add all peer bricks
	for _, peerIP := range allPeerIPs {
		if peerIP != myIP {
			bricks = append(bricks, fmt.Sprintf("%s:/data/glusterfs", peerIP))
		}
	}

	replicaCount := len(bricks)
	if replicaCount < 2 {
		replicaCount = 1 // Single node setup
	}

	brickList := strings.Join(bricks, " ")
	fmt.Printf("🧱 Bricks: %s\n", brickList)

	// Create volume with all bricks
	volumeCmd := fmt.Sprintf("gluster volume create %s replica %d %s force", VolumeName, replicaCount, brickList)
	if err := c.execInContainer(volumeCmd); err != nil {
		return fmt.Errorf("failed to create volume: %v", err)
	}

	// Start volume
	startCmd := fmt.Sprintf("gluster volume start %s", VolumeName)
	if err := c.execInContainer(startCmd); err != nil {
		return fmt.Errorf("failed to start volume: %v", err)
	}

	// Mount volume
	fmt.Println("📁 Mounting shared volume...")
	mountCmd := fmt.Sprintf("mount -t glusterfs localhost:/%s /mnt/shared", VolumeName)
	if err := c.execInContainer(mountCmd); err != nil {
		fmt.Printf("⚠️  Failed to mount volume: %v\n", err)
	} else {
		fmt.Println("✅ Shared volume mounted")
	}

	return nil
}

func (c *GlusterCLI) isNodeRunning() bool {
	cmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("name=%s", ContainerName), "--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), ContainerName)
}

func (c *GlusterCLI) isDockerAvailable() bool {
	cmd := exec.Command("docker", "--version")
	return cmd.Run() == nil
}

func (c *GlusterCLI) execInContainer(command string) error {
	cmd := exec.Command("docker", "exec", ContainerName, "sh", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("command failed: %s, output: %s", err, output)
	}
	return nil
}

func (c *GlusterCLI) runStatus(cmd *cobra.Command, args []string) error {
	if !c.isNodeRunning() {
		fmt.Println("❌ GlusterFS node is not running")
		fmt.Println("Use 'gluster-sync setup' to create a node")
		return nil
	}

	fmt.Println("📊 GlusterFS Cluster Status")
	fmt.Println("===========================")

	// Peer status
	fmt.Println("\n🔗 Peer Status:")
	if err := c.execInContainerPrint("gluster peer status"); err != nil {
		fmt.Printf("Error getting peer status: %v\n", err)
	}

	// Volume status
	fmt.Println("\n📦 Volume Status:")
	if err := c.execInContainerPrint("gluster volume status"); err != nil {
		fmt.Printf("Error getting volume status: %v\n", err)
	}

	// Container status
	fmt.Println("\n🐳 Container Status:")
	cmd2 := exec.Command("docker", "ps", "--filter", fmt.Sprintf("name=%s", ContainerName), "--format", "table {{.Names}}\\t{{.Status}}\\t{{.Ports}}")
	output, err := cmd2.Output()
	if err == nil {
		fmt.Print(string(output))
	}

	return nil
}

func (c *GlusterCLI) execInContainerPrint(command string) error {
	cmd := exec.Command("docker", "exec", ContainerName, "sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (c *GlusterCLI) runRemove(cmd *cobra.Command, args []string) error {
	if !c.isNodeRunning() {
		fmt.Println("ℹ️  No GlusterFS node is currently running")
		return nil
	}

	prompt := promptui.Prompt{
		Label:     "Are you sure you want to remove the GlusterFS node? (y/N)",
		IsConfirm: true,
	}

	if _, err := prompt.Run(); err != nil {
		fmt.Println("Cancelled")
		return nil
	}

	fmt.Println("🗑️  Removing GlusterFS container...")

	stopCmd := exec.Command("docker", "stop", ContainerName)
	stopCmd.Run() // Ignore errors

	removeCmd := exec.Command("docker", "rm", "-f", ContainerName)
	if err := removeCmd.Run(); err != nil {
		return fmt.Errorf("failed to remove container: %v", err)
	}

	fmt.Println("✅ GlusterFS node removed successfully")
	return nil
}
