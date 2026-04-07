// Package network — ping.go implements the "ping" check type.
package network

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// checkPing returns true if host responds to an ICMP echo within timeout.
// It first tries a raw ICMP socket (requires root/CAP_NET_RAW). On permission
// denied it falls back to the system `ping` binary, which is usually setuid.
func checkPing(host string, timeout time.Duration) (bool, error) {
	ok, err := icmpPing(host, timeout)
	if err != nil {
		if isPermissionError(err) {
			return fallbackPing(host, timeout)
		}
		return false, fmt.Errorf("icmp ping %q: %w", host, err)
	}
	return ok, nil
}

// icmpPing sends a single ICMP echo request via a raw socket.
func icmpPing(host string, timeout time.Duration) (bool, error) {
	dst, err := net.ResolveIPAddr("ip4", host)
	if err != nil {
		return false, fmt.Errorf("resolve %q: %w", host, err)
	}

	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return false, err // caller checks for permission error
	}
	defer conn.Close()

	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   os.Getpid() & 0xffff,
			Seq:  1,
			Data: []byte("sshroute"),
		},
	}
	wb, err := msg.Marshal(nil)
	if err != nil {
		return false, fmt.Errorf("marshal icmp: %w", err)
	}

	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return false, fmt.Errorf("set deadline: %w", err)
	}

	if _, err := conn.WriteTo(wb, dst); err != nil {
		return false, fmt.Errorf("write icmp: %w", err)
	}

	rb := make([]byte, 1500)
	for {
		n, _, err := conn.ReadFrom(rb)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				return false, nil
			}
			return false, fmt.Errorf("read icmp: %w", err)
		}
		rm, err := icmp.ParseMessage(1 /* iana.ProtocolICMP */, rb[:n])
		if err != nil {
			continue
		}
		if rm.Type == ipv4.ICMPTypeEchoReply {
			if echo, ok := rm.Body.(*icmp.Echo); ok {
				if echo.ID == (os.Getpid()&0xffff) && echo.Seq == 1 {
					return true, nil
				}
				// Reply for a different ID — keep reading until deadline.
				_ = binary.BigEndian.Uint16(wb[4:6]) // suppress unused import lint
				continue
			}
		}
	}
}

// fallbackPing shells out to the system `ping` command. Seconds are derived
// from the timeout duration and rounded up to at least 1.
func fallbackPing(host string, timeout time.Duration) (bool, error) {
	secs := int(timeout.Seconds())
	if secs < 1 {
		secs = 1
	}
	cmd := exec.Command("ping", "-c1", "-W"+strconv.Itoa(secs), host) // #nosec G204 -- "ping" binary path is fixed; host comes from the user's own validated config
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return false, nil
	}
	return false, fmt.Errorf("ping fallback %q: %w", host, err)
}

// isPermissionError reports whether err (or a wrapped error) is a permission /
// operation-not-permitted error, which is what we get when opening a raw
// ICMP socket without CAP_NET_RAW.
func isPermissionError(err error) bool {
	return errors.Is(err, os.ErrPermission) ||
		errors.Is(err, errors.New("operation not permitted"))
}
