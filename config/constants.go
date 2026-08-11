package config

const (
	MF_API_BASE_URL       = "https://api.mfapi.in/mf"
	MF_HISTORY_API_URL    = MF_API_BASE_URL + "/%s" // schemeCode
	STOCK_API_BASE_URL    = "https://query1.finance.yahoo.com"
	STOCK_SEARCH_API_URL  = STOCK_API_BASE_URL + "/v1/finance/search"
	STOCK_HISTORY_API_URL = STOCK_API_BASE_URL + "/v8/finance/chart/%s?interval=1d&range=%s" // symbol, rangeParam

	DB_CONNECTION_STRING = "postgres://postgres:example@localhost:5432/trading_dashboard?sslmode=disable"
	JWT_SECRET           = "c257465c25bc4daecac2ac3d36fdce297e2ba92c"

	CORS_ALLOWED_ORIGIN = "http://localhost:3000"
)
