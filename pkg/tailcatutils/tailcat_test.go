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
	"context"
	"net"
	"sync/atomic"
	"testing"

	"github.com/nicocha30/ligolo-ng/pkg/utils"
	"github.com/tailscale/tailcat"
)

func TestFormatURL(t *testing.T) {
	tests := []struct {
		addr string
		port uint16
		want string
	}{
		{"tcExample", 0, "tailcat://tcExample"},
		{"tcExample", utils.DefaultTailcatPort, "tailcat://tcExample"},
		{"tcExample", 443, "tailcat://tcExample:443"},
	}

	for _, tt := range tests {
		if got := FormatURL(tt.addr, tt.port); got != tt.want {
			t.Errorf("FormatURL(%q, %d) = %q, want %q", tt.addr, tt.port, got, tt.want)
		}
	}
}

func TestValidateAddr(t *testing.T) {
	if err := ValidateAddr(""); err == nil {
		t.Fatal("ValidateAddr(\"\") = nil, want error")
	}
	if err := ValidateAddr("not-a-tailcat-addr"); err == nil {
		t.Fatal("ValidateAddr(\"not-a-tailcat-addr\") = nil, want error")
	}

	// Contract test: generate a valid ConnInfo address with tailcat and confirm ValidateAddr accepts it.
	priv := tailcat.NewPrivateKey()
	priv.Public.RegionID = 1
	validAddr := string(priv.Public.Addr())
	if err := ValidateAddr(validAddr); err != nil {
		t.Fatalf("ValidateAddr(%q) returned unexpected error: %v", validAddr, err)
	}
}

func TestDialRejectsInvalidAddr(t *testing.T) {
	_, err := Dial(context.Background(), "invalid", utils.DefaultTailcatPort, nil)
	if err == nil {
		t.Fatal("Dial with invalid addr returned nil error")
	}
}

func TestListenAndDialURLValidation(t *testing.T) {
	if _, err := ListenURL(context.Background(), "tailcat://tcSomeAddr", nil); err == nil {
		t.Fatal("ListenURL with non-empty addr returned nil error")
	}
	if _, err := ListenURL(context.Background(), "http://127.0.0.1:8080", nil); err == nil {
		t.Fatal("ListenURL with non-tailcat scheme returned nil error")
	}
	if _, err := DialURL(context.Background(), "tailcat://", nil); err == nil {
		t.Fatal("DialURL with empty addr returned nil error")
	}
	if _, err := DialURL(context.Background(), "tailcat://invalid-addr", nil); err == nil {
		t.Fatal("DialURL with invalid tailcat addr returned nil error")
	}
}

func TestConnWithCloserIdempotent(t *testing.T) {
	c1, c2 := net.Pipe()
	defer c2.Close()

	var closeCount atomic.Int32
	wrapped := newConnWithCloser(c1, func() error {
		closeCount.Add(1)
		return nil
	})

	if err := wrapped.Close(); err != nil {
		t.Fatalf("first Close() error = %v", err)
	}
	if err := wrapped.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
	if got := closeCount.Load(); got != 1 {
		t.Fatalf("closeFn called %d times, want 1", got)
	}
}

func TestEndpointTrackerAndIsLocalEndpoint(t *testing.T) {
	tr := newEndpointTracker()
	defer tr.close()

	logf := newLogf(tr)
	logf("magicsock: endpoints changed: %s", "10.77.77.50:42945 (local), [fd00::50]:42945 (local), 198.51.100.2:54321 (stun)")

	if !IsLocalEndpoint("10.77.77.50", 42945) {
		t.Fatal("expected 10.77.77.50:42945 to match active tailcat endpoint")
	}
	if !IsLocalEndpoint("fd00::50", 42945) {
		t.Fatal("expected [fd00::50]:42945 to match active tailcat endpoint")
	}
	if !IsLocalEndpoint("198.51.100.2", 54321) {
		t.Fatal("expected stun endpoint 198.51.100.2:54321 to match active tailcat endpoint")
	}
	if !IsLocalEndpoint("127.0.0.1", 42945) {
		t.Fatal("expected loopback with local magicsock port 42945 to match")
	}
	if IsLocalEndpoint("10.77.77.50", 9999) {
		t.Fatal("unrelated port 9999 on 10.77.77.50 must not be blocked")
	}
	if IsLocalEndpoint("10.77.77.51", 42945) {
		t.Fatal("different host IP with same port must not be blocked")
	}

	tr.close()
	if IsLocalEndpoint("10.77.77.50", 42945) {
		t.Fatal("closed tracker must no longer match 10.77.77.50:42945")
	}
}

func TestNewYamuxConfig(t *testing.T) {
	cfg := NewYamuxConfig()
	if !cfg.EnableKeepAlive {
		t.Fatal("EnableKeepAlive = false, want true")
	}
	if cfg.ConnectionWriteTimeout < 60*1e9 {
		t.Fatalf("ConnectionWriteTimeout = %v, want >= 60s", cfg.ConnectionWriteTimeout)
	}
	if cfg.MaxStreamWindowSize < 16*1024*1024 {
		t.Fatalf("MaxStreamWindowSize = %d, want >= 16MB", cfg.MaxStreamWindowSize)
	}
}
