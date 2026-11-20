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

package agent

import "sync"

var (
	udpTargetFilterMu sync.RWMutex
	udpTargetFilter   func(addr string, port uint16) bool
)

// SetUDPTargetFilter registers an optional predicate invoked before dialing
// UDP targets in HandleConn. Returning true refuses the dial attempt.
func SetUDPTargetFilter(fn func(addr string, port uint16) bool) {
	udpTargetFilterMu.Lock()
	udpTargetFilter = fn
	udpTargetFilterMu.Unlock()
}

func isUDPTargetBlocked(addr string, port uint16) bool {
	udpTargetFilterMu.RLock()
	fn := udpTargetFilter
	udpTargetFilterMu.RUnlock()
	if fn == nil {
		return false
	}
	return fn(addr, port)
}
