package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ai-crypto-onramp/exchange-connectors/internal/adapters/binance"
	"github.com/ai-crypto-onramp/exchange-connectors/internal/adapters/kraken"
	"github.com/ai-crypto-onramp/exchange-connectors/internal/adapters/otc"
	"github.com/ai-crypto-onramp/exchange-connectors/internal/audit"
	"github.com/ai-crypto-onramp/exchange-connectors/internal/events"
	"github.com/ai-crypto-onramp/exchange-connectors/internal/server"
	"github.com/ai-crypto-onramp/exchange-connectors/internal/venue"
)

func main() {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	venueName := os.Getenv("VENUE_FAMILY")
	if venueName == "" {
		venueName = "dummy"
	}
	pairs := []string{"BTCUSDT", "ETHUSDT"}
	if p := os.Getenv("PAIRS"); p != "" {
		pairs = splitCSV(p)
	}

	conn := selectConnector(venueName)
	sink := audit.NewInMemorySink()
	svc, err := server.NewService(conn, sink, server.Config{VenueName: venueName, Pairs: pairs})
	if err != nil {
		log.Fatalf("server: %v", err)
	}

	// When EVENT_BUS_URL is set (kafka://host:9092), construct a Kafka
	// publisher for fill/balance events and close it on shutdown. The
	// publisher is exposed via svc for future wiring into the server.
	var eventPub *events.KafkaPublisher
	if busURL := os.Getenv("EVENT_BUS_URL"); busURL != "" {
		p, err := events.NewKafkaPublisherFromURL(busURL)
		if err != nil {
			log.Printf("events: kafka publisher init failed: %v", err)
		} else {
			eventPub = p
		}
	}

	srv := &http.Server{
		Addr:    addr,
		Handler: svc.Routes(),
	}

	go func() {
		log.Printf("exchange-connectors listening on %s (venue=%s)", addr, venueName)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	if eventPub != nil {
		_ = eventPub.Close()
	}
}

func selectConnector(name string) venue.VenueConnector {
	switch name {
	case "binance":
		return binance.NewConnector(os.Getenv("BINANCE_API_KEY"), os.Getenv("BINANCE_API_SECRET"), nil)
	case "kraken":
		return kraken.NewConnector(os.Getenv("KRAKEN_API_KEY"), os.Getenv("KRAKEN_API_SECRET"), nil)
	case "otc":
		return otc.NewConnector(os.Getenv("OTC_API_KEY"), nil)
	default:
		return venue.NewDummyVenueConnector()
	}
}

func splitCSV(s string) []string {
	var out []string
	cur := ""
	for _, c := range s {
		if c == ',' {
			if cur != "" {
				out = append(out, cur)
			}
			cur = ""
		} else {
			cur += string(c)
		}
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}