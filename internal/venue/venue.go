// Package venue defines the uniform VenueConnector interface and shared domain
// types that every venue adapter implements, so Liquidity Routing is agnostic to
// the underlying venue.
//
// No venue-specific code leaks into this package: adapters live under
// internal/adapters/<venue> and satisfy the VenueConnector interface declared
// here. All monetary and quantity fields use decimal.Decimal to avoid
// floating-point rounding errors across venues.
package venue

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

// Side indicates the direction of an order.
//
//go:generate stringer -type=Side
type Side int

const (
	// SideUnknown is the zero value and should not be used for real orders.
	SideUnknown Side = iota
	// SideBuy indicates a buy order.
	SideBuy
	// SideSell indicates a sell order.
	SideSell
)

// String returns a stable string representation of a Side used in serialization
// and logs.
func (s Side) String() string {
	switch s {
	case SideBuy:
		return "buy"
	case SideSell:
		return "sell"
	default:
		return "unknown"
	}
}

// OrderType indicates the execution semantics of an order.
type OrderType int

const (
	// OrderTypeUnknown is the zero value and should not be used for real orders.
	OrderTypeUnknown OrderType = iota
	// OrderTypeLimit is a limit order that executes at a specified price or better.
	OrderTypeLimit
	// OrderTypeMarket is a market order that executes immediately at the best
	// available price.
	OrderTypeMarket
)

// String returns a stable string representation of an OrderType.
func (t OrderType) String() string {
	switch t {
	case OrderTypeLimit:
		return "limit"
	case OrderTypeMarket:
		return "market"
	default:
		return "unknown"
	}
}

// OrderStatus indicates the lifecycle state of an order as reported by a venue.
type OrderStatus int

const (
	// OrderStatusUnknown is the zero value and indicates the venue did not report
	// a recognizable state.
	OrderStatusUnknown OrderStatus = iota
	// OrderStatusPending indicates the order has been accepted but not yet working
	// on the book.
	OrderStatusPending
	// OrderStatusOpen indicates the order is working on the venue book.
	OrderStatusOpen
	// OrderStatusPartiallyFilled indicates the order has been partially executed.
	OrderStatusPartiallyFilled
	// OrderStatusFilled indicates the order has been fully executed.
	OrderStatusFilled
	// OrderStatusCanceled indicates the order has been canceled.
	OrderStatusCanceled
	// OrderStatusRejected indicates the order was rejected by the venue.
	OrderStatusRejected
	// OrderStatusExpired indicates the order has expired per venue rules.
	OrderStatusExpired
)

// String returns a stable string representation of an OrderStatus.
func (s OrderStatus) String() string {
	switch s {
	case OrderStatusPending:
		return "pending"
	case OrderStatusOpen:
		return "open"
	case OrderStatusPartiallyFilled:
		return "partially_filled"
	case OrderStatusFilled:
		return "filled"
	case OrderStatusCanceled:
		return "canceled"
	case OrderStatusRejected:
		return "rejected"
	case OrderStatusExpired:
		return "expired"
	default:
		return "unknown"
	}
}

// VenueConfig holds non-secret, per-venue configuration used by adapters to
// build REST and WebSocket clients. Secret material (API keys, signing secrets)
// is intentionally excluded and must be supplied via the secrets package at
// runtime.
type VenueConfig struct {
	// Name is the canonical venue identifier (e.g. "binance", "kraken", "otc").
	Name string
	// RESTBaseURL is the base URL for the venue REST API.
	RESTBaseURL string
	// WSBaseURL is the base URL for the venue WebSocket stream API.
	WSBaseURL string
	// RequestTimeout is the per-REST-request timeout. Zero means no timeout;
	// adapters should apply a sensible default.
	RequestTimeout time.Duration
	// RecvWindow is the optional receive-window knob venues like Binance use to
	// bound request validity. Zero means venue default.
	RecvWindow time.Duration
}

// OrderRequest is the internal representation of an order placement intent,
// translated by adapters into venue-specific payloads.
type OrderRequest struct {
	// Symbol is the venue-native trading pair identifier (e.g. "BTCUSDT").
	Symbol string
	// Side is the direction of the order.
	Side Side
	// Type is the execution semantics of the order.
	Type OrderType
	// Quantity is the amount of the base asset to trade.
	Quantity decimal.Decimal
	// Price is the limit price. Zero for market orders.
	Price decimal.Decimal
	// ClientOrderID is the internal order identifier propagated to the venue for
	// idempotency and reconciliation.
	ClientOrderID string
}

// CancelRequest is the internal representation of an order cancellation intent.
// Either VenueOrderID or ClientOrderID may be populated; adapters should prefer
// VenueOrderID when present.
type CancelRequest struct {
	// Symbol is the venue-native trading pair identifier.
	Symbol string
	// VenueOrderID is the order identifier assigned by the venue.
	VenueOrderID string
	// ClientOrderID is the internal order identifier supplied on placement.
	ClientOrderID string
}

// FillQuery narrows the fills returned by GetFills to a symbol and a time range.
type FillQuery struct {
	// Symbol is the venue-native trading pair identifier. Empty means all symbols.
	Symbol string
	// VenueOrderID, when non-empty, restricts results to a single venue order.
	VenueOrderID string
	// StartTime is the inclusive lower bound on fill time. Zero means no lower bound.
	StartTime time.Time
	// EndTime is the inclusive upper bound on fill time. Zero means no upper bound.
	EndTime time.Time
}

// VenueOrder is the normalized representation of an order as reported by a venue.
type VenueOrder struct {
	// VenueOrderID is the order identifier assigned by the venue.
	VenueOrderID string
	// ClientOrderID is the internal order identifier propagated on placement.
	ClientOrderID string
	// Symbol is the venue-native trading pair identifier.
	Symbol string
	// Side is the direction of the order.
	Side Side
	// Type is the execution semantics of the order.
	Type OrderType
	// Status is the current lifecycle state of the order.
	Status OrderStatus
	// Quantity is the original ordered amount of the base asset.
	Quantity decimal.Decimal
	// Price is the limit price. Zero for market orders.
	Price decimal.Decimal
	// ExecutedQuantity is the cumulative amount of the base asset filled so far.
	ExecutedQuantity decimal.Decimal
	// AvgFillPrice is the volume-weighted average price of executed quantity.
	// Zero until any fill is reported.
	AvgFillPrice decimal.Decimal
	// CreatedAt is the venue-reported order creation time.
	CreatedAt time.Time
	// UpdatedAt is the venue-reported time of the most recent state change.
	UpdatedAt time.Time
}

// Fill is the normalized representation of a single execution against an order.
type Fill struct {
	// FillID is the venue-assigned execution identifier.
	FillID string
	// VenueOrderID is the order identifier assigned by the venue.
	VenueOrderID string
	// ClientOrderID is the internal order identifier propagated on placement.
	ClientOrderID string
	// Symbol is the venue-native trading pair identifier.
	Symbol string
	// Side is the direction of the parent order.
	Side Side
	// Price is the execution price.
	Price decimal.Decimal
	// Quantity is the executed amount of the base asset.
	Quantity decimal.Decimal
	// Fee is the fee charged for this execution, in the fee currency reported by
	// the venue.
	Fee decimal.Decimal
	// FeeCurrency is the currency the fee is denominated in.
	FeeCurrency string
	// Time is the venue-reported execution time.
	Time time.Time
}

// Balances is the normalized snapshot of asset balances at a venue.
type Balances struct {
	// UpdatedAt is the venue-reported time of the snapshot.
	UpdatedAt time.Time
	// Assets maps asset symbol to the available balance. Adapters must exclude
	// any asset with zero available and zero total balance.
	Assets map[string]Balance
}

// Balance is a single asset's balance breakdown at a venue.
type Balance struct {
	// Asset is the asset symbol (e.g. "BTC", "USDT").
	Asset string
	// Available is the amount available for trading or withdrawal.
	Available decimal.Decimal
	// Total is the total amount held, including amounts locked in open orders.
	Total decimal.Decimal
}

// BookUpdate is a normalized increment or snapshot of a venue order book for one
// symbol, emitted by SubscribeOrderBook.
type BookUpdate struct {
	// Symbol is the venue-native trading pair identifier.
	Symbol string
	// Snapshot is true when Bids/Asks represent a full book snapshot, false when
	// they represent an incremental diff.
	Snapshot bool
	// Sequence is the venue-reported sequence number for diff stream integrity.
	// Zero means the venue does not provide sequence numbers.
	Sequence uint64
	// Bids is a slice of [price, size] levels sorted best-first.
	Bids [][2]decimal.Decimal
	// Asks is a slice of [price, size] levels sorted best-first.
	Asks [][2]decimal.Decimal
	// Time is the venue-reported time of the update.
	Time time.Time
}

// VenueConnector is the uniform interface every venue adapter implements so
// that Liquidity Routing is agnostic to the underlying venue. Methods that
// perform IO must honor the supplied context for cancellation and deadlines.
type VenueConnector interface {
	// PlaceOrder submits a new order to the venue and returns the normalized
	// resulting VenueOrder.
	PlaceOrder(ctx context.Context, req OrderRequest) (VenueOrder, error)
	// CancelOrder requests cancellation of an existing order and returns the
	// normalized resulting VenueOrder.
	CancelOrder(ctx context.Context, req CancelRequest) (VenueOrder, error)
	// GetOrder fetches the current state of a single order identified by venue
	// order id and/or client order id, scoped to symbol where the venue requires
	// it.
	GetOrder(ctx context.Context, venueOrderID, clientOrderID, symbol string) (VenueOrder, error)
	// GetFills returns fills matching the supplied FillQuery.
	GetFills(ctx context.Context, q FillQuery) ([]Fill, error)
	// GetBalances returns the current balance snapshot at the venue.
	GetBalances(ctx context.Context) (Balances, error)
	// SubscribeOrderBook subscribes to order book updates for a single symbol and
	// emits BookUpdate messages on the returned channel. Closing the context
	// cancels the subscription and closes the channel. Adapters that do not
	// support market data streaming must return an error and a nil channel.
	SubscribeOrderBook(ctx context.Context, symbol string) (<-chan BookUpdate, error)
}
