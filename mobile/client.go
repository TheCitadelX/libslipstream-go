package mobile

import (
	"context"
	"fmt"
	"time"

	core "slipstream-go"
)

type Client struct {
	inner *core.Client
}

func NewClient(config ClientConfig) (*Client, error) {
	inner, err := core.NewClient(config.toCore())
	if err != nil {
		return nil, err
	}
	return &Client{inner: inner}, nil
}

func (c *Client) Start() error {
	return c.inner.Start()
}

func (c *Client) Stop() error {
	return c.inner.Stop()
}

func (c *Client) Connected() bool {
	return c.inner.Connected()
}

func (c *Client) DialTCP(target string) (*Stream, error) {
	stream, err := c.inner.DialTCP(target)
	if err != nil {
		return nil, err
	}
	return &Stream{inner: stream}, nil
}

func (c *Client) StartSOCKS5(listenAddr string) (string, error) {
	if listenAddr == "" {
		return c.inner.StartSOCKS5("")
	}
	return c.inner.StartSOCKS5(listenAddr)
}

func (c *Client) StopSOCKS5() error {
	return c.inner.StopSOCKS5()
}

func (c *Client) Ping(timeoutMs int) error {
	if !c.Connected() {
		return fmt.Errorf("client not connected")
	}
	if timeoutMs <= 0 {
		timeoutMs = 1000
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()
	stream, err := c.inner.OpenStreamSync(ctx)
	if err != nil {
		return err
	}
	return stream.Close()
}
