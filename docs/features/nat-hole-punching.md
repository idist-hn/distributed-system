# NAT Hole Punching

## Overview

NAT Hole Punching là kỹ thuật cho phép hai peers đằng sau NAT có thể kết nối trực tiếp với nhau mà không cần relay server. Kỹ thuật này sử dụng UDP để "đục lỗ" qua NAT firewall.

## Architecture

```
┌──────────────┐                           ┌──────────────┐
│   Peer A     │                           │   Peer B     │
│ (Behind NAT) │                           │ (Behind NAT) │
│              │                           │              │
│ Private:     │                           │ Private:     │
│ 192.168.1.10 │                           │ 10.0.0.20    │
│ UDP:5000     │                           │ UDP:5001     │
└──────┬───────┘                           └──────┬───────┘
       │                                          │
       │ NAT                                      │ NAT
       ▼                                          ▼
┌──────────────┐                           ┌──────────────┐
│ Public IP    │                           │ Public IP    │
│ 1.2.3.4:5000 │                           │ 5.6.7.8:5001 │
└──────┬───────┘                           └──────┬───────┘
       │                                          │
       │     ┌────────────────────────┐          │
       └────▶│   Tracker/Coordinator  │◀─────────┘
             │   UDP Coordinator      │
             │   (Public Server)      │
             └────────────────────────┘
```

## Hole Punch Process

### Step 1: Register with Coordinator
Mỗi peer gửi UDP packet đến coordinator để đăng ký public address.

```go
// Peer sends ping to coordinator
msg := Message{
    Type:     MsgPing,
    FromPeer: "peer-A",
}

// Coordinator responds with peer's public address
resp := {
    your_ip:   "1.2.3.4",
    your_port: 5000,
}
```

### Step 2: Get Target Peer's Endpoint
Peer A yêu cầu coordinator cung cấp endpoint của Peer B.

### Step 3: Simultaneous Punch
Cả hai peers đồng thời gửi UDP packets đến nhau.

```go
// Both peers send punch packets at the same time
puncherA.PunchTo(ctx, "peer-B", Endpoint{IP: "5.6.7.8", Port: 5001})
puncherB.PunchTo(ctx, "peer-A", Endpoint{IP: "1.2.3.4", Port: 5000})
```

### Step 4: NAT Table Updated
Khi cả hai gửi packets, NAT tables được cập nhật để cho phép incoming traffic.

### Step 5: Direct Communication
Sau khi punch thành công, hai peers có thể giao tiếp trực tiếp.

## Package Structure

```
pkg/holepunch/
├── holepunch.go      # Puncher implementation
├── coordinator.go    # Tracker-side coordinator
└── holepunch_test.go # Unit tests
```

## Key Components

### Puncher (Peer-side)

```go
// Create puncher
puncher, _ := holepunch.NewPuncher("peer-id", 0)
puncher.Start()
defer puncher.Stop()

// Set message handler
puncher.SetMessageHandler(func(from string, data []byte) {
    fmt.Printf("Received from %s: %s\n", from, data)
})

// Attempt hole punch
err := puncher.PunchTo(ctx, "target-peer", holepunch.Endpoint{
    IP:   "1.2.3.4",
    Port: 5000,
})

// Send data after punch
puncher.SendTo("target-peer", []byte("Hello!"))
```

### Coordinator (Tracker-side)

```go
// Create coordinator
coordinator, _ := holepunch.NewCoordinator(9999)
coordinator.Start()
defer coordinator.Stop()

// Get peer endpoint
endpoint, exists := coordinator.GetEndpoint("peer-id")
if exists {
    fmt.Printf("Peer at %s:%d\n", endpoint.PublicIP, endpoint.PublicPort)
}
```

## Message Types

| Type | Description |
|------|-------------|
| `punch` | Initial hole punch request |
| `punch_ack` | Acknowledge punch success |
| `data` | Application data |
| `ping` | Keep-alive/registration |
| `pong` | Ping response |

## NAT Types Compatibility

| NAT Type | Hole Punch Success Rate |
|----------|------------------------|
| Full Cone | ✅ 100% |
| Restricted Cone | ✅ 90% |
| Port Restricted | ⚠️ 70% |
| Symmetric | ❌ 10-20% |

## Connection Strategy Integration

```go
// Current connection order:
// 1. Direct TCP connection
// 2. UDP Hole Punch (NEW)
// 3. WebSocket Relay

func (m *Manager) RequestChunk(...) {
    // Try direct TCP
    data, err := m.tryDirectConnection(...)
    if err == nil { return data }
    
    // Try hole punch (NEW)
    data, err = m.tryHolePunch(...)
    if err == nil { return data }
    
    // Fallback to relay
    data, err = m.tryRelayConnection(...)
    return data
}
```

## Configuration

```bash
# Peer configuration
HOLE_PUNCH_ENABLED=true
HOLE_PUNCH_PORT=0  # 0 = random port

# Tracker configuration
HOLE_PUNCH_COORDINATOR_PORT=9999
```

## Limitations

1. **Symmetric NAT**: Không hoạt động với symmetric NAT (thay đổi port mỗi connection)
2. **Firewall**: Một số firewall block UDP traffic
3. **Timeout**: Cần retry nếu punch fail lần đầu

## Testing

```bash
# Run unit tests
go test -v ./pkg/holepunch/...

# Expected output: 13 tests passed
```

## Future Improvements

1. **STUN Integration**: Sử dụng STUN servers để detect NAT type
2. **TURN Fallback**: Fallback to TURN relay khi hole punch fail
3. **ICE Protocol**: Full ICE implementation cho reliability

