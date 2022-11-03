package main

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/awnumar/memguard"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/heyztb/plutus/plutus"
	"github.com/heyztb/plutus/routes"
)

func runServer() {
	memguard.CatchInterrupt()
	defer memguard.Purge()

	// we will be notified of new transactions. I just don't see any benefit from using the verbose notifications, hence why you see `false` here.
	err := routes.BitcoinRPCClient.NotifyNewTransactions(false)
	if err != nil {
		plutus.Logger.Warn("Could not connect to BTCWallet. Trying again")
		err := routes.BitcoinRPCClient.Connect(5)
		if err != nil {
			switch err {
			// This shouldn't ever happen under any circumstances
			case rpcclient.ErrNotWebsocketClient:
				plutus.Logger.Fatalf("Something has gone horribly wrong")

			// and neither should this, realistically
			case rpcclient.ErrClientAlreadyConnected:
				plutus.Logger.Info("Client already connected.. Continue on")
				// this means we couldn't connect to btcwallet after 5 attempts, something has gone wrong and will require operator intervention
			default:
				plutus.Logger.Fatal("Could not connect to btcwallet after multiple attempts. Operator intervention required")
			}
			// and this is our desired result
			plutus.Logger.Info("Client reconnected")
		}
	}

	// ensure that our database tables are laid out correctly
	plutus.SyncModels()
	// grab our router configuration to use later on
	router := configureRouter()

	// this TLS config should score 100%/A+ on SSL Labs
	tlsConfig := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
	}

	// create our https server using the environment to determine what address to listen on. this should be localhost, on any unprivileged port you desire. see .env.example
	// as well as the router and tls configurations we grabbed/created earlier
	server := &http.Server{
		Addr:      os.Getenv("BIND_PORT"),
		Handler:   router,
		TLSConfig: tlsConfig,
		// we have to effectively disable HTTP 2.0 in order for the above TLS configuration to work properly
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
	}

	// serve
	go func() {
		if err := server.ListenAndServeTLS("cert.pem", "key.pem"); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error listening: %s\n", err)
		}
	}()
	plutus.Logger.Infof("Server started on port %s", server.Addr)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// at this we point the code in this method blocks until we receive a signal on the channel created above. once we receive a signal, the remainder of the code in this method will run and shutdown the server gracefully. the fact that this blocks is not an issue to us because our server is being handled in its own goroutine, unaffected by anything going on over here
	<-quit
	plutus.Logger.Info("Server shutting down...")

	// ensure that any currently running requests are finished properly
	routes.BitcoinRPCClient.WaitForShutdown()

	// create a context with a 5 second timeout. currently this context is blank, and contains nothing at all, so this 5 second timeout is a bit redundant, however I've left it here because if we want to change the context used here we will be able to with relative ease
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// finally shutdown the server
	if err := server.Shutdown(ctx); err != nil {
		// if something goes wrong in this process, log to stderr/filelog and then use memguard to wipe application memory and force shutdown
		plutus.Logger.Errorf("Error shutting down server: %s\n", err)
		memguard.SafePanic(1)
	}
}
