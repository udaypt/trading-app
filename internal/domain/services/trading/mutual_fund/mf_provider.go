package mutualfund

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/udaypt/trading-app/config"
)

// MFSyncAPIURL is the mutual-fund master-list endpoint. Exported so it can
// be redirected to an httptest server from this and other packages' tests.
var MFSyncAPIURL = config.MF_API_BASE_URL

// MFStoreProvider is the external-API boundary: a single, non-retrying
// fetch of the mutual-fund master list. Retry policy lives in MFStoreSyncer.
type MFStoreProvider struct {
	client *http.Client
}

func NewMFStoreProvider() *MFStoreProvider {
	return &MFStoreProvider{client: &http.Client{Timeout: 15 * time.Second}}
}

// Fetch performs a single attempt at fetching and decoding the mfapi
// master-list response.
func (p *MFStoreProvider) Fetch(ctx context.Context) ([]Scheme, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, MFSyncAPIURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http fetch error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mfapi returned status code: %d", resp.StatusCode)
	}

	var schemes []Scheme
	if err := json.NewDecoder(resp.Body).Decode(&schemes); err != nil {
		return nil, fmt.Errorf("failed to decode scheme payload: %w", err)
	}

	return schemes, nil
}
