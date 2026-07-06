package venue

import (
	"context"
	"errors"
)

// stubConnector is a no-op VenueConnector implementation used to verify
// interface conformance and to serve as a reference for adapter authors. It
// performs no IO and returns empty results for every method.
type stubConnector struct{}

// newStubConnector returns a no-op VenueConnector.
func newStubConnector() VenueConnector { return stubConnector{} }

func (stubConnector) PlaceOrder(ctx context.Context, req OrderRequest) (VenueOrder, error) {
	return VenueOrder{}, nil
}

func (stubConnector) CancelOrder(ctx context.Context, req CancelRequest) (VenueOrder, error) {
	return VenueOrder{}, nil
}

func (stubConnector) GetOrder(ctx context.Context, venueOrderID, clientOrderID, symbol string) (VenueOrder, error) {
	return VenueOrder{}, nil
}

func (stubConnector) GetFills(ctx context.Context, q FillQuery) ([]Fill, error) {
	return nil, nil
}

func (stubConnector) GetBalances(ctx context.Context) (Balances, error) {
	return Balances{}, nil
}

// SubscribeOrderBook returns a clear unsupported error without blocking, since
// the stub owns no market data source.
func (stubConnector) SubscribeOrderBook(ctx context.Context, symbol string) (<-chan BookUpdate, error) {
	return nil, errors.New("stubConnector does not support order book subscription")
}
