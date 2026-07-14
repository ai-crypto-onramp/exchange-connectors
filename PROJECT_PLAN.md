# Project Plan — Exchange Connectors

Implementation plan for the Exchange Connectors service: venue-specific Go adapters
(Binance, Kraken, OTC desks) that translate internal trading intents into venue
protocol calls and stream fills, balances, and market data back to Liquidity Routing,
Reconciliation, and the Audit Event Log. Stages are ordered to build the shared
contract first, then layer venue adapters, resilience, security, streaming, and ops.

## Stage 1: Common VenueConnector interface

Goal: Define the uniform Go interface and shared types that every venue adapter
implements, so Liquidity Routing is agnostic to the underlying venue.

Tasks:
- [x] Create `internal/venue` package with the `VenueConnector` interface (`PlaceOrder`, `CancelOrder`, `GetOrder`, `GetFills`, `GetBalances`, `SubscribeOrderBook`).
- [x] Define shared domain types: `OrderRequest`, `CancelRequest`, `FillQuery`, `VenueOrder`, `Fill`, `Balances`, `BookUpdate`, `Side`, `OrderType`, `OrderStatus`.
- [ ] Use `decimal.Decimal` (shopspring/decimal) for all monetary/quantity fields.
- [x] Add `VenueConfig` struct holding REST/WS base URLs, venue name, and non-secret knobs.
- [x] Add a no-op `stubConnector` implementation for interface conformance tests.
- [x] Add unit tests covering type serialization and interface satisfaction.

Acceptance criteria:
- `go test ./internal/venue...` passes with the stub implementation satisfying `VenueConnector`.
- All domain types documented and used consistently across the package.
- No venue-specific code leaks into `internal/venue`.

## Stage 2: Binance adapter (REST + WS scaffolding)

Goal: Implement the Binance venue adapter REST client for order management and
balance queries, plus an initial WS client skeleton.

Tasks:
- [ ] Create `internal/adapters/binance` package implementing `VenueConnector`.
- [ ] Implement REST client for `PlaceOrder`, `CancelOrder`, `GetOrder`, `GetFills`, `GetBalances` against `https://api.binance.com`.
- [ ] Add Binance Edge auth: API key header, HMAC-SHA256 signing of query string with `timestamp` + `recvWindow`.
- [ ] Map Binance response payloads to shared `VenueOrder`, `Fill`, `Balances` types.
- [ ] Add WS client struct for `wss://stream.binance.com:9443/ws` (connection only; subscription wiring in Stage 5).
- [ ] Wire `VENUE_FAMILY=binance` selection in `cmd/exchange-connectors`.
- [ ] Add table-driven unit tests with recorded HTTP fixtures (httptest).

Acceptance criteria:
- `go test ./internal/adapters/binance...` passes against mock Binance REST endpoints.
- Signed requests include valid `signature`, `timestamp`, and `X-MBX-APIKEY` header.
- Adapter satisfies `venue.VenueConnector` via compile-time assertion.

## Stage 3: Kraken adapter

Goal: Implement the Kraken venue adapter REST client using Kraken's API key + nonce
signing and counter-based rate model.

Tasks:
- [ ] Create `internal/adapters/kraken` package implementing `VenueConnector`.
- [ ] Implement REST client for `PlaceOrder`, `CancelOrder`, `GetOrder`, `GetFills`, `GetBalances` against `https://api.kraken.com`.
- [ ] Add Kraken auth: API key header, SHA256+HMAC-SHA512 signing of path + nonce + POST data.
- [ ] Map Kraken response payloads (incl. `result`/`error` envelope) to shared types.
- [ ] Wire `VENUE_FAMILY=kraken` selection in `cmd/exchange-connectors`.
- [ ] Add table-driven unit tests with recorded HTTP fixtures.

Acceptance criteria:
- `go test ./internal/adapters/kraken...` passes against mock Kraken REST endpoints.
- Signed requests include valid `API-Key`, `API-Sign`, and incrementing nonce.
- Adapter satisfies `venue.VenueConnector` via compile-time assertion.

## Stage 4: OTC desk adapter

Goal: Implement the OTC desk adapter for bespoke quote-request / RFQ / manual
settlement workflows over REST/SFTP, with no public market data stream.

Tasks:
- [ ] Create `internal/adapters/otc` package implementing `VenueConnector`.
- [ ] Implement REST client for `PlaceOrder` (RFQ → quote → accept), `CancelOrder`, `GetOrder`, `GetFills`, `GetBalances` against `OTC_DESK_URL`.
- [ ] Add `SubscribeOrderBook` returning `errors.ErrUnsupported` (no public market data).
- [ ] Add SFTP fallback for settlement confirmations when configured.
- [ ] Map OTC desk response payloads to shared types; normalize manual settlement states.
- [ ] Wire `VENUE_FAMILY=otc` selection in `cmd/exchange-connectors`.
- [ ] Add table-driven unit tests with recorded HTTP/SFTP fixtures.

Acceptance criteria:
- `go test ./internal/adapters/otc...` passes against mock OTC endpoints.
- `SubscribeOrderBook` returns a clear unsupported error without blocking.
- Adapter satisfies `venue.VenueConnector` via compile-time assertion.

## Stage 5: WebSocket order book subscription + reconnect/backoff

Goal: Build a resilient WS order book client per symbol with snapshot + diff
reconstruction, sequence-gap detection, and reconnect with backoff.

Tasks:
- [ ] Create `internal/ws` package with a generic reconnecting WS client (exponential backoff + jitter, bounded by `WS_RECONNECT_MAX_BACKOFF`).
- [ ] Implement `SubscribeOrderBook` for Binance (diff depth stream + depth snapshot REST bootstrapping).
- [ ] Implement `SubscribeOrderBook` for Kraken (`book` channel with depth snapshots).
- [ ] Add sequence-gap detection: on gap, force a snapshot refresh and drop stale diffs.
- [ ] Resubscribe to book + private channels after reconnect; emit `BookUpdate` to caller channel.
- [ ] Add metrics: `ws_reconnect_count`, `ws_gap_count`, `ws_book_lag_seconds`.
- [ ] Add unit/integration tests with a local WS test server simulating gaps and disconnects.

Acceptance criteria:
- A forced socket close triggers reconnect with backoff and resubscribes without losing book integrity.
- Injected sequence gaps trigger a snapshot refresh and correct book reconstruction.
- `ws_reconnect_count` and `ws_gap_count` increment as expected in tests.

## Stage 6: Per-venue rate limit handling

Goal: Enforce per-venue rate limits with a token-bucket / weighted-cost model and
back off on 429/418, never exceeding venue limits.

Tasks:
- [ ] Create `internal/ratelimit` package with a weighted token-bucket limiter per venue.
- [ ] Binance: read `X-MBX-USED-WEIGHT-1M` response header; adjust budget; back off on 429/418.
- [ ] Kraken: model the Kraken counter system (decay-based) and gate requests by counter budget.
- [ ] OTC: simple RPS limit configurable via `RATE_LIMIT_WEIGHT_BUDGET`.
- [ ] When budget is exhausted, queue or reject internal requests rather than risk a ban; surface `rate_limit_headroom` metric.
- [ ] Add unit tests for budget accounting, header-driven adjustment, and 429 backoff.

Acceptance criteria:
- Requests are rejected/queued when the venue budget is exhausted.
- `rate_limit_headroom` Prometheus metric reflects remaining budget.
- 429/418 responses trigger backoff and reduced request rate in tests.

## Stage 7: Credential rotation from vault

Goal: Read API keys from the secrets vault at startup and on rotation signal;
never log or persist secrets to disk.

Tasks:
- [ ] Create `internal/secrets` package wrapping the platform vault client (HashiCorp Vault / cloud KMS).
- [ ] Read keys from `secret/exchange-connectors/<venue>/api-key` and `.../api-secret` at startup; hold a lease.
- [ ] Implement rotation via `POST /admin/rotate-credentials` and vault rotation signal.
- [ ] Old keys revoked only after the connector confirms the new key is active (probe call).
- [ ] Redact secrets from logs, API responses, and error messages; no disk persistence.
- [ ] Add `venue_credentials` rows referencing vault secret versions only (no secret material).
- [ ] Add unit tests with a fake vault client covering load, rotate, revoke, and redaction.

Acceptance criteria:
- Credentials load from vault at startup and a probe confirms activation before old keys are revoked.
- Rotation swaps keys without dropping in-flight requests.
- No secret material appears in logs, responses, or on disk (verified by test assertions).

## Stage 8: Fill/balance streaming to reconciliation

Goal: Emit each fill and balance update as a typed event on the event bus for
Reconciliation, including all required fields and venue/internal order id linkage.

Tasks:
- [ ] Create `internal/events` package with typed `FillEvent` and `BalanceEvent`.
- [ ] Publish fill events on every `GetFills`/WS user-data stream execution; include venue order id, internal order id, price, size, fee, timestamp.
- [ ] Publish balance events on periodic REST poll and WS balance stream (where supported); reconcile against internal ledger expectations.
- [ ] Wire event bus publisher to `RECON_EVENT_BUS` (NATS) with retries + dead-letter.
- [ ] Add Prometheus metrics: `fill_latency_seconds`, `balance_sync_lag_seconds`, `events_published_total`.
- [ ] Add integration tests with an in-process event bus subscriber.

Acceptance criteria:
- Every fill produces a `FillEvent` on the bus with all required fields populated.
- Balance updates stream within the configured sync interval and reconcile mismatches are logged.
- `fill_latency_seconds` and `events_published_total` metrics increment correctly in tests.

## Stage 9: Audit emission

Goal: Emit every order, fill, and credential event to the Audit Event Log
asynchronously over gRPC.

Tasks:
- [x] Create `internal/audit` package with a gRPC client to `AUDIT_EVENT_LOG_URL`.
- [x] Emit audit events for: order placed, order cancelled, order updated, fill received, balance snapshot, credential rotation.
- [ ] Propagate traces from Liquidity Routing through the connector to the audit call.
- [ ] Make audit emission non-blocking with bounded queue + drop-on-overflow policy (log + metric on drops).
- [ ] Add metric `audit_events_emitted_total` and `audit_dropped_total`.
- [x] Add unit tests verifying event payloads and trace propagation.

Acceptance criteria:
- Every order/fill/credential lifecycle action produces an audit event with full context.
- Audit failures do not block the transaction path; drops are metriced and logged.
- Trace context propagates end-to-end in tests.

## Stage 10: Tests, coverage, and Docker

Goal: Reach production-grade tests, wire CI, and finalize the
per-venue Docker images and Makefile targets.

Tasks:
- [x] Add integration test harness spinning up local WS + REST mock servers per venue.
- [x] Raise coverage across `internal/...`; report via `go test -cover` in CI.
- [x] Add `-race` runs and fuzz tests for signing, decimal parsing, and book reconstruction.
- [x] Finalize Dockerfile for per-venue images (`VENUE_FAMILY` build arg + runtime env).
- [x] Add Makefile targets: `test`, `cover`, `lint`, `docker`, `buf-generate`, `run-<venue>`.
- [x] Wire Codecov upload and golangci-lint in CI; ensure `make cover` reports coverage.

Acceptance criteria:
- `go test -race -cover ./...` reports coverage and CI gate is green.
- `make docker VENUE_FAMILY=binance` produces a runnable image.
- `make lint` and `make cover` both pass in CI.