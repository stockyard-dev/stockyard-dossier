package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct{ db *sql.DB }

type Contact struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	Company   string `json:"company"`
	Title     string `json:"title"`
	Tags      string `json:"tags"`
	Notes     string `json:"notes"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

func Open(d string) (*DB, error) {
	if err := os.MkdirAll(d, 0755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", filepath.Join(d, "dossier.db")+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}
	db.Exec(`CREATE TABLE IF NOT EXISTS contacts(
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		email TEXT DEFAULT '',
		phone TEXT DEFAULT '',
		company TEXT DEFAULT '',
		title TEXT DEFAULT '',
		tags TEXT DEFAULT '',
		notes TEXT DEFAULT '',
		status TEXT DEFAULT 'active',
		created_at TEXT DEFAULT(datetime('now'))
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS extras(
		resource TEXT NOT NULL,
		record_id TEXT NOT NULL,
		data TEXT NOT NULL DEFAULT '{}',
		PRIMARY KEY(resource, record_id)
	)`)
	return &DB{db: db}, nil
}

func (d *DB) Close() error { return d.db.Close() }

func genID() string { return fmt.Sprintf("%d", time.Now().UnixNano()) }
func now() string   { return time.Now().UTC().Format(time.RFC3339) }

func (d *DB) Create(e *Contact) error {
	e.ID = genID()
	e.CreatedAt = now()
	_, err := d.db.Exec(
		`INSERT INTO contacts(id, name, email, phone, company, title, tags, notes, status, created_at)
		 VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.Name, e.Email, e.Phone, e.Company, e.Title, e.Tags, e.Notes, e.Status, e.CreatedAt,
	)
	return err
}

func (d *DB) Get(id string) *Contact {
	var e Contact
	err := d.db.QueryRow(
		`SELECT id, name, email, phone, company, title, tags, notes, status, created_at
		 FROM contacts WHERE id=?`,
		id,
	).Scan(&e.ID, &e.Name, &e.Email, &e.Phone, &e.Company, &e.Title, &e.Tags, &e.Notes, &e.Status, &e.CreatedAt)
	if err != nil {
		return nil
	}
	return &e
}

func (d *DB) List() []Contact {
	rows, _ := d.db.Query(
		`SELECT id, name, email, phone, company, title, tags, notes, status, created_at
		 FROM contacts ORDER BY created_at DESC`,
	)
	if rows == nil {
		return nil
	}
	defer rows.Close()
	var o []Contact
	for rows.Next() {
		var e Contact
		rows.Scan(&e.ID, &e.Name, &e.Email, &e.Phone, &e.Company, &e.Title, &e.Tags, &e.Notes, &e.Status, &e.CreatedAt)
		o = append(o, e)
	}
	return o
}

func (d *DB) Update(e *Contact) error {
	_, err := d.db.Exec(
		`UPDATE contacts SET name=?, email=?, phone=?, company=?, title=?, tags=?, notes=?, status=?
		 WHERE id=?`,
		e.Name, e.Email, e.Phone, e.Company, e.Title, e.Tags, e.Notes, e.Status, e.ID,
	)
	return err
}

func (d *DB) Delete(id string) error {
	_, err := d.db.Exec(`DELETE FROM contacts WHERE id=?`, id)
	return err
}

func (d *DB) Count() int {
	var n int
	d.db.QueryRow(`SELECT COUNT(*) FROM contacts`).Scan(&n)
	return n
}

func (d *DB) Search(q string, filters map[string]string) []Contact {
	where := "1=1"
	args := []any{}
	if q != "" {
		where += " AND (name LIKE ? OR email LIKE ? OR company LIKE ? OR title LIKE ?)"
		args = append(args, "%"+q+"%", "%"+q+"%", "%"+q+"%", "%"+q+"%")
	}
	if v, ok := filters["status"]; ok && v != "" {
		where += " AND status=?"
		args = append(args, v)
	}
	rows, _ := d.db.Query(
		`SELECT id, name, email, phone, company, title, tags, notes, status, created_at
		 FROM contacts WHERE `+where+` ORDER BY created_at DESC`,
		args...,
	)
	if rows == nil {
		return nil
	}
	defer rows.Close()
	var o []Contact
	for rows.Next() {
		var e Contact
		rows.Scan(&e.ID, &e.Name, &e.Email, &e.Phone, &e.Company, &e.Title, &e.Tags, &e.Notes, &e.Status, &e.CreatedAt)
		o = append(o, e)
	}
	return o
}

func (d *DB) Stats() map[string]any {
	m := map[string]any{"total": d.Count()}
	rows, _ := d.db.Query(`SELECT status, COUNT(*) FROM contacts GROUP BY status`)
	if rows != nil {
		defer rows.Close()
		by := map[string]int{}
		for rows.Next() {
			var s string
			var c int
			rows.Scan(&s, &c)
			by[s] = c
		}
		m["by_status"] = by
	}
	return m
}

// ─── Extras: generic key-value storage for personalization custom fields ───

// GetExtras returns the JSON extras blob for a record. Returns "{}" if none.
func (d *DB) GetExtras(resource, recordID string) string {
	var data string
	err := d.db.QueryRow(
		`SELECT data FROM extras WHERE resource=? AND record_id=?`,
		resource, recordID,
	).Scan(&data)
	if err != nil || data == "" {
		return "{}"
	}
	return data
}

// SetExtras stores the JSON extras blob for a record (upsert).
func (d *DB) SetExtras(resource, recordID, data string) error {
	if data == "" {
		data = "{}"
	}
	_, err := d.db.Exec(
		`INSERT INTO extras(resource, record_id, data) VALUES(?, ?, ?)
		 ON CONFLICT(resource, record_id) DO UPDATE SET data=excluded.data`,
		resource, recordID, data,
	)
	return err
}

// DeleteExtras removes extras for a record. Called by delete handlers.
func (d *DB) DeleteExtras(resource, recordID string) error {
	_, err := d.db.Exec(
		`DELETE FROM extras WHERE resource=? AND record_id=?`,
		resource, recordID,
	)
	return err
}

// AllExtras returns all extras for a resource as a map of record_id → JSON string.
func (d *DB) AllExtras(resource string) map[string]string {
	out := make(map[string]string)
	rows, _ := d.db.Query(
		`SELECT record_id, data FROM extras WHERE resource=?`,
		resource,
	)
	if rows == nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id, data string
		rows.Scan(&id, &data)
		out[id] = data
	}
	return out
}
