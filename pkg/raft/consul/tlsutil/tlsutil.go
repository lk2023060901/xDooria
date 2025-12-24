// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

// Package tlsutil provides TLS configuration utilities for RPC connections.
package tlsutil

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"sync"
)

// ALPNWrapper is a function that is used to wrap a non-TLS connection and
// returns an appropriate TLS connection or error. This takes a datacenter and
// node name as argument to configure the desired SNI value and the desired
// next proto for configuring ALPN.
type ALPNWrapper func(dc, nodeName, alpnProto string, conn net.Conn) (net.Conn, error)

// DCWrapper is a function that is used to wrap a non-TLS connection
// and returns an appropriate TLS connection or error. This takes
// a datacenter as an argument.
type DCWrapper func(dc string, conn net.Conn) (net.Conn, error)

// Wrapper is a variant of DCWrapper, where the DC is provided as
// a constant value. This is usually done by currying DCWrapper.
type Wrapper func(conn net.Conn) (net.Conn, error)

// Config holds the TLS configuration for a node.
type Config struct {
	// VerifyIncoming is used to verify the authenticity of incoming connections.
	VerifyIncoming bool

	// VerifyOutgoing is used to verify the authenticity of outgoing connections.
	VerifyOutgoing bool

	// VerifyServerHostname is used to enable hostname verification of servers.
	VerifyServerHostname bool

	// CAFile is a path to a certificate authority file.
	CAFile string

	// CAPath is a path to a directory containing certificate authority files.
	CAPath string

	// CertFile is used to provide a TLS certificate for serving TLS connections.
	CertFile string

	// KeyFile is used to provide a TLS key for serving TLS connections.
	KeyFile string

	// ServerName is used with the TLS certificate to ensure the name matches.
	ServerName string

	// Domain is the Consul TLS domain (usually "consul")
	Domain string

	// NodeName is the name of the local node.
	NodeName string

	// Datacenter is the name of the local datacenter.
	Datacenter string

	// EnableAgentTLSForChecks is used to apply the agent's TLS settings to
	// check requests.
	EnableAgentTLSForChecks bool

	// InternalRPC is the configuration for internal RPC connections.
	InternalRPC ProtocolConfig
}

// ProtocolConfig contains TLS configuration for a specific protocol.
type ProtocolConfig struct {
	// VerifyIncoming enables certificate verification for incoming connections.
	VerifyIncoming bool

	// VerifyOutgoing enables certificate verification for outgoing connections.
	VerifyOutgoing bool

	// CAFile is a path to a certificate authority file.
	CAFile string

	// CAPath is a path to a directory containing certificate authority files.
	CAPath string

	// CertFile is used to provide a TLS certificate.
	CertFile string

	// KeyFile is used to provide a TLS key.
	KeyFile string

	// TLSMinVersion is the minimum accepted TLS version.
	TLSMinVersion string
}

// Configurator provides TLS configuration for connections.
type Configurator struct {
	sync.RWMutex

	config     *Config
	caPool     *x509.CertPool
	tlsConfig  *tls.Config
	useTLS     bool
	autoTLS    bool
	peerUseTLS map[string]bool
}

// NewConfigurator creates a new TLS configurator.
func NewConfigurator(config Config) (*Configurator, error) {
	c := &Configurator{
		peerUseTLS: make(map[string]bool),
	}
	if err := c.Update(config); err != nil {
		return nil, err
	}
	return c, nil
}

// Update updates the TLS configuration.
func (c *Configurator) Update(config Config) error {
	c.Lock()
	defer c.Unlock()

	c.config = &config

	// Build the CA pool
	caPool := x509.NewCertPool()
	if config.InternalRPC.CAFile != "" {
		data, err := os.ReadFile(config.InternalRPC.CAFile)
		if err != nil {
			return fmt.Errorf("failed to read CA file: %w", err)
		}
		if !caPool.AppendCertsFromPEM(data) {
			return fmt.Errorf("failed to parse CA certificates")
		}
	}
	c.caPool = caPool

	// Determine if TLS is enabled
	c.useTLS = config.InternalRPC.VerifyOutgoing ||
		config.InternalRPC.CertFile != "" ||
		config.InternalRPC.CAFile != ""

	// Build base TLS config
	if c.useTLS {
		tlsConfig := &tls.Config{
			RootCAs:            c.caPool,
			InsecureSkipVerify: !config.VerifyServerHostname,
			MinVersion:         tls.VersionTLS12,
		}

		// Load client certificate if provided
		if config.InternalRPC.CertFile != "" && config.InternalRPC.KeyFile != "" {
			cert, err := tls.LoadX509KeyPair(config.InternalRPC.CertFile, config.InternalRPC.KeyFile)
			if err != nil {
				return fmt.Errorf("failed to load key pair: %w", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}

		c.tlsConfig = tlsConfig
	}

	return nil
}

// UseTLS returns whether TLS should be used for the given datacenter.
func (c *Configurator) UseTLS(dc string) bool {
	c.RLock()
	defer c.RUnlock()

	// Check if there's a specific override for this datacenter
	if useTLS, ok := c.peerUseTLS[dc]; ok {
		return useTLS
	}

	return c.useTLS
}

// OutgoingRPCWrapper returns a DCWrapper for outgoing RPC connections.
func (c *Configurator) OutgoingRPCWrapper() DCWrapper {
	return func(dc string, conn net.Conn) (net.Conn, error) {
		c.RLock()
		defer c.RUnlock()

		if c.tlsConfig == nil {
			return conn, nil
		}

		tlsConfig := c.tlsConfig.Clone()
		if c.config != nil && c.config.ServerName != "" {
			tlsConfig.ServerName = c.config.ServerName
		}

		tlsConn := tls.Client(conn, tlsConfig)
		if err := tlsConn.Handshake(); err != nil {
			tlsConn.Close()
			return nil, err
		}

		return tlsConn, nil
	}
}

// OutgoingALPNRPCWrapper returns an ALPNWrapper for outgoing RPC connections
// that negotiate using ALPN.
func (c *Configurator) OutgoingALPNRPCWrapper() ALPNWrapper {
	return func(dc, nodeName, alpnProto string, conn net.Conn) (net.Conn, error) {
		c.RLock()
		defer c.RUnlock()

		if c.tlsConfig == nil {
			return nil, fmt.Errorf("TLS not configured")
		}

		tlsConfig := c.tlsConfig.Clone()
		tlsConfig.NextProtos = []string{alpnProto}
		tlsConfig.ServerName = fmt.Sprintf("%s.%s.consul", nodeName, dc)

		tlsConn := tls.Client(conn, tlsConfig)
		if err := tlsConn.Handshake(); err != nil {
			tlsConn.Close()
			return nil, err
		}

		return tlsConn, nil
	}
}

// IncomingRPCConfig returns the TLS config for incoming RPC connections.
func (c *Configurator) IncomingRPCConfig() *tls.Config {
	c.RLock()
	defer c.RUnlock()

	if c.tlsConfig == nil {
		return nil
	}

	tlsConfig := c.tlsConfig.Clone()
	if c.config != nil && c.config.InternalRPC.VerifyIncoming {
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		tlsConfig.ClientCAs = c.caPool
	}

	return tlsConfig
}

// UpdateAreaPeerDatacenterUseTLS sets the TLS mode for a specific peer datacenter.
func (c *Configurator) UpdateAreaPeerDatacenterUseTLS(peerDatacenter string, useTLS bool) {
	c.Lock()
	defer c.Unlock()
	c.peerUseTLS[peerDatacenter] = useTLS
}
