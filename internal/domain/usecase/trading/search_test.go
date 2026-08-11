package trading

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAssetTypes(t *testing.T) {
	assert.ElementsMatch(t, []string{"STOCK", "MUTUAL_FUND"}, AssetTypes)
	assert.Contains(t, AssetTypes, string(Stock))
	assert.Contains(t, AssetTypes, string(MutualFund))
}
