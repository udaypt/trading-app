package main

import (
	"context"
	"log"
	"net/http"

	mfstore "github.com/udaypt/trading-app/internal/domain/services/trading/mutual_fund/mf-store"
)

// main is not unit tested: http.ListenAndServe blocks until the process
// exits, so exercising it here would mean starting a real, permanently
// running server inside the test binary.
func main() {
	ctx := context.Background()

	cntr, err := getContainer(ctx)
	if err != nil {
		log.Fatalf("Server exited unexpectedly: %v", err)

		return
	}

	// Load the mutual-fund master list into memory before the server accepts
	// any traffic, so search has data to serve from the moment it's up.
	err = cntr.Invoke(func(initializer *mfstore.Initializer) error {
		initializer.Initialize(ctx)
		return nil
	})
	if err != nil {
		log.Fatalf("Server exited unexpectedly: %v", err)
	}

	port := ":8080"
	log.Printf("Trading dashboard Service is running on http://localhost%s\n", port)

	log.Printf(" - Endpoint 1: http://localhost%s/api/v1/auth/register\n", port)
	log.Printf(" - Endpoint 2: http://localhost%s/api/v1/auth/login\n", port)
	log.Printf(" - Endpoint 3: http://localhost%s/api/v1/search?q=HDFC\n", port)
	log.Printf(" - Endpoint 4: http://localhost%s/api/v1/market-data?id=RELIANCE.NS&type=STOCK&days=30\n", port)

	err = cntr.Invoke(func(r *Router) error {
		return http.ListenAndServe(port, nil)
	})

	if err != nil {
		log.Fatalf("Server exited unexpectedly: %v", err)
	}
}
