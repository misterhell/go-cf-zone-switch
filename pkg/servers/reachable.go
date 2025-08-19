package servers

import (
	"net"
	"time"
)

// IsServerReachable tries to connect to the given IP and port within a timeout.
// Returns true if reachable, false otherwise.
func IsServerReachable(host string, port string, timeout time.Duration) (bool, error) {
	address := net.JoinHostPort(host, port)
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return false, nil
	}
	err = conn.Close()
	return true, err
}