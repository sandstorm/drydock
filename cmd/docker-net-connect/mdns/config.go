// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

package mdns

import (
	"golang.zx2c4.com/wireguard/device"
	"net"
	"time"
)

const (
	// DefaultAddress is the default used by mDNS
	// and in most cases should be the address that the
	// net.Conn passed to Server is bound to
	DefaultAddress = "224.0.0.0:5353"
)

// Config is used to configure a mDNS client or server.
type Config struct {
	// QueryInterval controls how often we sends Queries until we
	// get a response for the requested name
	QueryInterval time.Duration

	// LocalNamesToIps are the names that we will generate answers for
	// when we get questions
	// !!! localName keys must end on ".local." (the last dot is also important!)
	LocalNamesToIps map[string]net.IP

	Logger *device.Logger
}
