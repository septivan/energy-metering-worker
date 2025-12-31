package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/septivank/energy-metering-worker/internal/db"
)

// Tx is an alias for pgx.Tx
type Tx = pgx.Tx

// Repository handles database operations
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// GetOrCreateClient retrieves or creates a meter client
func (r *Repository) GetOrCreateClient(ctx context.Context, fingerprint string, ipAddress string, userAgent *string) (*db.MeterClient, error) {
	// Try to get existing client
	query := `
		SELECT id, client_fingerprint, ip_address::text, user_agent, first_seen_at, last_seen_at, created_at
		FROM meter_clients
		WHERE client_fingerprint = $1
	`

	var client db.MeterClient
	err := r.pool.QueryRow(ctx, query, fingerprint).Scan(
		&client.ID,
		&client.ClientFingerprint,
		&client.IPAddress,
		&client.UserAgent,
		&client.FirstSeenAt,
		&client.LastSeenAt,
		&client.CreatedAt,
	)

	if err == nil {
		// Client exists, update last_seen_at
		updateQuery := `
			UPDATE meter_clients
			SET last_seen_at = $1
			WHERE id = $2
		`
		now := time.Now()
		_, err = r.pool.Exec(ctx, updateQuery, now, client.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to update client last_seen_at: %w", err)
		}
		client.LastSeenAt = now
		return &client, nil
	}

	if err != pgx.ErrNoRows {
		return nil, fmt.Errorf("failed to query client: %w", err)
	}

	// Client doesn't exist, create new one
	insertQuery := `
		INSERT INTO meter_clients (client_fingerprint, ip_address, user_agent, first_seen_at, last_seen_at, created_at)
		VALUES ($1, $2, $3, $4, $4, $4)
		RETURNING id, client_fingerprint, ip_address::text, user_agent, first_seen_at, last_seen_at, created_at
	`

	now := time.Now()
	err = r.pool.QueryRow(ctx, insertQuery, fingerprint, ipAddress, userAgent, now).Scan(
		&client.ID,
		&client.ClientFingerprint,
		&client.IPAddress,
		&client.UserAgent,
		&client.FirstSeenAt,
		&client.LastSeenAt,
		&client.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return &client, nil
}

// InsertMeterReading inserts a meter reading
func (r *Repository) InsertMeterReading(ctx context.Context, reading *db.MeterReading) error {
	query := `
		INSERT INTO meter_readings_raw (
			client_id, metric_name, metric_value, reading_timestamp,
			received_at, validation_status, anomaly_reason, raw_payload
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.pool.Exec(ctx, query,
		reading.ClientID,
		reading.MetricName,
		reading.MetricValue,
		reading.ReadingTimestamp,
		reading.ReceivedAt,
		reading.ValidationStatus,
		reading.AnomalyReason,
		reading.RawPayload,
	)

	if err != nil {
		return fmt.Errorf("failed to insert meter reading: %w", err)
	}

	return nil
}

// BeginTx starts a new transaction
func (r *Repository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.pool.Begin(ctx)
}

// InsertMeterReadingTx inserts a meter reading within a transaction
func (r *Repository) InsertMeterReadingTx(ctx context.Context, tx pgx.Tx, reading *db.MeterReading) error {
	query := `
		INSERT INTO meter_readings_raw (
			client_id, metric_name, metric_value, reading_timestamp,
			received_at, validation_status, anomaly_reason, raw_payload
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := tx.Exec(ctx, query,
		reading.ClientID,
		reading.MetricName,
		reading.MetricValue,
		reading.ReadingTimestamp,
		reading.ReceivedAt,
		reading.ValidationStatus,
		reading.AnomalyReason,
		reading.RawPayload,
	)

	if err != nil {
		return fmt.Errorf("failed to insert meter reading: %w", err)
	}

	return nil
}

// GetRecentReadingsForClient gets recent readings for anomaly detection
func (r *Repository) GetRecentReadingsForClient(ctx context.Context, clientID uuid.UUID, metricName string, limit int) ([]float64, error) {
	query := `
		SELECT metric_value
		FROM meter_readings_raw
		WHERE client_id = $1 AND metric_name = $2 AND validation_status = 'valid'
		ORDER BY reading_timestamp DESC
		LIMIT $3
	`

	rows, err := r.pool.Query(ctx, query, clientID, metricName, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent readings: %w", err)
	}
	defer rows.Close()

	var values []float64
	for rows.Next() {
		var value float64
		if err := rows.Scan(&value); err != nil {
			return nil, fmt.Errorf("failed to scan value: %w", err)
		}
		values = append(values, value)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return values, nil
}

// GetOrCreateClientTx retrieves or creates a meter client within a transaction
func (r *Repository) GetOrCreateClientTx(ctx context.Context, tx pgx.Tx, fingerprint string, ipAddress string, userAgent *string) (*db.MeterClient, error) {
	// Try to get existing client
	query := `
		SELECT id, client_fingerprint, ip_address::text, user_agent, first_seen_at, last_seen_at, created_at
		FROM meter_clients
		WHERE client_fingerprint = $1
	`

	var client db.MeterClient
	err := tx.QueryRow(ctx, query, fingerprint).Scan(
		&client.ID,
		&client.ClientFingerprint,
		&client.IPAddress,
		&client.UserAgent,
		&client.FirstSeenAt,
		&client.LastSeenAt,
		&client.CreatedAt,
	)

	if err == nil {
		// Client exists, update last_seen_at
		updateQuery := `
			UPDATE meter_clients
			SET last_seen_at = $1
			WHERE id = $2
		`
		now := time.Now()
		_, err = tx.Exec(ctx, updateQuery, now, client.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to update client last_seen_at: %w", err)
		}
		client.LastSeenAt = now
		return &client, nil
	}

	if err != pgx.ErrNoRows {
		return nil, fmt.Errorf("failed to query client: %w", err)
	}

	// Client doesn't exist, create new one
	insertQuery := `
		INSERT INTO meter_clients (client_fingerprint, ip_address, user_agent, first_seen_at, last_seen_at, created_at)
		VALUES ($1, $2, $3, $4, $4, $4)
		RETURNING id, client_fingerprint, ip_address::text, user_agent, first_seen_at, last_seen_at, created_at
	`

	now := time.Now()
	err = tx.QueryRow(ctx, insertQuery, fingerprint, ipAddress, userAgent, now).Scan(
		&client.ID,
		&client.ClientFingerprint,
		&client.IPAddress,
		&client.UserAgent,
		&client.FirstSeenAt,
		&client.LastSeenAt,
		&client.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return &client, nil
}
