package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct{ db *sql.DB }

type CleanRecord struct {
	ID, PlanID, Status, Payload string
	StartedAt, CompletedAt      time.Time
	ReclaimedBytes              int64
}

func Open(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", filepath.Join(dataDir, "ai-clear.db"))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	store := &Store{db: db}
	if err := store.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) migrate() error {
	statements := []string{
		`PRAGMA journal_mode=WAL`, `PRAGMA foreign_keys=ON`,
		`CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS settings (key TEXT PRIMARY KEY, value_json TEXT NOT NULL, updated_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS rule_packages (id TEXT PRIMARY KEY, version INTEGER NOT NULL, source TEXT NOT NULL, verified_at TEXT)`,
		`CREATE TABLE IF NOT EXISTS rules (id TEXT PRIMARY KEY, package_id TEXT, version INTEGER NOT NULL, enabled INTEGER NOT NULL, payload_json TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS scan_jobs (id TEXT PRIMARY KEY, status TEXT NOT NULL, mode TEXT NOT NULL, started_at TEXT NOT NULL, completed_at TEXT)`,
		`CREATE TABLE IF NOT EXISTS scan_items (id TEXT NOT NULL, scan_id TEXT NOT NULL, rule_id TEXT NOT NULL, path TEXT NOT NULL, allocated_size INTEGER NOT NULL, payload_json TEXT NOT NULL, PRIMARY KEY(id,scan_id))`,
		`CREATE TABLE IF NOT EXISTS clean_jobs (id TEXT PRIMARY KEY, plan_id TEXT NOT NULL, status TEXT NOT NULL, started_at TEXT NOT NULL, completed_at TEXT NOT NULL, reclaimed_bytes INTEGER NOT NULL, payload_json TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS clean_items (clean_id TEXT NOT NULL, item_id TEXT NOT NULL, status TEXT NOT NULL, action TEXT NOT NULL, payload_json TEXT NOT NULL, PRIMARY KEY(clean_id,item_id))`,
		`CREATE TABLE IF NOT EXISTS ai_providers (id TEXT PRIMARY KEY, enabled INTEGER NOT NULL, config_json TEXT NOT NULL, updated_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS ai_sessions (id TEXT PRIMARY KEY, provider_id TEXT NOT NULL, status TEXT NOT NULL, created_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS ai_messages (id TEXT PRIMARY KEY, session_id TEXT NOT NULL, role TEXT NOT NULL, summary TEXT NOT NULL, created_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS audit_events (id INTEGER PRIMARY KEY AUTOINCREMENT, event_type TEXT NOT NULL, subject_id TEXT, status TEXT NOT NULL, created_at TEXT NOT NULL, detail_json TEXT NOT NULL)`,
		`INSERT OR IGNORE INTO schema_migrations(version,applied_at) VALUES(1,CURRENT_TIMESTAMP)`,
	}
	for _, statement := range statements {
		if _, err := s.db.Exec(statement); err != nil {
			return fmt.Errorf("database migration: %w", err)
		}
	}
	return s.migrateLegacyCleanSchema()
}

func (s *Store) migrateLegacyCleanSchema() error {
	rows, err := s.db.Query(`PRAGMA table_info(clean_jobs)`)
	if err != nil {
		return err
	}
	hasLegacyColumn := false
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, kind string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &kind, &notNull, &defaultValue, &primaryKey); err != nil {
			rows.Close()
			return err
		}
		hasLegacyColumn = hasLegacyColumn || name == "quarantined_bytes"
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if hasLegacyColumn {
		tx, err := s.db.Begin()
		if err != nil {
			return err
		}
		statements := []string{`CREATE TABLE clean_jobs_v2 (id TEXT PRIMARY KEY, plan_id TEXT NOT NULL, status TEXT NOT NULL, started_at TEXT NOT NULL, completed_at TEXT NOT NULL, reclaimed_bytes INTEGER NOT NULL, payload_json TEXT NOT NULL)`, `INSERT INTO clean_jobs_v2(id,plan_id,status,started_at,completed_at,reclaimed_bytes,payload_json) SELECT id,plan_id,status,started_at,completed_at,reclaimed_bytes,payload_json FROM clean_jobs`, `DROP TABLE clean_jobs`, `ALTER TABLE clean_jobs_v2 RENAME TO clean_jobs`}
		for _, statement := range statements {
			if _, err := tx.Exec(statement); err != nil {
				_ = tx.Rollback()
				return err
			}
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) SaveClean(record CleanRecord) error {
	_, err := s.db.Exec(`INSERT OR REPLACE INTO clean_jobs(id,plan_id,status,started_at,completed_at,reclaimed_bytes,payload_json) VALUES(?,?,?,?,?,?,?)`, record.ID, record.PlanID, record.Status, record.StartedAt.Format(time.RFC3339Nano), record.CompletedAt.Format(time.RFC3339Nano), record.ReclaimedBytes, record.Payload)
	return err
}

func (s *Store) CleanRecords() ([]CleanRecord, error) {
	rows, err := s.db.Query(`SELECT id,plan_id,status,started_at,completed_at,reclaimed_bytes,payload_json FROM clean_jobs ORDER BY completed_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]CleanRecord, 0)
	for rows.Next() {
		var record CleanRecord
		var started, completed string
		if err := rows.Scan(&record.ID, &record.PlanID, &record.Status, &started, &completed, &record.ReclaimedBytes, &record.Payload); err != nil {
			return nil, err
		}
		record.StartedAt, _ = time.Parse(time.RFC3339Nano, started)
		record.CompletedAt, _ = time.Parse(time.RFC3339Nano, completed)
		result = append(result, record)
	}
	return result, rows.Err()
}

func (s *Store) SaveProvider(id string, enabled bool, payload string) error {
	_, err := s.db.Exec(`INSERT OR REPLACE INTO ai_providers(id,enabled,config_json,updated_at) VALUES(?,?,?,?)`, id, enabled, payload, time.Now().Format(time.RFC3339Nano))
	return err
}

func (s *Store) Provider(id string) (string, bool, error) {
	var payload string
	err := s.db.QueryRow(`SELECT config_json FROM ai_providers WHERE id=?`, id).Scan(&payload)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	return payload, err == nil, err
}

// SaveSetting stores a small application setting as JSON. Settings are kept
// beside the application database so portable installs do not write config to
// the user's profile.
func (s *Store) SaveSetting(key string, value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT OR REPLACE INTO settings(key,value_json,updated_at) VALUES(?,?,?)`, key, string(payload), time.Now().Format(time.RFC3339Nano))
	return err
}

func (s *Store) GetSetting(key string, target any) (bool, error) {
	var payload string
	err := s.db.QueryRow(`SELECT value_json FROM settings WHERE key=?`, key).Scan(&payload)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal([]byte(payload), target); err != nil {
		return false, err
	}
	return true, nil
}
