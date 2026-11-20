// Ligolo-ng
// Copyright (C) 2025 Nicolas Chatelain (nicocha30)

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package tailcatutils

import (
	"net"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/yamux"
)

const endpointsChangedMarker = "magicsock: endpoints changed: "

// NewYamuxConfig returns a yamux configuration tuned for tailcat tunnels,
// matching the agent's longer keepalive/write timeouts and 16MB stream window
// so DERP relay throttling or NAT transitions do not prematurely drop sessions.
func NewYamuxConfig() *yamux.Config {
	cfg := yamux.DefaultConfig()
	cfg.EnableKeepAlive = true
	cfg.KeepAliveInterval = 60 * time.Second
	cfg.ConnectionWriteTimeout = 120 * time.Second
	cfg.MaxStreamWindowSize = 16 * 1024 * 1024
	return cfg
}

type endpointTracker struct {
	mu         sync.RWMutex
	endpoints  map[netip.AddrPort]struct{}
	localPorts map[uint16]struct{}
}

var (
	trackersMu sync.RWMutex
	trackers   = make(map[*endpointTracker]struct{})
)

func newEndpointTracker() *endpointTracker {
	t := &endpointTracker{}
	trackersMu.Lock()
	trackers[t] = struct{}{}
	trackersMu.Unlock()
	return t
}

func (t *endpointTracker) close() {
	if t == nil {
		return
	}
	trackersMu.Lock()
	delete(trackers, t)
	trackersMu.Unlock()
}

func (t *endpointTracker) observeLog(msg string) {
	if t == nil {
		return
	}
	idx := strings.Index(msg, endpointsChangedMarker)
	if idx < 0 {
		return
	}
	eps, ports := parseEndpointsList(msg[idx+len(endpointsChangedMarker):])
	t.mu.Lock()
	t.endpoints = eps
	t.localPorts = ports
	t.mu.Unlock()
}

func parseEndpointsList(raw string) (map[netip.AddrPort]struct{}, map[uint16]struct{}) {
	eps := make(map[netip.AddrPort]struct{})
	ports := make(map[uint16]struct{})

	for _, entry := range strings.Split(raw, ",") {
		fields := strings.Fields(entry)
		if len(fields) == 0 {
			continue
		}
		ap, err := netip.ParseAddrPort(fields[0])
		if err != nil {
			continue
		}
		ap = netip.AddrPortFrom(ap.Addr().Unmap(), ap.Port())
		eps[ap] = struct{}{}
		if len(fields) > 1 && strings.Contains(fields[1], "local") {
			ports[ap.Port()] = struct{}{}
		}
	}
	return eps, ports
}

func (t *endpointTracker) matches(ip netip.Addr, port uint16) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if _, ok := t.endpoints[netip.AddrPortFrom(ip, port)]; ok {
		return true
	}
	if _, ok := t.localPorts[port]; ok {
		if ip.IsLoopback() || isHostLocalIP(ip) {
			return true
		}
	}
	return false
}

// IsLocalEndpoint reports whether addr:port matches an active local tailcat
// magicsock UDP endpoint on this process. Agents use this to refuse forwarding
// Ligolo TUN UDP packets back into their own tailcat socket, preventing
// self-encapsulation routing loops when the proxy routes the agent's LAN subnet
// into the Ligolo TUN interface.
func IsLocalEndpoint(addr string, port uint16) bool {
	ip, err := netip.ParseAddr(addr)
	if err != nil {
		return false
	}
	ip = ip.Unmap()

	trackersMu.RLock()
	defer trackersMu.RUnlock()
	for t := range trackers {
		if t.matches(ip, port) {
			return true
		}
	}
	return false
}

func isHostLocalIP(target netip.Addr) bool {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return false
	}
	for _, a := range addrs {
		var ip net.IP
		switch v := a.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		if ip == nil {
			continue
		}
		if addr, ok := netip.AddrFromSlice(ip); ok && addr.Unmap() == target {
			return true
		}
	}
	return false
}
