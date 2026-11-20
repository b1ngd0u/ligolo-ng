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

package app

import (
	"context"
	"errors"
	"time"

	"github.com/desertbit/grumble"
	"github.com/hashicorp/yamux"
	"github.com/nicocha30/ligolo-ng/pkg/controller"
	"github.com/nicocha30/ligolo-ng/pkg/tailcatutils"
	"github.com/sirupsen/logrus"
)

const tailcatDialTimeout = 30 * time.Second

func registerTailcatCommands() {
	App.AddCommand(&grumble.Command{
		Name:  "tailcat_addr",
		Help:  "Show the current tailcat listening address",
		Usage: "tailcat_addr",
		Run: func(c *grumble.Context) error {
			if ProxyController == nil {
				return errors.New("proxy controller is not initialized")
			}
			addr, err := ProxyController.GetTailcatAddr()
			if err != nil {
				return err
			}
			logrus.Printf("Tailcat listening address: %s\n", addr)
			return nil
		},
	})
}

func connectAgentTailcat(rawURL string) error {
	var tcCfg *tailcatutils.Config
	if ProxyController != nil {
		tcCfg = ProxyController.TailcatConfig
	}

	ctx, cancel := context.WithTimeout(context.Background(), tailcatDialTimeout)
	defer cancel()

	remoteConn, err := tailcatutils.DialURL(ctx, rawURL, tcCfg)
	if err != nil {
		return err
	}

	yamuxConn, err := yamux.Client(remoteConn, tailcatutils.NewYamuxConfig())
	if err != nil {
		_ = remoteConn.Close()
		return err
	}

	agent, err := controller.NewAgent(yamuxConn)
	if err != nil {
		_ = yamuxConn.Close()
		logrus.Errorf("could not register agent, error: %v", err)
		return err
	}

	logrus.WithFields(logrus.Fields{"remote": remoteConn.RemoteAddr(), "name": agent.Name, "id": agent.SessionID}).Info("Agent connected.")

	if err := RegisterAgent(agent); err != nil {
		logrus.Errorf("could not register agent: %s", err.Error())
	}
	return nil
}
