package slipstream

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
)

const (
	targetTypeIPv4   = 0x01
	targetTypeDomain = 0x03
	targetTypeIPv6   = 0x04
)

func writeTargetAddress(w io.Writer, addr string) error {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("parse address: %w", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("parse port: %w", err)
	}

	var buf []byte
	if ip := net.ParseIP(host); ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			buf = append(buf, targetTypeIPv4)
			buf = append(buf, ip4...)
		} else {
			buf = append(buf, targetTypeIPv6)
			buf = append(buf, ip...)
		}
	} else {
		if len(host) > 255 {
			return fmt.Errorf("domain name too long")
		}
		buf = append(buf, targetTypeDomain, byte(len(host)))
		buf = append(buf, host...)
	}
	buf = append(buf, byte(port>>8), byte(port))

	_, err = w.Write(buf)
	return err
}

func readTargetAddress(r io.Reader) (string, error) {
	typeBuf := make([]byte, 1)
	if _, err := io.ReadFull(r, typeBuf); err != nil {
		return "", fmt.Errorf("read address type: %w", err)
	}

	var host string
	switch typeBuf[0] {
	case targetTypeIPv4:
		ipBuf := make([]byte, 4)
		if _, err := io.ReadFull(r, ipBuf); err != nil {
			return "", fmt.Errorf("read IPv4: %w", err)
		}
		host = net.IP(ipBuf).String()
	case targetTypeDomain:
		lenBuf := make([]byte, 1)
		if _, err := io.ReadFull(r, lenBuf); err != nil {
			return "", fmt.Errorf("read domain length: %w", err)
		}
		domainBuf := make([]byte, lenBuf[0])
		if _, err := io.ReadFull(r, domainBuf); err != nil {
			return "", fmt.Errorf("read domain: %w", err)
		}
		host = string(domainBuf)
	case targetTypeIPv6:
		ipBuf := make([]byte, 16)
		if _, err := io.ReadFull(r, ipBuf); err != nil {
			return "", fmt.Errorf("read IPv6: %w", err)
		}
		host = net.IP(ipBuf).String()
	default:
		return "", fmt.Errorf("unknown address type: %d", typeBuf[0])
	}

	portBuf := make([]byte, 2)
	if _, err := io.ReadFull(r, portBuf); err != nil {
		return "", fmt.Errorf("read port: %w", err)
	}
	port := binary.BigEndian.Uint16(portBuf)
	return net.JoinHostPort(host, strconv.Itoa(int(port))), nil
}
