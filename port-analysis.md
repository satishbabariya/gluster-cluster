# GlusterFS Port Requirements Analysis

## Current CLI Configuration

The CLI currently uses these port settings:

```go
const (
    GlusterPort = "24007"  // GlusterFS daemon port
    BrickPort   = "49152"  // Brick communication port
)
```

## Docker Configuration Issues

### Problem 1: Redundant Configuration
```bash
docker run -d \
    --net host \                    # ← Uses host networking
    -p 24007:24007 \               # ← Redundant with host networking
    -p 49152:49152 \               # ← Redundant with host networking
```

**Issue**: When using `--net host`, port mappings (`-p`) are ignored and unnecessary.

## What Happens If Ports Are Not Exposed

### Scenario 1: Ports Blocked by Firewall
**Impact**: 
- ❌ Nodes cannot communicate with each other
- ❌ Peer probing fails
- ❌ Volume creation fails
- ❌ File replication stops

**Error Messages**:
```
connection failed: No route to host
peer probe: failed: Probe returned with Transport endpoint is not connected
```

### Scenario 2: Ports Already in Use
**Impact**:
- ❌ GlusterFS daemon fails to start
- ❌ Container startup fails
- ❌ CLI setup command fails

**Error Messages**:
```
bind: address already in use
failed to start container: Error response from daemon
```

### Scenario 3: Host Networking Disabled
**Impact**:
- ❌ Inter-node communication requires proper port mapping
- ❌ Complex networking setup needed
- ❌ Performance may be reduced

## Required Ports for GlusterFS

### Core Ports
- **24007**: GlusterFS daemon (glusterd)
- **24008**: GlusterFS management  
- **49152-49251**: Brick processes (dynamic range)

### Additional Ports (if needed)
- **111**: Portmapper (rpcbind)
- **38465-38469**: NFS services (if enabled)

## Solutions and Recommendations

### 1. Fix Current Implementation
Remove redundant port mappings:
```bash
docker run -d \
    --name glusterfs-node \
    --privileged \
    --net host \              # Sufficient for port access
    # Remove: -p 24007:24007 -p 49152:49152
```

### 2. Alternative: Bridge Networking
For environments where host networking is restricted:
```bash
docker run -d \
    --name glusterfs-node \
    --privileged \
    -p 24007:24007 \
    -p 24008:24008 \
    -p 49152-49251:49152-49251 \  # Full brick port range
```

### 3. Port Availability Check
Add pre-flight checks:
```bash
# Check if ports are available
netstat -tlnp | grep :24007
lsof -i :24007
```

### 4. Firewall Configuration
Ensure ports are open:
```bash
# Ubuntu/Debian
ufw allow 24007:24008/tcp
ufw allow 49152:49251/tcp

# CentOS/RHEL
firewall-cmd --add-port=24007-24008/tcp --permanent
firewall-cmd --add-port=49152-49251/tcp --permanent
```

## Impact Assessment

| Scenario | Cluster Formation | File Sync | Recovery |
|----------|------------------|-----------|----------|
| Ports Open | ✅ Works | ✅ Works | ✅ Works |
| Ports Blocked | ❌ Fails | ❌ Fails | ❌ Fails |
| Partial Ports | ⚠️ Limited | ⚠️ Limited | ⚠️ Limited |

## Recommendations for Production

1. **Use host networking** for simplicity and performance
2. **Configure firewall rules** to allow GlusterFS ports
3. **Add port availability checks** before container startup
4. **Provide clear error messages** when ports are unavailable
5. **Document port requirements** for system administrators