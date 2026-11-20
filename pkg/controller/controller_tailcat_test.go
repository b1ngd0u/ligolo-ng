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
	"testing"

	"github.com/nicocha30/ligolo-ng/pkg/tlsutils"
)

func TestControllerGetTailcatAddrNotEnabled(t *testing.T) {
	c := New(ControllerConfig{Address: "127.0.0.1:11601"})
	if _, err := c.GetTailcatAddr(); err == nil {
		t.Fatal("GetTailcatAddr() = nil error when tailcat not active, want error")
	}
}

func TestControllerGetSelfCertificateSignatureNilTLSConfig(t *testing.T) {
	c := New(ControllerConfig{
		Address: "tailcat://",
		CertManagerConfig: &tlsutils.CertManagerConfig{
			EnableSelfcert: true,
		},
	})
	if _, err := c.GetSelfCertificateSignature(); err == nil {
		t.Fatal("GetSelfCertificateSignature() = nil error with nil tlsConfig, want error")
	}
}

func TestControllerListenTailcatRejectsHostAddr(t *testing.T) {
	c := New(ControllerConfig{Address: "tailcat://tcInvalidListenHost"})
	go c.ListenAndServe()
	if err := c.WaitForReady(); err == nil {
		t.Fatal("WaitForReady() = nil error for tailcat listen address with host, want error")
	}
}
