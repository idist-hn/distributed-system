# User Guide - P2P File Sharing

## Quick Start

### 1. Start Peer Node

```bash
# Basic usage
./peer -tracker https://p2p.idist.dev -api-key YOUR_API_KEY

# With bandwidth limit (1 MB/s)
./peer -tracker https://p2p.idist.dev -api-key YOUR_API_KEY -bandwidth 1048576
```

### 2. Share a File

```bash
# In peer CLI
> share /path/to/myfile.zip

# Output:
Shared: myfile.zip
Hash: abc123def456...
Chunks: 42
Magnet: magnet:?xt=urn:p2p:abc123def456...&dn=myfile.zip&xl=10485760
```

Copy the **Magnet link** to share with others.

### 3. Download a File

#### Using File Hash
```bash
> download abc123def456
```

#### Using Magnet Link
```bash
./download "magnet:?xt=urn:p2p:abc123...&dn=myfile.zip"
```

## Commands Reference

| Command | Description | Example |
|---------|-------------|---------|
| `share <path>` | Share a file | `share ./video.mp4` |
| `download <hash>` | Download by hash | `download abc123` |
| `list` | List shared files | `list` |
| `peers` | Show connected peers | `peers` |
| `status` | Show download status | `status` |
| `help` | Show help | `help` |

## Magnet Links

Magnet links contain all info needed to download:

```
magnet:?xt=urn:p2p:<hash>&dn=<filename>&xl=<size>&tr=<tracker>&x.cs=<chunksize>&x.cn=<chunks>
```

| Parameter | Description |
|-----------|-------------|
| `xt` | File hash (urn:p2p:HASH) |
| `dn` | Display name |
| `xl` | File size in bytes |
| `tr` | Tracker URL |
| `x.cs` | Chunk size |
| `x.cn` | Number of chunks |

## Bandwidth Control

Limit download/upload speed:

```bash
# Start with 500 KB/s limit
./peer -bandwidth 512000

# Or set in CLI
> bandwidth 1048576  # 1 MB/s
```

## Data Integrity

Files are verified using:
- **SHA-256** hash per chunk
- **Merkle Tree** for overall file integrity

If verification fails, chunks are automatically re-downloaded.

## Troubleshooting

### Connection Issues
```bash
# Check tracker connectivity
curl https://p2p.idist.dev/api/health

# Check peer status
> peers
```

### Slow Downloads
- Check bandwidth limit setting
- Try different peers: `> peers`
- Check network connectivity

### File Not Found
- Ensure at least one peer has the file online
- Verify the file hash is correct

