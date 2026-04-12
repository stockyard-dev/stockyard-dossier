package store

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	db  *sql.DB
	pub Publisher // optional; nil when bus is disabled or unconfigured
}

// Publisher is the narrow slice of github.com/stockyard-dev/stockyard/bus
// that dossier uses. Declared as an interface here so the store package
// stays decoupled from the bus package (no import cycle risk, easy to
// stub in tests, and the store keeps working when the bus is disabled).
type Publisher interface {
	Publish(topic string, payload any) (int64, error)
}

// SetPublisher wires an optional bus publisher into the store. Safe to
// call with nil; subsequent mutations will skip publishing. Intended to
// be called once at boot after store.Open succeeds.
func (d *DB) SetPublisher(p Publisher) { d.pub = p }

// publish emits one event and swallows errors — the bus is a side
// channel, and a publish failure must not cause the mutation to appear
// failed to the caller. Errors surface only via the bus's own logs.
func (d *DB) publish(topic string, payload any) {
	if d.pub == nil {
		return
	}
	if _, err := d.pub.Publish(topic, payload); err != nil {
		// Intentional log-only. See README: the only signal that the
		// bus is misbehaving for a publisher is a log line.
		_ = err
	}
}

type Contact struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	Phone      string `json:"phone"`
	Company    string `json:"company"`
	Title      string `json:"title"`
	Tags       string `json:"tags"`
	Notes      string `json:"notes"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
	ImportHash string `json:"import_hash,omitempty"`
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

	// Migration: add import_hash column to contacts so CSV import can
	// dedupe on (name + email + phone) content. Idempotent — sqlite
	// errors with "duplicate column name" on subsequent boots, which
	// is the expected case after the first migration. We can't easily
	// distinguish that error from a real failure (modernc/sqlite returns
	// a string error, not a typed code), so we treat all errors as
	// "column probably exists" and rely on the next ALTER's behavior
	// to surface real corruption.
	db.Exec(`ALTER TABLE contacts ADD COLUMN import_hash TEXT DEFAULT ''`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_contacts_import_hash ON contacts(import_hash) WHERE import_hash != ''`)

	// Backfill: any contact row with an empty import_hash needs one
	// computed retroactively. This handles the upgrade path for
	// existing customers — without it, the first import after upgrade
	// would treat every legacy contact as new and create duplicates.
	//
	// Done in chunks to avoid loading the whole contacts table into
	// memory at once on customers who have tens of thousands of records.
	// Idempotent — once a row has a hash, subsequent boots skip it
	// because the WHERE filter excludes rows with import_hash != ''.
	if err := backfillImportHashes(db); err != nil {
		// Non-fatal: dedup just won't catch legacy contacts. Log for
		// triage but don't refuse to start the tool.
		fmt.Fprintf(os.Stderr, "dossier: warning: import_hash backfill failed: %v\n", err)
	}

	return &DB{db: db}, nil
}

// backfillImportHashes populates import_hash for any rows that have it
// empty, using the same ContactImportHash function the import path uses.
// Called once at Open(); subsequent boots are no-ops because the WHERE
// filter has nothing to do.
//
// Chunking avoids loading the entire contacts table into memory on
// customers with very large databases — we read 500 rows at a time
// and update each in its own statement inside a single transaction
// per chunk.
func backfillImportHashes(db *sql.DB) error {
	const chunkSize = 500
	for {
		rows, err := db.Query(
			`SELECT id, name, email, phone FROM contacts WHERE import_hash = '' LIMIT ?`,
			chunkSize,
		)
		if err != nil {
			return fmt.Errorf("query unhashed: %w", err)
		}
		type unhashed struct{ id, name, email, phone string }
		var batch []unhashed
		for rows.Next() {
			var u unhashed
			if err := rows.Scan(&u.id, &u.name, &u.email, &u.phone); err == nil {
				batch = append(batch, u)
			}
		}
		rows.Close()
		if len(batch) == 0 {
			return nil
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin chunk: %w", err)
		}
		stmt, err := tx.Prepare(`UPDATE contacts SET import_hash=? WHERE id=?`)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("prepare update: %w", err)
		}
		for _, u := range batch {
			h := ContactImportHash(u.name, u.email, u.phone)
			if h == "" {
				// All three identifying fields blank — set a sentinel
				// so we don't loop forever on this row. Empty contacts
				// won't dedup, but they also won't be re-processed.
				h = "-"
			}
			if _, err := stmt.Exec(h, u.id); err != nil {
				stmt.Close()
				tx.Rollback()
				return fmt.Errorf("update %s: %w", u.id, err)
			}
		}
		stmt.Close()
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit chunk: %w", err)
		}
		if len(batch) < chunkSize {
			return nil
		}
	}
}

func (d *DB) Close() error { return d.db.Close() }

func genID() string { return fmt.Sprintf("%d", time.Now().UnixNano()) }
func now() string   { return time.Now().UTC().Format(time.RFC3339) }

// ContactImportHash computes a stable content hash for dedup purposes.
// Two contacts with the same (lowercased name) + (lowercased email) +
// (digits-only phone) produce the same hash. This is the pragmatic
// definition of "duplicate" for import: it catches the common cases
// (same person re-imported from a fresh CSV export) without being so
// strict that case differences in capitalization or formatting create
// false negatives.
//
// Edge case: if a contact has only a name and no email/phone, the
// hash includes just the name. Two contacts named "John Smith" with
// no other identifying info will be treated as duplicates. This is
// intentional — the alternative is to skip dedup entirely for sparse
// records, which is worse than an occasional false positive.
//
// Empty contacts (no name, no email, no phone) hash to a sentinel
// value that is intentionally NOT inserted into the import_hash
// column on commit. This means rows with no identifying info don't
// participate in dedup at all and can be re-imported infinitely.
// That's the right behavior because such rows are usually CSV
// artifacts (trailing blank lines) that the parser should already
// have filtered out.
func ContactImportHash(name, email, phone string) string {
	n := strings.ToLower(strings.TrimSpace(name))
	e := strings.ToLower(strings.TrimSpace(email))
	// Phone normalization: keep only digits. "(555) 123-4567" and
	// "555-123-4567" and "5551234567" all hash the same.
	var pb strings.Builder
	for _, c := range phone {
		if c >= '0' && c <= '9' {
			pb.WriteRune(c)
		}
	}
	p := pb.String()
	if n == "" && e == "" && p == "" {
		return ""
	}
	h := sha256.Sum256([]byte(n + "|" + e + "|" + p))
	return hex.EncodeToString(h[:])
}

func (d *DB) Create(e *Contact) error {
	e.ID = genID()
	e.CreatedAt = now()
	if e.ImportHash == "" {
		e.ImportHash = ContactImportHash(e.Name, e.Email, e.Phone)
	}
	_, err := d.db.Exec(
		`INSERT INTO contacts(id, name, email, phone, company, title, tags, notes, status, created_at, import_hash)
		 VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.Name, e.Email, e.Phone, e.Company, e.Title, e.Tags, e.Notes, e.Status, e.CreatedAt, e.ImportHash,
	)
	if err == nil {
		d.publish("contacts.created", e)
	}
	return err
}

func (d *DB) Get(id string) *Contact {
	var e Contact
	err := d.db.QueryRow(
		`SELECT id, name, email, phone, company, title, tags, notes, status, created_at, import_hash
		 FROM contacts WHERE id=?`,
		id,
	).Scan(&e.ID, &e.Name, &e.Email, &e.Phone, &e.Company, &e.Title, &e.Tags, &e.Notes, &e.Status, &e.CreatedAt, &e.ImportHash)
	if err != nil {
		return nil
	}
	return &e
}

func (d *DB) List() []Contact {
	rows, _ := d.db.Query(
		`SELECT id, name, email, phone, company, title, tags, notes, status, created_at, import_hash
		 FROM contacts ORDER BY created_at DESC`,
	)
	if rows == nil {
		return nil
	}
	defer rows.Close()
	var o []Contact
	for rows.Next() {
		var e Contact
		rows.Scan(&e.ID, &e.Name, &e.Email, &e.Phone, &e.Company, &e.Title, &e.Tags, &e.Notes, &e.Status, &e.CreatedAt, &e.ImportHash)
		o = append(o, e)
	}
	return o
}

func (d *DB) Update(e *Contact) error {
	// Recompute import_hash on update — if name/email/phone changed,
	// the dedup key needs to track the new content. Without this, an
	// edited contact would no longer match itself on re-import.
	e.ImportHash = ContactImportHash(e.Name, e.Email, e.Phone)
	_, err := d.db.Exec(
		`UPDATE contacts SET name=?, email=?, phone=?, company=?, title=?, tags=?, notes=?, status=?, import_hash=?
		 WHERE id=?`,
		e.Name, e.Email, e.Phone, e.Company, e.Title, e.Tags, e.Notes, e.Status, e.ImportHash, e.ID,
	)
	if err == nil {
		d.publish("contacts.updated", e)
	}
	return err
}

func (d *DB) Delete(id string) error {
	_, err := d.db.Exec(`DELETE FROM contacts WHERE id=?`, id)
	if err == nil {
		d.publish("contacts.deleted", map[string]string{"id": id})
	}
	return err
}

func (d *DB) Count() int {
	var n int
	d.db.QueryRow(`SELECT COUNT(*) FROM contacts`).Scan(&n)
	return n
}

// CountNewInBatch returns how many records in the given batch would
// actually be inserted by ImportBatch — i.e. excluding duplicates
// (both against existing rows and against earlier rows in the same
// batch) and excluding rows with no name. Used by the import handler
// to do an accurate free-tier limit check before running the actual
// transactional insert.
//
// This is a read-only dry-run: no rows are written. The cost is one
// SELECT plus a hash computation per input row.
func (d *DB) CountNewInBatch(records []Contact) int {
	if len(records) == 0 {
		return 0
	}
	seen := make(map[string]bool)
	rows, err := d.db.Query(`SELECT import_hash FROM contacts WHERE import_hash != ''`)
	if err != nil {
		// Fall back to assuming nothing dedupes — this overcounts but
		// fails open (refuses too many imports) rather than fails
		// closed (lets the user blow past their limit).
		count := 0
		for _, r := range records {
			if strings.TrimSpace(r.Name) != "" {
				count++
			}
		}
		return count
	}
	defer rows.Close()
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err == nil && h != "" {
			seen[h] = true
		}
	}

	count := 0
	for _, r := range records {
		if strings.TrimSpace(r.Name) == "" {
			continue
		}
		hash := ContactImportHash(r.Name, r.Email, r.Phone)
		if hash != "" && seen[hash] {
			continue
		}
		count++
		if hash != "" {
			seen[hash] = true
		}
	}
	return count
}

// ImportResult is what ImportBatch returns: per-row outcome counts plus
// any per-row errors so the UI can show "imported 47, skipped 3 dups,
// 0 failed" without needing a second query.
type ImportResult struct {
	Created int      `json:"created"`
	Skipped int      `json:"skipped"`
	Failed  int      `json:"failed"`
	Errors  []string `json:"errors,omitempty"`
}

// ImportBatch inserts a slice of contacts in a single transaction with
// dedup. Records whose import_hash already exists in the table are
// skipped (counted as Skipped, not as errors). Records that fail to
// insert for any other reason are counted as Failed and the error
// message is appended to Errors.
//
// Why a single transaction: a CSV upload of 1000 contacts is ~1000
// INSERTs. Without a transaction sqlite would fsync after every row,
// which on a real disk takes ~1ms each = 1 second of waiting. Inside
// a transaction the whole batch fsyncs once. Empirically this is the
// difference between "feels broken" and "feels instant" for any CSV
// over a few hundred rows.
//
// The dedup query is a SELECT inside the transaction, which on sqlite
// reads from the same connection's transaction view. This means two
// rows in the same import batch with identical content will dedup
// against each other (the first wins, the second is skipped) — not
// just against pre-existing rows. That's the right behavior; users
// don't expect a single import to insert duplicates of itself.
func (d *DB) ImportBatch(records []Contact) (*ImportResult, error) {
	out := &ImportResult{}
	if len(records) == 0 {
		return out, nil
	}

	tx, err := d.db.Begin()
	if err != nil {
		return out, fmt.Errorf("begin: %w", err)
	}
	// Defer rollback; the explicit commit below sets tx to nil so the
	// rollback becomes a no-op on the success path.
	defer func() {
		if tx != nil {
			tx.Rollback()
		}
	}()

	// Pre-compute hashes and seed the seen set with everything already
	// in the table. We could do per-row SELECTs but a single load up
	// front turns N round trips into 1 — much faster for any non-tiny
	// batch.
	seen := make(map[string]bool)
	rows, err := tx.Query(`SELECT import_hash FROM contacts WHERE import_hash != ''`)
	if err != nil {
		return out, fmt.Errorf("scan existing hashes: %w", err)
	}
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err == nil && h != "" {
			seen[h] = true
		}
	}
	rows.Close()

	stmt, err := tx.Prepare(
		`INSERT INTO contacts(id, name, email, phone, company, title, tags, notes, status, created_at, import_hash)
		 VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return out, fmt.Errorf("prepare insert: %w", err)
	}
	defer stmt.Close()

	for i := range records {
		r := &records[i]
		if strings.TrimSpace(r.Name) == "" {
			out.Failed++
			out.Errors = append(out.Errors, fmt.Sprintf("row %d: name is required", i+1))
			continue
		}
		if r.Status == "" {
			r.Status = "active"
		}
		hash := ContactImportHash(r.Name, r.Email, r.Phone)
		if hash != "" && seen[hash] {
			out.Skipped++
			continue
		}
		r.ID = genID()
		r.CreatedAt = now()
		r.ImportHash = hash
		if _, err := stmt.Exec(
			r.ID, r.Name, r.Email, r.Phone, r.Company, r.Title, r.Tags, r.Notes, r.Status, r.CreatedAt, r.ImportHash,
		); err != nil {
			out.Failed++
			if len(out.Errors) < 20 {
				out.Errors = append(out.Errors, fmt.Sprintf("row %d (%s): %v", i+1, r.Name, err))
			}
			continue
		}
		out.Created++
		if hash != "" {
			seen[hash] = true
		}
		// genID uses time.Now().UnixNano() and will collide if two
		// inserts happen in the same nanosecond — sleep a hair to
		// guarantee monotonic IDs across the whole batch. Cheaper
		// than uuid generation, sufficient for the import use case.
		time.Sleep(time.Nanosecond)
	}

	if err := tx.Commit(); err != nil {
		return out, fmt.Errorf("commit: %w", err)
	}
	tx = nil // suppress deferred rollback
	return out, nil
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
		`SELECT id, name, email, phone, company, title, tags, notes, status, created_at, import_hash
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
		rows.Scan(&e.ID, &e.Name, &e.Email, &e.Phone, &e.Company, &e.Title, &e.Tags, &e.Notes, &e.Status, &e.CreatedAt, &e.ImportHash)
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
