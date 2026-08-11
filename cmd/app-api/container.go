package main

import (
	"context"

	"github.com/pkg/errors"
	"go.uber.org/dig"

	httphandlers "github.com/udaypt/trading-app/cmd/app-api/http-handlers"
	mf "github.com/udaypt/trading-app/internal/domain/services/trading/mutual_fund"
	mfstore "github.com/udaypt/trading-app/internal/domain/services/trading/mutual_fund/mf-store"
	db "github.com/udaypt/trading-app/internal/infra/db/postgres"
)

// getContainer's error branches are not unit tested: every provider below is
// a fixed, distinct constructor, so dig.Provide cannot fail here without
// editing this function to introduce a conflicting registration.
func getContainer(ctx context.Context) (*dig.Container, error) {
	// Create new dig container
	container := dig.New()

	// Provide context.Context to the container so constructor dependencies can resolve it
	if err := container.Provide(func() context.Context {
		return ctx
	}); err != nil {
		return nil, errors.Wrap(err, "provide context.Context")
	}

	// Provide resources
	for _, provide := range []struct {
		Name      string
		Resources []any
		Options   []dig.ProvideOption
	}{
		{
			Name:      "http routers",
			Resources: []any{newRouter},
		},
		{
			Name:      "Db repository",
			Resources: []any{db.NewDBRepository},
		},
		{
			Name:      "In-memory store for Mutual-fund's data",
			Resources: []any{mf.NewMFStore},
		},
		{
			Name:      "Mutual-fund external API provider",
			Resources: []any{mfstore.NewProvider},
		},
		{
			Name:      "Mutual-fund scheme syncer (retry + persist)",
			Resources: []any{mfstore.NewSyncer},
		},
		{
			Name:      "Mutual-fund scheme loader (Postgres read)",
			Resources: []any{mfstore.NewLoader},
		},
		{
			Name:      "Mutual-fund store initializer",
			Resources: []any{mfstore.NewInitializer},
		},
		{
			Name:      "Sign in http handler",
			Resources: []any{httphandlers.NewSignIn},
		},
		{
			Name:      "Sign up http handler",
			Resources: []any{httphandlers.NewSignUp},
		},
		{
			Name:      "Search http handler",
			Resources: []any{httphandlers.NewSearch},
		},
		{
			Name:      "marketing data http handler",
			Resources: []any{httphandlers.NewMarketData},
		},
	} {
		for _, resource := range provide.Resources {
			provideErr := container.Provide(resource, provide.Options...)
			if provideErr != nil {
				return nil, errors.Wrapf(provideErr, "provide %s", provide.Name)
			}
		}
	}
	return container, nil
}
