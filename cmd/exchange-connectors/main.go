package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ai-crypto-onramp/exchange-connectors/internal/audit"
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

	conn := venue.NewDummyVenueConnector()
	sink := audit.NewInMemorySink()
	svc, err := server.NewService(conn, sink, server.Config{VenueName: venueName, Pairs: pairs})
	if err != nil {
		log.Fatalf("server: %v", err)
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