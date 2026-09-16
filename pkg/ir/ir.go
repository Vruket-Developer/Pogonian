package ir

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// SchemaVersion defines the current Pogonian Universal IR schema version.
const SchemaVersion = "2.0.0"

// TargetPlatform represents an operating system or client environment target.
type TargetPlatform string

const (
	TargetWeb       TargetPlatform = "web"
	TargetAndroid   TargetPlatform = "android"
	TargetIOS       TargetPlatform = "ios"
	TargetLinux     TargetPlatform = "linux"
	TargetWindows   TargetPlatform = "windows"
	TargetMacOS     TargetPlatform = "macos"
	TargetAndroidTV TargetPlatform = "android_tv"
)

// AllTargetPlatforms lists all recognized Pogonian compilation targets.
func AllTargetPlatforms() []TargetPlatform {
	return []TargetPlatform{
		TargetWeb,
		TargetAndroid,
		TargetIOS,
		TargetLinux,
		TargetWindows,
		TargetMacOS,
		TargetAndroidTV,
	}
}

// IsValidTarget checks if a platform string is a recognized target.
func IsValidTarget(t string) bool {
	switch TargetPlatform(t) {
	case TargetWeb, TargetAndroid, TargetIOS, TargetLinux, TargetWindows, TargetMacOS, TargetAndroidTV:
		return true
	default:
		return false
	}
}

// IsNativeTarget returns true if the target renders natively without browser engines.
func (tp TargetPlatform) IsNativeTarget() bool {
	return tp != TargetWeb
}

// IsDesktopTarget returns true if the platform is a standard desktop OS.
func (tp TargetPlatform) IsDesktopTarget() bool {
	return tp == TargetLinux || tp == TargetWindows || tp == TargetMacOS
}

// IsMobileTarget returns true for phone/tablet mobile platforms.
func (tp TargetPlatform) IsMobileTarget() bool {
	return tp == TargetAndroid || tp == TargetIOS
}

// IsTVTarget returns true for 10-foot, D-pad driven TV platforms.
func (tp TargetPlatform) IsTVTarget() bool {
	return tp == TargetAndroidTV
}

// Application represents the complete Pogonian Intermediate Representation of a full-stack, cross-platform project.
type Application struct {
	SchemaVersion    string            `json:"schema_version" yaml:"schema_version"`
	Name             string            `json:"name" yaml:"name"`
	PackageID        string            `json:"package_id,omitempty" yaml:"package_id,omitempty"`
	Version          string            `json:"version" yaml:"version"`
	Framework        string            `json:"framework" yaml:"framework"`
	Architecture     string            `json:"architecture" yaml:"architecture"`
	SupportedTargets []TargetPlatform  `json:"supported_targets,omitempty" yaml:"supported_targets,omitempty"`
	Routes           []*RouteIR        `json:"routes" yaml:"routes"`
	UIComponents     []*UIComponentIR  `json:"ui_components,omitempty" yaml:"ui_components,omitempty"`
	State            *StateIR          `json:"state,omitempty" yaml:"state,omitempty"`
	Capabilities     []*CapabilityIR   `json:"capabilities,omitempty" yaml:"capabilities,omitempty"`
	Entities         []*EntityIR       `json:"entities" yaml:"entities"`
	Handlers         []*HandlerIR      `json:"handlers" yaml:"handlers"`
	Services         []*ServiceIR      `json:"services" yaml:"services"`
	Repositories     []*RepositoryIR   `json:"repositories" yaml:"repositories"`
	Database         *DatabaseIR       `json:"database,omitempty" yaml:"database,omitempty"`
	Contracts        []*ContractIR     `json:"contracts,omitempty" yaml:"contracts,omitempty"`
	Features         []*FeatureIR      `json:"features,omitempty" yaml:"features,omitempty"`
	Mounts           []*MountIR        `json:"mounts,omitempty" yaml:"mounts,omitempty"`
	Metadata         map[string]any    `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// RouteIR represents an endpoint or page in the application.
type RouteIR struct {
	Path        string         `json:"path" yaml:"path"`
	Methods     []string       `json:"methods" yaml:"methods"`
	PagePath    string         `json:"page_path,omitempty" yaml:"page_path,omitempty"`
	LayoutChain []string       `json:"layout_chain,omitempty" yaml:"layout_chain,omitempty"`
	Params      []string       `json:"params,omitempty" yaml:"params,omitempty"`
	Handler     string         `json:"handler,omitempty" yaml:"handler,omitempty"`
	UIRoot      *UIElementIR   `json:"ui_root,omitempty" yaml:"ui_root,omitempty"`
	StateScope  []string       `json:"state_scope,omitempty" yaml:"state_scope,omitempty"`
	Mount       *MountIR       `json:"mount,omitempty" yaml:"mount,omitempty"`
}

// UIElementKind defines semantic UI element primitives.
type UIElementKind string

const (
	UIElementStackVertical   UIElementKind = "StackVertical"
	UIElementStackHorizontal UIElementKind = "StackHorizontal"
	UIElementContainer       UIElementKind = "Container"
	UIElementHeading         UIElementKind = "Heading"
	UIElementText            UIElementKind = "Text"
	UIElementButton          UIElementKind = "Button"
	UIElementInput           UIElementKind = "Input"
	UIElementForm            UIElementKind = "Form"
	UIElementList            UIElementKind = "List"
	UIElementImage           UIElementKind = "Image"
	UIElementCard            UIElementKind = "Card"
	UIElementNavigation      UIElementKind = "Navigation"
	UIElementGrid            UIElementKind = "Grid"
	UIElementModal           UIElementKind = "Modal"
	UIElementSpacer          UIElementKind = "Spacer"
	UIElementDivider         UIElementKind = "Divider"
	UIElementIcon            UIElementKind = "Icon"
)

// UIPropertyIR represents a layout, styling, or accessibility attribute.
type UIPropertyIR struct {
	Key   string `json:"key" yaml:"key"`
	Value string `json:"value" yaml:"value"`
}

// UIElementIR represents a platform-independent node in a UI tree.
type UIElementIR struct {
	ID         string            `json:"id" yaml:"id"`
	Kind       UIElementKind     `json:"kind" yaml:"kind"`
	Name       string            `json:"name,omitempty" yaml:"name,omitempty"`
	Text       string            `json:"text,omitempty" yaml:"text,omitempty"`
	Binding    string            `json:"binding,omitempty" yaml:"binding,omitempty"`
	Properties map[string]string `json:"properties,omitempty" yaml:"properties,omitempty"`
	Events     []*EventIR        `json:"events,omitempty" yaml:"events,omitempty"`
	Children   []*UIElementIR    `json:"children,omitempty" yaml:"children,omitempty"`
}

// UIComponentIR represents a reusable, self-contained UI module.
type UIComponentIR struct {
	Name        string            `json:"name" yaml:"name"`
	File        string            `json:"file,omitempty" yaml:"file,omitempty"`
	Props       []FieldIR         `json:"props,omitempty" yaml:"props,omitempty"`
	StateFields []*StateFieldIR   `json:"state_fields,omitempty" yaml:"state_fields,omitempty"`
	Root        *UIElementIR      `json:"root,omitempty" yaml:"root,omitempty"`
}

// StateScope defines lifetime and visibility of application state fields.
type StateScope string

const (
	StateScopeLocal      StateScope = "local"      // ephemeral to component
	StateScopeRoute      StateScope = "route"      // scoped to active route/page
	StateScopeSession    StateScope = "session"    // per-user authenticated session
	StateScopePersistent StateScope = "persistent" // saved to local storage/db
	StateScopeSync       StateScope = "sync"       // portable across devices
)

// StateFieldIR represents a typed reactive state variable.
type StateFieldIR struct {
	Name         string     `json:"name" yaml:"name"`
	Type         string     `json:"type" yaml:"type"` // "string", "int", "bool", "list", "map", "entity:<Name>"
	InitialValue any        `json:"initial_value,omitempty" yaml:"initial_value,omitempty"`
	Scope        StateScope `json:"scope" yaml:"scope"`
	Description  string     `json:"description,omitempty" yaml:"description,omitempty"`
}

// StateTransitionIR represents valid state machine mutations.
type StateTransitionIR struct {
	ID           string `json:"id" yaml:"id"`
	FromState    string `json:"from" yaml:"from"`
	ToState      string `json:"to" yaml:"to"`
	TriggerEvent string `json:"trigger_event" yaml:"trigger_event"`
	Action       string `json:"action" yaml:"action"`
}

// StateIR encapsulates the application's overall portable state model.
type StateIR struct {
	Fields      []*StateFieldIR      `json:"fields" yaml:"fields"`
	Transitions []*StateTransitionIR `json:"transitions,omitempty" yaml:"transitions,omitempty"`
}

// EventType represents recognized UI, lifecycle, and hardware inputs.
type EventType string

const (
	EventClick      EventType = "click"
	EventChange     EventType = "change"
	EventSubmit     EventType = "submit"
	EventNavigate   EventType = "navigate"
	EventMount      EventType = "mount"
	EventUnmount    EventType = "unmount"
	EventFocus      EventType = "focus"
	EventBlur       EventType = "blur"
	EventDPadUp     EventType = "dpad_up"
	EventDPadDown   EventType = "dpad_down"
	EventDPadLeft   EventType = "dpad_left"
	EventDPadRight  EventType = "dpad_right"
	EventDPadSelect EventType = "dpad_select"
	EventBackPress  EventType = "back_press"
)

// ActionType represents an action triggered by an event.
type ActionType string

const (
	ActionMutateState       ActionType = "mutate_state"
	ActionNavigateTo        ActionType = "navigate_to"
	ActionCallService       ActionType = "call_service"
	ActionRequestCapability ActionType = "request_capability"
	ActionTransferState     ActionType = "transfer_state"
	ActionCustom            ActionType = "custom"
)

// ActionIR defines an action and its parameters.
type ActionIR struct {
	Type    ActionType     `json:"type" yaml:"type"`
	Target  string         `json:"target" yaml:"target"`
	Payload map[string]any `json:"payload,omitempty" yaml:"payload,omitempty"`
}

// EventIR binds an event type to corresponding actions.
type EventIR struct {
	Type    EventType   `json:"type" yaml:"type"`
	Handler string      `json:"handler,omitempty" yaml:"handler,omitempty"`
	Actions []*ActionIR `json:"actions,omitempty" yaml:"actions,omitempty"`
}

// CapabilityType represents hardware, system, or environment capabilities.
type CapabilityType string

const (
	CapabilityCamera         CapabilityType = "camera"
	CapabilityMicrophone     CapabilityType = "microphone"
	CapabilityFileSystem     CapabilityType = "filesystem"
	CapabilityNotifications  CapabilityType = "notifications"
	CapabilityLocation       CapabilityType = "location"
	CapabilityBluetooth      CapabilityType = "bluetooth"
	CapabilityNetwork        CapabilityType = "network"
	CapabilitySensors        CapabilityType = "sensors"
	CapabilityBiometrics     CapabilityType = "biometrics"
	CapabilityClipboard      CapabilityType = "clipboard"
	CapabilityKeyboard       CapabilityType = "keyboard"
	CapabilityMouse          CapabilityType = "mouse"
	CapabilityTouch          CapabilityType = "touch"
	CapabilityGameController CapabilityType = "game_controller"
	CapabilityTVRemote       CapabilityType = "tv_remote"
	CapabilityBackgroundExec CapabilityType = "background_execution"
)

// CapabilityIR represents a requested device capability with fallback policies.
type CapabilityIR struct {
	Type           CapabilityType `json:"type" yaml:"type"`
	Required       bool           `json:"required" yaml:"required"`
	Permissions    []string       `json:"permissions,omitempty" yaml:"permissions,omitempty"`
	FallbackAction string         `json:"fallback_action,omitempty" yaml:"fallback_action,omitempty"`
	Description    string         `json:"description,omitempty" yaml:"description,omitempty"`
}

// EntityIR represents a domain model.
type EntityIR struct {
	Name      string    `json:"name" yaml:"name"`
	Package   string    `json:"package,omitempty" yaml:"package,omitempty"`
	File      string    `json:"file,omitempty" yaml:"file,omitempty"`
	Fields    []FieldIR `json:"fields" yaml:"fields"`
	TableName string    `json:"table_name,omitempty" yaml:"table_name,omitempty"`
}

// FieldIR represents a field within an Entity or Component Prop.
type FieldIR struct {
	Name     string `json:"name" yaml:"name"`
	Type     string `json:"type" yaml:"type"`
	Required bool   `json:"required" yaml:"required"`
	PK       bool   `json:"pk,omitempty" yaml:"pk,omitempty"`
	JSONTag  string `json:"json_tag,omitempty" yaml:"json_tag,omitempty"`
}

// MethodIR represents a Go method signature.
type MethodIR struct {
	Name       string   `json:"name" yaml:"name"`
	Parameters []string `json:"parameters" yaml:"parameters"`
	Returns    []string `json:"returns" yaml:"returns"`
}

// HandlerIR represents an HTTP/Action Handler layer in the backend.
type HandlerIR struct {
	Name           string      `json:"name" yaml:"name"`
	Package        string      `json:"package" yaml:"package"`
	File           string      `json:"file" yaml:"file"`
	CallsServices  []string    `json:"calls_services" yaml:"calls_services"`
	DirectDBAccess bool        `json:"direct_db_access" yaml:"direct_db_access"`
	HandledMethods []string    `json:"handled_methods" yaml:"handled_methods"`
	Methods        []*MethodIR `json:"methods,omitempty" yaml:"methods,omitempty"`
}

// ServiceIR represents a Business Logic layer in the backend.
type ServiceIR struct {
	Name       string      `json:"name" yaml:"name"`
	Package    string      `json:"package" yaml:"package"`
	File       string      `json:"file" yaml:"file"`
	CallsRepos []string    `json:"calls_repos" yaml:"calls_repos"`
	Methods    []*MethodIR `json:"methods,omitempty" yaml:"methods,omitempty"`
}

// RepositoryIR represents a Data Access layer in the backend.
type RepositoryIR struct {
	Name        string      `json:"name" yaml:"name"`
	Package     string      `json:"package" yaml:"package"`
	File        string      `json:"file" yaml:"file"`
	TargetTable string      `json:"target_table" yaml:"target_table"`
	Methods     []*MethodIR `json:"methods,omitempty" yaml:"methods,omitempty"`
}

// DatabaseIR represents database configurations and schemas.
type DatabaseIR struct {
	Driver string    `json:"driver" yaml:"driver"`
	Tables []TableIR `json:"tables" yaml:"tables"`
}

// TableIR represents a database table schema.
type TableIR struct {
	Name    string   `json:"name" yaml:"name"`
	Columns []string `json:"columns" yaml:"columns"`
}

// ContractIR represents an architectural rule or boundary guarantee.
type ContractIR struct {
	Name        string `json:"name" yaml:"name"`
	SourceLayer string `json:"source_layer" yaml:"source_layer"`
	TargetLayer string `json:"target_layer" yaml:"target_layer"`
	Rule        string `json:"rule" yaml:"rule"` // "ALLOW", "DENY"
	Enforced    bool   `json:"enforced" yaml:"enforced"`
}

// FeatureIR represents a cohesive full-stack module.
type FeatureIR struct {
	Name         string   `json:"name" yaml:"name"`
	Route        string   `json:"route" yaml:"route"`
	Entity       string   `json:"entity" yaml:"entity"`
	Handler      string   `json:"handler" yaml:"handler"`
	Service      string   `json:"service" yaml:"service"`
	Repository   string   `json:"repository" yaml:"repository"`
	Capabilities []string `json:"capabilities" yaml:"capabilities"`
}

// MountIR formalizes the authoritative connection binding application resources, UI routes, backend handlers, services, repositories, and data entities.
type MountIR struct {
	ID           string   `json:"id" yaml:"id"`
	Resource     string   `json:"resource,omitempty" yaml:"resource,omitempty"`
	RoutePath    string   `json:"route_path,omitempty" yaml:"route_path,omitempty"`
	Entity       string   `json:"entity,omitempty" yaml:"entity,omitempty"`
	Table        string   `json:"table,omitempty" yaml:"table,omitempty"`
	Handler      string   `json:"handler,omitempty" yaml:"handler,omitempty"`
	Service      string   `json:"service,omitempty" yaml:"service,omitempty"`
	Repository   string   `json:"repository,omitempty" yaml:"repository,omitempty"`
	PrimaryKey   string   `json:"primary_key,omitempty" yaml:"primary_key,omitempty"`
	ParamBinding string   `json:"param_binding,omitempty" yaml:"param_binding,omitempty"`
	DataLoader   string   `json:"data_loader,omitempty" yaml:"data_loader,omitempty"`
	Action       string   `json:"action,omitempty" yaml:"action,omitempty"`
	AutoWired    bool     `json:"auto_wired" yaml:"auto_wired"`
	Permissions  []string `json:"permissions,omitempty" yaml:"permissions,omitempty"`
}

// FindMountByResource retrieves a mount configuration by canonical resource name.
func (app *Application) FindMountByResource(resource string) *MountIR {
	clean := strings.ToLower(resource)
	for _, m := range app.Mounts {
		res := strings.ToLower(m.Resource)
		if res == clean || res+"s" == clean || strings.TrimSuffix(res, "s") == clean {
			return m
		}
	}
	return nil
}

// FindMountByRoute retrieves a mount configuration by route path (supporting both [id] and :id syntax).
func (app *Application) FindMountByRoute(path string) *MountIR {
	normalized := normalizeRoutePath(path)
	for _, m := range app.Mounts {
		if m.RoutePath == path || normalizeRoutePath(m.RoutePath) == normalized {
			return m
		}
	}
	return nil
}

func normalizeRoutePath(p string) string {
	parts := strings.Split(p, "/")
	for i, part := range parts {
		if strings.HasPrefix(part, "[") && strings.HasSuffix(part, "]") {
			parts[i] = ":" + strings.Trim(part, "[]")
		}
	}
	return strings.Join(parts, "/")
}

// FindMountByEntity retrieves a mount configuration by entity name.
func (app *Application) FindMountByEntity(entity string) *MountIR {
	for _, m := range app.Mounts {
		if m.Entity == entity {
			return m
		}
	}
	return nil
}

// OperationStatus defines the execution state of an active operation.
type OperationStatus string

const (
	OpStatusPending   OperationStatus = "pending"
	OpStatusRunning   OperationStatus = "running"
	OpStatusPaused    OperationStatus = "paused"
	OpStatusCompleted OperationStatus = "completed"
	OpStatusFailed    OperationStatus = "failed"
)

// ActiveOperationIR captures an in-flight operation during cross-device migration.
type ActiveOperationIR struct {
	ID               string          `json:"id" yaml:"id"`
	Type             string          `json:"type" yaml:"type"` // "api_call", "form_submission", "resumable_task", "data_sync"
	Target           string          `json:"target" yaml:"target"`
	Status           OperationStatus `json:"status" yaml:"status"`
	CurrentStep      string          `json:"current_step,omitempty" yaml:"current_step,omitempty"`
	Progress         float64         `json:"progress" yaml:"progress"` // 0.0 to 1.0
	Payload          map[string]any  `json:"payload,omitempty" yaml:"payload,omitempty"`
	Checkpoint       map[string]any  `json:"checkpoint,omitempty" yaml:"checkpoint,omitempty"`
	ExecutionContext map[string]any  `json:"execution_context,omitempty" yaml:"execution_context,omitempty"`
	Idempotency      string          `json:"idempotency,omitempty" yaml:"idempotency,omitempty"`
	CreatedAt        int64           `json:"created_at" yaml:"created_at"`
	UpdatedAt        int64           `json:"updated_at" yaml:"updated_at"`
}

// StateSnapshot captures a portable point-in-time application state for cross-device transfer.
type StateSnapshot struct {
	SnapshotID           string               `json:"snapshot_id"`
	Nonce                string               `json:"nonce,omitempty"`
	Timestamp            int64                `json:"timestamp"`
	AppVersion           string               `json:"app_version"`
	OriginPlatform       TargetPlatform       `json:"origin_platform"`
	ActiveRoute          string               `json:"active_route"`
	ActiveComponentID    string               `json:"active_component_id,omitempty"`
	SchemaChecksum       string               `json:"schema_checksum"`
	StateData            map[string]any       `json:"state_data"`
	ActiveOperations     []*ActiveOperationIR `json:"active_operations,omitempty"`
	MountedResource      *MountIR             `json:"mounted_resource,omitempty"`
	VersionClock         int64                `json:"version_clock,omitempty"`
	StateVector          map[string]int64     `json:"state_vector,omitempty"`
	RequiredCapabilities []CapabilityType     `json:"required_capabilities,omitempty"`
}

// ComputeChecksum calculates a SHA-256 hash of the snapshot payload for integrity verification.
func (s *StateSnapshot) ComputeChecksum() (string, error) {
	data, err := json.Marshal(s.StateData)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%s:%s:%s:%s:%d:", s.SnapshotID, s.Nonce, s.AppVersion, s.ActiveRoute, s.VersionClock)))
	h.Write(data)
	if len(s.ActiveOperations) > 0 {
		opData, _ := json.Marshal(s.ActiveOperations)
		h.Write(opData)
	}
	if s.MountedResource != nil {
		h.Write([]byte(s.MountedResource.ID))
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// StateEnvelope encapsulates a signed, tamper-evident state transfer payload.
type StateEnvelope struct {
	Version   string         `json:"version"`
	Signature string         `json:"signature"`
	ExpiresAt int64          `json:"expires_at"`
	Snapshot  *StateSnapshot `json:"snapshot"`
}

// SignEnvelope generates an HMAC-SHA256 signature for a state envelope.
func SignEnvelope(snapshot *StateSnapshot, secretKey string, ttl time.Duration) (*StateEnvelope, error) {
	chk, err := snapshot.ComputeChecksum()
	if err != nil {
		return nil, err
	}
	snapshot.SchemaChecksum = chk

	expiresAt := time.Now().Add(ttl).Unix()
	envelope := &StateEnvelope{
		Version:   SchemaVersion,
		ExpiresAt: expiresAt,
		Snapshot:  snapshot,
	}

	payload := fmt.Sprintf("%s:%d:%s", envelope.Version, envelope.ExpiresAt, snapshot.SchemaChecksum)
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(payload))
	envelope.Signature = hex.EncodeToString(mac.Sum(nil))

	return envelope, nil
}

// Verify verifies the HMAC signature and expiration timestamp of the state envelope.
func (e *StateEnvelope) Verify(secretKey string) error {
	if time.Now().Unix() > e.ExpiresAt {
		return fmt.Errorf("state envelope expired at timestamp %d", e.ExpiresAt)
	}

	computedChk, err := e.Snapshot.ComputeChecksum()
	if err != nil {
		return fmt.Errorf("failed to compute snapshot checksum: %w", err)
	}
	if computedChk != e.Snapshot.SchemaChecksum {
		return fmt.Errorf("snapshot data integrity mismatch: expected checksum %s, got %s", e.Snapshot.SchemaChecksum, computedChk)
	}

	payload := fmt.Sprintf("%s:%d:%s", e.Version, e.ExpiresAt, e.Snapshot.SchemaChecksum)
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(payload))
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(e.Signature), []byte(expectedSig)) {
		return fmt.Errorf("state envelope cryptographic signature verification failed")
	}

	return nil
}

// ToJSON serializes the Pogonian Universal IR application to JSON.
func (app *Application) ToJSON() ([]byte, error) {
	if app.SchemaVersion == "" {
		app.SchemaVersion = SchemaVersion
	}
	return json.MarshalIndent(app, "", "  ")
}

// ToYAML serializes the Pogonian Universal IR application to YAML.
func (app *Application) ToYAML() ([]byte, error) {
	if app.SchemaVersion == "" {
		app.SchemaVersion = SchemaVersion
	}
	return yaml.Marshal(app)
}

// FromJSON deserializes JSON data into a Pogonian Universal IR application.
func FromJSON(data []byte) (*Application, error) {
	var app Application
	if err := json.Unmarshal(data, &app); err != nil {
		return nil, err
	}
	return &app, nil
}
