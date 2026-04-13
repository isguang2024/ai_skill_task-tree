package tasktree

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func (a *App) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, jsonMap{"ok": true, "server": "task-tree-go"})
}

func (a *App) handleAPI(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1")
	switch {
	case path == "/tasks" && r.Method == http.MethodPost:
		var body taskCreate
		if err := decodeJSON(r, &body); err != nil {
			writeErr(w, err)
			return
		}
		item, err := a.createTask(r.Context(), body)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case path == "/tasks" && r.Method == http.MethodGet:
		items, err := a.listTasksByProject(
			r.Context(),
			r.URL.Query().Get("status"),
			r.URL.Query().Get("q"),
			r.URL.Query().Get("project_id"),
			r.URL.Query().Get("include_deleted") == "true",
			false,
			parseIntDefault(r.URL.Query().Get("limit"), 100),
		)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, items)
	case path == "/projects" && r.Method == http.MethodGet:
		items, err := a.listProjects(r.Context(), r.URL.Query().Get("q"), r.URL.Query().Get("include_deleted") == "true", parseIntDefault(r.URL.Query().Get("limit"), 100))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, items)
	case path == "/projects" && r.Method == http.MethodPost:
		var body projectCreate
		if err := decodeJSON(r, &body); err != nil {
			writeErr(w, err)
			return
		}
		item, err := a.createProject(r.Context(), body)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasPrefix(path, "/projects/") && strings.HasSuffix(path, "/overview") && r.Method == http.MethodGet:
		item, err := a.projectOverview(r.Context(), lastSegment(strings.TrimSuffix(path, "/overview")), r.URL.Query().Get("include_deleted") == "true", parseIntDefault(r.URL.Query().Get("limit"), 100))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasPrefix(path, "/tasks/") && strings.HasSuffix(path, "/memory") && r.Method == http.MethodGet:
		item, err := a.getTaskMemory(r.Context(), taskIDFromPath(path))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasPrefix(path, "/tasks/") && strings.HasSuffix(path, "/context") && r.Method == http.MethodGet:
		item, err := a.getTaskMemory(r.Context(), taskIDFromPath(path))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasPrefix(path, "/tasks/") && strings.HasSuffix(path, "/context") && r.Method == http.MethodPatch:
		var body taskContextPatchBody
		if err := decodeJSON(r, &body); err != nil {
			writeErr(w, err)
			return
		}
		item, err := a.patchTaskContext(r.Context(), taskIDFromPath(path), body)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasPrefix(path, "/tasks/") && strings.HasSuffix(path, "/memory") && r.Method == http.MethodPatch:
		var body memoryPatchBody
		if err := decodeJSON(r, &body); err != nil {
			writeErr(w, err)
			return
		}
		item, err := a.patchTaskMemoryManualNote(r.Context(), taskIDFromPath(path), body.ManualNoteText, body.ExpectedVersion)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasPrefix(path, "/tasks/") && strings.HasSuffix(path, "/memory/snapshot") && r.Method == http.MethodPost:
		item, err := a.snapshotMemoryManually(r.Context(), "task", taskIDFromPath(path))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasPrefix(path, "/projects/") && strings.HasSuffix(path, "/tasks") && r.Method == http.MethodGet:
		projectID := lastSegment(strings.TrimSuffix(path, "/tasks"))
		items, err := a.listTasksByProject(r.Context(), r.URL.Query().Get("status"), r.URL.Query().Get("q"), projectID, r.URL.Query().Get("include_deleted") == "true", false, parseIntDefault(r.URL.Query().Get("limit"), 100))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, items)
	case strings.HasPrefix(path, "/projects/") && r.Method == http.MethodGet && !strings.Contains(strings.TrimPrefix(path, "/projects/"), "/"):
		item, err := a.getProject(r.Context(), lastSegment(path), r.URL.Query().Get("include_deleted") == "true")
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasPrefix(path, "/projects/") && r.Method == http.MethodPatch && !strings.Contains(strings.TrimPrefix(path, "/projects/"), "/"):
		var body projectUpdate
		if err := decodeJSON(r, &body); err != nil {
			writeErr(w, err)
			return
		}
		item, err := a.updateProject(r.Context(), lastSegment(path), body)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasPrefix(path, "/projects/") && r.Method == http.MethodDelete && !strings.Contains(strings.TrimPrefix(path, "/projects/"), "/"):
		if err := a.deleteProject(r.Context(), lastSegment(path)); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, jsonMap{"ok": true})
	case strings.HasPrefix(path, "/tasks/") && r.Method == http.MethodPatch && !strings.Contains(strings.TrimPrefix(path, "/tasks/"), "/"):
		var body taskUpdate
		if err := decodeJSON(r, &body); err != nil {
			writeErr(w, err)
			return
		}
		item, err := a.updateTask(r.Context(), lastSegment(path), body)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasSuffix(path, "/transition") && strings.Contains(path, "/nodes/") && r.Method == http.MethodPost:
		var body transitionBody
		if err := decodeJSON(r, &body); err != nil {
			writeErr(w, err)
			return
		}
		item, err := a.transitionNode(r.Context(), nodeIDFromPath(path), body)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasSuffix(path, "/transition") && strings.HasPrefix(path, "/tasks/") && r.Method == http.MethodPost:
		var body transitionBody
		if err := decodeJSON(r, &body); err != nil {
			writeErr(w, err)
			return
		}
		item, err := a.transitionTask(r.Context(), taskIDFromPath(path), body)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasSuffix(path, "/nodes/batch") && r.Method == http.MethodPost:
		var bodies []nodeCreate
		if err := decodeJSON(r, &bodies); err != nil {
			writeErr(w, err)
			return
		}
		items, err := a.batchCreateNodes(r.Context(), taskIDFromPath(path), bodies)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, jsonMap{"created": items, "count": len(items)})
	case strings.HasSuffix(path, "/nodes") && r.Method == http.MethodPost:
		var body nodeCreate
		if err := decodeJSON(r, &body); err != nil {
			writeErr(w, err)
			return
		}
		item, err := a.createNode(r.Context(), taskIDFromPath(path), body)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasSuffix(path, "/stages") && r.Method == http.MethodPost:
		var body stageCreate
		if err := decodeJSON(r, &body); err != nil {
			writeErr(w, err)
			return
		}
		item, err := a.createStage(r.Context(), taskIDFromPath(path), body)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasSuffix(path, "/stages/batch") && r.Method == http.MethodPost:
		var body stageBatchCreate
		if err := decodeJSON(r, &body); err != nil {
			writeErr(w, err)
			return
		}
		items, err := a.batchCreateStages(r.Context(), taskIDFromPath(path), body.Stages)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, jsonMap{"created": items, "count": len(items)})
	case strings.HasSuffix(path, "/stages") && r.Method == http.MethodGet:
		items, err := a.listStages(r.Context(), taskIDFromPath(path))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, items)
	case strings.Contains(path, "/stages/") && strings.HasSuffix(path, "/activate") && r.Method == http.MethodPost:
		var body stageActivate
		if err := decodeJSON(r, &body); err != nil {
			writeErr(w, err)
			return
		}
		item, err := a.activateStage(r.Context(), taskIDFromPath(path), stageNodeIDFromPath(path), body)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasSuffix(path, "/runs") && strings.Contains(path, "/nodes/") && r.Method == http.MethodPost:
		var body runStartBody
		if err := decodeJSON(r, &body); err != nil {
			writeErr(w, err)
			return
		}
		item, err := a.startRun(r.Context(), nodeIDFromPath(path), runStart{
			Actor:            body.Actor,
			TriggerKind:      body.TriggerKind,
			InputSummary:     body.InputSummary,
			OutputPreview:    body.OutputPreview,
			OutputRef:        body.OutputRef,
			StructuredResult: body.StructuredResult,
		})
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasSuffix(path, "/runs") && strings.Contains(path, "/nodes/") && r.Method == http.MethodGet:
		items, err := a.listNodeRuns(r.Context(), nodeIDFromPath(path), parseIntDefault(r.URL.Query().Get("limit"), 20))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, items)
	case strings.HasPrefix(path, "/runs/") && strings.HasSuffix(path, "/finish") && r.Method == http.MethodPost:
		var body runFinishBody
		if err := decodeJSON(r, &body); err != nil {
			writeErr(w, err)
			return
		}
		item, err := a.finishRun(r.Context(), runIDFromPath(path), runFinish{
			Result:           body.Result,
			Status:           body.Status,
			OutputPreview:    body.OutputPreview,
			OutputRef:        body.OutputRef,
			StructuredResult: body.StructuredResult,
			ErrorText:        body.ErrorText,
		})
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasPrefix(path, "/runs/") && strings.HasSuffix(path, "/logs") && r.Method == http.MethodPost:
		var body runLogBody
		if err := decodeJSON(r, &body); err != nil {
			writeErr(w, err)
			return
		}
		item, err := a.addRunLog(r.Context(), runIDFromPath(path), runLogCreate{
			Kind:    body.Kind,
			Content: body.Content,
			Payload: body.Payload,
		})
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasPrefix(path, "/runs/") && r.Method == http.MethodGet && !strings.Contains(strings.TrimPrefix(path, "/runs/"), "/"):
		item, err := a.getRun(r.Context(), lastSegment(path))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasPrefix(path, "/stages/") && strings.HasSuffix(path, "/memory") && r.Method == http.MethodGet:
		item, err := a.getStageMemory(r.Context(), stageNodeIDFromPath(path))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasPrefix(path, "/stages/") && strings.HasSuffix(path, "/memory") && r.Method == http.MethodPatch:
		var body memoryPatchBody
		if err := decodeJSON(r, &body); err != nil {
			writeErr(w, err)
			return
		}
		item, err := a.patchStageMemoryManualNote(r.Context(), stageNodeIDFromPath(path), body.ManualNoteText, body.ExpectedVersion)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasPrefix(path, "/stages/") && strings.HasSuffix(path, "/memory/snapshot") && r.Method == http.MethodPost:
		item, err := a.snapshotMemoryManually(r.Context(), "stage", stageNodeIDFromPath(path))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasPrefix(path, "/nodes/") && strings.HasSuffix(path, "/memory") && r.Method == http.MethodGet:
		item, err := a.getNodeMemory(r.Context(), nodeIDFromPath(path))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasPrefix(path, "/nodes/") && strings.HasSuffix(path, "/memory") && r.Method == http.MethodPatch:
		var body memoryFullPatchBody
		if err := decodeJSON(r, &body); err != nil {
			writeErr(w, err)
			return
		}
		// 兼容旧接口：如果只传了 manual_note_text（结构化字段都为nil），走原路径
		if body.SummaryText == nil && body.Conclusions == nil && body.Decisions == nil && body.Risks == nil && body.Blockers == nil && body.NextActions == nil && body.Evidence == nil {
			noteText := ""
			if body.ManualNoteText != nil {
				noteText = *body.ManualNoteText
			}
			item, err := a.patchNodeMemoryManualNote(r.Context(), nodeIDFromPath(path), noteText, body.ExpectedVersion)
			if err != nil {
				writeErr(w, err)
				return
			}
			writeJSON(w, http.StatusOK, item)
			return
		}
		item, err := a.patchNodeMemoryFull(r.Context(), nodeIDFromPath(path), body)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasPrefix(path, "/nodes/") && strings.HasSuffix(path, "/memory/snapshot") && r.Method == http.MethodPost:
		item, err := a.snapshotMemoryManually(r.Context(), "node", nodeIDFromPath(path))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasPrefix(path, "/nodes/") && strings.HasSuffix(path, "/context") && r.Method == http.MethodGet:
		item, err := a.buildNodeContext(r.Context(), nodeIDFromPath(path))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasSuffix(path, "/nodes") && r.Method == http.MethodGet:
		if !hasNodeListOptionQuery(r) {
			items, err := a.listNodes(r.Context(), taskIDFromPath(path))
			if err != nil {
				writeErr(w, err)
				return
			}
			writeJSON(w, http.StatusOK, jsonMap{"items": items, "has_more": false})
			return
		}
		items, err := a.listNodesWithOptions(r.Context(), taskIDFromPath(path), parseNodeListOptions(r))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, items)
	case strings.HasPrefix(path, "/nodes/") && r.Method == http.MethodPatch:
		var body nodeUpdate
		if err := decodeJSON(r, &body); err != nil {
			writeErr(w, err)
			return
		}
		item, err := a.updateNode(r.Context(), lastSegment(path), body)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasSuffix(path, "/reorder") && r.Method == http.MethodPost:
		var body reorderBody
		if err := decodeJSON(r, &body); err != nil {
			writeErr(w, err)
			return
		}
		items, err := a.reorderNodes(r.Context(), body.NodeIDs)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, items)
	case strings.HasSuffix(path, "/move") && r.Method == http.MethodPost:
		var body moveNodeBody
		if err := decodeJSON(r, &body); err != nil {
			writeErr(w, err)
			return
		}
		item, err := a.moveNode(r.Context(), nodeIDFromPath(path), body)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasSuffix(path, "/progress") && r.Method == http.MethodPost:
		var body progressBody
		if err := decodeJSON(r, &body); err != nil {
			writeErr(w, err)
			return
		}
		item, err := a.reportProgress(r.Context(), nodeIDFromPath(path), body)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasSuffix(path, "/complete") && r.Method == http.MethodPost:
		var body completeBody
		if err := decodeJSON(r, &body); err != nil {
			writeErr(w, err)
			return
		}
		item, err := a.completeNode(r.Context(), nodeIDFromPath(path), body)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasSuffix(path, "/block") && r.Method == http.MethodPost:
		var body blockBody
		if err := decodeJSON(r, &body); err != nil {
			writeErr(w, err)
			return
		}
		item, err := a.blockNode(r.Context(), nodeIDFromPath(path), body)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasSuffix(path, "/claim-and-start-run") && r.Method == http.MethodPost:
		var body claimStartBody
		if err := decodeJSON(r, &body); err != nil {
			writeErr(w, err)
			return
		}
		item, err := a.claimAndStartRun(r.Context(), nodeIDFromPath(path), body)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasSuffix(path, "/claim") && r.Method == http.MethodPost:
		var body claimBody
		if err := decodeJSON(r, &body); err != nil {
			writeErr(w, err)
			return
		}
		item, err := a.claimNode(r.Context(), nodeIDFromPath(path), body)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasSuffix(path, "/release") && r.Method == http.MethodPost:
		item, err := a.releaseNode(r.Context(), nodeIDFromPath(path))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasSuffix(path, "/retype") && strings.Contains(path, "/nodes/") && r.Method == http.MethodPost:
		var body retypeBody
		if err := decodeJSON(r, &body); err != nil {
			writeErr(w, err)
			return
		}
		item, err := a.retypeNodeToLeaf(r.Context(), nodeIDFromPath(path), body)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case path == "/admin/sweep-leases" && r.Method == http.MethodPost:
		cleared, err := a.sweepExpiredLeases(r.Context())
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, jsonMap{"cleared": cleared})
	case strings.HasPrefix(path, "/nodes/") && r.Method == http.MethodGet:
		item, err := a.findNode(r.Context(), lastSegment(path), false)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.Contains(path, "/nodes/") && r.Method == http.MethodGet && !strings.HasSuffix(path, "/resume-context") && !strings.HasSuffix(path, "/artifacts"):
		item, err := a.findNode(r.Context(), nodeIDFromPath(path), false)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasPrefix(path, "/tasks/") && r.Method == http.MethodGet && !strings.Contains(strings.TrimPrefix(path, "/tasks/"), "/"):
		item, err := a.getTask(r.Context(), lastSegment(path), false)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasSuffix(path, "/next-node") && r.Method == http.MethodGet:
		item, err := a.findNextNode(r.Context(), taskIDFromPath(path))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasSuffix(path, "/remaining") && r.Method == http.MethodGet:
		item, err := a.getRemaining(r.Context(), taskIDFromPath(path))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasSuffix(path, "/resume-context") && r.Method == http.MethodGet:
		item, err := a.getResumeContext(r.Context(), taskIDFromPath(path), nodeIDFromPath(path), 10)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasSuffix(path, "/resume") && r.Method == http.MethodGet:
		item, err := a.resumeTaskWithOptions(
			r.Context(),
			taskIDFromPath(path),
			parseNodeListOptions(r),
			parseEventListOptions(r),
		)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasSuffix(path, "/tree-view") && strings.HasPrefix(path, "/tasks/") && r.Method == http.MethodGet:
		stageNodeID := strings.TrimSpace(r.URL.Query().Get("stage_node_id"))
		var stageRef *string
		if stageNodeID != "" {
			stageRef = &stageNodeID
		}
		item, err := a.treeView(r.Context(), taskIDFromPath(path), stageRef, r.URL.Query().Get("only_executable") == "true")
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case path == "/import-plan" && r.Method == http.MethodPost:
		var body importPlanBody
		if err := decodeJSON(r, &body); err != nil {
			writeErr(w, err)
			return
		}
		item, err := a.importPlan(r.Context(), body)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasSuffix(path, "/wrapup") && strings.HasPrefix(path, "/tasks/") && r.Method == http.MethodGet:
		item, err := a.getWrapup(r.Context(), taskIDFromPath(path))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasSuffix(path, "/wrapup") && strings.HasPrefix(path, "/tasks/") && r.Method == http.MethodPost:
		var body struct {
			Summary string `json:"summary"`
		}
		if err := decodeJSON(r, &body); err != nil {
			writeErr(w, err)
			return
		}
		item, err := a.wrapupTask(r.Context(), taskIDFromPath(path), body.Summary)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasSuffix(path, "/events/stream") && strings.HasPrefix(path, "/tasks/") && r.Method == http.MethodGet:
		a.handleTaskEventsStream(w, r, taskIDFromPath(path))
	case path == "/work-items" && r.Method == http.MethodGet:
		items, err := a.listWorkItems(r.Context(), r.URL.Query().Get("status"), r.URL.Query().Get("include_claimed") == "true", parseIntDefault(r.URL.Query().Get("limit"), 50))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, items)
	case path == "/search" && r.Method == http.MethodGet:
		item, err := a.search(r.Context(), r.URL.Query().Get("q"), r.URL.Query().Get("kind"), parseIntDefault(r.URL.Query().Get("limit"), 30))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case path == "/smart-search" && r.Method == http.MethodGet:
		item, err := a.smartSearch(r.Context(), r.URL.Query().Get("q"), r.URL.Query().Get("scope"), r.URL.Query().Get("task_id"), parseIntDefault(r.URL.Query().Get("limit"), 20))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case path == "/admin/rebuild-index" && r.Method == http.MethodPost:
		if err := a.rebuildSearchIndex(r.Context()); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, jsonMap{"status": "ok", "message": "索引重建完成"})
	case path == "/events" && r.Method == http.MethodGet:
		eventOpts := parseEventListOptions(r)
		item, err := a.listEventsScoped(
			r.Context(),
			r.URL.Query().Get("task_id"),
			r.URL.Query().Get("node_id"),
			r.URL.Query().Get("include_descendants") == "true",
			eventOpts.Before,
			eventOpts.After,
			eventOpts.Limit,
			eventOpts,
		)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasSuffix(path, "/artifacts") && strings.Contains(path, "/nodes/") && r.Method == http.MethodGet:
		item, err := a.listArtifacts(r.Context(), taskIDFromPath(path), strPtr(nodeIDFromPath(path)))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasSuffix(path, "/artifacts") && r.Method == http.MethodGet:
		item, err := a.listArtifacts(r.Context(), taskIDFromPath(path), nil)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasSuffix(path, "/artifacts") && r.Method == http.MethodPost:
		var body artifactCreate
		if err := decodeJSON(r, &body); err != nil {
			writeErr(w, err)
			return
		}
		item, err := a.createArtifact(r.Context(), taskIDFromPath(path), body)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasSuffix(path, "/artifacts/upload") && r.Method == http.MethodPost:
		item, err := a.uploadArtifact(r.Context(), taskIDFromPath(path), r)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasPrefix(path, "/artifacts/") && strings.HasSuffix(path, "/download") && r.Method == http.MethodGet:
		artifactID := strings.TrimSuffix(strings.TrimPrefix(path, "/artifacts/"), "/download")
		data, filename, err := a.getArtifactFile(r.Context(), artifactID)
		if err != nil {
			writeErr(w, err)
			return
		}
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	case strings.HasSuffix(path, "/hard") && r.Method == http.MethodDelete:
		item, err := a.hardDeleteTask(r.Context(), taskIDFromPath(path))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasSuffix(path, "/restore") && r.Method == http.MethodPost:
		item, err := a.restoreTask(r.Context(), taskIDFromPath(path))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case path == "/admin/empty-trash" && r.Method == http.MethodPost:
		item, err := a.emptyTrash(r.Context())
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case strings.HasPrefix(path, "/tasks/") && r.Method == http.MethodDelete && !strings.HasSuffix(path, "/hard"):
		item, err := a.softDeleteTask(r.Context(), lastSegment(path))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	default:
		writeJSON(w, http.StatusNotFound, jsonMap{"detail": "not implemented yet"})
	}
}

func (a *App) handleUI(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost && r.URL.Path == "/ui/tasks/create" {
		a.handleUICreateTask(w, r)
		return
	}
	if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/ui/tasks/") && strings.HasSuffix(r.URL.Path, "/nodes/create") {
		a.handleUICreateNode(w, r, taskIDFromPath(strings.TrimPrefix(r.URL.Path, "/ui")))
		return
	}
	if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/ui/tasks/") && strings.HasSuffix(r.URL.Path, "/delete") {
		a.handleUISoftDeleteTask(w, r, taskIDFromPath(strings.TrimPrefix(r.URL.Path, "/ui")))
		return
	}
	if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/ui/tasks/") && strings.HasSuffix(r.URL.Path, "/hard-delete") {
		a.handleUIHardDeleteTask(w, r, taskIDFromPath(strings.TrimPrefix(r.URL.Path, "/ui")))
		return
	}
	if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/ui/tasks/") && strings.HasSuffix(r.URL.Path, "/save") {
		a.handleUIUpdateTask(w, r, taskIDFromPath(strings.TrimPrefix(r.URL.Path, "/ui")))
		return
	}
	if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/ui/tasks/") && strings.HasSuffix(r.URL.Path, "/transition") {
		a.handleUITaskTransition(w, r, taskIDFromPath(strings.TrimPrefix(r.URL.Path, "/ui")))
		return
	}
	if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/ui/tasks/") && strings.HasSuffix(r.URL.Path, "/restore") {
		a.handleUIRestoreTask(w, r, taskIDFromPath(strings.TrimPrefix(r.URL.Path, "/ui")))
		return
	}
	if r.Method == http.MethodPost && r.URL.Path == "/ui/trash/empty" {
		a.handleUIEmptyTrash(w, r)
		return
	}
	if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/ui/nodes/") && strings.HasSuffix(r.URL.Path, "/claim") {
		a.handleUIClaimNode(w, r, nodeIDFromPath(strings.TrimPrefix(r.URL.Path, "/ui")))
		return
	}
	if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/ui/nodes/") && strings.HasSuffix(r.URL.Path, "/release") {
		a.handleUIReleaseNode(w, r, nodeIDFromPath(strings.TrimPrefix(r.URL.Path, "/ui")))
		return
	}
	if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/ui/nodes/") && strings.HasSuffix(r.URL.Path, "/retype") {
		a.handleUIRetypeNode(w, r, nodeIDFromPath(strings.TrimPrefix(r.URL.Path, "/ui")))
		return
	}
	if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/ui/nodes/") && strings.HasSuffix(r.URL.Path, "/progress") {
		a.handleUIProgressNode(w, r, nodeIDFromPath(strings.TrimPrefix(r.URL.Path, "/ui")))
		return
	}
	if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/ui/nodes/") && strings.HasSuffix(r.URL.Path, "/complete") {
		a.handleUICompleteNode(w, r, nodeIDFromPath(strings.TrimPrefix(r.URL.Path, "/ui")))
		return
	}
	if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/ui/nodes/") && strings.HasSuffix(r.URL.Path, "/block") {
		a.handleUIBlockNode(w, r, nodeIDFromPath(strings.TrimPrefix(r.URL.Path, "/ui")))
		return
	}
	if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/ui/nodes/") && strings.HasSuffix(r.URL.Path, "/save") {
		a.handleUIUpdateNode(w, r, nodeIDFromPath(strings.TrimPrefix(r.URL.Path, "/ui")))
		return
	}
	if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/ui/nodes/") && strings.HasSuffix(r.URL.Path, "/transition") {
		a.handleUINodeTransition(w, r, nodeIDFromPath(strings.TrimPrefix(r.URL.Path, "/ui")))
		return
	}
	if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/ui/tasks/") && strings.HasSuffix(r.URL.Path, "/artifacts/create") {
		a.handleUICreateArtifact(w, r, taskIDFromPath(strings.TrimPrefix(r.URL.Path, "/ui")))
		return
	}
	if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/ui/tasks/") && strings.HasSuffix(r.URL.Path, "/artifacts/upload") {
		a.handleUIUploadArtifact(w, r, taskIDFromPath(strings.TrimPrefix(r.URL.Path, "/ui")))
		return
	}
	if r.URL.Path == "/" {
		a.renderTaskListPageV2(w, r, "任务总览", "", false)
		return
	}
	if r.URL.Path == "/new-task" {
		a.renderCreateTaskPage(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/tasks/") {
		if strings.HasSuffix(r.URL.Path, "/new-node") {
			a.renderCreateNodePage(w, r, taskIDFromPath(r.URL.Path))
			return
		}
		a.renderTaskDetailPageV2(w, r, lastSegment(r.URL.Path))
		return
	}
	if strings.HasPrefix(r.URL.Path, "/nodes/") && strings.HasSuffix(r.URL.Path, "/actions") {
		a.renderNodeActionsPage(w, r, nodeIDFromPath(r.URL.Path))
		return
	}
	if r.URL.Path == "/work" {
		a.renderWorkPageV2(w, r)
		return
	}
	if r.URL.Path == "/trash" {
		a.renderTaskListPageV2(w, r, "回收站", "", true)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/search") {
		a.renderSearchPageV2(w, r)
		return
	}
	a.renderUIV2(w, workspacePageData{
		Title:   "404 · Task Tree",
		Section: "页面不存在",
		Error:   "找不到页面：" + r.URL.Path,
	})
}

func taskIDFromPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i, part := range parts {
		if part == "tasks" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func nodeIDFromPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i, part := range parts {
		if part == "nodes" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func runIDFromPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i, part := range parts {
		if part == "runs" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func stageNodeIDFromPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i, part := range parts {
		if part == "stages" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func lastSegment(path string) string {
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}

func parseNodeListOptions(r *http.Request) nodeListOptions {
	q := r.URL.Query()
	var depth *int
	if v := q.Get("depth"); v != "" {
		n := parseIntDefault(v, -1)
		if n >= 0 {
			depth = &n
		}
	}
	var maxDepth *int
	if v := q.Get("max_depth"); v != "" {
		n := parseIntDefault(v, -1)
		if n >= 0 {
			maxDepth = &n
		}
	}
	var hasChildren *bool
	if v := strings.ToLower(strings.TrimSpace(q.Get("has_children"))); v == "true" || v == "false" {
		b := v == "true"
		hasChildren = &b
	}
	return nodeListOptions{
		Statuses:        splitCSV(q.Get("status")),
		Kinds:           splitCSV(q.Get("kind")),
		Depth:           depth,
		MaxDepth:        maxDepth,
		UpdatedAfter:    q.Get("updated_after"),
		HasChildren:     hasChildren,
		Query:           q.Get("q"),
		Limit:           parseIntDefault(q.Get("limit"), 100),
		Cursor:          q.Get("cursor"),
		SortBy:          q.Get("sort_by"),
		SortOrder:       q.Get("sort_order"),
		ViewMode:        q.Get("view_mode"),
		FilterMode:      q.Get("filter_mode"),
		IncludeFullTree: q.Get("include_full_tree") == "true",
		IncludeHidden:   q.Get("include_deleted") == "true",
	}
}

func parseEventListOptions(r *http.Request) eventListOptions {
	q := r.URL.Query()
	return eventListOptions{
		Types:       splitCSV(q.Get("type")),
		Query:       q.Get("q"),
		ViewMode:    q.Get("view_mode"),
		SortOrder:   q.Get("sort_order"),
		Limit:       parseIntDefault(q.Get("limit"), 100),
		Cursor:      q.Get("cursor"),
		Before:      q.Get("before"),
		After:       q.Get("after"),
		IncludeDesc: q.Get("include_descendants") == "true",
	}
}

func hasNodeListOptionQuery(r *http.Request) bool {
	q := r.URL.Query()
	keys := []string{
		"status", "kind", "depth", "max_depth", "updated_after",
		"has_children", "q", "limit", "cursor", "sort_by", "sort_order",
		"view_mode", "filter_mode", "include_deleted",
	}
	for _, key := range keys {
		if strings.TrimSpace(q.Get(key)) != "" {
			return true
		}
	}
	return false
}

func strPtr(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

func (a *App) handleTaskEventsStream(w http.ResponseWriter, r *http.Request, taskID string) {
	if _, err := a.getTask(r.Context(), taskID, false); err != nil {
		writeErr(w, err)
		return
	}
	nodeID := strings.TrimSpace(r.URL.Query().Get("node_id"))
	includeDesc := r.URL.Query().Get("include_descendants") == "true"
	cursor := strings.TrimSpace(r.URL.Query().Get("after"))
	if cursor == "" {
		// Start tailing from "now" to avoid replaying old events and causing reload loops.
		cursor = utcNowISO()
	}

	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, jsonMap{"detail": "streaming not supported"})
		return
	}
	_, _ = w.Write([]byte(": connected\n\n"))
	flusher.Flush()

	ticker := time.NewTicker(1200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			eventsWrap, err := a.listEventsScoped(r.Context(), taskID, nodeID, includeDesc, "", cursor, 50, eventListOptions{})
			if err != nil {
				writeSSE(w, "error", jsonMap{"detail": err.Error()})
				flusher.Flush()
				return
			}
			items := workspaceAsItems(eventsWrap["items"])
			if len(items) == 0 {
				continue
			}
			// listEventsScoped returns DESC order; emit oldest->newest to keep timeline natural.
			for i := len(items) - 1; i >= 0; i-- {
				event := items[i]
				var node jsonMap
				if nodeRef := strings.TrimSpace(asString(event["node_id"])); nodeRef != "" {
					node, _ = a.findNode(r.Context(), nodeRef, false)
				}
				payload, _ := json.Marshal(dirtyEnvelopeForEvent(event, node))
				_, _ = fmt.Fprintf(w, "id: %s\n", asString(event["id"]))
				_, _ = w.Write([]byte("event: task_event\n"))
				for _, line := range strings.Split(string(payload), "\n") {
					_, _ = fmt.Fprintf(w, "data: %s\n", line)
				}
				_, _ = w.Write([]byte("\n"))
				cursor = asString(event["created_at"])
			}
			flusher.Flush()
		}
	}
}
