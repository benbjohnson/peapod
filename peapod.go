package peapod

import "net"

// General errors.
const (
	ErrInvalidURL = Error("invalid url")
)

// IsLocal returns true if the host represents the local machine.
// This function assumes the hostname has no port.
func IsLocal(hostname string) bool {
	// Check for localhost.
	if hostname == "localhost" {
		return true
	}

	// Check if an IP.
	if ip := net.ParseIP(hostname); ip != nil && ip.IsLoopback() {
		return true
	}

	return false
}
