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

import "testing"

func TestParseLigoloURLTailcat(t *testing.T) {
	sampleAddr := "tcomFwWCCcjS5nKNqAod034nWoJZW0LZqDhhC8U_dKdnDRYQ8uNGFpGQEu-123"

	tests := []struct {
		name     string
		rawURL   string
		wantAddr string
		wantPort uint16
		wantErr  bool
	}{
		{
			name:     "listen default port",
			rawURL:   "tailcat://",
			wantAddr: "",
			wantPort: DefaultTailcatPort,
		},
		{
			name:     "listen custom port",
			rawURL:   "tailcat://:443",
			wantAddr: "",
			wantPort: 443,
		},
		{
			name:     "connect default port preserving case and base64url chars",
			rawURL:   "tailcat://" + sampleAddr,
			wantAddr: sampleAddr,
			wantPort: DefaultTailcatPort,
		},
		{
			name:     "connect custom port",
			rawURL:   "tailcat://" + sampleAddr + ":8443",
			wantAddr: sampleAddr,
			wantPort: 8443,
		},
		{
			name:    "invalid port zero",
			rawURL:  "tailcat://:0",
			wantErr: true,
		},
		{
			name:    "invalid port overflow",
			rawURL:  "tailcat://:70000",
			wantErr: true,
		},
		{
			name:    "unexpected path",
			rawURL:  "tailcat://" + sampleAddr + "/unexpected",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := ParseLigoloURL(tt.rawURL)
			if err != nil {
				t.Fatalf("ParseLigoloURL(%q) returned unexpected error: %v", tt.rawURL, err)
			}
			if !u.IsTailcat() {
				t.Fatalf("ParseLigoloURL(%q).IsTailcat() = false, want true", tt.rawURL)
			}
			if u.IsWebsocket() {
				t.Errorf("ParseLigoloURL(%q).IsWebsocket() = true, want false", tt.rawURL)
			}
			if !u.IsValid() {
				t.Errorf("ParseLigoloURL(%q).IsValid() = false, want true", tt.rawURL)
			}

			addr, port, err := u.TailcatAddrAndPort()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("TailcatAddrAndPort() succeeded (%q, %d), want error", addr, port)
				}
				return
			}
			if err != nil {
				t.Fatalf("TailcatAddrAndPort() returned unexpected error: %v", err)
			}
			if addr != tt.wantAddr {
				t.Errorf("TailcatAddrAndPort() addr = %q, want %q", addr, tt.wantAddr)
			}
			if port != tt.wantPort {
				t.Errorf("TailcatAddrAndPort() port = %d, want %d", port, tt.wantPort)
			}
		})
	}
}
