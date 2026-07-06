package venue

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

// Compile-time assertions that the stub satisfies VenueConnector and that the
// interface is wired correctly.
var (
	_ VenueConnector = (*stubConnector)(nil)
	_ VenueConnector = stubConnector{}
	_ VenueConnector = newStubConnector()
)

func TestStubSatisfiesVenueConnector(t *testing.T) {
	t.Parallel()
	c := newStubConnector()
	ctx := context.Background()

	if _, err := c.PlaceOrder(ctx, OrderRequest{}); err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}
	if _, err := c.CancelOrder(ctx, CancelRequest{}); err != nil {
		t.Fatalf("CancelOrder: %v", err)
	}
	if _, err := c.GetOrder(ctx, "", "", ""); err != nil {
		t.Fatalf("GetOrder: %v", err)
	}
	if fills, err := c.GetFills(ctx, FillQuery{}); err != nil || fills != nil {
		t.Fatalf("GetFills: fills=%v err=%v", fills, err)
	}
	if _, err := c.GetBalances(ctx); err != nil {
		t.Fatalf("GetBalances: %v", err)
	}
	ch, err := c.SubscribeOrderBook(ctx, "BTCUSDT")
	if ch != nil {
		t.Fatalf("SubscribeOrderBook: expected nil channel, got %v", ch)
	}
	if err == nil {
		t.Fatalf("SubscribeOrderBook: expected error, got nil")
	}
}

func TestSideString(t *testing.T) {
	t.Parallel()
	cases := map[Side]string{
		SideUnknown: "unknown",
		SideBuy:     "buy",
		SideSell:    "sell",
	}
	for in, want := range cases {
		if got := in.String(); got != want {
			t.Errorf("Side(%d).String() = %q, want %q", in, got, want)
		}
	}
}

func TestOrderTypeString(t *testing.T) {
	t.Parallel()
	cases := map[OrderType]string{
		OrderTypeUnknown: "unknown",
		OrderTypeLimit:   "limit",
		OrderTypeMarket:  "market",
	}
	for in, want := range cases {
		if got := in.String(); got != want {
			t.Errorf("OrderType(%d).String() = %q, want %q", in, got, want)
		}
	}
}

func TestOrderStatusString(t *testing.T) {
	t.Parallel()
	cases := map[OrderStatus]string{
		OrderStatusUnknown:         "unknown",
		OrderStatusPending:         "pending",
		OrderStatusOpen:            "open",
		OrderStatusPartiallyFilled: "partially_filled",
		OrderStatusFilled:          "filled",
		OrderStatusCanceled:        "canceled",
		OrderStatusRejected:        "rejected",
		OrderStatusExpired:         "expired",
	}
	for in, want := range cases {
		if got := in.String(); got != want {
			t.Errorf("OrderStatus(%d).String() = %q, want %q", in, got, want)
		}
	}
}

func TestVenueConfigRoundTrip(t *testing.T) {
	t.Parallel()
	in := VenueConfig{
		Name:           "binance",
		RESTBaseURL:    "https://api.binance.com",
		WSBaseURL:      "wss://stream.binance.com:9443/ws",
		RequestTimeout: 5 * time.Second,
		RecvWindow:     1000 * time.Millisecond,
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out VenueConfig
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out != in {
		t.Errorf("VenueConfig round trip mismatch:\n in=%+v\nout=%+v", in, out)
	}
}

func TestOrderRequestSerialization(t *testing.T) {
	t.Parallel()
	in := OrderRequest{
		Symbol:        "BTCUSDT",
		Side:          SideBuy,
		Type:          OrderTypeLimit,
		Quantity:      decimal.NewFromFloat(0.5),
		Price:         decimal.NewFromFloat(30000.5),
		ClientOrderID: "client-1",
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out OrderRequest
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Symbol != in.Symbol || out.Side != in.Side || out.Type != in.Type ||
		out.ClientOrderID != in.ClientOrderID ||
		!out.Quantity.Equal(in.Quantity) || !out.Price.Equal(in.Price) {
		t.Errorf("OrderRequest round trip mismatch:\n in=%+v\nout=%+v", in, out)
	}
}

func TestVenueOrderSerialization(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)
	in := VenueOrder{
		VenueOrderID:     "v-1",
		ClientOrderID:    "client-1",
		Symbol:           "BTCUSDT",
		Side:             SideSell,
		Type:             OrderTypeMarket,
		Status:           OrderStatusPartiallyFilled,
		Quantity:         decimal.NewFromFloat(1.25),
		Price:            decimal.Zero,
		ExecutedQuantity: decimal.NewFromFloat(0.25),
		AvgFillPrice:     decimal.NewFromFloat(30100.0),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out VenueOrder
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.VenueOrderID != in.VenueOrderID || out.ClientOrderID != in.ClientOrderID ||
		out.Symbol != in.Symbol || out.Side != in.Side || out.Type != in.Type ||
		out.Status != in.Status || !out.Quantity.Equal(in.Quantity) ||
		!out.ExecutedQuantity.Equal(in.ExecutedQuantity) || !out.AvgFillPrice.Equal(in.AvgFillPrice) ||
		!out.CreatedAt.Equal(in.CreatedAt) || !out.UpdatedAt.Equal(in.UpdatedAt) {
		t.Errorf("VenueOrder round trip mismatch:\n in=%+v\nout=%+v", in, out)
	}
}

func TestFillSerialization(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)
	in := Fill{
		FillID:        "f-1",
		VenueOrderID:  "v-1",
		ClientOrderID: "client-1",
		Symbol:        "BTCUSDT",
		Side:          SideBuy,
		Price:         decimal.NewFromFloat(30100.5),
		Quantity:      decimal.NewFromFloat(0.25),
		Fee:           decimal.NewFromFloat(0.0001),
		FeeCurrency:   "BNB",
		Time:          now,
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out Fill
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.FillID != in.FillID || out.VenueOrderID != in.VenueOrderID ||
		out.ClientOrderID != in.ClientOrderID || out.Symbol != in.Symbol ||
		out.Side != in.Side || !out.Price.Equal(in.Price) ||
		!out.Quantity.Equal(in.Quantity) || !out.Fee.Equal(in.Fee) ||
		out.FeeCurrency != in.FeeCurrency || !out.Time.Equal(in.Time) {
		t.Errorf("Fill round trip mismatch:\n in=%+v\nout=%+v", in, out)
	}
}

func TestBalancesSerialization(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)
	in := Balances{
		UpdatedAt: now,
		Assets: map[string]Balance{
			"BTC":  {Asset: "BTC", Available: decimal.NewFromFloat(1.0), Total: decimal.NewFromFloat(1.5)},
			"USDT": {Asset: "USDT", Available: decimal.NewFromFloat(1000.0), Total: decimal.NewFromFloat(1200.0)},
		},
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out Balances
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !out.UpdatedAt.Equal(in.UpdatedAt) || len(out.Assets) != len(in.Assets) {
		t.Errorf("Balances round trip mismatch:\n in=%+v\nout=%+v", in, out)
	}
	for k, v := range in.Assets {
		got, ok := out.Assets[k]
		if !ok {
			t.Errorf("missing asset %q", k)
			continue
		}
		if got.Asset != v.Asset || !got.Available.Equal(v.Available) || !got.Total.Equal(v.Total) {
			t.Errorf("asset %q mismatch:\n in=%+v\nout=%+v", k, v, got)
		}
	}
}

func TestBookUpdateSerialization(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)
	in := BookUpdate{
		Symbol:   "BTCUSDT",
		Snapshot: true,
		Sequence: 42,
		Bids:     [][2]decimal.Decimal{{decimal.NewFromFloat(30100.0), decimal.NewFromFloat(1.5)}},
		Asks:     [][2]decimal.Decimal{{decimal.NewFromFloat(30100.5), decimal.NewFromFloat(0.75)}},
		Time:     now,
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out BookUpdate
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Symbol != in.Symbol || out.Snapshot != in.Snapshot || out.Sequence != in.Sequence ||
		!out.Time.Equal(in.Time) || len(out.Bids) != len(in.Bids) || len(out.Asks) != len(in.Asks) {
		t.Errorf("BookUpdate round trip mismatch:\n in=%+v\nout=%+v", in, out)
	}
	for i, lvl := range in.Bids {
		if !out.Bids[i][0].Equal(lvl[0]) || !out.Bids[i][1].Equal(lvl[1]) {
			t.Errorf("bid level %d mismatch: in=%v out=%v", i, lvl, out.Bids[i])
		}
	}
	for i, lvl := range in.Asks {
		if !out.Asks[i][0].Equal(lvl[0]) || !out.Asks[i][1].Equal(lvl[1]) {
			t.Errorf("ask level %d mismatch: in=%v out=%v", i, lvl, out.Asks[i])
		}
	}
}

func TestSubscribeOrderBookStubUnsupported(t *testing.T) {
	t.Parallel()
	c := newStubConnector()
	ch, err := c.SubscribeOrderBook(context.Background(), "BTCUSDT")
	if ch != nil {
		t.Fatalf("expected nil channel, got %v", ch)
	}
	if err == nil {
		t.Fatalf("expected non-nil error, got nil")
	}
}
