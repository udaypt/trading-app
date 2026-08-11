package mutualfund

import (
	"testing"

	"trading-dashboard/internal/infra/db/postgres"

	"github.com/stretchr/testify/assert"
)

func TestConvertToDBRecords(t *testing.T) {
	tests := []struct {
		name    string
		schemes []Scheme
		want    []postgres.SchemeRecord
	}{
		{
			name:    "nil input produces empty slice",
			schemes: nil,
			want:    []postgres.SchemeRecord{},
		},
		{
			name:    "empty input produces empty slice",
			schemes: []Scheme{},
			want:    []postgres.SchemeRecord{},
		},
		{
			name: "maps fields in order",
			schemes: []Scheme{
				{SchemeCode: 101, SchemeName: "Alpha Fund"},
				{SchemeCode: 102, SchemeName: "Beta Fund"},
			},
			want: []postgres.SchemeRecord{
				{SchemeCode: 101, SchemeName: "Alpha Fund"},
				{SchemeCode: 102, SchemeName: "Beta Fund"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertToDBRecords(tt.schemes)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConvertFromDBRecords(t *testing.T) {
	tests := []struct {
		name    string
		records []postgres.SchemeRecord
		want    []Scheme
	}{
		{
			name:    "nil input produces empty slice",
			records: nil,
			want:    []Scheme{},
		},
		{
			name:    "empty input produces empty slice",
			records: []postgres.SchemeRecord{},
			want:    []Scheme{},
		},
		{
			name: "maps fields in order",
			records: []postgres.SchemeRecord{
				{SchemeCode: 201, SchemeName: "Gamma Fund"},
			},
			want: []Scheme{
				{SchemeCode: 201, SchemeName: "Gamma Fund"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertFromDBRecords(tt.records)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConvertRoundTrip(t *testing.T) {
	original := []Scheme{{SchemeCode: 5, SchemeName: "Round Trip Fund"}}
	got := convertFromDBRecords(convertToDBRecords(original))
	assert.Equal(t, original, got)
}
