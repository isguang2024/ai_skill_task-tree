package tasktree

import (
	"context"
	"fmt"
	"strings"
)

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, raw := range values {
		v := strings.TrimSpace(raw)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func (a *App) resolveNodeDependencies(ctx context.Context, taskID string, dependsOnIDs, dependsOnKeys []string) ([]string, error) {
	resolved := uniqueStrings(dependsOnIDs)
	keys := uniqueStrings(dependsOnKeys)
	if len(keys) == 0 {
		return resolved, nil
	}
	for _, key := range keys {
		rows, err := a.queryContext(ctx, `SELECT id, path FROM nodes WHERE task_id = ? AND node_key = ? AND deleted_at IS NULL ORDER BY path`, taskID, key)
		if err != nil {
			return nil, err
		}
		items, err := scanRows(rows)
		if err != nil {
			return nil, err
		}
		switch len(items) {
		case 0:
			return nil, &appError{Code: 400, Msg: fmt.Sprintf("depends_on_keys 中的 %q 未匹配到节点", key)}
		case 1:
			resolved = append(resolved, asString(items[0]["id"]))
		default:
			paths := make([]string, 0, len(items))
			for _, item := range items {
				paths = append(paths, asString(item["path"]))
			}
			return nil, &appError{Code: 400, Msg: fmt.Sprintf("depends_on_keys 中的 %q 匹配到多个节点：%s", key, strings.Join(paths, ", "))}
		}
	}
	return uniqueStrings(resolved), nil
}
