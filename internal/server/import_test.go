package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stockyard-dev/stockyard-dossier/internal/store"
)

// Tests for the CSV import flow. These exercise parseCSV, autoMapHeader,
// and ImportBatch end-to-end without going through HTTP — the goal is to
// catch parser, mapping, and dedup bugs in isolation. HTTP-level tests
// would just add ceremony around the same assertions.

func TestParseCSV_basic(t *testing.T) {
	csv := "Name,Email,Phone\nAlice,alice@x.com,555-1234\nBob,bob@y.com,555-5678\n"
	headers, rows, warnings, err := parseCSV(csv)
	if err != nil {
		t.Fatalf("parseCSV: %v", err)
	}
	if got := len(headers); got != 3 {
		t.Errorf("headers: got %d, want 3", got)
	}
	if headers[0] != "Name" || headers[1] != "Email" || headers[2] != "Phone" {
		t.Errorf("headers content: got %v", headers)
	}
	if got := len(rows); got != 2 {
		t.Errorf("rows: got %d, want 2", got)
	}
	if len(warnings) > 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
}

func TestParseCSV_BOM_strippedFromFirstHeader(t *testing.T) {
	// Excel saves CSVs with a UTF-8 BOM (\ufeff) at the very start.
	// Without explicit handling this becomes part of the first header
	// and breaks the auto-map (the header reads "\ufeffName" which
	// matches no alias).
	csv := "\ufeffName,Email\nAlice,a@x.com\n"
	headers, _, _, err := parseCSV(csv)
	if err != nil {
		t.Fatalf("parseCSV: %v", err)
	}
	if headers[0] != "Name" {
		t.Errorf("BOM not stripped: header[0]=%q", headers[0])
	}
}

func TestParseCSV_quotedCommasAndNewlines(t *testing.T) {
	// Real CSVs from CRMs often contain quoted fields with commas and
	// embedded newlines (multi-line notes). encoding/csv handles this
	// correctly when fields are properly quoted.
	csv := `Name,Notes
Alice,"prefers morning, weekends only"
Bob,"line one
line two"
`
	_, rows, _, err := parseCSV(csv)
	if err != nil {
		t.Fatalf("parseCSV: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows: got %d, want 2", len(rows))
	}
	if !strings.Contains(rows[0][1], "morning, weekends") {
		t.Errorf("quoted comma lost: %q", rows[0][1])
	}
	if !strings.Contains(rows[1][1], "line one\nline two") {
		t.Errorf("embedded newline lost: %q", rows[1][1])
	}
}

func TestParseCSV_strayQuoteToleratedByLazyQuotes(t *testing.T) {
	// Real-world CSVs sometimes have stray unescaped quotes from
	// hand-edited Excel files. With LazyQuotes=true these don't
	// kill the whole import.
	csv := "Name,Notes\nAlice,5'2\" tall\nBob,normal\n"
	_, rows, _, err := parseCSV(csv)
	if err != nil {
		t.Fatalf("parseCSV with stray quote: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("rows: got %d, want 2", len(rows))
	}
}

func TestParseCSV_trailingBlankRowsStripped(t *testing.T) {
	csv := "Name,Email\nAlice,a@x.com\n\n\n,\n"
	_, rows, _, err := parseCSV(csv)
	if err != nil {
		t.Fatalf("parseCSV: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("trailing blanks not stripped: got %d rows, want 1", len(rows))
	}
}

func TestParseCSV_inlineBlankRowsWarned(t *testing.T) {
	csv := "Name,Email\nAlice,a@x.com\n,\nBob,b@x.com\n"
	_, rows, warnings, err := parseCSV(csv)
	if err != nil {
		t.Fatalf("parseCSV: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("inline blanks not skipped: got %d rows, want 2", len(rows))
	}
	if len(warnings) == 0 {
		t.Error("expected warning about skipped blank rows")
	}
}

func TestParseCSV_duplicateHeadersWarned(t *testing.T) {
	csv := "Name,Email,Email\nAlice,a@x.com,a2@x.com\n"
	_, _, warnings, err := parseCSV(csv)
	if err != nil {
		t.Fatalf("parseCSV: %v", err)
	}
	foundDupWarning := false
	for _, w := range warnings {
		if strings.Contains(w, "duplicate") {
			foundDupWarning = true
		}
	}
	if !foundDupWarning {
		t.Errorf("expected duplicate-header warning, got: %v", warnings)
	}
}

func TestAutoMapHeader_commonAliases(t *testing.T) {
	cases := map[string]string{
		"Name":               "name",
		"Full Name":          "name",
		"Customer Name":      "name",
		"Email":              "email",
		"Email Address":      "email",
		"E-mail":             "email",
		"Phone":              "phone",
		"Phone Number":       "phone",
		"Mobile":             "phone",
		"Cell":               "phone",
		"Company":            "company",
		"Organization":       "company",
		"Title":              "title",
		"Job Title":          "title",
		"Tags":               "tags",
		"Category":           "tags",
		"Notes":              "notes",
		"Comments":           "notes",
		"Status":             "status",
		"Some Random Column": "",
		"Birthday":           "", // we don't have a birthday field
	}
	for header, want := range cases {
		got := autoMapHeader(header)
		if got != want {
			t.Errorf("autoMapHeader(%q) = %q, want %q", header, got, want)
		}
	}
}

func TestImportBatch_basic(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	contacts := []store.Contact{
		{Name: "Alice", Email: "alice@x.com", Phone: "555-1234"},
		{Name: "Bob", Email: "bob@x.com", Phone: "555-5678"},
		{Name: "Carol", Email: "carol@x.com"},
	}
	res, err := db.ImportBatch(contacts)
	if err != nil {
		t.Fatalf("ImportBatch: %v", err)
	}
	if res.Created != 3 {
		t.Errorf("Created: got %d, want 3", res.Created)
	}
	if res.Skipped != 0 {
		t.Errorf("Skipped: got %d, want 0", res.Skipped)
	}
	if res.Failed != 0 {
		t.Errorf("Failed: got %d, want 0 (errors=%v)", res.Failed, res.Errors)
	}
	if got := db.Count(); got != 3 {
		t.Errorf("db.Count(): got %d, want 3", got)
	}
}

func TestImportBatch_dedupesAgainstExisting(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	// Insert one contact via the normal Create path so the dedup
	// hash gets set, then re-import the same contact in a batch.
	if err := db.Create(&store.Contact{Name: "Alice", Email: "alice@x.com", Phone: "555-1234"}); err != nil {
		t.Fatalf("seed create: %v", err)
	}

	contacts := []store.Contact{
		{Name: "Alice", Email: "alice@x.com", Phone: "555-1234"}, // dup
		{Name: "Bob", Email: "bob@x.com", Phone: "555-5678"},     // new
	}
	res, err := db.ImportBatch(contacts)
	if err != nil {
		t.Fatalf("ImportBatch: %v", err)
	}
	if res.Created != 1 {
		t.Errorf("Created: got %d, want 1 (Bob only)", res.Created)
	}
	if res.Skipped != 1 {
		t.Errorf("Skipped: got %d, want 1 (Alice dup)", res.Skipped)
	}
	if got := db.Count(); got != 2 {
		t.Errorf("db.Count(): got %d, want 2", got)
	}
}

func TestImportBatch_dedupesWithinBatch(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	contacts := []store.Contact{
		{Name: "Alice", Email: "alice@x.com", Phone: "555-1234"},
		{Name: "Alice", Email: "alice@x.com", Phone: "555-1234"}, // intra-batch dup
		{Name: "Alice", Email: "alice@x.com", Phone: "555-1234"}, // another intra-batch dup
	}
	res, err := db.ImportBatch(contacts)
	if err != nil {
		t.Fatalf("ImportBatch: %v", err)
	}
	if res.Created != 1 {
		t.Errorf("Created: got %d, want 1", res.Created)
	}
	if res.Skipped != 2 {
		t.Errorf("Skipped: got %d, want 2", res.Skipped)
	}
}

func TestImportBatch_dedupNormalizesPhoneFormatting(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	// Same person, different phone formats. Should be one contact
	// with two skips, because phone normalization strips formatting.
	contacts := []store.Contact{
		{Name: "Alice", Email: "alice@x.com", Phone: "555-123-4567"},
		{Name: "Alice", Email: "alice@x.com", Phone: "(555) 123-4567"},
		{Name: "Alice", Email: "alice@x.com", Phone: "5551234567"},
	}
	res, err := db.ImportBatch(contacts)
	if err != nil {
		t.Fatalf("ImportBatch: %v", err)
	}
	if res.Created != 1 || res.Skipped != 2 {
		t.Errorf("phone normalization broken: created=%d skipped=%d", res.Created, res.Skipped)
	}
}

func TestImportBatch_dedupCaseInsensitiveOnNameAndEmail(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	contacts := []store.Contact{
		{Name: "Alice Johnson", Email: "Alice@Example.com"},
		{Name: "alice johnson", Email: "alice@example.com"}, // same person, different case
		{Name: "ALICE JOHNSON", Email: "ALICE@EXAMPLE.COM"}, // ditto
	}
	res, err := db.ImportBatch(contacts)
	if err != nil {
		t.Fatalf("ImportBatch: %v", err)
	}
	if res.Created != 1 || res.Skipped != 2 {
		t.Errorf("case dedup broken: created=%d skipped=%d", res.Created, res.Skipped)
	}
}

func TestImportBatch_emptyNameFails(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	contacts := []store.Contact{
		{Name: "", Email: "blank@x.com"},
		{Name: "Alice", Email: "alice@x.com"},
	}
	res, err := db.ImportBatch(contacts)
	if err != nil {
		t.Fatalf("ImportBatch: %v", err)
	}
	if res.Created != 1 {
		t.Errorf("Created: got %d, want 1", res.Created)
	}
	if res.Failed != 1 {
		t.Errorf("Failed: got %d, want 1", res.Failed)
	}
	if len(res.Errors) == 0 {
		t.Error("expected error for blank-name row")
	}
}

func TestImportBatch_setsDefaultStatus(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	contacts := []store.Contact{
		{Name: "Alice", Email: "alice@x.com"}, // no Status set
	}
	if _, err := db.ImportBatch(contacts); err != nil {
		t.Fatalf("ImportBatch: %v", err)
	}
	got := db.List()
	if len(got) != 1 {
		t.Fatalf("List: got %d, want 1", len(got))
	}
	if got[0].Status != "active" {
		t.Errorf("Status: got %q, want %q", got[0].Status, "active")
	}
}

func TestImportBatch_emptyBatch(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	res, err := db.ImportBatch([]store.Contact{})
	if err != nil {
		t.Fatalf("ImportBatch: %v", err)
	}
	if res.Created != 0 || res.Skipped != 0 || res.Failed != 0 {
		t.Errorf("empty batch should be all zeros: %+v", res)
	}
}

func TestImportBatch_largeBatchPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("skip large batch in short mode")
	}
	db := newTestDB(t)
	defer db.Close()

	// 1000 unique contacts. The whole import should complete in
	// well under a second on any reasonable hardware thanks to
	// the single-transaction design.
	contacts := make([]store.Contact, 1000)
	for i := 0; i < 1000; i++ {
		contacts[i] = store.Contact{
			Name:  "Person " + itoa(i),
			Email: "person" + itoa(i) + "@x.com",
		}
	}
	res, err := db.ImportBatch(contacts)
	if err != nil {
		t.Fatalf("ImportBatch: %v", err)
	}
	if res.Created != 1000 {
		t.Errorf("Created: got %d, want 1000 (failed=%d errors=%v)",
			res.Created, res.Failed, res.Errors)
	}
}

// ─── helpers ──────────────────────────────────────────────────────────

func newTestDB(t *testing.T) *store.DB {
	t.Helper()
	dir, err := os.MkdirTemp("", "dossier-import-test-*")
	if err != nil {
		t.Fatalf("mkdtemp: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	db, err := store.Open(filepath.Join(dir, "data"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	return db
}

// itoa is a tiny stdlib-free int-to-string for the perf test, to keep
// the test file from importing strconv just for one call.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
