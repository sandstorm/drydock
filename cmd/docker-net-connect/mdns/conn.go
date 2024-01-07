// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

package mdns

import (
	"errors"
	"golang.org/x/net/dns/dnsmessage"
	"golang.org/x/net/ipv4"
	"golang.zx2c4.com/wireguard/device"
	"math/big"
	"net"
	"sync"
)

// Conn represents a mDNS Server
type Conn struct {
	mu sync.RWMutex

	socket  *ipv4.PacketConn
	dstAddr *net.UDPAddr

	ifaces []net.Interface

	closed chan interface{}
	logger *device.Logger
	config *Config
}

const (
	destinationAddress = "224.0.0.251:5353"
	maxMessageRecords  = 3
	responseTTL        = 2
)

var errNoPositiveMTUFound = errors.New("no positive MTU found")

// Server establishes a mDNS connection over an existing conn
func Server(conn *ipv4.PacketConn, config *Config) (*Conn, error) {
	if config == nil {
		return nil, errNilConfig
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	inboundBufferSize := 0
	joinErrCount := 0
	ifacesToUse := make([]net.Interface, 0, len(ifaces))
	for i, ifc := range ifaces {
		if err = conn.JoinGroup(&ifaces[i], &net.UDPAddr{IP: net.IPv4(224, 0, 0, 251)}); err != nil {
			joinErrCount++
			continue
		}

		ifcCopy := ifc
		ifacesToUse = append(ifacesToUse, ifcCopy)
		if ifaces[i].MTU > inboundBufferSize {
			inboundBufferSize = ifaces[i].MTU
		}
	}

	if inboundBufferSize == 0 {
		return nil, errNoPositiveMTUFound
	}
	if joinErrCount >= len(ifaces) {
		return nil, errJoiningMulticastGroup
	}

	dstAddr, err := net.ResolveUDPAddr("udp", destinationAddress)
	if err != nil {
		return nil, err
	}

	c := &Conn{
		socket:  conn,
		dstAddr: dstAddr,
		config:  config,
		ifaces:  ifacesToUse,
		closed:  make(chan interface{}),
		logger:  config.Logger,
	}

	if err := conn.SetControlMessage(ipv4.FlagInterface, true); err != nil {
		config.Logger.Errorf("Failed to SetControlMessage on PacketConn %v", err)
	}

	// https://www.rfc-editor.org/rfc/rfc6762.html#section-17
	// Multicast DNS messages carried by UDP may be up to the IP MTU of the
	// physical interface, less the space required for the IP header (20
	// bytes for IPv4; 40 bytes for IPv6) and the UDP header (8 bytes).
	go c.start(inboundBufferSize-20-8, config)
	return c, nil
}

// Close closes the mDNS Conn
func (c *Conn) Close() error {
	select {
	case <-c.closed:
		return nil
	default:
	}

	if err := c.socket.Close(); err != nil {
		return err
	}

	<-c.closed
	return nil
}

func ipToBytes(ip net.IP) (out [4]byte) {
	rawIP := ip.To4()
	if rawIP == nil {
		return
	}

	ipInt := big.NewInt(0)
	ipInt.SetBytes(rawIP)
	copy(out[:], ipInt.Bytes())
	return
}

func (c *Conn) writeToSocket(ifIndex int, b []byte, onlyLooback bool) {
	if ifIndex != 0 {
		ifc, err := net.InterfaceByIndex(ifIndex)
		if err != nil {
			c.logger.Errorf("Failed to get interface interface for %d: %v", ifIndex, err)
			return
		}
		if onlyLooback && ifc.Flags&net.FlagLoopback == 0 {
			// avoid accidentally tricking the destination that itself is the same as us
			c.logger.Errorf("Interface is not loopback %d", ifIndex)
			return
		}
		if err := c.socket.SetMulticastInterface(ifc); err != nil {
			c.logger.Errorf("Failed to set multicast interface for %d: %v", ifIndex, err)
		} else {
			if _, err := c.socket.WriteTo(b, nil, c.dstAddr); err != nil {
				c.logger.Errorf("Failed to send mDNS packet on interface %d: %v", ifIndex, err)
			}
		}
		return
	}
	for ifcIdx := range c.ifaces {
		if onlyLooback && c.ifaces[ifcIdx].Flags&net.FlagLoopback == 0 {
			// avoid accidentally tricking the destination that itself is the same as us
			continue
		}
		if err := c.socket.SetMulticastInterface(&c.ifaces[ifcIdx]); err != nil {
			c.logger.Errorf("Failed to set multicast interface for %d: %v", c.ifaces[ifcIdx].Index, err)
		} else {
			if _, err := c.socket.WriteTo(b, nil, c.dstAddr); err != nil {
				c.logger.Errorf("Failed to send mDNS packet on interface %d: %v", c.ifaces[ifcIdx].Index, err)
			}
		}
	}
}

func (c *Conn) sendAnswer(name string, ifIndex int, dst net.IP) {
	packedName, err := dnsmessage.NewName(name)
	if err != nil {
		c.logger.Errorf("Failed to construct mDNS packet %v", err)
		return
	}

	msg := dnsmessage.Message{
		Header: dnsmessage.Header{
			Response:      true,
			Authoritative: true,
		},
		Answers: []dnsmessage.Resource{
			{
				Header: dnsmessage.ResourceHeader{
					Type:  dnsmessage.TypeA,
					Class: dnsmessage.ClassINET,
					Name:  packedName,
					TTL:   responseTTL,
				},
				Body: &dnsmessage.AResource{
					A: ipToBytes(dst),
				},
			},
		},
	}

	rawAnswer, err := msg.Pack()
	if err != nil {
		c.logger.Errorf("Failed to construct mDNS packet %v", err)
		return
	}

	c.writeToSocket(ifIndex, rawAnswer, dst.IsLoopback())
}

func (c *Conn) start(inboundBufferSize int, config *Config) { //nolint gocognit
	defer func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		close(c.closed)
	}()

	b := make([]byte, inboundBufferSize)
	p := dnsmessage.Parser{}

	for {
		n, cm, src, err := c.socket.ReadFrom(b)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			c.logger.Errorf("Failed to ReadFrom %q %v", src, err)
			continue
		}
		var ifIndex int
		if cm != nil {
			ifIndex = cm.IfIndex
		}

		func() {
			c.mu.RLock()
			defer c.mu.RUnlock()

			if _, err := p.Start(b[:n]); err != nil {
				c.logger.Errorf("Failed to parse mDNS packet %v", err)
				return
			}

			for i := 0; i <= maxMessageRecords; i++ {
				q, err := p.Question()
				if errors.Is(err, dnsmessage.ErrSectionDone) {
					break
				} else if err != nil {
					c.logger.Errorf("Failed to parse mDNS packet %v", err)
					return
				}

				for localName, ip := range c.config.LocalNamesToIps {
					if localName == q.Name.String() {
						c.sendAnswer(q.Name.String(), ifIndex, ip)
					}
				}
			}
		}()
	}
}
