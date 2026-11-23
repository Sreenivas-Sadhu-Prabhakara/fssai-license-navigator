// Command fssai-license-navigator serves a small local web tool that works out
// which FSSAI licence tier a food business needs and watches the expiry dates
// of the licences it already holds.
//
// It is one binary with no dependencies. The user interface is embedded in the
// binary, it listens on the loopback address only, it stores everything in a
// single local JSON file, and it makes no outbound network calls of any kind.
package main

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sreenivas-sadhu-prabhakara/fssai-license-navigator/internal/fssai"
)

//go:embed web
var webFiles embed.FS

// storeVersion is the schema version written into the data file.
const storeVersion = 1

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "address to listen on; loopback only by default")
	data := flag.String("data", "./fssai-license-navigator-data.json", "path to the local data file")
	flag.Parse()

	st, err := openStore(*data)
	if err != nil {
		log.Fatalf("data file %s: %v", *data, err)
	}

	srv := &http.Server{
		Addr:              *addr,
		Handler:           newServer(st),
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("FSSAI licence navigator listening on http://%s", *addr)
	log.Printf("data file: %s", st.path)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

// ---------------------------------------------------------------- data store

// document is the whole contents of the data file.
type document struct {
	Version int `json:"version"`
	// Params is present only when the user has overridden a published figure.
	Params   *fssai.Params   `json:"params,omitempty"`
	Licences []fssai.Licence `json:"licences"`
}

// store keeps the document in memory and writes it back atomically.
type store struct {
	mu   sync.Mutex
	path string
	doc  document
}

// openStore loads the data file, creating nothing until the first write. An
// overridden figure set that no longer validates is dropped rather than
// allowed to produce a wrong tier.
func openStore(path string) (*store, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	s := &store{path: abs, doc: document{Version: storeVersion, Licences: []fssai.Licence{}}}

	raw, err := os.ReadFile(abs)
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return s, nil
	}
	var doc document
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("cannot read the data file: %w", err)
	}
	if doc.Licences == nil {
		doc.Licences = []fssai.Licence{}
	}
	if doc.Params != nil {
		if err := doc.Params.Validate(); err != nil {
			log.Printf("ignoring the stored thresholds and using the published ones: %v", err)
			doc.Params = nil
		}
	}
	doc.Version = storeVersion
	s.doc = doc
	return s, nil
}

// params returns the figure set in force: the user's overrides when they are
// present, otherwise the published ones.
func (s *store) params() fssai.Params {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.paramsLocked()
}

func (s *store) paramsLocked() fssai.Params {
	if s.doc.Params != nil {
		return *s.doc.Params
	}
	return fssai.DefaultParams()
}

func (s *store) overridden() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.doc.Params != nil
}

func (s *store) licences() []fssai.Licence {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]fssai.Licence, len(s.doc.Licences))
	copy(out, s.doc.Licences)
	return out
}

func (s *store) addLicence(l fssai.Licence) (fssai.Licence, error) {
	if err := l.Validate(); err != nil {
		return fssai.Licence{}, err
	}
	l.BusinessName = strings.TrimSpace(l.BusinessName)
	l.LicenceNumber = fssai.NormalizeLicenceNumber(l.LicenceNumber)
	l.Notes = strings.TrimSpace(l.Notes)

	s.mu.Lock()
	defer s.mu.Unlock()
	if l.ID == "" {
		l.ID = newID()
	}
	replaced := false
	for i, existing := range s.doc.Licences {
		if existing.ID == l.ID {
			s.doc.Licences[i] = l
			replaced = true
			break
		}
	}
	if !replaced {
		s.doc.Licences = append(s.doc.Licences, l)
	}
	if err := s.saveLocked(); err != nil {
		return fssai.Licence{}, err
	}
	return l, nil
}

// errNoSuchLicence reports a delete for an identifier that is not stored.
var errNoSuchLicence = errors.New("no licence with that identifier")

func (s *store) deleteLicence(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.doc.Licences {
		if existing.ID == id {
			s.doc.Licences = append(s.doc.Licences[:i], s.doc.Licences[i+1:]...)
			return s.saveLocked()
		}
	}
	return errNoSuchLicence
}

func (s *store) setParams(p fssai.Params) error {
	if err := p.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.doc.Params = &p
	return s.saveLocked()
}

func (s *store) resetParams() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.doc.Params = nil
	return s.saveLocked()
}

// saveLocked writes the document through a temporary file in the same
// directory and renames it over the data file, so an interrupted write cannot
// leave a half-written file behind.
func (s *store) saveLocked() error {
	s.doc.Version = storeVersion
	if s.doc.Licences == nil {
		s.doc.Licences = []fssai.Licence{}
	}
	body, err := json.MarshalIndent(s.doc, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(s.path)+".tmp*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(body); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		return err
	}
	return os.Rename(tmpName, s.path)
}

// newID returns a short random identifier for a saved licence.
func newID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(b[:])
}

// ------------------------------------------------------------------- serving

// newServer wires the embedded user interface and the local JSON API.
func newServer(st *store) http.Handler {
	ui, err := fs.Sub(webFiles, "web")
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /", http.FileServerFS(ui))

	mux.HandleFunc("GET /api/state", func(w http.ResponseWriter, r *http.Request) {
		asOf, err := asOfFrom(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		p := st.params()
		type option struct {
			Key   string `json:"key"`
			Label string `json:"label"`
		}
		activities := make([]option, 0, len(fssai.Activities()))
		for _, a := range fssai.Activities() {
			activities = append(activities, option{Key: string(a), Label: a.Label()})
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"asOf":       asOf.Format(fssai.DateLayout),
			"dataFile":   st.path,
			"params":     p,
			"defaults":   fssai.DefaultParams(),
			"overridden": st.overridden(),
			"activities": activities,
			"units":      []option{{Key: string(fssai.UnitKgPerDay), Label: "kg per day"}, {Key: string(fssai.UnitMTPerDay), Label: "tonnes per day"}},
			"tiers": []option{
				{Key: fssai.TierBasicRegistration.Slug(), Label: fssai.TierBasicRegistration.Name()},
				{Key: fssai.TierStateLicence.Slug(), Label: fssai.TierStateLicence.Name()},
				{Key: fssai.TierCentralLicence.Slug(), Label: fssai.TierCentralLicence.Name()},
			},
			"licences":  st.licences(),
			"watchlist": fssai.WatchlistWith(p, st.licences(), asOf),
		})
	})

	mux.HandleFunc("GET /api/tier", func(w http.ResponseWriter, r *http.Request) {
		in, err := inputFromQuery(r.URL.Query())
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, fssai.TierWith(st.params(), in))
	})

	mux.HandleFunc("GET /api/checklist.csv", func(w http.ResponseWriter, r *http.Request) {
		in, err := inputFromQuery(r.URL.Query())
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		d := fssai.TierWith(st.params(), in)
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition",
			fmt.Sprintf("attachment; filename=%q", "fssai-"+d.Tier.Slug()+"-checklist.csv"))
		if err := fssai.WriteChecklistCSV(w, d); err != nil {
			log.Printf("checklist export: %v", err)
		}
	})

	mux.HandleFunc("GET /api/licences", func(w http.ResponseWriter, r *http.Request) {
		asOf, err := asOfFrom(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeLicences(w, st, asOf)
	})

	mux.HandleFunc("POST /api/licences", func(w http.ResponseWriter, r *http.Request) {
		asOf, err := asOfFrom(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		var l fssai.Licence
		if err := decodeJSON(w, r, &l); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if _, err := st.addLicence(l); err != nil {
			if errors.Is(err, fssai.ErrInvalidLicence) {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeLicences(w, st, asOf)
	})

	mux.HandleFunc("DELETE /api/licences/{id}", func(w http.ResponseWriter, r *http.Request) {
		asOf, err := asOfFrom(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := st.deleteLicence(r.PathValue("id")); err != nil {
			if errors.Is(err, errNoSuchLicence) {
				writeError(w, http.StatusNotFound, err)
				return
			}
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeLicences(w, st, asOf)
	})

	mux.HandleFunc("PUT /api/params", func(w http.ResponseWriter, r *http.Request) {
		p := st.params()
		if err := decodeJSON(w, r, &p); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := st.setParams(p); err != nil {
			if errors.Is(err, fssai.ErrInvalidParams) {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"params": st.params(), "overridden": st.overridden()})
	})

	mux.HandleFunc("DELETE /api/params", func(w http.ResponseWriter, r *http.Request) {
		if err := st.resetParams(); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"params": st.params(), "overridden": st.overridden()})
	})

	return withGuards(mux)
}

func writeLicences(w http.ResponseWriter, st *store, asOf time.Time) {
	p := st.params()
	ls := st.licences()
	writeJSON(w, http.StatusOK, map[string]any{
		"asOf":      asOf.Format(fssai.DateLayout),
		"licences":  ls,
		"watchlist": fssai.WatchlistWith(p, ls, asOf),
	})
}

// withGuards adds the headers a local tool should send and refuses a
// cross-origin write, so that a page in another tab cannot reach in and change
// the data file.
func withGuards(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")

		if origin := r.Header.Get("Origin"); origin != "" {
			if origin != "http://"+r.Host && origin != "https://"+r.Host {
				writeError(w, http.StatusForbidden, errors.New("cross-origin requests are not accepted"))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// ------------------------------------------------------------------- plumbing

// asOfFrom reads the optional asOf date, which lets the watchdog be asked how
// things will stand on another day. It defaults to today.
func asOfFrom(r *http.Request) (time.Time, error) {
	raw := r.URL.Query().Get("asOf")
	if raw == "" {
		return time.Now(), nil
	}
	return fssai.ParseDate(raw)
}

// inputFromQuery reads a questionnaire answer set from the query string, which
// is what both the live calculation and the CSV export are driven by.
func inputFromQuery(q map[string][]string) (fssai.Input, error) {
	get := func(key string) string {
		if v, ok := q[key]; ok && len(v) > 0 {
			return strings.TrimSpace(v[0])
		}
		return ""
	}
	num := func(key string) (float64, error) {
		raw := get(key)
		if raw == "" {
			return 0, nil
		}
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return 0, fmt.Errorf("%s must be a number", key)
		}
		return v, nil
	}
	flagOf := func(key string) bool {
		switch strings.ToLower(get(key)) {
		case "1", "true", "yes", "on":
			return true
		}
		return false
	}

	var in fssai.Input
	var err error

	if in.AnnualTurnoverINR, err = num("turnover"); err != nil {
		return in, err
	}
	if in.ProductionCapacityPerDay, err = num("capacity"); err != nil {
		return in, err
	}
	if in.DairyLitresPerDay, err = num("dairy"); err != nil {
		return in, err
	}
	if in.DairyMilkSolidsMTPerYear, err = num("solids"); err != nil {
		return in, err
	}

	activity := get("activity")
	if activity == "" {
		activity = string(fssai.ActivityManufacturer)
	}
	a, err := fssai.ParseActivity(activity)
	if err != nil {
		return in, err
	}
	in.Activity = a

	unit, err := fssai.ParseCapacityUnit(get("unit"))
	if err != nil {
		return in, err
	}
	in.CapacityUnit = unit

	in.StatesOfOperation = 1
	if raw := get("states"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return in, errors.New("states must be a whole number")
		}
		in.StatesOfOperation = n
	}

	in.IsImporter = flagOf("importer")
	in.IsExporter = flagOf("exporter")
	in.IsEcommerce = flagOf("ecommerce")
	in.Is100PctExportOriented = flagOf("eou")
	in.OperatesAtAirportOrSeaport = flagOf("port")

	if err := in.Validate(); err != nil {
		return in, err
	}
	return in, nil
}

// maxBody caps a request body; nothing this tool accepts is large.
const maxBody = 64 << 10

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("cannot read the request: %w", err)
	}
	return nil
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("writing a response: %v", err)
	}
}

func writeError(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]string{"error": err.Error()})
}
