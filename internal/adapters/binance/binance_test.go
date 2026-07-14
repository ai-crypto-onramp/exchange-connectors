package binance

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ai-crypto-onramp/exchange-connectors/internal/ratelimit"
	"github.com/ai-crypto-onramp/exchange-connectors/internal/venue"
	"github.com/shopspring/decimal"
)

func newTestConnector(t *testing.T, fn func(w http.ResponseWriter, r *http.Request)) (*Connector, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(fn))
	t.Cleanup(srv.Close)
	c := NewConnector("testkey", "testsecret", ratelimit.NewWeightedLimiter("binance", 1200))
	c.cfg.RESTBase = srv.URL
	return c, srv
}

func TestPlaceOrder(t *testing.T) {
	c, _ := newTestConnector(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-MBX-APIKEY") != "testkey" {
			t.Errorf("missing api key header")
		}
		if r.URL.Query().Get("signature") == "" {
			t.Errorf("missing signature")
		}
		if r.URL.Query().Get("timestamp") == "" {
			t.Errorf("missing timestamp")
		}
		if r.URL.Query().Get("recvWindow") == "" {
			t.Errorf("missing recvWindow")
		}
		_, _ = w.Write([]byte(`{"orderId":123456,"clientOrderId":"c1","status":"FILLED","executedQty":"0.5","cummulativeQuoteQty":"25000","fills":[{"price":"50000","qty":"0.5","commission":"5","commissionAsset":"USDT","tradeId":777}]}`))
	})
	resp, err := c.PlaceOrder(context.Background(), venue.OrderRequest{
		ClientOrderID: "c1",
		Symbol:       "BTCUSDT",
		Side:         venue.SideBuy,
		Type:         venue.OrderTypeMarket,
		Quantity:     decimal.NewFromFloat(0.5),
	})
	if err != nil {
		t.Fatalf("place: %v", err)
	}
	if resp.VenueOrderID != "123456" {
		t.Fatalf("order id: %s", resp.VenueOrderID)
	}
	if resp.Status != venue.OrderStatusFilled {
		t.Fatalf("status: %s", resp.Status)
	}
	if !resp.FilledQty.Equal(decimal.NewFromFloat(0.5)) {
		t.Fatalf("filled: %v", resp.FilledQty)
	}
	if !resp.AvgPrice.Equal(decimal.NewFromInt(50000)) {
		t.Fatalf("avg: %v", resp.AvgPrice)
	}
	if len(resp.Fills) != 1 || resp.Fills[0].TradeID != "777" {
		t.Fatalf("fills: %+v", resp.Fills)
	}
}

func TestPlaceOrderLimit(t *testing.T) {
	c, _ := newTestConnector(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("price") != "51000" {
			t.Errorf("price param: %s", r.URL.Query().Get("price"))
		}
		if r.URL.Query().Get("timeInForce") != "GTC" {
			t.Errorf("tif: %s", r.URL.Query().Get("timeInForce"))
		}
		_, _ = w.Write([]byte(`{"orderId":1,"clientOrderId":"c2","status":"NEW","executedQty":"0","cummulativeQuoteQty":"0"}`))
	})
	resp, err := c.PlaceOrder(context.Background(), venue.OrderRequest{
		ClientOrderID: "c2",
		Symbol:       "BTCUSDT",
		Side:         venue.SideBuy,
		Type:         venue.OrderTypeLimit,
		Quantity:     decimal.NewFromInt(1),
		Price:        decimal.NewFromInt(51000),
	})
	if err != nil {
		t.Fatalf("place: %v", err)
	}
	if resp.Status != venue.OrderStatusNew {
		t.Fatalf("status: %s", resp.Status)
	}
}

func TestGetOrder(t *testing.T) {
	c, _ := newTestConnector(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("orderId") != "123" {
			t.Errorf("orderId: %s", r.URL.Query().Get("orderId"))
		}
		_, _ = w.Write([]byte(`{"orderId":123,"clientOrderId":"c","status":"PARTIALLY_FILLED","executedQty":"0.25","cummulativeQuoteQty":"12500"}`))
	})
	o, err := c.GetOrder(context.Background(), "123")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if o.Status != venue.OrderStatusPartial {
		t.Fatalf("status: %s", o.Status)
	}
	if !o.FilledQty.Equal(decimal.NewFromFloat(0.25)) {
		t.Fatalf("filled: %v", o.FilledQty)
	}
	if !o.AvgPrice.Equal(decimal.NewFromInt(50000)) {
		t.Fatalf("avg: %v", o.AvgPrice)
	}
}

func TestCancelOrder(t *testing.T) {
	c, _ := newTestConnector(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method: %s", r.Method)
		}
		if r.URL.Query().Get("orderId") != "9" {
			t.Errorf("orderId: %s", r.URL.Query().Get("orderId"))
		}
		_, _ = w.Write([]byte(`{}`))
	})
	if err := c.CancelOrder(context.Background(), venue.CancelRequest{VenueOrderID: "9"}); err != nil {
		t.Fatalf("cancel: %v", err)
	}
}

func TestCancelOrderClientID(t *testing.T) {
	c, _ := newTestConnector(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("origClientOrderId") != "c9" {
			t.Errorf("client: %s", r.URL.Query().Get("origClientOrderId"))
		}
		_, _ = w.Write([]byte(`{}`))
	})
	if err := c.CancelOrder(context.Background(), venue.CancelRequest{ClientOrderID: "c9"}); err != nil {
		t.Fatalf("cancel: %v", err)
	}
}

func TestCancelOrderMissingID(t *testing.T) {
	c, _ := newTestConnector(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	})
	if err := c.CancelOrder(context.Background(), venue.CancelRequest{}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestGetFills(t *testing.T) {
	c, _ := newTestConnector(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("orderId") != "123" {
			t.Errorf("orderId: %s", r.URL.Query().Get("orderId"))
		}
		_, _ = w.Write([]byte(`{"fills":[{"id":1,"price":"50000","qty":"0.5","commission":"5","commissionAsset":"USDT","time":1700000000000}]}`))
	})
	fills, err := c.GetFills(context.Background(), venue.FillQuery{VenueOrderID: "123"})
	if err != nil {
		t.Fatalf("fills: %v", err)
	}
	if len(fills) != 1 {
		t.Fatalf("len: %d", len(fills))
	}
	if !fills[0].Price.Equal(decimal.NewFromInt(50000)) {
		t.Fatalf("price: %v", fills[0].Price)
	}
	if !fills[0].Quantity.Equal(decimal.NewFromFloat(0.5)) {
		t.Fatalf("qty: %v", fills[0].Quantity)
	}
	if fills[0].FeeAsset != "USDT" {
		t.Fatalf("feeAsset: %s", fills[0].FeeAsset)
	}
}

func TestGetBalances(t *testing.T) {
	c, _ := newTestConnector(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/api/v3/account") {
			t.Errorf("path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"balances":[{"asset":"BTC","free":"1.5","locked":"0.1"},{"asset":"USDT","free":"100","locked":"0"}]}`))
	})
	b, err := c.GetBalances(context.Background())
	if err != nil {
		t.Fatalf("balances: %v", err)
	}
	if !b.Assets["BTC"].Free.Equal(decimal.NewFromFloat(1.5)) {
		t.Fatalf("btc free: %v", b.Assets["BTC"].Free)
	}
	if !b.Assets["USDT"].Locked.Equal(decimal.Zero) {
		t.Fatalf("usdt locked: %v", b.Assets["USDT"].Locked)
	}
}

func TestRateLimit429(t *testing.T) {
	c, _ := newTestConnector(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-MBX-USED-WEIGHT-1M", "1200")
		w.WriteHeader(429)
		_, _ = w.Write([]byte(`{"code":-1003,"msg":"rate limited"}`))
	})
	_, err := c.GetBalances(context.Background())
	if err == nil || !strings.Contains(err.Error(), "rate limited") {
		t.Fatalf("expected rate limit err, got %v", err)
	}
}

func TestSignConsistent(t *testing.T) {
	c := NewConnector("k", "s", ratelimit.NewWeightedLimiter("binance", 1200))
	q := "symbol=BTCUSDT&side=buy&type=market&quantity=0.5&timestamp=123&recvWindow=5000"
	sig := c.sign(q)
	if sig == "" {
		t.Fatalf("empty signature")
	}
	sig2 := c.sign(q)
	if sig != sig2 {
		t.Fatalf("signature not deterministic: %s vs %s", sig, sig2)
	}
}

func TestVenueConfig(t *testing.T) {
	c := NewConnector("k", "s", ratelimit.NewWeightedLimiter("binance", 1200))
	cfg := c.Config()
	if cfg.Name != "binance" {
		t.Fatalf("name: %s", cfg.Name)
	}
	if cfg.RESTBase == "" {
		t.Fatalf("missing rest base")
	}
	if cfg.WSBase == "" {
		t.Fatalf("missing ws base")
	}
}

func TestVenueInterfaceSatisfaction(t *testing.T) {
	var _ venue.VenueConnector = (*Connector)(nil)
}

func TestParseURL(t *testing.T) {
	u, _ := url.Parse("https://api.binance.com/api/v3/order?symbol=BTCUSDT&signature=abc")
	if u.Path != "/api/v3/order" {
		t.Fatalf("path: %s", u.Path)
	}
}