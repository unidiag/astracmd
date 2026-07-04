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
	return AstraLoad(ctx, c.Conn)
}

func (c *Client) GetStatus(ctx context.Context) StatusResult {
	return AstraGetStatus(ctx, c.Conn)
}

func (c *Client) Restart(ctx context.Context) RestartResult {
	return AstraRestart(ctx, c.Conn)
}

func (c *Client) SetLicense(ctx context.Context, license string) SetLicenseResult {
	return AstraSetLicense(ctx, c.Conn, license)
}

func (c *Client) RestartStream(ctx context.Context, streamID string) RestartItemResult {
	return AstraRestartStream(ctx, c.Conn, streamID)
}

func (c *Client) DeleteStream(ctx context.Context, streamID string) DeleteItemResult {
	return AstraDeleteStream(ctx, c.Conn, streamID)
}

func (c *Client) RestartAdapter(ctx context.Context, adapterID string) RestartItemResult {
	return AstraRestartAdapter(ctx, c.Conn, adapterID)
}

func (c *Client) DeleteAdapter(ctx context.Context, adapterID string) DeleteItemResult {
	return AstraDeleteAdapter(ctx, c.Conn, adapterID)
}

func (c *Client) ControlRequest(ctx context.Context, body []byte) ([]byte, error) {
	return astraControlRequest(ctx, c.Conn, body)
}
