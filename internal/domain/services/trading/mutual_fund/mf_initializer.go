package mutualfund

import (
	"context"
	"log"
	"time"
)

// backgroundSyncInterval is how often the mutual-fund master list is
// refreshed from the external API after startup.
const backgroundSyncInterval = 24 * time.Hour

// MFStoreInitializer is the entry point main.go calls at startup: it makes
// sure MFStore has data before the server accepts traffic, and keeps it
// fresh afterwards.
type MFStoreInitializer struct {
	store  *MFStore
	loader *MFStoreLoader
	syncer *MFStoreSyncer
}

func NewMFStoreInitializer(store *MFStore, loader *MFStoreLoader, syncer *MFStoreSyncer) *MFStoreInitializer {
	return &MFStoreInitializer{store: store, loader: loader, syncer: syncer}
}

// Initialize loads the mutual-fund master list from Postgres, falling back
// to a synced fetch from the external API when Postgres has nothing cached.
// If neither source can produce data, it panics: the server has no useful
// mutual-fund search without it, so it must not start.
func (i *MFStoreInitializer) Initialize(ctx context.Context) *MFStore {
	schemes, err := i.loader.Load(ctx)
	if err != nil {
		log.Printf("[MFStoreInitializer] %v — syncing from API instead...", err)

		schemes, err = i.syncer.Sync(ctx)
		if err != nil {
			log.Panicf("[MFStoreInitializer] could not load mutual fund schemes from Postgres or the external API: %v", err)
		}
	}

	i.store.SetSchemes(schemes)
	i.syncer.StartBackgroundSync(ctx, backgroundSyncInterval, i.store.SetSchemes)

	return i.store
}
