// Package clientlib provides a client for interacting with the clt netlink family.
// It provides methods to enable and disable trace functionality.
package clientlib

import (
	"errors"
	"fmt"

	"github.com/mdlayher/genetlink"
	"github.com/mdlayher/netlink"
)

var (
	ErrDialGenetlink = errors.New("failed to dial genetlink")
	ErrGetFamily     = errors.New("failed to get genetlink family")
	ErrEnableTrace   = errors.New("failed to enable trace")
	ErrDisableTrace  = errors.New("failed to disable trace")
)

const (
	cltFamilyName = "clt"
)

type CLTClient struct {
	conn   *genetlink.Conn
	family genetlink.Family
}

func NewCLTClient() (*CLTClient, error) {
	conn, err := genetlink.Dial(nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDialGenetlink, err)
	}

	family, err := conn.GetFamily(cltFamilyName)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("%w: %v", ErrGetFamily, err)
	}

	return &CLTClient{
		conn:   conn,
		family: family,
	}, nil
}

func (c *CLTClient) Close() error {
	return c.conn.Close()
}

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

	if _, err := c.conn.Execute(req, c.family.ID, netlink.Request|netlink.Acknowledge); err != nil {
		return fmt.Errorf("%w: %v", ErrEnableTrace, err)
	}

	return nil
}

func (c *CLTClient) DisableTrace() error {
	req := genetlink.Message{
		Header: genetlink.Header{
			Command: 2,
			Version: c.family.Version,
		},
	}

	if _, err := c.conn.Execute(req, c.family.ID, netlink.Request|netlink.Acknowledge); err != nil {
		return fmt.Errorf("%w: %v", ErrDisableTrace, err)
	}

	return nil
}
