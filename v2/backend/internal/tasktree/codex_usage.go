package tasktree

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
)

type codexUsageSnapshot struct {
	ThreadID string
	Tokens   int
}

func codexStateDBPath() string {
	if explicit := strings.TrimSpace(os.Getenv("CODEX_STATE_DB_PATH")); explicit != "" {
		return explicit
	}
	base := strings.TrimSpace(os.Getenv("CODEX_HOME"))
	if base == "" {
		if home, err := os.UserHomeDir(); err == nil {
			base = home
		}
	}
	if base == "" {
		return ""
	}
	if filepath.Base(base) == ".codex" {
		return filepath.Join(base, "state_5.sqlite")
	}
	return filepath.Join(base, ".codex", "state_5.sqlite")
}

func (a *App) currentCodexUsageSnapshot(ctx context.Context) (codexUsageSnapshot, bool) {
	threadID := strings.TrimSpace(os.Getenv("CODEX_THREAD_ID"))
	if threadID == "" {
		return codexUsageSnapshot{}, false
	}
	path := codexStateDBPath()
	if path == "" {
		return codexUsageSnapshot{}, false
	}
	if _, err := os.Stat(path); err != nil {
		return codexUsageSnapshot{}, false
	}
	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(1000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return codexUsageSnapshot{}, false
	}
	defer db.Close()
	var tokens sql.NullInt64
	if err := db.QueryRowContext(ctx, `SELECT tokens_used FROM threads WHERE id = ?`, threadID).Scan(&tokens); err != nil {
		return codexUsageSnapshot{}, false
	}
	if !tokens.Valid {
		return codexUsageSnapshot{}, false
	}
	return codexUsageSnapshot{
		ThreadID: threadID,
		Tokens:   int(tokens.Int64),
	}, true
}

func resolveRunUsageTokens(ctx context.Context, run jsonMap, explicit *int, app *App) *int {
	if explicit != nil {
		value := *explicit
		return &value
	}
	startThreadID := strings.TrimSpace(asString(run["usage_thread_id"]))
	if startThreadID == "" {
		return nil
	}
	startTokens := asInt(run["usage_start_tokens"])
	snapshot, ok := app.currentCodexUsageSnapshot(ctx)
	if !ok || snapshot.ThreadID != startThreadID {
		return nil
	}
	usage := snapshot.Tokens - startTokens
	if usage < 0 {
		usage = 0
	}
	return &usage
}
