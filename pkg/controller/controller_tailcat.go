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

package controller

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/nicocha30/ligolo-ng/pkg/tailcatutils"
	"github.com/sirupsen/logrus"
)

const tailcatStartupTimeout = 30 * time.Second

// GetTailcatAddr returns the active tailcat:// URL if the controller is
// listening over tailcat.
func (c *Controller) GetTailcatAddr() (string, error) {
	c.tailcatMu.RLock()
	defer c.tailcatMu.RUnlock()
	if c.tailcatURL == "" {
		return "", errors.New("tailcat listener is not enabled")
	}
	return c.tailcatURL, nil
}

// IsTailcat reports whether the controller is actively listening over tailcat.
func (c *Controller) IsTailcat() bool {
	c.tailcatMu.RLock()
	defer c.tailcatMu.RUnlock()
	return c.tailcatURL != ""
}

func (c *Controller) setTailcatURL(u string) {
	c.tailcatMu.Lock()
	c.tailcatURL = u
	c.tailcatMu.Unlock()
}

func (c *Controller) listenTailcat() {
	ctx, cancel := context.WithTimeout(context.Background(), tailcatStartupTimeout)
	listener, err := tailcatutils.ListenURL(ctx, c.Address, c.TailcatConfig)
	cancel()
	if err != nil {
		c.startchan <- err
		return
	}
	defer listener.Close()

	c.setTailcatURL(listener.TailcatURL())
	c.startchan <- nil
	logrus.Infof("Listening tailcat on %s", listener.TailcatURL())

	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			logrus.Error(err)
			continue
		}
		c.Connection <- conn
	}
}
