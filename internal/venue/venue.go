package venue

import (
	"context"
	"time"
)

type Side string

const (
	SideBuy  Side = "buy"
	SideSell Side = "sell"
)

type OrderType string

const (
	OrderTypeMarket OrderType = "market"
	OrderTypeLimit  OrderType = "limit"
)

type OrderStatus string

const (
	OrderStatusNew      OrderStatus = "new"
	OrderStatusFilled   OrderStatus = "filled"
	OrderStatusCanceled OrderStatus = "canceled"
)

type OrderRequest struct {
	ClientOrderID string
	Symbol        string
	Side          Side
	Type          OrderType
	Quantity      float64
	Price         float64
}

type CancelRequest struct {
	VenueOrderID  string
	ClientOrderID string
}

type FillQuery struct {
	VenueOrderID string
	Start        time.Time
	End          time.Time
	Limit        int
}

type OrderResponse struct {
	VenueOrderID string
	ClientOrderID string
	Status        OrderStatus
	FilledQty     float64
	AvgPrice      float64
	Fills         []Fill
}

type Fill struct {
	VenueOrderID string
	TradeID      string
	Price        float64
	Quantity     float64
	Fee          float64
	FeeAsset     string
	Timestamp    time.Time
}

type Balances struct {
	Assets map[string]Balance
}

type Balance struct {
	Asset  string
	Free   float64
	Locked float64
}

type BookLevel struct {
	Price float64
	Size  float64
}

type BookUpdate struct {
	Pair   string
	Bids   []BookLevel
	Asks   []BookLevel
	TS     time.Time
}

type VenueConnector interface {
	PlaceOrder(ctx context.Context, req OrderRequest) (*OrderResponse, error)
	CancelOrder(ctx context.Context, req CancelRequest) error
	GetFills(ctx context.Context, query FillQuery) ([]Fill, error)
	GetBalances(ctx context.Context) (*Balances, error)
	SubscribeBook(ctx context.Context, pairs []string) (<-chan BookUpdate, error)
}

type VenueConfig struct {
	Name    string
	Price   float64
	Balance map[string]Balance
}