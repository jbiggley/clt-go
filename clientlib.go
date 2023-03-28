// Package clientlib provides a client for interacting with the CLT netlink family.
// It provides methods to enable and disable trace functionality.
package clientlib

import (
	"errors"
	"fmt"

	"github.com/mdlayher/genetlink"
	"github.com/mdlayher/netlink"
)

const (
	cltFamilyName = "clt"
)

var (
	ErrFailedToDialGenetlink = errors.New("failed to dial genetlink")
	ErrFailedToGetFamily     = errors.New("failed to get genetlink family")
	ErrFailedToEnableTrace   = errors.New("failed to enable trace")
	ErrFailedToDisableTrace  = errors.New("failed to disable trace")
)

// CLTClient represents a client for interacting with the CLT netlink family.
type CLTClient struct {
	conn   *genetlink.Conn
	family genetlink.Family
}

// NewCLTClient returns a new CLTClient instance.
func NewCLTClient() (*CLTClient, error) {
	conn, err := genetlink.Dial(nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFailedToDialGenetlink, err)
	}

	family, err := conn.GetFamily(cltFamilyName)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("%w: %v", ErrFailedToGetFamily, err)
	}

	return &CLTClient{
		conn:   conn,
		family: family,
	}, nil
}

// Close closes the connection to the CLTClient.
func (c *CLTClient) Close() error {
	return c.conn.Close()
}

// EnableTrace enables CLT tracing with the specified spanID and traceID.
func (c *CLTClient) EnableTrace(spanID, traceID uint64) error {
	attr := netlink.AttributeEncoder{}
	attr.Uint64(1, spanID)
	attr.Uint64(2, traceID)

	req := genetlink.Message{
		Header: genetlink.Header{
			Command: 1,
			Version: c.family.Version,
		},
		Data: attr.Encode(),
	}

	// Execute the request and check for errors.
	_, err := c.conn.Execute(req, c.family.ID, netlink.Request|netlink.Acknowledge)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrFailedToEnableTrace, err)
	}

	return nil
}

// DisableTrace disables CLT tracing.
func (c *CLTClient) DisableTrace() error {
	req := genetlink.Message{
		Header: genetlink.Header{
			Command: 2,
			Version: c.family.Version,
		},
	}

	// Execute the request and check for errors.
	_, err := c.conn.Execute(req, c.family.ID, netlink.Request|netlink.Acknowledge)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrFailedToDisableTrace, err)
	}

	return nil
}
