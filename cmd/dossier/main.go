package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/stockyard-dev/stockyard-dossier/internal/server"
	"github.com/stockyard-dev/stockyard-dossier/internal/store"
	"github.com/stockyard-dev/stockyard/bus"
)

// version is set at build time via -ldflags="-X main.version=v3.6.0"
// (see .github/workflows/release.yml). The release workflow uses the
// git tag verbatim, which already has a "v" prefix, so we strip it
// here before formatting to avoid the "vv3.6.0" double-v we saw in
// production output.
var version = "dev"

func main() {
	// Handle --version / -v / version BEFORE flag.Parse so we don't
	// crash with "flag provided but not defined: -version" when a
	// customer runs the standard CLI version check.
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v", "version":
			fmt.Printf("dossier %s\n", displayVersion())
			os.Exit(0)
		}
	}

	portFlag := flag.String("port", "", "HTTP port (overrides PORT env var)")
	dataFlag := flag.String("data", "", "Data directory (overrides DATA_DIR env var)")
	flag.Parse()

	port := *portFlag
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = "9700"
	}

	dataDir := *dataFlag
	if dataDir == "" {
		dataDir = os.Getenv("DATA_DIR")
	}
	if dataDir == "" {
		dataDir = "./dossier-data"
	}

	db, err := store.Open(dataDir)
	if err != nil {
		log.Fatalf("dossier: %v", err)
	}
	defer db.Close()

	// Bus lives one level up from the per-tool data dir so all tools in
	// a bundle share a single _bus.db. Bus failures are non-fatal —
	// dossier still serves its users when the bus is unavailable.
	if b, berr := bus.Open(filepath.Dir(dataDir), "dossier"); berr != nil {
		log.Printf("dossier: bus disabled: %v", berr)
	} else {
		defer b.Close()
		db.SetPublisher(b)
		log.Printf("dossier: bus enabled at %s", filepath.Join(filepath.Dir(dataDir), "_bus.db"))
	}

	srv := server.New(db, server.DefaultLimits(dataDir), dataDir)

	fmt.Printf("\n  Dossier %s — Self-hosted contact and CRM manager\n", displayVersion())
	fmt.Printf("  Dashboard:  http://localhost:%s/ui\n", port)
	fmt.Printf("  API:        http://localhost:%s/api\n", port)
	fmt.Printf("  Data:       %s\n", dataDir)
	fmt.Printf("  Questions?  hello@stockyard.dev — I read every message\n\n")

	log.Printf("dossier: listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, srv))
}

// displayVersion normalizes the version string to always show as "vX.Y.Z"
// (with exactly one leading v) regardless of whether the build-time value
// already had a v prefix. Fixes the "vv3.6.0" double-v from previous
// releases where the tag-based ldflag value was naively concatenated
// with a literal "v" in the printf.
func displayVersion() string {
	v := strings.TrimPrefix(version, "v")
	if v == "" || v == "dev" {
		return "dev"
	}
	return "v" + v
}
