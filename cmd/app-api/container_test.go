package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetContainer(t *testing.T) {
	ctx := context.Background()

	container, err := getContainer(ctx)
	require.NoError(t, err)
	require.NotNil(t, container)

	// Providers are registered lazily; resolving the plain context.Context
	// dependency confirms the container was wired without needing a real
	// Postgres connection (which NewDBRepository would otherwise require).
	err = container.Invoke(func(gotCtx context.Context) error {
		assert.Equal(t, ctx, gotCtx)
		return nil
	})
	assert.NoError(t, err)
}
