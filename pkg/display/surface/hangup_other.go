//go:build !linux

package surface

func (c *Client) hungUp() bool { return false }
