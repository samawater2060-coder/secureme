package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// PostgresStore is a PostgreSQL-backed Store implementation.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore opens a connection to the given DSN and pings it.
func NewPostgresStore(dsn string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}
	return &PostgresStore{db: db}, nil
}

// Migrate runs all *.sql files from dir in sorted order.
func (p *PostgresStore) Migrate(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(files)
	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read %s: %w", f, err)
		}
		if _, err := p.db.Exec(string(content)); err != nil {
			return fmt.Errorf("exec %s: %w", f, err)
		}
	}
	return nil
}

func (p *PostgresStore) UpsertDevice(ctx context.Context, d *Device) error {
	_, err := p.db.ExecContext(ctx, `
		INSERT INTO devices (id, hostname, os, ip, agent_token, enrolled_at, last_seen)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE
		  SET hostname    = EXCLUDED.hostname,
		      os          = EXCLUDED.os,
		      ip          = EXCLUDED.ip,
		      last_seen   = EXCLUDED.last_seen`,
		d.ID, d.Hostname, d.OS, d.IP, d.AgentToken, d.EnrolledAt, d.LastSeen,
	)
	return err
}

func (p *PostgresStore) UpdateLastSeen(ctx context.Context, id string) error {
	res, err := p.db.ExecContext(ctx,
		`UPDATE devices SET last_seen = $1 WHERE id = $2`,
		time.Now().UTC(), id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *PostgresStore) ListDevices(ctx context.Context) ([]*Device, error) {
	rows, err := p.db.QueryContext(ctx,
		`SELECT id, hostname, os, ip, enrolled_at, last_seen FROM devices ORDER BY enrolled_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var devices []*Device
	for rows.Next() {
		d := &Device{}
		if err := rows.Scan(&d.ID, &d.Hostname, &d.OS, &d.IP, &d.EnrolledAt, &d.LastSeen); err != nil {
			return nil, err
		}
		devices = append(devices, d)
	}
	return devices, rows.Err()
}

func (p *PostgresStore) GetDevice(ctx context.Context, id string) (*Device, error) {
	d := &Device{}
	err := p.db.QueryRowContext(ctx,
		`SELECT id, hostname, os, ip, enrolled_at, last_seen FROM devices WHERE id = $1`, id,
	).Scan(&d.ID, &d.Hostname, &d.OS, &d.IP, &d.EnrolledAt, &d.LastSeen)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return d, err
}
