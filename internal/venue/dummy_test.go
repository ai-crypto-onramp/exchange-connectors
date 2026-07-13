package venue

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestDummyPlaceOrderFill(t *testing.T) {
	c := NewDummyVenueConnector()
	resp, err := c.PlaceOrder(context.Background(), OrderRequest{
		ClientOrderID: "c1",
		Symbol:        "BTCUSDT",
		Side:          SideBuy,
		Type:          OrderTypeMarket,
		Quantity:      0.5,
	})
	if err != nil {
		t.Fatalf("place: %v", err)
	}
	if resp.Status != OrderStatusFilled {
		t.Fatalf("expected filled, got %s", resp.Status)
	}
	if resp.FilledQty != 0.5 {
		t.Fatalf("filled qty: %v", resp.FilledQty)
	}
	if resp.VenueOrderID != "c1" {
		t.Fatalf("venue order id: %s", resp.VenueOrderID)
	}
	if len(resp.Fills) != 1 {
		t.Fatalf("expected 1 fill, got %d", len(resp.Fills))
	}
	if resp.Fills[0].Price <= 0 {
		t.Fatalf("price should be positive: %v", resp.Fills[0].Price)
	}
	if resp.Fills[0].FeeAsset != "USDT" {
		t.Fatalf("fee asset: %s", resp.Fills[0].FeeAsset)
	}
}

func TestDummyPlaceOrderLimitPrice(t *testing.T) {
	c := NewDummyVenueConnector()
	resp, err := c.PlaceOrder(context.Background(), OrderRequest{
		ClientOrderID: "c2",
		Symbol:        "BTCUSDT",
		Side:          SideSell,
		Type:          OrderTypeLimit,
		Quantity:      1,
		Price:         51000,
	})
	if err != nil {
		t.Fatalf("place: %v", err)
	}
	if resp.AvgPrice != 51000 {
		t.Fatalf("expected limit price 51000, got %v", resp.AvgPrice)
	}
}

func TestDummyCancelOrder(t *testing.T) {
	c := NewDummyVenueConnector()
	_, _ = c.PlaceOrder(context.Background(), OrderRequest{
		ClientOrderID: "c3", Symbol: "BTCUSDT", Side: SideBuy, Type: OrderTypeMarket, Quantity: 1,
	})
	if err := c.CancelOrder(context.Background(), CancelRequest{VenueOrderID: "c3"}); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	c.mu.Lock()
	o := c.orders["c3"]
	c.mu.Unlock()
	if o.Status != OrderStatusCanceled {
		t.Fatalf("expected canceled, got %s", o.Status)
	}
}

func TestDummyCancelOrderClientID(t *testing.T) {
	c := NewDummyVenueConnector()
	_, _ = c.PlaceOrder(context.Background(), OrderRequest{
		ClientOrderID: "c4", Symbol: "BTCUSDT", Side: SideBuy, Type: OrderTypeMarket, Quantity: 1,
	})
	if err := c.CancelOrder(context.Background(), CancelRequest{ClientOrderID: "c4"}); err != nil {
		t.Fatalf("cancel: %v", err)
	}
}

func TestDummyGetFillsByOrder(t *testing.T) {
	c := NewDummyVenueConnector()
	_, _ = c.PlaceOrder(context.Background(), OrderRequest{
		ClientOrderID: "c5", Symbol: "BTCUSDT", Side: SideBuy, Type: OrderTypeMarket, Quantity: 1,
	})
	_, _ = c.PlaceOrder(context.Background(), OrderRequest{
		ClientOrderID: "c6", Symbol: "BTCUSDT", Side: SideBuy, Type: OrderTypeMarket, Quantity: 1,
	})
	fills, err := c.GetFills(context.Background(), FillQuery{VenueOrderID: "c5"})
	if err != nil {
		t.Fatalf("getfills: %v", err)
	}
	if len(fills) != 1 {
		t.Fatalf("expected 1 fill, got %d", len(fills))
	}
	if fills[0].VenueOrderID != "c5" {
		t.Fatalf("wrong fill order: %s", fills[0].VenueOrderID)
	}
}

func TestDummyGetFillsTimeWindow(t *testing.T) {
	c := NewDummyVenueConnector()
	start := time.Now().Add(-1 * time.Hour)
	_, _ = c.PlaceOrder(context.Background(), OrderRequest{
		ClientOrderID: "c7", Symbol: "BTCUSDT", Side: SideBuy, Type: OrderTypeMarket, Quantity: 1,
	})
	end := time.Now().Add(1 * time.Hour)
	fills, err := c.GetFills(context.Background(), FillQuery{Start: start, End: end})
	if err != nil {
		t.Fatalf("getfills: %v", err)
	}
	if len(fills) != 1 {
		t.Fatalf("expected 1 fill in window, got %d", len(fills))
	}
	fills2, _ := c.GetFills(context.Background(), FillQuery{Start: time.Now().Add(1 * time.Minute)})
	if len(fills2) != 0 {
		t.Fatalf("expected 0 fills in future window, got %d", len(fills2))
	}
}

func TestDummyGetFillsLimit(t *testing.T) {
	c := NewDummyVenueConnector()
	for i := 0; i < 5; i++ {
		_, _ = c.PlaceOrder(context.Background(), OrderRequest{
			ClientOrderID: "o" + strconv.Itoa(i), Symbol: "BTCUSDT", Side: SideBuy, Type: OrderTypeMarket, Quantity: 1,
		})
	}
	fills, _ := c.GetFills(context.Background(), FillQuery{Limit: 2})
	if len(fills) != 2 {
		t.Fatalf("expected 2 fills, got %d", len(fills))
	}
}

func TestDummyGetBalances(t *testing.T) {
	c := NewDummyVenueConnector()
	b, err := c.GetBalances(context.Background())
	if err != nil {
		t.Fatalf("balances: %v", err)
	}
	if b.Assets["BTC"].Free != 1.5 {
		t.Fatalf("btc free: %v", b.Assets["BTC"].Free)
	}
	if b.Assets["USDT"].Locked != 1000 {
		t.Fatalf("usdt locked: %v", b.Assets["USDT"].Locked)
	}
}

func TestDummySubscribeBook(t *testing.T) {
	c := NewDummyVenueConnector()
	ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
	defer cancel()
	ch, err := c.SubscribeBook(ctx, []string{"BTCUSDT"})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	got := 0
	for upd := range ch {
		if upd.Pair != "BTCUSDT" {
			t.Fatalf("unexpected pair: %s", upd.Pair)
		}
		if len(upd.Bids) == 0 || len(upd.Asks) == 0 {
			t.Fatalf("empty book")
		}
		if upd.Bids[0].Price >= upd.Asks[0].Price {
			t.Fatalf("bid >= ask: %v >= %v", upd.Bids[0].Price, upd.Asks[0].Price)
		}
		got++
	}
	if got == 0 {
		t.Fatalf("expected at least 1 update")
	}
}

func TestDummyLastFill(t *testing.T) {
	c := NewDummyVenueConnector()
	if lf := c.LastFill(); lf != nil {
		t.Fatalf("expected nil last fill, got %+v", lf)
	}
	_, _ = c.PlaceOrder(context.Background(), OrderRequest{
		ClientOrderID: "c8", Symbol: "BTCUSDT", Side: SideBuy, Type: OrderTypeMarket, Quantity: 1,
	})
	lf := c.LastFill()
	if lf == nil {
		t.Fatalf("expected last fill")
	}
	if lf.VenueOrderID != "c8" {
		t.Fatalf("last fill order: %s", lf.VenueOrderID)
	}
}

func TestDummyPriceFromEnv(t *testing.T) {
	os.Setenv("DUMMY_PRICE", "12345")
	defer os.Unsetenv("DUMMY_PRICE")
	c := NewDummyVenueConnector()
	resp, _ := c.PlaceOrder(context.Background(), OrderRequest{
		ClientOrderID: "c9", Symbol: "BTCUSDT", Side: SideBuy, Type: OrderTypeMarket, Quantity: 1,
	})
	if resp.AvgPrice != 12345 {
		t.Fatalf("expected env price 12345, got %v", resp.AvgPrice)
	}
}

func TestVenueInterfaceSatisfaction(t *testing.T) {
	var _ VenueConnector = (*DummyVenueConnector)(nil)
}