-- QUERIES FOR POSTGRESQL

CREATE DATABASE trading_dashboard OWNER postgres;

-- Switch to the target database
\c trading_dashboard


-- Users master records for authentication handling
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL DEFAULT '',
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    last_login TIMESTAMP WITH TIME ZONE DEFAULT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Metadata for Stocks and Mutual Funds
CREATE TABLE IF NOT EXISTS assets (
    id VARCHAR(50) PRIMARY KEY, -- "RELIANCE.NS" or "125497"
    name VARCHAR(255) NOT NULL,
    asset_type VARCHAR(20) NOT NULL, -- "STOCK" or "MUTUAL_FUND"
    exchange VARCHAR(20) NOT NULL, -- "NSE", "BSE", or "AMC"
    last_nday_fetched_date DATE default NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Historical Price / NAV Time-Series Data
CREATE TABLE IF NOT EXISTS price_history (
    id BIGSERIAL PRIMARY KEY,
    asset_id VARCHAR(50) NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    price_date DATE NOT NULL,
    price NUMERIC(12, 4) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Ensure no duplicate price entries for the same asset on the same day
    CONSTRAINT unique_asset_date UNIQUE (asset_id, price_date)
);

-- Index for fast time-series range queries by asset_id and date
CREATE INDEX IF NOT EXISTS idx_price_history_asset_date 
ON price_history(asset_id, price_date DESC);


-- mutual funds store, to be used for loading all these records into the memory when system starts and these records will be refreshed in 24 hours through the external apis
CREATE TABLE IF NOT EXISTS mutual_fund_schemes (
    scheme_code INT PRIMARY KEY,
    scheme_name VARCHAR(255) NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_mf_schemes_name ON mutual_fund_schemes(scheme_name);