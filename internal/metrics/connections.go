package metrics

import (
	"context"
	"fmt"
	"sort"

	gnet "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

// PortInfo describes a single listening socket (a "port in use").
type PortInfo struct {
	Proto   string // "tcp" or "udp"
	Addr    string // bound local address (e.g. "*", "127.0.0.1")
	Port    uint32
	PID     int32
	Process string
}

// ConnectionInfo describes an active (non-listening) network connection.
type ConnectionInfo struct {
	Proto   string
	Laddr   string // local  ip:port
	Raddr   string // remote ip:port
	Status  string // ESTABLISHED, CLOSE_WAIT, ...
	PID     int32
	Process string
}

// NetConnections is the collected view of listening ports and active
// connections. Available is false when the data could not be gathered.
type NetConnections struct {
	Available   bool
	Listening   []PortInfo
	Connections []ConnectionInfo
}

// socketType values from the OS socket API (syscall.SOCK_*).
const (
	sockStream = 1 // TCP
	sockDgram  = 2 // UDP
)

// CollectConnections lists listening ports and active connections. It resolves
// process names for the PIDs involved (cached per call). No elevated
// privileges are required; connections owned by other users may be omitted by
// the OS. On macOS this shells out to lsof, so it is comparatively slow and
// should be collected on a throttled cadence rather than every tick.
func CollectConnections(ctx context.Context) (NetConnections, error) {
	stats, err := gnet.ConnectionsWithContext(ctx, "inet")
	if err != nil {
		return NetConnections{}, err
	}

	nameCache := make(map[int32]string)
	resolve := func(pid int32) string {
		if pid <= 0 {
			return ""
		}
		if n, ok := nameCache[pid]; ok {
			return n
		}
		name := ""
		if p, err := process.NewProcessWithContext(ctx, pid); err == nil {
			if nm, err := p.NameWithContext(ctx); err == nil {
				name = nm
			}
		}
		nameCache[pid] = name
		return name
	}

	seen := make(map[string]bool)
	var listening []PortInfo
	var connections []ConnectionInfo

	for _, c := range stats {
		proto := "tcp"
		if c.Type == sockDgram {
			proto = "udp"
		}

		isListen := c.Status == "LISTEN" ||
			(proto == "udp" && (c.Raddr.IP == "" || c.Raddr.Port == 0))

		if isListen {
			key := fmt.Sprintf("%s/%d/%d", proto, c.Laddr.Port, c.Pid)
			if seen[key] {
				continue
			}
			seen[key] = true
			addr := c.Laddr.IP
			if addr == "" || addr == "0.0.0.0" || addr == "::" {
				addr = "*"
			}
			listening = append(listening, PortInfo{
				Proto:   proto,
				Addr:    addr,
				Port:    c.Laddr.Port,
				PID:     c.Pid,
				Process: resolve(c.Pid),
			})
			continue
		}

		if c.Raddr.IP == "" && c.Raddr.Port == 0 {
			continue // not a real active connection
		}
		connections = append(connections, ConnectionInfo{
			Proto:   proto,
			Laddr:   fmt.Sprintf("%s:%d", c.Laddr.IP, c.Laddr.Port),
			Raddr:   fmt.Sprintf("%s:%d", c.Raddr.IP, c.Raddr.Port),
			Status:  c.Status,
			PID:     c.Pid,
			Process: resolve(c.Pid),
		})
	}

	sort.Slice(listening, func(i, j int) bool {
		if listening[i].Port != listening[j].Port {
			return listening[i].Port < listening[j].Port
		}
		return listening[i].Proto < listening[j].Proto
	})
	sort.Slice(connections, func(i, j int) bool {
		if connections[i].Process != connections[j].Process {
			return connections[i].Process < connections[j].Process
		}
		return connections[i].Raddr < connections[j].Raddr
	})

	return NetConnections{
		Available:   true,
		Listening:   listening,
		Connections: connections,
	}, nil
}
