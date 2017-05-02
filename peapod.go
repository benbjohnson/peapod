package peapod

import (
	"crypto/rand"
	"fmt"
	"net"
)

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

// GenerateToken returns a random string.
func GenerateToken() string {
	buf := make([]byte, 20)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", buf)
}
