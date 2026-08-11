package mutualfund

import "github.com/udaypt/trading-app/internal/infra/db/postgres"

func convertToDBRecords(schemes []Scheme) []postgres.SchemeRecord {
	records := make([]postgres.SchemeRecord, len(schemes))
	for i, s := range schemes {
		records[i] = postgres.SchemeRecord{
			SchemeCode: s.SchemeCode,
			SchemeName: s.SchemeName,
		}
	}
	return records
}

func convertFromDBRecords(records []postgres.SchemeRecord) []Scheme {
	schemes := make([]Scheme, len(records))
	for i, r := range records {
		schemes[i] = Scheme{
			SchemeCode: r.SchemeCode,
			SchemeName: r.SchemeName,
		}
	}
	return schemes
}
