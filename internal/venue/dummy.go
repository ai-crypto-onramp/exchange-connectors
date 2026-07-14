package venue

import (
	"context"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type DummyVenueConnector struct {
	mu     sync.Mutex
	price  decimal.Decimal
	fills  []Fill
	orders map[string]*OrderResponse
}

func NewDummyVenueConnector() *DummyVenueConnector {
	price := decimal.NewFromFloat(50000)
	if v := os.Getenv("DUMMY_PRICE"); v != "" {
		if p, err := decimal.NewFromString(v); err == nil {
			price = p
		}
	}
	return &DummyVenueConnector{
		price:  price,
		orders: make(map[string]*OrderResponse),
	}
}

var _ VenueConnector = (*DummyVenueConnector)(nil)

func (d *DummyVenueConnector) PlaceOrder(ctx context.Context, req OrderRequest) (*OrderResponse, error) {
	if req.ClientOrderID == "" {
		req.ClientOrderID = uuid.NewString()
	}
	price := req.Price
	if price.IsZero() {
		d.mu.Lock()
		price = d.price
		d.mu.Unlock()
	}
	qty := req.Quantity
	fill := Fill{
		VenueOrderID: req.ClientOrderID,
		TradeID:      uuid.NewString(),
		Price:        price,
		Quantity:     qty,
		Fee:          price.Mul(qty).Mul(decimal.NewFromFloat(0.001)),
		FeeAsset:     "USDT",
		Timestamp:    time.Now().UTC(),
	}
	resp := &OrderResponse{
		VenueOrderID:  req.ClientOrderID,
		ClientOrderID: req.ClientOrderID,
		Status:        OrderStatusFilled,
		FilledQty:     qty,
		AvgPrice:      price,
		Fills:         []Fill{fill},
	}
	d.mu.Lock()
	d.fills = append(d.fills, fill)
	d.orders[req.ClientOrderID] = resp
	d.mu.Unlock()
	return resp, nil
}

func (d *DummyVenueConnector) CancelOrder(ctx context.Context, req CancelRequest) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	id := req.VenueOrderID
	if id == "" {
		id = req.ClientOrderID
	}
	if o, ok := d.orders[id]; ok {
		o.Status = OrderStatusCanceled
	}
	return nil
}

func (d *DummyVenueConnector) GetOrder(ctx context.Context, venueOrderID string) (*OrderResponse, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if o, ok := d.orders[venueOrderID]; ok {
		cp := *o
		return &cp, nil
	}
	return nil, ErrOrderNotFound
}

func (d *DummyVenueConnector) GetFills(ctx context.Context, query FillQuery) ([]Fill, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	var out []Fill
	for _, f := range d.fills {
		if query.VenueOrderID != "" && f.VenueOrderID != query.VenueOrderID {
			continue
		}
		if !query.Start.IsZero() && f.Timestamp.Before(query.Start) {
			continue
		}
		if !query.End.IsZero() && f.Timestamp.After(query.End) {
			continue
		}
		out = append(out, f)
	}
	if query.Limit > 0 && len(out) > query.Limit {
		out = out[:query.Limit]
	}
	return out, nil
}

func (d *DummyVenueConnector) GetBalances(ctx context.Context) (*Balances, error) {
	assets := map[string]Balance{
		"BTC":  {Asset: "BTC", Free: decimal.NewFromFloat(1.5), Locked: decimal.NewFromFloat(0.1)},
		"USDT": {Asset: "USDT", Free: decimal.NewFromFloat(250000), Locked: decimal.NewFromFloat(1000)},
		"ETH":  {Asset: "ETH", Free: decimal.NewFromFloat(20), Locked: decimal.Zero},
	}
	return &Balances{Assets: assets}, nil
}

func (d *DummyVenueConnector) SubscribeBook(ctx context.Context, pairs []string) (<-chan BookUpdate, error) {
	ch := make(chan BookUpdate, 16)
	go func() {
		defer close(ch)
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				d.mu.Lock()
				base := d.price
				d.mu.Unlock()
				for _, pair := range pairs {
					delta := decimal.NewFromFloat(rand.Float64() - 0.5).Mul(base).Mul(decimal.NewFromFloat(0.001))
					mid := base.Add(delta)
					upd := BookUpdate{
						Pair: pair,
						Bids: []BookLevel{{Price: mid.Sub(decimal.NewFromInt(1)), Size: decimal.NewFromFloat(0.5)}},
						Asks: []BookLevel{{Price: mid.Add(decimal.NewFromInt(1)), Size: decimal.NewFromFloat(0.5)}},
						TS:   time.Now().UTC(),
					}
					select {
					case ch <- upd:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()
	return ch, nil
}

func (d *DummyVenueConnector) LastFill() *Fill {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.fills) == 0 {
		return nil
	}
	f := d.fills[len(d.fills)-1]
	return &f
}