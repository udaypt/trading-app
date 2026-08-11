package mfstore

import (
	mutualfund "github.com/udaypt/trading-app/internal/domain/services/trading/mutual_fund"
	"github.com/udaypt/trading-app/internal/infra/db/postgres"
)

func convertToDBRecords(schemes []mutualfund.Scheme) []postgres.SchemeRecord {
	records := make([]postgres.SchemeRecord, len(schemes))
	for i, s := range schemes {
		records[i] = postgres.SchemeRecord{
			SchemeCode: s.SchemeCode,
			SchemeName: s.SchemeName,
		}
	}
	return records
}

func convertFromDBRecords(records []postgres.SchemeRecord) []mutualfund.Scheme {
	schemes := make([]mutualfund.Scheme, len(records))
	for i, r := range records {
		schemes[i] = mutualfund.Scheme{
			SchemeCode: r.SchemeCode,
			SchemeName: r.SchemeName,
		}
	}
	return schemes
}
