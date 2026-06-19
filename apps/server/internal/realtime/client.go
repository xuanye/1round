package realtime

import (
	"context"
	"time"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type Client struct {
	ID            string
	UserID        string
	GameSessionID string
	Conn          *websocket.Conn
	Send          chan Event
}

func (c *Client) ReadLoop(ctx context.Context, onDone func()) {
	defer onDone()
	for {
		if _, _, err := c.Conn.Read(ctx); err != nil {
			return
		}
	}
}

func (c *Client) WriteLoop(ctx context.Context, writeTimeout time.Duration, onDone func()) {
	defer onDone()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-c.Send:
			if !ok {
				return
			}
			wctx, cancel := context.WithTimeout(ctx, writeTimeout)
			err := wsjsonWrite(wctx, c.Conn, ev)
			cancel()
			if err != nil {
				return
			}
		}
	}
}

var wsjsonWrite = func(ctx context.Context, conn *websocket.Conn, event Event) error {
	return wsjson.Write(ctx, conn, event)
}
