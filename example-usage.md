# Multi-Machine Setup Examples

## Example 1: 3-Machine Cluster

### Scenario
You have 3 machines and want to sync folders between all of them:
- Machine A: 192.168.1.10 → `/home/user/documents`
- Machine B: 192.168.1.20 → `/opt/shared-data`  
- Machine C: 192.168.1.30 → `/tmp/sync-folder`

### Setup Process

**Run on ALL machines (same IP list everywhere):**

```bash
./gluster-sync setup
```

**When prompted for IPs, provide ALL machine IPs:**

**Option 1 - Bulk Entry (Recommended):**
```
All peer IPs (comma-separated): 192.168.1.10,192.168.1.20,192.168.1.30
✅ Added node: 192.168.1.10
✅ Added node: 192.168.1.20  
✅ Added node: 192.168.1.30
📋 Cluster nodes: 192.168.1.10, 192.168.1.20, 192.168.1.30
```

**Option 2 - Individual Entry:**
```
Node IP [1]: 192.168.1.10
✅ Added node: 192.168.1.10
Node IP [2]: 192.168.1.20
✅ Added node: 192.168.1.20
Node IP [3]: 192.168.1.30
✅ Added node: 192.168.1.30
Node IP [4]: [Enter to finish]
```

### Result
- All machines know about each other
- Perfect 3-way replication
- Any machine can fail and cluster continues
- Files sync instantly between all folders

## Example 2: Development Team Setup

### Scenario
Development team with laptops and servers:
- Developer Laptop: 192.168.1.100 → `/home/dev/projects`
- Build Server: 192.168.1.200 → `/opt/ci-builds`
- Staging Server: 192.168.1.201 → `/var/staging`

### Setup
**Same process on all 3 machines:**
```bash
./gluster-sync setup
Enter local folder: [respective folder for each machine]
All peer IPs: 192.168.1.100,192.168.1.200,192.168.1.201
```

### Benefits
- Code changes sync from dev laptop to all servers
- Build artifacts available everywhere
- Staging deployment sees latest code instantly

## Example 3: Backup Strategy

### Scenario  
Primary server with 2 backup locations:
- Main Server: 192.168.1.50 → `/opt/data`
- Backup Site 1: 192.168.1.51 → `/backup/data`
- Backup Site 2: 192.168.1.52 → `/backup2/data`

### Setup
```bash
# Same on all machines
./gluster-sync setup
All peer IPs: 192.168.1.50,192.168.1.51,192.168.1.52
```

### Result
- Real-time replication to both backup sites
- If main server fails, backups have latest data
- Perfect disaster recovery setup

## Key Benefits of This Approach

✅ **Complete Mesh Network**: Every node knows about every other node  
✅ **Better Fault Tolerance**: No single point of failure  
✅ **Automatic Replication**: Data replicated across all nodes  
✅ **Easy Management**: Same configuration everywhere  
✅ **Scalable**: Easy to add/remove nodes  

## Management Commands

```bash
# Check cluster status from any machine
./gluster-sync status

# Remove a node (run on that machine)
./gluster-sync remove

# View all connected peers
docker exec glusterfs-node gluster peer status

# Check volume health
docker exec glusterfs-node gluster volume status shared
```

This approach makes your GlusterFS cluster incredibly robust and easy to manage!