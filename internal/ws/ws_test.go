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