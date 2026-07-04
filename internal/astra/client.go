package astra

import "context"

type AstraClient struct {
	Conn AstraConnection
}

func NewAstraClient(conn AstraConnection) *AstraClient {
	return &AstraClient{
		Conn: conn,
	}
}

func (c *AstraClient) Load(ctx context.Context) AstraLoadResult {
	return AstraLoad(ctx, c.Conn)
}

func (c *AstraClient) GetStatus(ctx context.Context) AstraStatusResult {
	return AstraGetStatus(ctx, c.Conn)
}

func (c *AstraClient) Restart(ctx context.Context) AstraRestartResult {
	return AstraRestart(ctx, c.Conn)
}

func (c *AstraClient) SetLicense(ctx context.Context, license string) AstraSetLicenseResult {
	return AstraSetLicense(ctx, c.Conn, license)
}

func (c *AstraClient) RestartStream(ctx context.Context, streamID string) AstraRestartItemResult {
	return AstraRestartStream(ctx, c.Conn, streamID)
}

func (c *AstraClient) DeleteStream(ctx context.Context, streamID string) AstraDeleteItemResult {
	return AstraDeleteStream(ctx, c.Conn, streamID)
}

func (c *AstraClient) RestartAdapter(ctx context.Context, adapterID string) AstraRestartItemResult {
	return AstraRestartAdapter(ctx, c.Conn, adapterID)
}

func (c *AstraClient) DeleteAdapter(ctx context.Context, adapterID string) AstraDeleteItemResult {
	return AstraDeleteAdapter(ctx, c.Conn, adapterID)
}

func (c *AstraClient) ControlRequest(ctx context.Context, body []byte) ([]byte, error) {
	return astraControlRequest(ctx, c.Conn, body)
}
