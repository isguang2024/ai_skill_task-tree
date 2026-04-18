package tasktree

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var (
	alphabet = []rune("0123456789ABCDEFGHJKMNPQRSTVWXYZ")
)

type App struct {
	db           *sql.DB
	dbPath       string
	artifactRoot string
	mcpSessions  *mcpSessionStore
	aiSessions   *aiSessionStore
	mux          *http.ServeMux
}

type jsonMap map[string]any

type appError struct {
	Code int
	Msg  string
}

func (e *appError) Error() string { return e.Msg }

type trackingResponseWriter struct {
	http.ResponseWriter
	wroteHeader bool
	statusCode  int
}

func (w *trackingResponseWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *trackingResponseWriter) Write(p []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(p)
}

func (w *trackingResponseWriter) Written() bool {
	return w.wroteHeader
}

func (w *trackingResponseWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *trackingResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("hijack not supported")
}

func (w *trackingResponseWriter) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := w.ResponseWriter.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}
	return http.ErrNotSupported
}

func NewApp() (*App, error) {
	dbPath := os.Getenv("TTS_DB_PATH")
	if dbPath == "" {
		dbPath = "./data/tasks.db"
	}
	absDB, err := filepath.Abs(dbPath)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(absDB), 0o755); err != nil {
		return nil, err
	}
	dsn := absDB + "?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// Allow concurrent local reads to run in parallel.
	// SQLite WAL still serializes writes, but a single connection would
	// unnecessarily queue independent GET requests behind each other.
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)

	app := &App{
		db:           db,
		dbPath:       absDB,
		artifactRoot: filepath.Join(filepath.Dir(absDB), "artifacts"),
		mcpSessions:  newMCPSessionStore(),
		aiSessions:   newAISessionStore(),
		mux:          http.NewServeMux(),
	}
	if err := app.migrate(); err != nil {
		return nil, err
	}
	bootCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := app.ensureDefaultProject(bootCtx); err != nil {
		return nil, err
	}
	if _, err := app.sweepExpiredLeases(bootCtx); err != nil {
		return nil, err
	}
	app.registerRoutes()
	return app, nil
}

func (a *App) ListenAndServe(addr string) error {
	return http.ListenAndServe(addr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.mux.ServeHTTP(&trackingResponseWriter{ResponseWriter: w}, r)
	}))
}

func (a *App) HTTPHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.mux.ServeHTTP(&trackingResponseWriter{ResponseWriter: w}, r)
	})
}

func (a *App) Close() error {
	if a == nil || a.db == nil {
		return nil
	}
	if _, err := a.db.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		return err
	}
	return a.db.Close()
}

func (a *App) registerRoutes() {
	a.mux.HandleFunc("/healthz", a.handleHealthz)
	a.mux.HandleFunc("/favicon.ico", a.handleFavicon)
	a.mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(resolveUpwardPath("frontend/dist")))))
	a.mux.HandleFunc("/mcp", a.handleMCPHTTP)
	a.mux.HandleFunc("/v1/", a.handleAPI)
	a.mux.HandleFunc("/ai/chat", a.handleAIChat)
	a.mux.HandleFunc("/ai/chat/stream", a.handleAIChatStream)
	a.mux.HandleFunc("/ai/clear", a.handleAIClearSession)
	a.mux.HandleFunc("/ai/status", a.handleAIStatus)
	a.mux.HandleFunc("/ui/", a.handleUI)
	a.mux.HandleFunc("/", a.handleSPA)
}

func (a *App) handleSPA(w http.ResponseWriter, r *http.Request) {
	distDir := resolveUpwardPath("frontend/dist")
	// Try to serve the requested file from dist
	filePath := filepath.Join(distDir, r.URL.Path)
	if r.URL.Path != "/" {
		if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
			http.ServeFile(w, r, filePath)
			return
		}
	}
	// Fallback: serve index.html for SPA routing
	indexPath := filepath.Join(distDir, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		// SPA not built yet, fall back to old UI
		a.handleUI(w, r)
		return
	}
	http.ServeFile(w, r, indexPath)
}

func (a *App) handleFavicon(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) migrate() error {
	if _, err := a.db.Exec(`CREATE TABLE IF NOT EXISTS schema_version (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL);`); err != nil {
		return err
	}
	applied := map[int]struct{}{}
	rows, err := a.db.Query(`SELECT version FROM schema_version`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return err
		}
		applied[version] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	entries, err := os.ReadDir(resolveUpwardPath("migrations"))
	if err != nil {
		return err
	}
	type migrationFile struct {
		version int
		name    string
	}
	files := make([]migrationFile, 0, len(entries))
	seenVersions := map[int]string{}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			parts := strings.SplitN(entry.Name(), "_", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid migration filename: %s", entry.Name())
			}
			version, err := strconv.Atoi(parts[0])
			if err != nil || version <= 0 {
				return fmt.Errorf("invalid migration version in %s", entry.Name())
			}
			if prev, ok := seenVersions[version]; ok {
				return fmt.Errorf("duplicate migration version %03d: %s and %s", version, prev, entry.Name())
			}
			seenVersions[version] = entry.Name()
			files = append(files, migrationFile{version: version, name: entry.Name()})
		}
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].version == files[j].version {
			return files[i].name < files[j].name
		}
		return files[i].version < files[j].version
	})
	for _, file := range files {
		if _, ok := applied[file.version]; ok {
			continue
		}
		content, err := os.ReadFile(filepath.Join(resolveUpwardPath("migrations"), file.name))
		if err != nil {
			return err
		}
		if _, err := a.db.Exec(string(content)); err != nil {
			return err
		}
		if _, err := a.db.Exec(`INSERT INTO schema_version(version, applied_at) VALUES (?, ?)`, file.version, utcNowISO()); err != nil {
			return err
		}
	}
	return nil
}

func scanRows(rows *sql.Rows) ([]jsonMap, error) {
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	items := []jsonMap{}
	for rows.Next() {
		values := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		item := jsonMap{}
		for i, col := range cols {
			item[col] = normalizeDBValue(values[i])
		}
		for key, value := range item {
			if strings.HasSuffix(key, "_json") {
				base := strings.TrimSuffix(key, "_json")
				var decoded any
				if s := asString(value); s != "" {
					if err := json.Unmarshal([]byte(s), &decoded); err != nil {
						decoded = s
					}
				}
				item[base] = decoded
				delete(item, key)
			}
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func normalizeDBValue(v any) any {
	switch t := v.(type) {
	case []byte:
		return string(t)
	default:
		return t
	}
}

func newID(prefix string) string {
	ts := time.Now().UnixMilli()
	randVal := rand.Uint64() & ((1 << 60) - 1)
	return fmt.Sprintf("%s_%s%s", prefix, encodeBase32(ts, 10), encodeBase32(int64(randVal), 12))
}

func encodeBase32(n int64, length int) string {
	out := make([]rune, length)
	for i := length - 1; i >= 0; i-- {
		out[i] = alphabet[n&0x1f]
		n >>= 5
	}
	return string(out)
}

func utcNowISO() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05.000000Z")
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	body, err := json.Marshal(data)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if written, ok := w.(interface{ Written() bool }); ok {
		if !written.Written() && status != http.StatusOK {
			w.WriteHeader(status)
		}
	} else if status != http.StatusOK {
		w.WriteHeader(status)
	}
	if _, err := w.Write(append(body, '\n')); err != nil {
		return
	}
}

func writeErr(w http.ResponseWriter, err error) {
	if appErr, ok := err.(*appError); ok {
		writeJSON(w, appErr.Code, jsonMap{"detail": appErr.Msg})
		return
	}
	writeJSON(w, http.StatusInternalServerError, jsonMap{"detail": err.Error()})
}

func decodeJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		return &appError{Code: 400, Msg: err.Error()}
	}
	return nil
}

func omitEmpty(m jsonMap) jsonMap {
	for k, v := range m {
		switch v.(type) {
		case nil:
			delete(m, k)
		case map[string]any:
			if len(v.(map[string]any)) == 0 {
				delete(m, k)
			}
		case jsonMap:
			if len(v.(jsonMap)) == 0 {
				delete(m, k)
			}
		}
	}
	return m
}

func omitEmptySlice(items []jsonMap) []jsonMap {
	for i := range items {
		omitEmpty(items[i])
	}
	return items
}

func placeholders(n int) string {
	if n <= 0 {
		return "?"
	}
	parts := make([]string, n)
	for i := range parts {
		parts[i] = "?"
	}
	return strings.Join(parts, ",")
}

func splitCSV(v string) []string {
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func parseIntDefault(v string, fallback int) int {
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func asString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case []byte:
		return string(t)
	default:
		return fmt.Sprint(t)
	}
}

func asInt(v any) int {
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	case string:
		n, _ := strconv.Atoi(t)
		return n
	default:
		return 0
	}
}

func asAnyMap(v any) map[string]any {
	switch t := v.(type) {
	case map[string]any:
		return t
	case jsonMap:
		return t
	default:
		return nil
	}
}

func asFloat(v any) float64 {
	switch t := v.(type) {
	case nil:
		return 0
	case float64:
		return t
	case int64:
		return float64(t)
	case int:
		return float64(t)
	case string:
		n, _ := strconv.ParseFloat(t, 64)
		return n
	default:
		return 0
	}
}

func mustJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Errorf("mustJSON marshal failed: %w", err))
	}
	return string(data)
}
