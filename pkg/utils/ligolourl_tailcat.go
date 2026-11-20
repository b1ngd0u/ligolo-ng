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

package utils

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// DefaultTailcatPort is the default virtual TCP port used inside a tailcat tunnel
// when no explicit port is provided in a tailcat:// URL.
const DefaultTailcatPort uint16 = 11601

// IsTailcat reports whether the URL uses the tailcat:// transport scheme.
func (l *LigoloURL) IsTailcat() bool {
	return l != nil && l.URL != nil && l.Scheme == "tailcat"
}

// TailcatAddrAndPort extracts the tailcat address (empty for listen/bind URLs such as
// "tailcat://" or "tailcat://:11601") and virtual TCP port (defaults to DefaultTailcatPort)
// from a tailcat:// URL.
func (l *LigoloURL) TailcatAddrAndPort() (string, uint16, error) {
	if !l.IsTailcat() {
		return "", 0, fmt.Errorf("not a tailcat URL: %v", l)
	}
	if l.Path != "" && l.Path != "/" {
		return "", 0, fmt.Errorf("invalid tailcat URL path %q", l.Path)
	}

	host := l.Host
	if host == "" {
		return "", DefaultTailcatPort, nil
	}

	if !strings.Contains(host, ":") {
		return host, DefaultTailcatPort, nil
	}

	addr, portStr, err := net.SplitHostPort(host)
	if err != nil {
		return "", 0, fmt.Errorf("invalid tailcat host:port %q: %w", host, err)
	}
	if portStr == "" {
		return addr, DefaultTailcatPort, nil
	}

	portNum, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil || portNum == 0 {
		return "", 0, fmt.Errorf("invalid tailcat port %q", portStr)
	}

	return addr, uint16(portNum), nil
}
