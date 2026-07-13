.PHONY: build test run lint docker-build docker-run clean

build:
	go build -o bin/exchange-connectors ./cmd/exchange-connectors

test:
	go test ./... -race -coverprofile=coverage.out -coverpkg=./...

run:
	go run ./cmd/exchange-connectors

lint:
	go vet ./...

docker-build:
	docker build -t ai-crypto-onramp/exchange-connectors .

docker-run:
	docker run --rm -p 8080:8080 ai-crypto-onramp/exchange-connectors

clean:
	rm -rf bin/ coverage.out
