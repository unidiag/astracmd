package astra

import "context"

type Client struct {
	Conn Connection
}

func NewClient(conn Connection) *Client {
	return &Client{
		Conn: conn,
	}
}

func (c *Client) Load(ctx context.Context) LoadResult {
	return load(ctx, c.Conn)
}

func (c *Client) GetStatus(ctx context.Context) StatusResult {
	return getStatus(ctx, c.Conn)
}

func (c *Client) Restart(ctx context.Context) RestartResult {
	return restart(ctx, c.Conn)
}

func (c *Client) SetLicense(ctx context.Context, license string) SetLicenseResult {
	return setLicense(ctx, c.Conn, license)
}

func (c *Client) RestartStream(ctx context.Context, streamID string) RestartItemResult {
	return restartStream(ctx, c.Conn, streamID)
}

func (c *Client) DeleteStream(ctx context.Context, streamID string) DeleteItemResult {
	return deleteStream(ctx, c.Conn, streamID)
}

func (c *Client) RestartAdapter(ctx context.Context, adapterID string) RestartItemResult {
	return restartAdapter(ctx, c.Conn, adapterID)
}

func (c *Client) DeleteAdapter(ctx context.Context, adapterID string) DeleteItemResult {
	return deleteAdapter(ctx, c.Conn, adapterID)
}

func (c *Client) ControlRequest(ctx context.Context, body []byte) ([]byte, error) {
	return controlRequest(ctx, c.Conn, body)
}
