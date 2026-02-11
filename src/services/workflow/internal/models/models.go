package models

// Process represents a workflow process definition.
// It corresponds to the 'Process' schema in the OpenAPI specification.
type Process struct {
	ID          string      `json:"id,omitempty" db:"id" gorm:"column:id;primaryKey;type:uuid;default:uuid_generate_v4()"`
	TenantID    string      `json:"-" db:"tenant_id" gorm:"column:tenant_id;not null"` // Never serialize tenantId in JSON - comes from header only
	Name        string      `json:"name" db:"name" gorm:"column:name;not null"`
	Code        string      `json:"code" db:"code" gorm:"column:code;not null"`
	Description *string     `json:"description,omitempty" db:"description" gorm:"column:description"`
	Version     *string     `json:"version,omitempty" db:"version" gorm:"column:version"`
	SLA         *int64      `json:"sla,omitempty" db:"sla" gorm:"column:sla"`
	AuditDetail AuditDetail `json:"auditDetail,omitempty" db:",inline" gorm:"embedded"`
}

// TableName specifies the table name for Process
func (Process) TableName() string {
	return "processes"
}

// State represents a state within a workflow process.
// It corresponds to the 'State' schema in the OpenAPI specification.
type State struct {
	ID           string      `json:"id,omitempty" db:"id" gorm:"column:id;primaryKey;type:uuid;default:uuid_generate_v4()"`
	TenantID     string      `json:"-" db:"tenant_id" gorm:"column:tenant_id;not null"` // Never serialize tenantId in JSON - comes from header only
	ProcessID    string      `json:"processId" db:"process_id" gorm:"column:process_id;not null;type:uuid"`
	Code         string      `json:"code" db:"code" gorm:"column:code;not null"`
	Name         string      `json:"name" db:"name" gorm:"column:name;not null"`
	Description  *string     `json:"description,omitempty" db:"description" gorm:"column:description"`
	SLA          *int64      `json:"sla,omitempty" db:"sla" gorm:"column:sla"`
	IsInitial    bool        `json:"isInitial,omitempty" db:"is_initial" gorm:"column:is_initial;default:false"`
	IsParallel   bool        `json:"isParallel,omitempty" db:"is_parallel" gorm:"column:is_parallel;default:false"`
	IsJoin       bool        `json:"isJoin,omitempty" db:"is_join" gorm:"column:is_join;default:false"`
	BranchStates []string    `json:"branchStates,omitempty" db:"branch_states" gorm:"column:branch_states;type:jsonb;serializer:json"`
	AuditDetail  AuditDetail `json:"auditDetail,omitempty" db:",inline" gorm:"embedded"`
}

// TableName specifies the table name for State
func (State) TableName() string {
	return "states"
}

// Action represents a transition between two states in a process.
// It corresponds to the 'Action' schema in the OpenAPI specification.
type Action struct {
	ID                    string               `json:"id,omitempty" db:"id" gorm:"column:id;primaryKey;type:uuid;default:uuid_generate_v4()"`
	TenantID              string               `json:"-" db:"tenant_id" gorm:"column:tenant_id;not null"` // Never serialize tenantId in JSON - comes from header only
	Name                  string               `json:"name" db:"name" gorm:"column:name;not null"`
	Label                 *string              `json:"label,omitempty" db:"label" gorm:"column:label"`
	CurrentState          string               `json:"currentState" db:"current_state_id" gorm:"column:current_state_id;not null;type:uuid"`
	NextState             string               `json:"nextState" db:"next_state_id" gorm:"column:next_state_id;not null;type:uuid"`
	AttributeValidationID *string              `json:"-" db:"attribute_validation_id" gorm:"column:attribute_validation_id;type:uuid"` // Internal DB relation
	AttributeValidation   *AttributeValidation `json:"attributeValidation,omitempty" db:"-" gorm:"-"`
	AuditDetail           AuditDetail          `json:"auditDetail,omitempty" db:",inline" gorm:"embedded"`
}

// TableName specifies the table name for Action
func (Action) TableName() string {
	return "actions"
}

// AttributeValidation defines guard conditions for an action.
// It corresponds to the 'AttributeValidation' schema in the OpenAPI specification.
type AttributeValidation struct {
	ID            string              `json:"id,omitempty" db:"id" gorm:"column:id;primaryKey;type:uuid;default:uuid_generate_v4()"`
	TenantID      string              `json:"-" db:"tenant_id" gorm:"column:tenant_id;not null"` // Never serialize tenantId in JSON - comes from header only
	Attributes    map[string][]string `json:"attributes,omitempty" db:"attributes" gorm:"column:attributes;type:jsonb;serializer:json"`
	AssigneeCheck bool                `json:"assigneeCheck,omitempty" db:"assignee_check" gorm:"column:assignee_check;default:false"`
	AuditDetail   AuditDetail         `json:"auditDetail,omitempty" db:",inline" gorm:"embedded"`
}

// TableName specifies the table name for AttributeValidation
func (AttributeValidation) TableName() string {
	return "attribute_validations"
}

// ProcessInstance represents a running instance of a process for a specific entity.
// It corresponds to the 'ProcessInstance' schema in the OpenAPI specification.
type ProcessInstance struct {
	ID           string              `json:"id,omitempty" db:"id" gorm:"column:id;primaryKey;type:uuid;default:uuid_generate_v4()"`
	TenantID     string              `json:"-" db:"tenant_id" gorm:"column:tenant_id;not null"` // Never serialize tenantId in JSON - comes from header only
	ProcessID    string              `json:"processId" db:"process_id" gorm:"column:process_id;not null;type:uuid"`
	EntityID     string              `json:"entityId" db:"entity_id" gorm:"column:entity_id;not null"`
	Action       string              `json:"action,omitempty" db:"action" gorm:"column:action"`
	Status       string              `json:"status,omitempty" db:"status" gorm:"column:status;not null"`
	Comment      *string             `json:"comment,omitempty" db:"comment" gorm:"column:comment"`
	Documents    []Document          `json:"documents,omitempty" db:"documents" gorm:"column:documents;type:jsonb;serializer:json"`
	Assigner     *string             `json:"assigner,omitempty" db:"assigner" gorm:"column:assigner"`
	Assignees    []string            `json:"assignees,omitempty" db:"assignees" gorm:"column:assignees;type:jsonb;serializer:json"`
	CurrentState string              `json:"currentState" db:"current_state_id" gorm:"column:current_state_id;not null;type:uuid"`
	StateSLA     *int64              `json:"stateSla,omitempty" db:"state_sla" gorm:"column:state_sla"`
	ProcessSLA   *int64              `json:"processSla,omitempty" db:"process_sla" gorm:"column:process_sla"`
	Attributes   map[string][]string `json:"attributes,omitempty" db:"attributes" gorm:"column:attributes;type:jsonb;serializer:json"`
	NextActions  []string            `json:"nextActions,omitempty" db:"-" gorm:"-"` // Not stored in DB, computed from current state's actions
	// Parallel workflow fields
	ParentInstanceID *string `json:"parentInstanceId,omitempty" db:"parent_instance_id" gorm:"column:parent_instance_id;type:uuid"`     // For tracking parallel branches
	BranchID         *string `json:"branchId,omitempty" db:"branch_id" gorm:"column:branch_id"`                                         // Which parallel branch this represents
	IsParallelBranch bool    `json:"isParallelBranch,omitempty" db:"is_parallel_branch" gorm:"column:is_parallel_branch;default:false"` // Flag for parallel instances
	IsLatest         bool    `json:"isLatest,omitempty" db:"is_latest" gorm:"column:is_latest;default:false"`

	// Auto-escalation tracking field (following Java service pattern)
	Escalated bool `json:"escalated" db:"escalated" gorm:"column:escalated;default:false"`

	AuditDetails AuditDetail `json:"auditDetails,omitempty" db:",inline" gorm:"embedded"`
}

// TableName specifies the table name for ProcessInstance
func (ProcessInstance) TableName() string {
	return "process_instances"
}

// ProcessInstanceSearchCriteria captures optional filters for querying process instances.
type ProcessInstanceSearchCriteria struct {
	EntityID       *string `json:"-"`
	ProcessID      *string `json:"-"`
	CurrentStateID *string `json:"-"`
	AssigneeID     *string `json:"-"`
	History        bool    `json:"-"`
}

// ParallelExecution tracks the coordination of parallel workflow branches
type ParallelExecution struct {
	ID                string      `json:"id,omitempty" db:"id" gorm:"column:id;primaryKey;type:uuid;default:uuid_generate_v4()"`
	TenantID          string      `json:"-" db:"tenant_id" gorm:"column:tenant_id;not null"`
	EntityID          string      `json:"entityId" db:"entity_id" gorm:"column:entity_id;not null"`
	ProcessID         string      `json:"processId" db:"process_id" gorm:"column:process_id;not null;type:uuid"`
	ParallelStateID   string      `json:"parallelStateId" db:"parallel_state_id" gorm:"column:parallel_state_id;not null;type:uuid"`                          // State that created branches
	JoinStateID       string      `json:"joinStateId" db:"join_state_id" gorm:"column:join_state_id;not null;type:uuid"`                                      // State where branches merge
	ActiveBranches    []string    `json:"activeBranches" db:"active_branches" gorm:"column:active_branches;type:jsonb;serializer:json;default:'[]'"`          // Which branches are still running
	CompletedBranches []string    `json:"completedBranches" db:"completed_branches" gorm:"column:completed_branches;type:jsonb;serializer:json;default:'[]'"` // Which branches reached join
	Status            string      `json:"status" db:"status" gorm:"column:status;default:'ACTIVE'"`                                                           // ACTIVE, WAITING_FOR_JOIN, COMPLETED
	AuditDetail       AuditDetail `json:"auditDetail,omitempty" db:",inline" gorm:"embedded"`
}

// TableName specifies the table name for ParallelExecution
func (ParallelExecution) TableName() string {
	return "parallel_executions"
}

// EscalationConfig represents an auto-escalation rule for a process state.
// It defines when and how to automatically escalate workflow instances based on SLA breaches.
type EscalationConfig struct {
	ID                string      `json:"id,omitempty" db:"id" gorm:"column:id;primaryKey;type:uuid;default:uuid_generate_v4()"`
	TenantID          string      `json:"-" db:"tenant_id" gorm:"column:tenant_id;not null"` // Never serialize tenantId in JSON - comes from header only
	ProcessID         string      `json:"processId" db:"process_id" gorm:"column:process_id;not null;type:uuid"`
	StateCode         string      `json:"stateCode" db:"state_code" gorm:"column:state_code;not null"`
	EscalationAction  string      `json:"escalationAction" db:"escalation_action" gorm:"column:escalation_action;not null"`
	StateSlaMinutes   *int        `json:"stateSlaMinutes,omitempty" db:"state_sla_minutes" gorm:"column:state_sla_minutes"`
	ProcessSlaMinutes *int        `json:"processSlaMinutes,omitempty" db:"process_sla_minutes" gorm:"column:process_sla_minutes"`
	AuditDetail       AuditDetail `json:"auditDetail,omitempty" db:",inline" gorm:"embedded"`
}

// TableName specifies the table name for EscalationConfig
func (EscalationConfig) TableName() string {
	return "escalation_configs"
}

// EscalationResult represents the result of an auto-escalation operation.
type EscalationResult struct {
	TotalFound         int                `json:"totalFound"`
	TotalEscalated     int                `json:"totalEscalated"`
	EscalatedInstances []*ProcessInstance `json:"escalatedInstances,omitempty"`
	Errors             []string           `json:"errors,omitempty"`
}

// StateDetail is a response model that includes a state's information along with its possible actions.
type StateDetail struct {
	State
	Actions []Action `json:"actions,omitempty"`
}

// ProcessDefinitionDetail is a response model for the full definition of a process, including all states and actions.
type ProcessDefinitionDetail struct {
	Process
	States []StateDetail `json:"states,omitempty"`
}
