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

// Package tailcatutils is the sole integration boundary (anti-corruption layer)
// around github.com/tailscale/tailcat. Keeping all tailcat/tailscale imports
// confined to this package isolates the rest of ligolo-ng from upstream
// tailcat API changes.
package tailcatutils

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/nicocha30/ligolo-ng/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/tailscale/tailcat"
	"tailscale.com/envknob"
	"tailscale.com/tailcfg"
	"tailscale.com/types/logger"
)

// Config holds transport options for tailcat listeners and dialers using
// standard Go types so callers do not depend on tailscale/tailcat packages.
type Config struct {
	// DERPMapURL overrides tailcat's default DERP map URL when non-empty.
	DERPMapURL string
	// RegionID pins a specific DERP region for listeners (0 = auto-select nearest).
	RegionID int
	// DERPOnly forces all tailcat traffic through DERP relays and disables direct
	// UDP hole-punching (equivalent to TS_DEBUG_ALWAYS_USE_DERP=true).
	DERPOnly bool
}

func applyConfig(cfg *Config) {
	// Disable Linux UDP GSO by default to prevent packet blackholing on paths
	// with MTU < 1440 (PPPoE, VPNs, Tailscale, nested tunnels; tailcat#168).
	// Users can still opt back in by explicitly setting TS_DEBUG_DISABLE_UDP_GSO=false.
	if _, set := os.LookupEnv("TS_DEBUG_DISABLE_UDP_GSO"); !set {
		envknob.Setenv("TS_DEBUG_DISABLE_UDP_GSO", "true")
	}
	if cfg != nil && cfg.DERPOnly {
		envknob.Setenv("TS_DEBUG_ALWAYS_USE_DERP", "true")
	}
}

// Listener wraps a tailcat server listener and manages the underlying
// tailcat.Server lifecycle behind standard net.Listener semantics.
type Listener struct {
	net.Listener
	server    *tailcat.Server
	tracker   *endpointTracker
	addr      string
	port      uint16
	closeOnce sync.Once
	closeErr  error
}

// TailcatAddr returns the raw tailcat address string ("tc...") for this listener.
func (l *Listener) TailcatAddr() string {
	return l.addr
}

// TailcatURL returns the formatted tailcat:// URL that peers can pass to
// -connect or connect_agent.
func (l *Listener) TailcatURL() string {
	return FormatURL(l.addr, l.port)
}

// Close closes both the listener and its backing tailcat.Server.
func (l *Listener) Close() error {
	l.closeOnce.Do(func() {
		var errs []error
		if l.Listener != nil {
			if err := l.Listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
				errs = append(errs, err)
			}
		}
		if l.server != nil {
			if err := l.server.Close(); err != nil {
				errs = append(errs, err)
			}
		}
		l.tracker.close()
		l.closeErr = errors.Join(errs...)
	})
	return l.closeErr
}

// FormatURL formats a raw tailcat address and virtual port into a ligolo-ng
// tailcat:// URL, omitting the port when it equals utils.DefaultTailcatPort.
func FormatURL(addr string, port uint16) string {
	if port == 0 || port == utils.DefaultTailcatPort {
		return "tailcat://" + addr
	}
	return fmt.Sprintf("tailcat://%s:%d", addr, port)
}

// ValidateAddr verifies that addr is a syntactically valid tailcat address.
func ValidateAddr(addr string) error {
	if addr == "" {
		return errors.New("tailcat address is empty")
	}
	if _, err := tailcat.ParseAddr(tailcat.Addr(addr)); err != nil {
		return fmt.Errorf("invalid tailcat address: %w", err)
	}
	return nil
}

// ListenURL parses a tailcat:// listen URL (e.g. "tailcat://" or "tailcat://:443")
// and starts a tailcat listener on the parsed virtual port.
func ListenURL(ctx context.Context, rawURL string, cfg *Config) (*Listener, error) {
	ligoloURL, err := utils.ParseLigoloURL(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid tailcat listen URL: %w", err)
	}
	addr, port, err := ligoloURL.TailcatAddrAndPort()
	if err != nil {
		return nil, err
	}
	if addr != "" {
		return nil, fmt.Errorf("invalid tailcat listen address %q: expected tailcat:// or tailcat://:port", rawURL)
	}
	return Listen(ctx, port, cfg)
}

// DialURL parses a tailcat://<addr>[:port] URL and connects to the target tailcat server.
func DialURL(ctx context.Context, rawURL string, cfg *Config) (net.Conn, error) {
	ligoloURL, err := utils.ParseLigoloURL(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid tailcat connect URL: %w", err)
	}
	addr, port, err := ligoloURL.TailcatAddrAndPort()
	if err != nil {
		return nil, err
	}
	if addr == "" {
		return nil, errors.New("missing tailcat address, expected tailcat://<addr>[:port]")
	}
	return Dial(ctx, addr, port, cfg)
}

// Listen starts an ephemeral WireGuard+PSK tailcat server and listens for TCP
// connections on the specified virtual port.
func Listen(ctx context.Context, port uint16, cfg *Config) (*Listener, error) {
	if port == 0 {
		port = utils.DefaultTailcatPort
	}
	applyConfig(cfg)

	tracker := newEndpointTracker()
	srv := &tailcat.Server{
		Logf: newLogf(tracker),
	}
	if cfg != nil {
		if cfg.DERPMapURL != "" {
			srv.DERPMapURL = cfg.DERPMapURL
		}
		if cfg.RegionID > 0 {
			srv.RegionID = tailcfg.DERPRegionID(cfg.RegionID)
		}
	}

	ln, err := srv.Listen(ctx, "tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		tracker.close()
		return nil, fmt.Errorf("tailcat listen failed: %w", err)
	}

	return &Listener{
		Listener: ln,
		server:   srv,
		tracker:  tracker,
		addr:     string(srv.TailcatAddr()),
		port:     port,
	}, nil
}

// Dial connects to a tailcat server on the given virtual TCP port and returns a
// net.Conn whose Close method also shuts down the underlying tailcat.Client.
func Dial(ctx context.Context, addr string, port uint16, cfg *Config) (net.Conn, error) {
	if err := ValidateAddr(addr); err != nil {
		return nil, err
	}
	if port == 0 {
		port = utils.DefaultTailcatPort
	}
	applyConfig(cfg)

	tracker := newEndpointTracker()
	client := &tailcat.Client{
		Server: tailcat.Addr(addr),
		Logf:   newLogf(tracker),
	}
	if cfg != nil && cfg.DERPMapURL != "" {
		client.DERPMapURL = cfg.DERPMapURL
	}

	conn, err := client.DialTCPPort(ctx, port)
	if err != nil {
		_ = client.Close()
		tracker.close()
		return nil, fmt.Errorf("tailcat dial failed: %w", err)
	}

	return newConnWithCloser(conn, func() error {
		tracker.close()
		return client.Close()
	}), nil
}

type connWithCloser struct {
	net.Conn
	closeFn   func() error
	closeOnce sync.Once
	closeErr  error
}

func newConnWithCloser(conn net.Conn, closeFn func() error) net.Conn {
	return &connWithCloser{
		Conn:    conn,
		closeFn: closeFn,
	}
}

func (c *connWithCloser) Close() error {
	c.closeOnce.Do(func() {
		var errs []error
		if c.Conn != nil {
			if err := c.Conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
				errs = append(errs, err)
			}
		}
		if c.closeFn != nil {
			if err := c.closeFn(); err != nil {
				errs = append(errs, err)
			}
		}
		c.closeErr = errors.Join(errs...)
	})
	return c.closeErr
}

func newLogf(tracker *endpointTracker) logger.Logf {
	return func(format string, args ...any) {
		debugEnabled := logrus.IsLevelEnabled(logrus.DebugLevel)
		watchEndpoints := tracker != nil && (strings.Contains(format, "endpoints changed:") || format == "%s")
		if !debugEnabled && !watchEndpoints {
			return
		}
		msg := fmt.Sprintf(format, args...)
		if watchEndpoints {
			tracker.observeLog(msg)
		}
		if debugEnabled {
			logrus.Debugf("[tailcat] %s", msg)
		}
	}
}
