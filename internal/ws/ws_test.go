package ws

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func startWSServer(t *testing.T, handler func(conn *websocket.Conn)) *httptest.Server {
	t.Helper()
	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		handler(conn)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestClientConnectAndReceive(t *testing.T) {
	srv := startWSServer(t, func(conn *websocket.Conn) {
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"hello":"world"}`))
		_ = conn.Close()
	})
	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	c := NewClient(url, "test")
	var got []byte
	c.SetHandler(func(msg []byte) {
		got = msg
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = c.Run(ctx)
	if string(got) != `{"hello":"world"}` {
		t.Fatalf("got: %s", got)
	}
}

func TestClientReconnect(t *testing.T) {
	connects := 0
	srv := startWSServer(t, func(conn *websocket.Conn) {
		connects++
		_ = conn.WriteMessage(websocket.TextMessage, []byte("msg"))
		_ = conn.Close()
	})
	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	c := NewClient(url, "test")
	c.maxBackoff = 50 * time.Millisecond
	c.SetHandler(func(msg []byte) {})
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_ = c.Run(ctx)
	if connects < 2 {
		t.Fatalf("expected reconnects, got %d", connects)
	}
}

func TestBookReconstructorSnapshotAndDiff(t *testing.T) {
	r := NewBookReconstructor("binance", "BTCUSDT")
	r.ApplySnapshot([][2]string{{"50000", "1"}, {"49999", "2"}}, [][2]string{{"50001", "0.5"}})
	bids, asks := r.TopN(10)
	if len(bids) != 2 || bids[0][0] != "50000" {
		t.Fatalf("bids: %+v", bids)
	}
	if len(asks) != 1 || asks[0][0] != "50001" {
		t.Fatalf("asks: %+v", asks)
	}
}

func TestBookReconstructorGapDetection(t *testing.T) {
	r := NewBookReconstructor("binance", "BTCUSDT")
	r.ApplySnapshot([][2]string{{"50000", "1"}}, [][2]string{{"50001", "1"}})
	_, gap := r.ApplyDiff(100, [][2]string{{"49999", "1"}}, nil)
	if gap {
		t.Fatalf("first diff should not be gap")
	}
	_, gap = r.ApplyDiff(105, [][2]string{{"49998", "1"}}, nil)
	if !gap {
		t.Fatalf("expected gap on seq jump")
	}
}

func TestBookReconstructorRemoveLevel(t *testing.T) {
	r := NewBookReconstructor("binance", "BTCUSDT")
	r.ApplySnapshot([][2]string{{"50000", "1"}, {"49999", "2"}}, [][2]string{{"50001", "0.5"}})
	_, gap := r.ApplyDiff(2, [][2]string{{"50000", "0"}}, nil)
	if gap {
		t.Fatalf("unexpected gap")
	}
	bids, _ := r.TopN(10)
	if len(bids) != 1 {
		t.Fatalf("bids after remove: %+v", bids)
	}
}

func TestBookReconstructorMarkGapResetsSeq(t *testing.T) {
	r := NewBookReconstructor("binance", "BTCUSDT")
	r.ApplySnapshot([][2]string{{"50000", "1"}}, [][2]string{{"50001", "1"}})
	_, _ = r.ApplyDiff(5, [][2]string{{"49999", "1"}}, nil)
	r.MarkGap()
	ok, _ := r.ApplyDiff(6, [][2]string{{"49998", "1"}}, nil)
	if !ok {
		t.Fatalf("after MarkGap, diff should apply (snapshot refresh)")
	}
}

func TestNextBackoff(t *testing.T) {
	b := nextBackoff(100*time.Millisecond, 30*time.Second)
	if b < 100*time.Millisecond {
		t.Fatalf("backoff too small: %v", b)
	}
	big := nextBackoff(60*time.Second, 30*time.Second)
	if big != 30*time.Second {
		t.Fatalf("expected capped at max, got %v", big)
	}
}

func TestParseMessage(t *testing.T) {
	m, err := ParseMessage([]byte(`{"a":1}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if m["a"].(float64) != 1 {
		t.Fatalf("a: %v", m["a"])
	}
}

func TestClientConnect(t *testing.T) {
	srv := startWSServer(t, func(conn *websocket.Conn) {
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"ok":true}`))
		_ = conn.Close()
	})
	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	c := NewClient(url, "test")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := c.Connect(ctx); err != nil {
		t.Fatalf("connect: %v", err)
	}
	if err := c.WriteJSON(map[string]string{"a": "1"}); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := c.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}

func TestClientConnectError(t *testing.T) {
	c := NewClient("ws://127.0.0.1:0/nonexistent", "test")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := c.Connect(ctx); err == nil {
		t.Fatalf("expected connect error")
	}
	if err := c.WriteJSON(map[string]string{"a": "1"}); err == nil {
		t.Fatalf("expected write error when not connected")
	}
}

func TestClientCloseNoConn(t *testing.T) {
	c := NewClient("ws://127.0.0.1:0/x", "test")
	if err := c.Close(); err != nil {
		t.Fatalf("close nil conn: %v", err)
	}
}

func TestBookReconstructorLagZero(t *testing.T) {
	r := NewBookReconstructor("binance", "BTCUSDT")
	if r.Lag() != 0 {
		t.Fatalf("lag should be 0 for fresh reconstructor: %v", r.Lag())
	}
}

func TestBookReconstructorLagAfterUpdate(t *testing.T) {
	r := NewBookReconstructor("binance", "BTCUSDT")
	r.ApplySnapshot([][2]string{{"50000", "1"}}, [][2]string{{"50001", "1"}})
	if r.Lag() < 0 {
		t.Fatalf("lag should be non-negative after update: %v", r.Lag())
	}
	_, gap := r.ApplyDiff(2, [][2]string{{"49999", "1"}}, nil)
	if gap {
		t.Fatalf("unexpected gap")
	}
	if r.Lag() < 0 {
		t.Fatalf("lag should be non-negative after diff: %v", r.Lag())
	}
}

func TestBookReconstructorApplyDiffStaleSeq(t *testing.T) {
	r := NewBookReconstructor("binance", "BTCUSDT")
	r.ApplySnapshot([][2]string{{"50000", "1"}}, [][2]string{{"50001", "1"}})
	_, gap := r.ApplyDiff(5, [][2]string{{"49999", "1"}}, nil)
	if gap {
		t.Fatalf("unexpected gap on first diff")
	}
	ok, gap := r.ApplyDiff(5, [][2]string{{"49998", "1"}}, nil)
	if ok || gap {
		t.Fatalf("stale seq should return false,false, got ok=%v gap=%v", ok, gap)
	}
}

func TestBookReconstructorTopNEmpty(t *testing.T) {
	r := NewBookReconstructor("binance", "BTCUSDT")
	bids, asks := r.TopN(10)
	if len(bids) != 0 || len(asks) != 0 {
		t.Fatalf("expected empty, got bids=%v asks=%v", bids, asks)
	}
}

func TestParseMessageError(t *testing.T) {
	if _, err := ParseMessage([]byte(`{bad json`)); err == nil {
		t.Fatalf("expected parse error")
	}
}