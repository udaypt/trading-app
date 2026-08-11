package httphandler

import (
	"context"
	"log"
	"net/http"

	"github.com/pkg/errors"
	"go.uber.org/dig"

	router "github.com/udaypt/trading-app/internal/httphandler/usecase/router"
)

func HandleHttp(writer http.ResponseWriter, request *http.Request, handle http.HandlerFunc) {
	handle(writer, request)
}

// RunHTTPServer's success path is not unit tested: it's only reached once
// http.ListenAndServe is actually called, which blocks until the server
// exits. TestRunHTTPServer_UnresolvableHandlerDependency covers the
// error-wrapping path via a container that fails to resolve its handler.
func RunHTTPServer(container *dig.Container, addr string) error {
	log.Printf("Start HTTP server on %s", addr)
	err := container.Invoke(func(ctx context.Context, handler router.Handler) error {
		return http.ListenAndServe(addr, nil)
	})

	if err != nil {
		return errors.Wrap(err, "ListenAndServe")
	}
	log.Println("Stopped HTTP server")
	return nil
}
