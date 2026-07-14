package venue

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
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
	OrderStatusPartial  OrderStatus = "partial"
	OrderStatusFilled   OrderStatus = "filled"
	OrderStatusCanceled OrderStatus = "canceled"
)

type OrderRequest struct {
	ClientOrderID string
	Symbol        string
	Side          Side
	Type          OrderType
	Quantity      decimal.Decimal
	Price         decimal.Decimal
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
	VenueOrderID  string
	ClientOrderID string
	Status        OrderStatus
	FilledQty     decimal.Decimal
	AvgPrice      decimal.Decimal
	Fills         []Fill
}

type Fill struct {
	VenueOrderID string
	TradeID      string
	Price        decimal.Decimal
	Quantity     decimal.Decimal
	Fee          decimal.Decimal
	FeeAsset     string
	Timestamp    time.Time
}

type Balances struct {
	Assets map[string]Balance
}

type Balance struct {
	Asset  string
	Free   decimal.Decimal
	Locked decimal.Decimal
}

type BookLevel struct {
	Price decimal.Decimal
	Size  decimal.Decimal
}

type BookUpdate struct {
	Pair string
	Bids []BookLevel
	Asks []BookLevel
	TS   time.Time
}

type VenueConnector interface {
	PlaceOrder(ctx context.Context, req OrderRequest) (*OrderResponse, error)
	CancelOrder(ctx context.Context, req CancelRequest) error
	GetOrder(ctx context.Context, venueOrderID string) (*OrderResponse, error)
	GetFills(ctx context.Context, query FillQuery) ([]Fill, error)
	GetBalances(ctx context.Context) (*Balances, error)
	SubscribeBook(ctx context.Context, pairs []string) (<-chan BookUpdate, error)
}

type VenueConfig struct {
	Name     string
	RESTBase string
	WSBase   string
	Knobs    map[string]string
}