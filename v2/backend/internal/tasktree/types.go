package tasktree

type actor struct {
	Tool    *string `json:"tool"`
	AgentID *string `json:"agent_id"`
}

type taskCreate struct {
	TaskKey         *string        `json:"task_key"`
	Title           string         `json:"title"`
	Goal            *string        `json:"goal"`
	ProjectID       *string        `json:"project_id"`
	ProjectKey      *string        `json:"project_key"`
	SourceTool      *string        `json:"source_tool"`
	SourceSessionID *string        `json:"source_session_id"`
	Tags            []string       `json:"tags"`
	Nodes           []taskNodeSeed `json:"nodes"`
	Stages          []stageCreate  `json:"stages"`
	DryRun          *bool          `json:"dry_run"`
	Metadata        map[string]any `json:"metadata"`
	CreatedByType   *string        `json:"created_by_type"`
	CreatedByID     *string        `json:"created_by_id"`
	CreationReason  *string        `json:"creation_reason"`
}

type taskNodeSeed struct {
	NodeKey            *string        `json:"node_key"`
	Kind               string         `json:"kind"`
	Title              string         `json:"title"`
	StageKey           *string        `json:"stage_key"`
	Instruction        *string        `json:"instruction"`
	AcceptanceCriteria []string       `json:"acceptance_criteria"`
	DependsOn          []string       `json:"depends_on"`
	DependsOnKeys      []string       `json:"depends_on_keys"`
	Estimate           *float64       `json:"estimate"`
	Status             *string        `json:"status"`
	SortOrder          *int           `json:"sort_order"`
	Metadata           map[string]any `json:"metadata"`
	CreatedByType      *string        `json:"created_by_type"`
	CreatedByID        *string        `json:"created_by_id"`
	CreationReason     *string        `json:"creation_reason"`
	Children           []taskNodeSeed `json:"children"`
}

type taskUpdate struct {
	TaskKey         *string `json:"task_key"`
	Title           *string `json:"title"`
	Goal            *string `json:"goal"`
	ProjectID       *string `json:"project_id"`
	ExpectedVersion *int    `json:"expected_version"`
}

type nodeCreate struct {
	ParentNodeID       *string        `json:"parent_node_id"`
	StageNodeID        *string        `json:"stage_node_id"`
	NodeKey            *string        `json:"node_key"`
	Kind               string         `json:"kind"`
	Role               *string        `json:"role"`
	Title              string         `json:"title"`
	Instruction        *string        `json:"instruction"`
	AcceptanceCriteria []string       `json:"acceptance_criteria"`
	DependsOn          []string       `json:"depends_on"`
	DependsOnKeys      []string       `json:"depends_on_keys"`
	Estimate           *float64       `json:"estimate"`
	Status             *string        `json:"status"`
	SortOrder          *int           `json:"sort_order"`
	Metadata           map[string]any `json:"metadata"`
	CreatedByType      *string        `json:"created_by_type"`
	CreatedByID        *string        `json:"created_by_id"`
	CreationReason     *string        `json:"creation_reason"`
}

type stageCreate struct {
	Key                *string        `json:"key"`
	NodeKey            *string        `json:"node_key"`
	Title              string         `json:"title"`
	Instruction        *string        `json:"instruction"`
	AcceptanceCriteria []string       `json:"acceptance_criteria"`
	Estimate           *float64       `json:"estimate"`
	SortOrder          *int           `json:"sort_order"`
	Metadata           map[string]any `json:"metadata"`
	Activate           *bool          `json:"activate"`
	ExpectedVersion    *int           `json:"expected_version"`
}

type stageBatchCreate struct {
	Stages []stageCreate `json:"stages"`
}

type stageActivate struct {
	ExpectedVersion *int    `json:"expected_version"`
	Message         *string `json:"message"`
	Actor           *actor  `json:"actor"`
}

type runStartBody struct {
	Actor            *actor         `json:"actor"`
	TriggerKind      *string        `json:"trigger_kind"`
	InputSummary     *string        `json:"input_summary"`
	OutputPreview    *string        `json:"output_preview"`
	OutputRef        *string        `json:"output_ref"`
	StructuredResult map[string]any `json:"structured_result"`
}

type runFinishBody struct {
	Result           *string        `json:"result"`
	Status           *string        `json:"status"`
	OutputPreview    *string        `json:"output_preview"`
	OutputRef        *string        `json:"output_ref"`
	StructuredResult map[string]any `json:"structured_result"`
	ErrorText        *string        `json:"error_text"`
}

type runLogBody struct {
	Kind    string         `json:"kind"`
	Content *string        `json:"content"`
	Payload map[string]any `json:"payload"`
}

type nodeUpdate struct {
	Title              *string   `json:"title"`
	Instruction        *string   `json:"instruction"`
	AcceptanceCriteria *[]string `json:"acceptance_criteria"`
	DependsOn          *[]string `json:"depends_on"`
	DependsOnKeys      *[]string `json:"depends_on_keys"`
	Estimate           *float64  `json:"estimate"`
	SortOrder          *int      `json:"sort_order"`
	ExpectedVersion    *int      `json:"expected_version"`
}

type reorderBody struct {
	NodeIDs []string `json:"node_ids"`
}

type moveNodeBody struct {
	AfterNodeID  *string `json:"after_node_id"`
	BeforeNodeID *string `json:"before_node_id"`
}

type progressBody struct {
	DeltaProgress   *float64 `json:"delta_progress"`
	Progress        *float64 `json:"progress"`
	Message         *string  `json:"message"`
	LogContent      *string  `json:"log_content"`
	Actor           *actor   `json:"actor"`
	IdempotencyKey  *string  `json:"idempotency_key"`
	ExpectedVersion *int     `json:"expected_version"`
}

type completeBody struct {
	Message         *string              `json:"message"`
	Actor           *actor               `json:"actor"`
	IdempotencyKey  *string              `json:"idempotency_key"`
	ExpectedVersion *int                 `json:"expected_version"`
	Memory          *memoryFullPatchBody `json:"memory"`
	ResultPayload   map[string]any       `json:"result_payload"`
}

type claimStartBody struct {
	Actor        actor          `json:"actor"`
	LeaseSeconds *int           `json:"lease_seconds"`
	InputSummary *string        `json:"input_summary"`
	TriggerKind  *string        `json:"trigger_kind"`
	Metadata     map[string]any `json:"metadata"`
}

type blockBody struct {
	Reason          string `json:"reason"`
	Actor           *actor `json:"actor"`
	ExpectedVersion *int   `json:"expected_version"`
}

type claimBody struct {
	Actor           actor `json:"actor"`
	LeaseSeconds    *int  `json:"lease_seconds"`
	ExpectedVersion *int  `json:"expected_version"`
}

type retypeBody struct {
	Message         *string `json:"message"`
	Actor           *actor  `json:"actor"`
	ExpectedVersion *int    `json:"expected_version"`
}

type artifactCreate struct {
	NodeID *string        `json:"node_id"`
	RunID  *string        `json:"run_id"`
	Kind   *string        `json:"kind"`
	Title  *string        `json:"title"`
	URI    string         `json:"uri"`
	Meta   map[string]any `json:"meta"`
}

type artifactUpload struct {
	NodeID      *string        `json:"node_id"`
	RunID       *string        `json:"run_id"`
	Kind        *string        `json:"kind"`
	Title       *string        `json:"title"`
	Filename    string         `json:"filename"`
	ContentBase string         `json:"content_base64"`
	Meta        map[string]any `json:"meta"`
}

type memoryPatchBody struct {
	ManualNoteText        string   `json:"manual_note_text"`
	ArchitectureDecisions []string `json:"architecture_decisions"`
	ReferenceFiles        []string `json:"reference_files"`
	ContextDocText        string   `json:"context_doc_text"`
	ExpectedVersion       *int     `json:"expected_version"`
}

type memoryFullPatchBody struct {
	SummaryText        *string  `json:"summary_text"`
	Conclusions        []string `json:"conclusions"`
	Decisions          []string `json:"decisions"`
	Risks              []string `json:"risks"`
	Blockers           []string `json:"blockers"`
	NextActions        []string `json:"next_actions"`
	Evidence           []string `json:"evidence"`
	ExecutionLog       *string  `json:"execution_log"`        // deprecated: ignored, auto-generated from run_logs
	AppendExecutionLog *string  `json:"append_execution_log"` // deprecated: ignored, auto-generated from run_logs
	ManualNoteText     *string  `json:"manual_note_text"`
	ExpectedVersion    *int     `json:"expected_version"`
}

type transitionBody struct {
	Action          string  `json:"action"`
	Message         *string `json:"message"`
	Actor           *actor  `json:"actor"`
	ExpectedVersion *int    `json:"expected_version"`
}

type projectCreate struct {
	ProjectKey  *string        `json:"project_key"`
	Name        string         `json:"name"`
	Description *string        `json:"description"`
	IsDefault   *bool          `json:"is_default"`
	Metadata    map[string]any `json:"metadata"`
}

type projectUpdate struct {
	ProjectKey      *string        `json:"project_key"`
	Name            *string        `json:"name"`
	Description     *string        `json:"description"`
	IsDefault       *bool          `json:"is_default"`
	Metadata        map[string]any `json:"metadata"`
	ExpectedVersion *int           `json:"expected_version"`
}

type nodeListOptions struct {
	Statuses          []string
	Kinds             []string
	Depth             *int
	MaxDepth          *int
	ParentNodeID      string
	SubtreeRootNodeID string
	MaxRelativeDepth  *int
	UpdatedAfter      string
	HasChildren       *bool
	Query             string
	Limit             int
	Cursor            string
	SortBy            string
	SortOrder         string
	ViewMode          string
	FilterMode        string
	IncludeFullTree   bool
	IncludeHidden     bool
}

type nodeContextOptions struct {
	Preset string
}

type runListOptions struct {
	Limit       int
	Cursor      string
	ViewMode    string
	IncludeLogs bool
}

type artifactListOptions struct {
	Limit    int
	Cursor   string
	ViewMode string
	Kind     string
}

type eventListOptions struct {
	Types       []string
	Query       string
	ViewMode    string
	SortOrder   string
	Limit       int
	Cursor      string
	Before      string
	After       string
	IncludeDesc bool
}

type resumeOptions struct {
	IncludeEvents          bool
	IncludeRuns            bool
	IncludeArtifacts       bool
	IncludeNextNodeContext bool
	IncludeTaskMemory      bool
	IncludeStageMemory     bool
}

type taskContextPatchBody struct {
	ArchitectureDecisions *[]string `json:"architecture_decisions"`
	ReferenceFiles        *[]string `json:"reference_files"`
	ContextDocText        *string   `json:"context_doc_text"`
	ExpectedVersion       *int      `json:"expected_version"`
}

type importPlanBody struct {
	Format string `json:"format"`
	Data   string `json:"data"`
	Apply  *bool  `json:"apply"`
}
