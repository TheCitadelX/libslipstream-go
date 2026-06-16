package mobile

import (
	"context"
	"fmt"
	"time"

	core "github.com/TheCitadelX/libslipstream-go"
)

type Client struct {
	inner  *core.Client
	events *EventQueue
}

func NewClient(config *ClientConfig) (*Client, error) {
	if config == nil {
		return nil, fmt.Errorf("client config is required")
	}
	inner, err := core.NewClient(config.toCore())
	if err != nil {
		return nil, err
	}
	events := NewEventQueue(config.EventQueueSize)
	events.emit("info", "client", "client created")
	return &Client{inner: inner, events: events}, nil
}

func (c *Client) Start() error {
	c.events.emit("info", "client", "starting client")
	if err := c.inner.Start(); err != nil {
		c.events.emit("error", "client", "start failed: "+err.Error())
		return err
	}
	c.events.emit("info", "client", "client started")
	return nil
}

func (c *Client) Stop() error {
	c.events.emit("info", "client", "stopping client")
	if err := c.inner.Stop(); err != nil {
		c.events.emit("error", "client", "stop failed: "+err.Error())
		return err
	}
	c.events.emit("info", "client", "client stopped")
	return nil
}

func (c *Client) Connected() bool {
	return c.inner.Connected()
}

func (c *Client) Events() *EventQueue {
	return c.events
}

func (c *Client) DialTCP(target string) (*Stream, error) {
	c.events.emit("info", "client", "dial tcp "+target)
	stream, err := c.inner.DialTCP(target)
	if err != nil {
		c.events.emit("error", "client", "dial tcp failed: "+err.Error())
		return nil, err
	}
	c.events.emit("info", "client", "tcp stream opened")
	return &Stream{inner: stream}, nil
}

func (c *Client) StartSOCKS5(listenAddr string) (string, error) {
	c.events.emit("info", "client", "starting socks5 proxy")
	if listenAddr == "" {
		addr, err := c.inner.StartSOCKS5("")
		if err != nil {
			c.events.emit("error", "client", "socks5 start failed: "+err.Error())
			return "", err
		}
		c.events.emit("info", "client", "socks5 proxy listening on "+addr)
		return addr, nil
	}
	addr, err := c.inner.StartSOCKS5(listenAddr)
	if err != nil {
		c.events.emit("error", "client", "socks5 start failed: "+err.Error())
		return "", err
	}
	c.events.emit("info", "client", "socks5 proxy listening on "+addr)
	return addr, nil
}

func (c *Client) StopSOCKS5() error {
	c.events.emit("info", "client", "stopping socks5 proxy")
	if err := c.inner.StopSOCKS5(); err != nil {
		c.events.emit("error", "client", "socks5 stop failed: "+err.Error())
		return err
	}
	c.events.emit("info", "client", "socks5 proxy stopped")
	return nil
}

func (c *Client) Ping(timeoutMs int) error {
	if !c.Connected() {
		c.events.emit("error", "client", "ping failed: client not connected")
		return fmt.Errorf("client not connected")
	}
	if timeoutMs <= 0 {
		timeoutMs = 1000
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()
	stream, err := c.inner.OpenStreamSync(ctx)
	if err != nil {
		c.events.emit("error", "client", "ping failed: "+err.Error())
		return err
	}
	c.events.emit("info", "client", "ping succeeded")
	return stream.Close()
}
