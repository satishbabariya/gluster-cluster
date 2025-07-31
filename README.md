# GlusterFS Sync CLI

A simple, user-friendly CLI tool that sets up GlusterFS clusters using Docker. Just run it on any machine and it will help you create or join a distributed file system.

## 🚀 Features

- **Simple Setup**: One command to set up a GlusterFS node
- **Interactive**: Prompts for local folder and peer IPs
- **Docker-based**: No complex installation, just needs Docker
- **Cross-platform**: Works on any system with Docker
- **User-friendly**: Clear prompts and status messages

## 📦 Installation

### Option 1: Download Binary (Recommended)
```bash
# Download the latest release
curl -L https://github.com/your-repo/gluster-sync-cli/releases/latest/download/gluster-sync -o gluster-sync
chmod +x gluster-sync
```

### Option 2: Build from Source
```bash
git clone https://github.com/your-repo/gluster-sync-cli
cd gluster-sync-cli
go build -o gluster-sync main.go
```

## 🎯 Quick Start

### Step 1: Set up first node
```bash
./gluster-sync setup
```

The CLI will ask you:
1. **Local folder path** - Which folder to sync (e.g., `/home/user/shared`)
2. **Peer IPs** - Leave empty for the first node, OR enter ALL planned node IPs

### Step 2: Set up additional nodes
On other machines:
```bash
./gluster-sync setup
```

The CLI will ask you:
1. **Local folder path** - Same or different folder to sync
2. **Peer IPs** - Enter ALL node IPs (including the first node and this node)

💡 **Pro Tip**: You can enter all IPs at once using comma-separated format:
`192.168.1.10,192.168.1.20,192.168.1.30`

### Step 3: Check status
```bash
./gluster-sync status
```

## 📋 Commands

| Command | Description |
|---------|-------------|
| `setup` | Interactive setup of a GlusterFS node |
| `status` | Show cluster and container status |
| `remove` | Stop and remove the GlusterFS container |
| `--help` | Show help information |

## 🔧 How it works

1. **Creates a Docker container** running GlusterFS on each machine
2. **Mounts your local folder** into the container
3. **Connects nodes** using GlusterFS peer probing
4. **Automatically replicates files** between all nodes
5. **Files added to the local folder** appear on all other nodes

## 📁 Example Workflow

### Option 1: Progressive Setup (Node by Node)
```bash
# Machine 1 (192.168.1.10)
./gluster-sync setup
# Enter local folder: /home/user/documents
# Enter peer IPs: (leave empty - first node)

# Machine 2 (192.168.1.20)  
./gluster-sync setup
# Enter local folder: /home/user/shared
# Enter peer IPs: 192.168.1.10

# Machine 3 (192.168.1.30)
./gluster-sync setup  
# Enter local folder: /opt/shared-data
# Enter peer IPs: 192.168.1.10

# Now all three folders are synced!
```

### Option 2: Cluster-wide Setup (Recommended)
```bash
# On ALL machines, use the same IP list:
./gluster-sync setup
# Enter local folder: [respective folder]
# All peer IPs: 192.168.1.10,192.168.1.20,192.168.1.30

# Benefits:
# ✅ Every node knows about every other node
# ✅ Better fault tolerance
# ✅ Perfect mesh topology
```

## 🛠️ Requirements

- **Docker** installed and running
- **Network connectivity** between machines
- **Ports 24007 and 49152** open for GlusterFS communication

## ❓ FAQ

**Q: What happens if a node goes down?**  
A: Other nodes continue working. When the node comes back up, files sync automatically.

**Q: Can I use different local folders on each machine?**  
A: Yes! Each machine can have a different local path.

**Q: How do I add a new machine to an existing cluster?**  
A: Just run `./gluster-sync setup` and enter any existing node's IP.

**Q: How do I remove a node from the cluster?**  
A: Run `./gluster-sync remove` on that machine.

## 🔍 Troubleshooting

**Container won't start:**
```bash
# Check Docker is running
docker ps

# Check if port is available
netstat -tulpn | grep 24007
```

**Can't connect to peers:**
```bash
# Check network connectivity
ping <peer-ip>

# Check firewall allows ports 24007 and 49152
```

**Files not syncing:**
```bash
# Check cluster status
./gluster-sync status

# Check container logs
docker logs glusterfs-node
```

## 🏗️ Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Machine 1     │    │   Machine 2     │    │   Machine 3     │
│                 │    │                 │    │                 │
│ /home/docs/ ────┼────┼─── /opt/data/ ──┼────┼─── /tmp/sync/   │
│                 │    │                 │    │                 │
│ [GlusterFS]     │    │ [GlusterFS]     │    │ [GlusterFS]     │
│                 │    │                 │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         ↑                       ↑                       ↑
    Docker Container       Docker Container       Docker Container
```

Files added to any folder automatically appear in all other folders across all machines.

## 📄 License

MIT License - feel free to use this in your projects!