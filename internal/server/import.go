package server

// CSV import for dossier contacts.
//
// Two-step flow:
//
//   1. POST /api/import/preview — accepts raw CSV text, returns the
//      detected headers, the first 5 sample rows, and an auto-suggested
//      column mapping ({"Email Address" → "email", "Full Name" → "name"}).
//      The client uses this to show a mapping UI before any data is
//      written.
//
//   2. POST /api/import/commit — accepts raw CSV text + the final
//      column mapping (the user may have adjusted the suggestions),
//      parses the whole file into Contact records, and runs them
//      through ImportBatch which dedupes against existing rows.
//      Returns ImportResult.
//
// Why two steps instead of one: a single-step "upload and import"
// flow gives the user no chance to fix wrong column detection. If
// the CSV has columns "First Name", "Last Name", "Email", a naive
// auto-detect might pick "First Name" as the name field and lose
// the last names. The preview step lets the user confirm or adjust
// before any data is written.
//
// Why server-side parsing instead of client-side JS: encoding/csv
// is rock solid and handles quoted commas, escaped quotes, and CRLF
// line endings correctly. The equivalent JS is brittle and bug-prone,
// and shipping a JS CSV library would bloat the dashboard. Server-side
// parsing also means the same logic can be used by a future CLI
// import command without code duplication.

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/stockyard-dev/stockyard-dossier/internal/store"
)

// Maximum CSV size we accept in a single import. 5MB is enough for
// roughly 50,000 typical contacts (~100 bytes/row), which is well past
// what any small business would have. Limits prevent a malicious or
// confused upload from blowing up the process or filling the disk.
const maxImportBytes = 5 << 20

// importableField is one of the Contact fields a CSV column can map to.
// "" means the column is intentionally skipped (not imported).
type importableField struct {
	Key     string   `json:"key"`               // canonical field name; "" = skip
	Label   string   `json:"label"`             // human label for the dropdown
	Aliases []string `json:"aliases,omitempty"` // header substrings that auto-map to this field
}

// importableFields is the list of fields the import UI shows in each
// column's mapping dropdown. Order matters — it controls dropdown order.
// Aliases are matched case-insensitively as substrings, so "Customer
// Email Address" matches the email field via the "email" alias.
var importableFields = []importableField{
	{Key: "name", Label: "Name", Aliases: []string{"name", "full name", "contact", "customer", "client"}},
	{Key: "email", Label: "Email", Aliases: []string{"email", "e-mail", "mail address"}},
	{Key: "phone", Label: "Phone", Aliases: []string{"phone", "tel", "mobile", "cell", "number"}},
	{Key: "company", Label: "Company", Aliases: []string{"company", "organization", "org", "business", "employer"}},
	{Key: "title", Label: "Title / Role", Aliases: []string{"title", "role", "position", "job"}},
	{Key: "tags", Label: "Tags", Aliases: []string{"tag", "category", "label", "group"}},
	{Key: "notes", Label: "Notes", Aliases: []string{"note", "comment", "description", "memo"}},
	{Key: "status", Label: "Status", Aliases: []string{"status", "stage", "state"}},
	{Key: "", Label: "— Skip this column —"},
}

// importPreviewRequest is what the client POSTs to /api/import/preview.
// The CSV is sent as a string in the JSON body rather than as a file
// upload because (a) it's simpler than parsing multipart, (b) the file
// is already in JS memory after the user picks it, and (c) we cap the
// total request size in maxImportBytes anyway.
type importPreviewRequest struct {
	CSV string `json:"csv"`
}

// importPreviewResponse is the shape returned to the client.
type importPreviewResponse struct {
	Headers    []string          `json:"headers"`     // raw column names from CSV
	SampleRows [][]string        `json:"sample_rows"` // first 5 data rows
	TotalRows  int               `json:"total_rows"`  // total parseable rows in the CSV
	Mapping    map[string]string `json:"mapping"`     // column header → field key (auto-suggested)
	Fields     []importableField `json:"fields"`      // available target fields for the dropdown
	Warnings   []string          `json:"warnings,omitempty"`
}

// importCommitRequest is the second-step request body. The mapping is
// trusted as-is — if the client sends an unknown field key it's
// silently ignored, mirroring how the API handles extra JSON fields
// elsewhere.
type importCommitRequest struct {
	CSV     string            `json:"csv"`
	Mapping map[string]string `json:"mapping"` // column header → field key
}

// previewImport handles step 1: parse the CSV, detect headers, build
// auto-suggested mapping, return sample rows and totals. Does NOT
// touch the database.
func (s *Server) previewImport(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxImportBytes+1))
	if err != nil {
		we(w, 400, "could not read request body")
		return
	}
	if len(body) > maxImportBytes {
		we(w, 413, fmt.Sprintf("CSV too large — max %d bytes (about 50,000 contacts)", maxImportBytes))
		return
	}

	var req importPreviewRequest
	if err := json.Unmarshal(body, &req); err != nil {
		we(w, 400, "invalid json: "+err.Error())
		return
	}
	if strings.TrimSpace(req.CSV) == "" {
		we(w, 400, "csv field is empty")
		return
	}

	headers, allRows, warnings, err := parseCSV(req.CSV)
	if err != nil {
		we(w, 400, "could not parse CSV: "+err.Error())
		return
	}
	if len(headers) == 0 {
		we(w, 400, "CSV has no header row")
		return
	}

	// Sample the first 5 rows for the UI preview.
	sample := allRows
	if len(sample) > 5 {
		sample = sample[:5]
	}

	// Auto-suggest mapping by matching each header against the alias
	// list of each field. First match wins; ties are broken by the
	// order in importableFields. Headers with no match get "" (skip).
	mapping := make(map[string]string, len(headers))
	for _, h := range headers {
		mapping[h] = autoMapHeader(h)
	}

	wj(w, 200, importPreviewResponse{
		Headers:    headers,
		SampleRows: sample,
		TotalRows:  len(allRows),
		Mapping:    mapping,
		Fields:     importableFields,
		Warnings:   warnings,
	})
}

// commitImport handles step 2: parse the CSV, apply the user's column
// mapping, and bulk-insert via ImportBatch. Returns the ImportResult.
func (s *Server) commitImport(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxImportBytes+1))
	if err != nil {
		we(w, 400, "could not read request body")
		return
	}
	if len(body) > maxImportBytes {
		we(w, 413, fmt.Sprintf("CSV too large — max %d bytes", maxImportBytes))
		return
	}

	var req importCommitRequest
	if err := json.Unmarshal(body, &req); err != nil {
		we(w, 400, "invalid json: "+err.Error())
		return
	}
	if strings.TrimSpace(req.CSV) == "" {
		we(w, 400, "csv field is empty")
		return
	}
	if len(req.Mapping) == 0 {
		we(w, 400, "mapping is required — call /api/import/preview first")
		return
	}

	headers, allRows, _, err := parseCSV(req.CSV)
	if err != nil {
		we(w, 400, "could not parse CSV: "+err.Error())
		return
	}

	// Validate that the user's mapping references headers that actually
	// exist in the CSV. Without this check, a stale mapping (e.g. user
	// previewed one CSV, then committed a different one) would silently
	// produce zero successful imports — every row would fail with
	// "name required" because the name column never gets set.
	headerSet := make(map[string]bool, len(headers))
	for _, h := range headers {
		headerSet[h] = true
	}
	hasNameHeader := false
	for header, field := range req.Mapping {
		if field == "" {
			continue // explicit skip is fine even if header is missing
		}
		if !headerSet[header] {
			we(w, 400, fmt.Sprintf("mapping references header %q that does not exist in the uploaded CSV — re-run preview to refresh the mapping", header))
			return
		}
		if field == "name" {
			hasNameHeader = true
		}
	}
	if !hasNameHeader {
		we(w, 400, "at least one CSV column must be mapped to 'name'")
		return
	}

	// Build a header-index → field-key map for fast row processing.
	// Anything mapped to "" (skip) is omitted from this index.
	colToField := make(map[int]string, len(headers))
	for i, h := range headers {
		if field, ok := req.Mapping[h]; ok && field != "" {
			colToField[i] = field
		}
	}

	// Convert each row to a Contact via the column mapping.
	contacts := make([]store.Contact, 0, len(allRows))
	for _, row := range allRows {
		c := store.Contact{}
		for i, val := range row {
			if i >= len(headers) {
				continue // ragged row, skip extras
			}
			field, ok := colToField[i]
			if !ok {
				continue
			}
			val = strings.TrimSpace(val)
			switch field {
			case "name":
				c.Name = val
			case "email":
				c.Email = val
			case "phone":
				c.Phone = val
			case "company":
				c.Company = val
			case "title":
				c.Title = val
			case "tags":
				c.Tags = val
			case "notes":
				c.Notes = val
			case "status":
				c.Status = val
			}
		}
		contacts = append(contacts, c)
	}

	// Free-tier limit check, AFTER building the contact slice so we can
	// run the dedup dry-run. Previously this used len(allRows) which
	// counted duplicates and rows-with-no-name toward the limit, falsely
	// rejecting imports that would only insert a few new records. Now
	// we count exactly what would be inserted.
	if s.limits.MaxItems > 0 {
		current := s.db.Count()
		wouldInsert := s.db.CountNewInBatch(contacts)
		if current+wouldInsert > s.limits.MaxItems {
			we(w, 402, fmt.Sprintf(
				"this import would push you to %d records (current %d, %d new after dedup) — free tier limit is %d. Upgrade at https://stockyard.dev/dossier/",
				current+wouldInsert, current, wouldInsert, s.limits.MaxItems,
			))
			return
		}
	}

	result, err := s.db.ImportBatch(contacts)
	if err != nil {
		log.Printf("dossier: import failed: %v", err)
		// SQLITE_BUSY (database locked) means another write is in
		// flight — usually a concurrent import or the autosave from
		// the dashboard form. Surface a clearer message than the
		// raw sqlite error so the customer knows to retry rather
		// than thinking their data is corrupt.
		if strings.Contains(err.Error(), "database is locked") || strings.Contains(err.Error(), "SQLITE_BUSY") {
			we(w, 503, "another change is in progress — please wait a moment and try again")
			return
		}
		we(w, 500, "import failed: "+err.Error())
		return
	}

	log.Printf("dossier: import complete — created=%d skipped=%d failed=%d",
		result.Created, result.Skipped, result.Failed)
	wj(w, 200, result)
}

// parseCSV is the shared CSV reader used by both preview and commit.
// Returns headers (first row) + data rows + any non-fatal warnings.
//
// Resilience choices:
//   - LazyQuotes=true so a stray unescaped quote in the middle of a
//     field doesn't kill the whole import. Many real-world CSVs from
//     Excel and Google Sheets have this issue.
//   - FieldsPerRecord=-1 so ragged rows (some columns missing) don't
//     error. The mapping logic handles missing columns gracefully.
//   - Trailing blank rows are stripped — they're a common artifact
//     of CSVs hand-edited in Excel.
func parseCSV(text string) (headers []string, rows [][]string, warnings []string, err error) {
	r := csv.NewReader(strings.NewReader(text))
	r.LazyQuotes = true
	r.FieldsPerRecord = -1
	r.TrimLeadingSpace = true

	all, err := r.ReadAll()
	if err != nil {
		return nil, nil, nil, err
	}
	if len(all) == 0 {
		return nil, nil, nil, fmt.Errorf("CSV is empty")
	}
	headers = all[0]
	// Strip BOM from the first header if present (Excel-saved CSVs).
	if len(headers) > 0 && strings.HasPrefix(headers[0], "\ufeff") {
		headers[0] = strings.TrimPrefix(headers[0], "\ufeff")
	}
	for i := range headers {
		headers[i] = strings.TrimSpace(headers[i])
	}

	rawRows := all[1:]
	// Strip trailing blank rows.
	for len(rawRows) > 0 {
		last := rawRows[len(rawRows)-1]
		isBlank := true
		for _, v := range last {
			if strings.TrimSpace(v) != "" {
				isBlank = false
				break
			}
		}
		if !isBlank {
			break
		}
		rawRows = rawRows[:len(rawRows)-1]
	}

	// Filter inline blank rows too — they don't error, just get warned.
	skipped := 0
	for _, row := range rawRows {
		isBlank := true
		for _, v := range row {
			if strings.TrimSpace(v) != "" {
				isBlank = false
				break
			}
		}
		if isBlank {
			skipped++
			continue
		}
		rows = append(rows, row)
	}
	if skipped > 0 {
		warnings = append(warnings, fmt.Sprintf("skipped %d blank rows", skipped))
	}

	// Check for duplicate headers, which would make column-mapping
	// ambiguous (which "Email" column does the user mean?). We don't
	// fail — just warn and let the second occurrence win in the map.
	seen := make(map[string]int)
	for _, h := range headers {
		seen[h]++
	}
	for h, count := range seen {
		if count > 1 {
			warnings = append(warnings, fmt.Sprintf("duplicate header %q appears %d times — only the last will be used", h, count))
		}
	}

	return headers, rows, warnings, nil
}

// autoMapHeader picks the best target field for a CSV header by
// substring-matching against each field's aliases. Case-insensitive.
// Returns "" (skip) if no field matches.
func autoMapHeader(header string) string {
	h := strings.ToLower(strings.TrimSpace(header))
	if h == "" {
		return ""
	}
	for _, f := range importableFields {
		if f.Key == "" {
			continue
		}
		for _, alias := range f.Aliases {
			if strings.Contains(h, alias) {
				return f.Key
			}
		}
	}
	return ""
}
