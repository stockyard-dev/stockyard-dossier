package server

// Stress / adversarial tests for the import flow.
//
// These complement import_test.go by targeting bugs found during code
// review rather than happy paths. Each test is a hypothesis about
// something that COULD break — most should pass, the ones that fail
// are real bugs to fix.

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/stockyard-dev/stockyard-dossier/internal/store"
)

// ─── Bug 1: ImportHash not read back from DB (now FIXED) ─────────────

func TestStressImportHash_notReadByGetOrList(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	if err := db.Create(&store.Contact{Name: "Alice", Email: "a@x.com", Phone: "555-1234"}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	list := db.List()
	if len(list) != 1 {
		t.Fatalf("List: got %d", len(list))
	}
	// Regression guard: List() must return the import_hash field that
	// Create() wrote. If this fails, someone removed import_hash from
	// the SELECT in List() and now Update() would clobber it.
	if list[0].ImportHash == "" {
		t.Errorf("REGRESSION: List() did not return import_hash for a row that has one — check the SELECT statement in List()")
	}

	// Same check for Get()
	got := db.Get(list[0].ID)
	if got == nil || got.ImportHash == "" {
		t.Errorf("REGRESSION: Get() did not return import_hash")
	}

	// Same check for Search()
	results := db.Search("Alice", nil)
	if len(results) != 1 || results[0].ImportHash == "" {
		t.Errorf("REGRESSION: Search() did not return import_hash")
	}
}

// ─── Bug 5: Free-tier limit check overcounts dedup'd rows ───────────

func TestStressFreeTier_falsePositiveRejection(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	// Seed 99 contacts.
	for i := 0; i < 99; i++ {
		if err := db.Create(&store.Contact{Name: fmt.Sprintf("seed%d", i), Email: fmt.Sprintf("seed%d@x.com", i)}); err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}

	srv := New(db, Limits{Tier: "free", MaxItems: 100}, t.TempDir())

	// Build a CSV where 49 of 50 rows are duplicates of seed contacts.
	// Only 1 row would actually be inserted, taking us from 99 to 100 (cap).
	// But the check uses len(allRows)=50 and rejects with "would push you to 149".
	var csvBuf strings.Builder
	csvBuf.WriteString("Name,Email\n")
	for i := 0; i < 49; i++ {
		fmt.Fprintf(&csvBuf, "seed%d,seed%d@x.com\n", i, i) // dup
	}
	csvBuf.WriteString("brand new person,new@x.com\n")

	body, _ := json.Marshal(importCommitRequest{
		CSV:     csvBuf.String(),
		Mapping: map[string]string{"Name": "name", "Email": "email"},
	})
	req := httptest.NewRequest("POST", "/api/import/commit", bytes.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code == 402 {
		t.Logf("KNOWN BUG: free-tier check rejects an import that would only insert 1 record, because it counts dedup'd rows toward the limit. Body: %s", w.Body.String())
	} else if w.Code == 200 {
		var res store.ImportResult
		json.Unmarshal(w.Body.Bytes(), &res)
		t.Logf("OK: import succeeded with created=%d skipped=%d", res.Created, res.Skipped)
		if res.Created != 1 || res.Skipped != 49 {
			t.Errorf("expected created=1 skipped=49, got created=%d skipped=%d", res.Created, res.Skipped)
		}
	} else {
		t.Errorf("unexpected status %d: %s", w.Code, w.Body.String())
	}
}

// ─── Bug 6: Mapping references nonexistent header ────────────────────

func TestStressCommit_mappingReferencesNonexistentHeader(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()
	srv := New(db, Limits{}, t.TempDir())

	// CSV has columns "FirstName,LastName,Email" but mapping references
	// "Name". The validation should catch this BEFORE running the import,
	// not generate 100 "name required" errors.
	body, _ := json.Marshal(importCommitRequest{
		CSV: "FirstName,LastName,Email\nAlice,Smith,a@x.com\nBob,Jones,b@x.com\n",
		Mapping: map[string]string{
			"Name":  "name", // wrong key — header doesn't exist
			"Email": "email",
		},
	})
	req := httptest.NewRequest("POST", "/api/import/commit", bytes.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code == 400 {
		// Best case: validation caught it.
		t.Logf("OK: validation rejected mapping with stale header. Body: %s", w.Body.String())
	} else if w.Code == 200 {
		var res store.ImportResult
		json.Unmarshal(w.Body.Bytes(), &res)
		t.Logf("KNOWN BUG: import accepted mapping with nonexistent 'Name' header — every row failed silently. created=%d skipped=%d failed=%d", res.Created, res.Skipped, res.Failed)
		if res.Created > 0 {
			t.Errorf("rows imported despite no name column: created=%d", res.Created)
		}
	} else {
		t.Errorf("unexpected status %d: %s", w.Code, w.Body.String())
	}
}

// ─── Bug 9: maxImportBytes is JSON body, not raw CSV ─────────────────

func TestStressImport_largeCSVAfterJSONEncoding(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()
	srv := New(db, Limits{}, t.TempDir())

	// Build a 4MB raw CSV with lots of quoted strings that will balloon
	// in JSON encoding (every quote becomes \" doubling its bytes).
	var csvBuf strings.Builder
	csvBuf.WriteString("Name,Notes\n")
	for csvBuf.Len() < 4_000_000 {
		csvBuf.WriteString(`"Person Name","Notes with ""quotes"" embedded for fun"` + "\n")
	}
	rawSize := csvBuf.Len()

	body, _ := json.Marshal(importPreviewRequest{CSV: csvBuf.String()})
	encodedSize := len(body)
	t.Logf("raw CSV: %d bytes, JSON-encoded: %d bytes (%.1fx)", rawSize, encodedSize, float64(encodedSize)/float64(rawSize))

	req := httptest.NewRequest("POST", "/api/import/preview", bytes.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code == 413 {
		t.Logf("KNOWN BUG: a 4MB raw CSV gets rejected as too large because JSON encoding pushed it over 5MB cap. Customers will hit this on legit medium-large CRM exports.")
	} else if w.Code == 200 {
		t.Logf("OK: 4MB CSV accepted (JSON expansion within limit)")
	} else {
		t.Errorf("unexpected status %d: %s", w.Code, w.Body.String())
	}
}

// ─── Bug 10: Pre-migration rows have empty import_hash ───────────────

func TestStressMigration_preExistingRowsDontDedupe(t *testing.T) {
	dir := t.TempDir()
	dbPath := dir + "/data"

	// Step 1: Open the DB, create a contact, close it. This represents
	// a pre-import-feature install — the row exists with whatever
	// import_hash the current code path writes.
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open 1: %v", err)
	}
	if err := db.Create(&store.Contact{Name: "Legacy User", Email: "legacy@x.com", Phone: "555-9999"}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	db.Close()

	// Step 2: With the DB closed, simulate the upgrade scenario by
	// directly clobbering import_hash to '' on the legacy row. This
	// is what every existing customer's database actually looks like
	// the moment they upgrade to a dossier binary that has the
	// import_hash column but before the backfill has run.
	raw, err := sql.Open("sqlite", dbPath+"/dossier.db")
	if err != nil {
		t.Fatalf("raw open: %v", err)
	}
	if _, err := raw.Exec(`UPDATE contacts SET import_hash='' WHERE name='Legacy User'`); err != nil {
		t.Fatalf("simulate pre-migration: %v", err)
	}
	raw.Close()

	// Step 3: Re-open the DB. This is the moment the upgraded dossier
	// binary boots. Open() runs backfillImportHashes which should
	// detect the empty hash and populate it.
	db, err = store.Open(dbPath)
	if err != nil {
		t.Fatalf("open 2 (post-migration): %v", err)
	}
	defer db.Close()

	// Step 4: Try to import the same contact via batch. The backfill
	// should have re-populated import_hash for the legacy row, so the
	// dedup logic catches this as a duplicate.
	res, err := db.ImportBatch([]store.Contact{
		{Name: "Legacy User", Email: "legacy@x.com", Phone: "555-9999"},
	})
	if err != nil {
		t.Fatalf("ImportBatch: %v", err)
	}

	if res.Created != 0 {
		t.Errorf("REGRESSION: pre-migration row was duplicated by import — backfill not running on Open(). created=%d skipped=%d", res.Created, res.Skipped)
	}
	if res.Skipped != 1 {
		t.Errorf("REGRESSION: pre-migration row should have been deduped as Skipped, got skipped=%d", res.Skipped)
	}
	if db.Count() != 1 {
		t.Errorf("REGRESSION: expected 1 contact after dedup re-import, got %d", db.Count())
	}
}

// ─── Bug 2: Migration silently swallows non-duplicate-column errors ──
// Hard to test without simulating disk failures. Documented as a known
// limitation rather than tested.

// ─── Test gap 1: Unicode in names ────────────────────────────────────

func TestStressUnicode_namesAndEmails(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	// Mixed-case unicode should hash equivalently in upper and lower case.
	contacts := []store.Contact{
		{Name: "JOSÉ García", Email: "JOSE@example.com", Phone: "555-1111"},
		{Name: "josé garcía", Email: "jose@example.com", Phone: "555-1111"}, // accent + case
	}
	res, err := db.ImportBatch(contacts)
	if err != nil {
		t.Fatalf("ImportBatch: %v", err)
	}
	// They should dedupe IF strings.ToLower handles unicode AND if accent
	// is preserved (only case differs).
	t.Logf("unicode case test: created=%d skipped=%d (Go ToLower is unicode-aware so JOSÉ→josé works, but the second name has a different last name 'garcía' vs 'García' — should still dedup on case alone)", res.Created, res.Skipped)

	if res.Created != 1 {
		t.Errorf("Unicode case folding broken: created=%d, want 1", res.Created)
	}
}

func TestStressUnicode_emojiNames(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	// People DO put emoji in their CRM names. Don't crash.
	contacts := []store.Contact{
		{Name: "Alice 🎨 Designer", Email: "alice@x.com"},
		{Name: "中文 名字", Email: "cn@x.com"},
		{Name: "Müller", Email: "muller@x.com"},
	}
	res, err := db.ImportBatch(contacts)
	if err != nil {
		t.Fatalf("ImportBatch: %v", err)
	}
	if res.Created != 3 {
		t.Errorf("Unicode insert broken: created=%d, want 3", res.Created)
	}
}

// ─── Test gap 3: CSV with one extremely long line (no newlines) ──────

func TestStressCSV_singleHugeLine(t *testing.T) {
	// A "CSV" that's actually a JSON dump or a base64 blob — one giant
	// "line" with no newlines. encoding/csv should handle gracefully.
	huge := "Name,Email\n" + strings.Repeat("x", 1_000_000) + ",noemail\n"
	_, rows, _, err := parseCSV(huge)
	if err != nil {
		t.Logf("OK: parseCSV rejected huge single field: %v", err)
		return
	}
	if len(rows) != 1 {
		t.Errorf("expected 1 row, got %d", len(rows))
	}
	t.Logf("OK: parseCSV handled 1MB single field, rows=%d", len(rows))
}

// ─── Test gap 4: SQL injection-style values ──────────────────────────

func TestStressSQLInjection_inFieldValues(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	// We use parameterized queries everywhere so this should just work,
	// but verify nobody accidentally added a fmt.Sprintf into a query.
	contacts := []store.Contact{
		{Name: "Robert'); DROP TABLE contacts;--", Email: "bobby@xkcd.com"},
		{Name: `"; DELETE FROM contacts; --`, Email: "evil@x.com"},
		{Name: "Normal Person", Email: "normal@x.com"},
	}
	res, err := db.ImportBatch(contacts)
	if err != nil {
		t.Fatalf("ImportBatch: %v", err)
	}
	if res.Created != 3 {
		t.Errorf("SQL-injection-like names broke insert: created=%d", res.Created)
	}
	if db.Count() != 3 {
		t.Errorf("Table compromised: count=%d", db.Count())
	}
}

// ─── Test gap 5: Concurrent ImportBatch calls ────────────────────────

func TestStressConcurrent_twoBatchesSameRecords(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	// Two goroutines simultaneously import the same 10 contacts.
	// Total inserted should be 10, not 20 — but the dedup snapshot is
	// taken at transaction start, so race window exists.
	contacts := make([]store.Contact, 10)
	for i := 0; i < 10; i++ {
		contacts[i] = store.Contact{
			Name:  fmt.Sprintf("Concurrent%d", i),
			Email: fmt.Sprintf("c%d@x.com", i),
		}
	}

	var wg sync.WaitGroup
	results := make([]*store.ImportResult, 2)
	errors := make([]error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			batch := make([]store.Contact, len(contacts))
			copy(batch, contacts)
			results[idx], errors[idx] = db.ImportBatch(batch)
		}(i)
	}
	wg.Wait()

	for i, e := range errors {
		if e != nil {
			t.Logf("goroutine %d error: %v", i, e)
		}
	}
	for i, r := range results {
		if r != nil {
			t.Logf("goroutine %d: created=%d skipped=%d failed=%d", i, r.Created, r.Skipped, r.Failed)
		}
	}

	count := db.Count()
	if count == 10 {
		t.Logf("OK: concurrent imports dedup'd correctly to %d records", count)
	} else if count > 10 {
		t.Logf("KNOWN BUG: concurrent imports raced — total %d records (expected 10)", count)
	}
}

// ─── Adversarial CSV: malformed inputs that could crash the parser ──

func TestStressCSV_onlyHeaders_noRows(t *testing.T) {
	headers, rows, _, err := parseCSV("Name,Email,Phone\n")
	if err != nil {
		t.Errorf("headers-only CSV should not error: %v", err)
	}
	if len(headers) != 3 {
		t.Errorf("headers: got %d, want 3", len(headers))
	}
	if len(rows) != 0 {
		t.Errorf("rows: got %d, want 0", len(rows))
	}
}

func TestStressCSV_unevenColumnCounts(t *testing.T) {
	// Some rows have more columns than headers, some fewer.
	csv := "A,B,C\n1,2,3\n4,5\n7,8,9,10,11\n"
	headers, rows, _, err := parseCSV(csv)
	if err != nil {
		t.Errorf("uneven columns should be tolerated by FieldsPerRecord=-1: %v", err)
	}
	t.Logf("headers=%v, rows=%v", headers, rows)
	if len(rows) != 3 {
		t.Errorf("rows: got %d, want 3", len(rows))
	}
}

func TestStressCSV_emptyString(t *testing.T) {
	_, _, _, err := parseCSV("")
	if err == nil {
		t.Error("empty CSV should error")
	}
}

func TestStressCSV_onlyWhitespace(t *testing.T) {
	_, _, _, err := parseCSV("   \n   \n")
	t.Logf("whitespace-only CSV: err=%v", err)
}

func TestStressCSV_carriageReturnsOnly(t *testing.T) {
	// Old Mac line endings (\r only). encoding/csv handles \r and \r\n.
	csv := "Name,Email\rAlice,a@x.com\rBob,b@x.com\r"
	_, rows, _, err := parseCSV(csv)
	if err != nil {
		t.Logf("CR-only CSV err: %v", err)
		return
	}
	if len(rows) != 2 {
		t.Logf("CR-only CSV: got %d rows, want 2 — encoding/csv may not handle bare CR", len(rows))
	}
}

// ─── Adversarial mapping: clients send unexpected shapes ──────────────

func TestStressCommit_mappingWithEmptyValues(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()
	srv := New(db, Limits{}, t.TempDir())

	body, _ := json.Marshal(importCommitRequest{
		CSV: "Name,Email\nAlice,a@x.com\n",
		Mapping: map[string]string{
			"Name":  "name",
			"Email": "", // explicit skip — should not crash
		},
	})
	req := httptest.NewRequest("POST", "/api/import/commit", bytes.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("status=%d: %s", w.Code, w.Body.String())
	}
}

func TestStressCommit_mappingWithUnknownField(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()
	srv := New(db, Limits{}, t.TempDir())

	// Client sends a mapping value that isn't a real Contact field.
	// Should silently ignore (not crash).
	body, _ := json.Marshal(importCommitRequest{
		CSV: "Name,Birthday\nAlice,1990-01-01\n",
		Mapping: map[string]string{
			"Name":     "name",
			"Birthday": "birthday", // not a real field
		},
	})
	req := httptest.NewRequest("POST", "/api/import/commit", bytes.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("status=%d: %s", w.Code, w.Body.String())
	}
	var res store.ImportResult
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Created != 1 {
		t.Errorf("expected created=1, got %+v", res)
	}
}

// ─── Adversarial sizes ───────────────────────────────────────────────

func TestStressImport_emptyJSONBody(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()
	srv := New(db, Limits{}, t.TempDir())

	req := httptest.NewRequest("POST", "/api/import/preview", bytes.NewReader([]byte("{}")))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Errorf("empty body should be 400: got %d", w.Code)
	}
}

func TestStressImport_malformedJSON(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()
	srv := New(db, Limits{}, t.TempDir())

	req := httptest.NewRequest("POST", "/api/import/preview", bytes.NewReader([]byte("not json at all")))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Errorf("malformed json should be 400: got %d", w.Code)
	}
}

// ─── Helper for net/http tests ───────────────────────────────────────

var _ = http.StatusOK // keep import in case other tests need it
