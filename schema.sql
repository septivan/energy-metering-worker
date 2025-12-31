-- Create database schema for energy metering worker

-- Meter clients table
CREATE TABLE IF NOT EXISTS meter_clients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_fingerprint TEXT UNIQUE NOT NULL,
    ip_address INET NOT NULL,
    user_agent TEXT,
    first_seen_at TIMESTAMPTZ NOT NULL,
    last_seen_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- Index for fast client lookup
CREATE UNIQUE INDEX IF NOT EXISTS idx_meter_clients_fingerprint ON meter_clients (client_fingerprint);

-- Meter readings raw table
CREATE TABLE IF NOT EXISTS meter_readings_raw (
    id UUID DEFAULT gen_random_uuid(),
    client_id UUID REFERENCES meter_clients(id),
    metric_name TEXT NOT NULL,
    metric_value DOUBLE PRECISION NOT NULL,
    reading_timestamp TIMESTAMPTZ NOT NULL,
    received_at TIMESTAMPTZ NOT NULL,
    validation_status TEXT NOT NULL,
    anomaly_reason TEXT,
    raw_payload JSONB NOT NULL,
    PRIMARY KEY (id, reading_timestamp)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_mrr_client_ts ON meter_readings_raw (client_id, reading_timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_mrr_metric_ts ON meter_readings_raw (metric_name, reading_timestamp DESC);

-- Convert to TimescaleDB hypertable (run this only if TimescaleDB extension is enabled)
SELECT create_hypertable('meter_readings_raw', 'reading_timestamp', if_not_exists => TRUE);

-- Enable compression and retention
ALTER TABLE meter_readings_raw SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'client_id,metric_name'
);

-- Create compression policy (compress chunks older than 7 days)
SELECT add_compression_policy('meter_readings_raw', INTERVAL '7 days');

-- Add retention (drop chunks older than 90 days)
SELECT add_retention_policy('meter_readings_raw', INTERVAL '90 days');
