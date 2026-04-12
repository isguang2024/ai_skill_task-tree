param(
  [string]$RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
)

$ErrorActionPreference = 'Stop'

$routesPath = Join-Path $RepoRoot 'internal/tasktree/routes.go'
$mcpPath = Join-Path $RepoRoot 'internal/tasktree/mcp.go'

if (-not (Test-Path $routesPath)) { throw "routes.go not found: $routesPath" }
if (-not (Test-Path $mcpPath)) { throw "mcp.go not found: $mcpPath" }

$routesText = Get-Content -Encoding UTF8 $routesPath -Raw
$mcpText = Get-Content -Encoding UTF8 $mcpPath -Raw

$toolNames = [regex]::Matches($mcpText, 'Name:\s+"([^"]+)"') | ForEach-Object { $_.Groups[1].Value } | Sort-Object -Unique

$checks = @(
  @{ Http = '/v1/tasks'; Method = 'GET'; Tool = 'task_tree_list_tasks'; RouteHint = 'path == "/tasks" && r.Method == http.MethodGet' }
  @{ Http = '/v1/tasks'; Method = 'POST'; Tool = 'task_tree_create_task'; RouteHint = 'path == "/tasks" && r.Method == http.MethodPost' }
  @{ Http = '/v1/tasks/{id}'; Method = 'GET'; Tool = 'task_tree_get_task'; RouteHint = 'strings.HasPrefix(path, "/tasks/") && r.Method == http.MethodGet && !strings.Contains(strings.TrimPrefix(path, "/tasks/"), "/")' }
  @{ Http = '/v1/tasks/{id}'; Method = 'PATCH'; Tool = 'task_tree_update_task'; RouteHint = 'strings.HasPrefix(path, "/tasks/") && r.Method == http.MethodPatch && !strings.Contains(strings.TrimPrefix(path, "/tasks/"), "/")' }
  @{ Http = '/v1/tasks/{id}/transition'; Method = 'POST'; Tool = 'task_tree_transition_task'; RouteHint = 'strings.HasSuffix(path, "/transition") && strings.HasPrefix(path, "/tasks/") && r.Method == http.MethodPost' }
  @{ Http = '/v1/tasks/{id}/nodes'; Method = 'GET'; Tool = 'task_tree_list_nodes'; RouteHint = 'strings.HasSuffix(path, "/nodes") && r.Method == http.MethodGet' }
  @{ Http = '/v1/tasks/{id}/nodes'; Method = 'POST'; Tool = 'task_tree_create_node'; RouteHint = 'strings.HasSuffix(path, "/nodes") && r.Method == http.MethodPost' }
  @{ Http = '/v1/nodes/{id}'; Method = 'GET'; Tool = 'task_tree_get_node'; RouteHint = 'strings.HasPrefix(path, "/nodes/") && r.Method == http.MethodGet' }
  @{ Http = '/v1/nodes/{id}'; Method = 'PATCH'; Tool = 'task_tree_update_node'; RouteHint = 'strings.HasPrefix(path, "/nodes/") && r.Method == http.MethodPatch' }
  @{ Http = '/v1/nodes/{id}/progress'; Method = 'POST'; Tool = 'task_tree_progress'; RouteHint = 'strings.HasSuffix(path, "/progress") && r.Method == http.MethodPost' }
  @{ Http = '/v1/nodes/{id}/complete'; Method = 'POST'; Tool = 'task_tree_complete'; RouteHint = 'strings.HasSuffix(path, "/complete") && r.Method == http.MethodPost' }
  @{ Http = '/v1/nodes/{id}/block'; Method = 'POST'; Tool = 'task_tree_block_node'; RouteHint = 'strings.HasSuffix(path, "/block") && r.Method == http.MethodPost' }
  @{ Http = '/v1/nodes/{id}/claim'; Method = 'POST'; Tool = 'task_tree_claim'; RouteHint = 'strings.HasSuffix(path, "/claim") && r.Method == http.MethodPost' }
  @{ Http = '/v1/nodes/{id}/release'; Method = 'POST'; Tool = 'task_tree_release'; RouteHint = 'strings.HasSuffix(path, "/release") && r.Method == http.MethodPost' }
  @{ Http = '/v1/nodes/{id}/retype'; Method = 'POST'; Tool = 'task_tree_retype_node'; RouteHint = 'strings.HasSuffix(path, "/retype") && strings.Contains(path, "/nodes/") && r.Method == http.MethodPost' }
  @{ Http = '/v1/tasks/{id}/remaining'; Method = 'GET'; Tool = 'task_tree_get_remaining'; RouteHint = 'strings.HasSuffix(path, "/remaining") && r.Method == http.MethodGet' }
  @{ Http = '/v1/tasks/{id}/resume-context'; Method = 'GET'; Tool = 'task_tree_get_resume_context'; RouteHint = 'strings.HasSuffix(path, "/resume-context") && r.Method == http.MethodGet' }
  @{ Http = '/v1/tasks/{id}/resume'; Method = 'GET'; Tool = 'task_tree_resume'; RouteHint = 'strings.HasSuffix(path, "/resume") && r.Method == http.MethodGet' }
  @{ Http = '/v1/events'; Method = 'GET'; Tool = 'task_tree_list_events'; RouteHint = 'path == "/events" && r.Method == http.MethodGet' }
  @{ Http = '/v1/tasks/{id}/artifacts'; Method = 'GET'; Tool = 'task_tree_list_artifacts'; RouteHint = 'strings.HasSuffix(path, "/artifacts") && r.Method == http.MethodGet' }
  @{ Http = '/v1/tasks/{id}/artifacts'; Method = 'POST'; Tool = 'task_tree_create_artifact'; RouteHint = 'strings.HasSuffix(path, "/artifacts") && r.Method == http.MethodPost' }
  @{ Http = '/v1/tasks/{id}/artifacts/upload'; Method = 'POST'; Tool = 'task_tree_upload_artifact'; RouteHint = 'strings.HasSuffix(path, "/artifacts/upload") && r.Method == http.MethodPost' }
  @{ Http = '/v1/tasks/{id}'; Method = 'DELETE'; Tool = 'task_tree_delete_task'; RouteHint = 'strings.HasPrefix(path, "/tasks/") && r.Method == http.MethodDelete && !strings.HasSuffix(path, "/hard")' }
  @{ Http = '/v1/tasks/{id}/restore'; Method = 'POST'; Tool = 'task_tree_restore_task'; RouteHint = 'strings.HasSuffix(path, "/restore") && r.Method == http.MethodPost' }
  @{ Http = '/v1/tasks/{id}/hard'; Method = 'DELETE'; Tool = 'task_tree_hard_delete_task'; RouteHint = 'strings.HasSuffix(path, "/hard") && r.Method == http.MethodDelete' }
  @{ Http = '/v1/admin/empty-trash'; Method = 'POST'; Tool = 'task_tree_empty_trash'; RouteHint = 'path == "/admin/empty-trash" && r.Method == http.MethodPost' }
  @{ Http = '/v1/admin/sweep-leases'; Method = 'POST'; Tool = 'task_tree_sweep_leases'; RouteHint = 'path == "/admin/sweep-leases" && r.Method == http.MethodPost' }
  @{ Http = '/v1/search'; Method = 'GET'; Tool = 'task_tree_search'; RouteHint = 'path == "/search" && r.Method == http.MethodGet' }
  @{ Http = '/v1/work-items'; Method = 'GET'; Tool = 'task_tree_work_items'; RouteHint = 'path == "/work-items" && r.Method == http.MethodGet' }
  @{ Http = '/v1/projects'; Method = 'GET'; Tool = 'task_tree_list_projects'; RouteHint = 'path == "/projects" && r.Method == http.MethodGet' }
  @{ Http = '/v1/projects'; Method = 'POST'; Tool = 'task_tree_create_project'; RouteHint = 'path == "/projects" && r.Method == http.MethodPost' }
  @{ Http = '/v1/projects/{id}'; Method = 'GET'; Tool = 'task_tree_get_project'; RouteHint = 'strings.HasPrefix(path, "/projects/") && r.Method == http.MethodGet && !strings.Contains(strings.TrimPrefix(path, "/projects/"), "/")' }
  @{ Http = '/v1/projects/{id}'; Method = 'PATCH'; Tool = 'task_tree_update_project'; RouteHint = 'strings.HasPrefix(path, "/projects/") && r.Method == http.MethodPatch && !strings.Contains(strings.TrimPrefix(path, "/projects/"), "/")' }
  @{ Http = '/v1/projects/{id}'; Method = 'DELETE'; Tool = 'task_tree_delete_project'; RouteHint = 'strings.HasPrefix(path, "/projects/") && r.Method == http.MethodDelete && !strings.Contains(strings.TrimPrefix(path, "/projects/"), "/")' }
  @{ Http = '/v1/projects/{id}/overview'; Method = 'GET'; Tool = 'task_tree_project_overview'; RouteHint = 'strings.HasPrefix(path, "/projects/") && strings.HasSuffix(path, "/overview") && r.Method == http.MethodGet' }
  @{ Http = '/v1/projects/{id}/tasks'; Method = 'GET'; Tool = 'task_tree_list_tasks'; RouteHint = 'strings.HasPrefix(path, "/projects/") && strings.HasSuffix(path, "/tasks") && r.Method == http.MethodGet' }
)

$missing = @()
foreach ($check in $checks) {
  if ($toolNames -notcontains $check.Tool) {
    $missing += "MCP tool missing: $($check.Tool) for $($check.Http) [$($check.Method)]"
  }
  if ($routesText -notmatch [regex]::Escape($check.RouteHint)) {
    $missing += "HTTP route hint missing in routes.go: $($check.Http) [$($check.Method)] -> $($check.RouteHint)"
  }
}

if ($missing.Count -gt 0) {
  $missing | ForEach-Object { Write-Host $_ -ForegroundColor Red }
  exit 1
}

Write-Host "MCP parity check passed for $($checks.Count) business endpoints."
