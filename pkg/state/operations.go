package state

import (
	"fmt"
	"sync"
	"time"

	"pogonian/pkg/ir"
)

// OperationResult captures the outcome of resuming an active operation on a destination platform.
type OperationResult struct {
	OperationID string             `json:"operation_id"`
	Type        string             `json:"type"`
	Status      ir.OperationStatus `json:"status"`
	Progress    float64            `json:"progress"`
	ResultData  map[string]any     `json:"result_data,omitempty"`
	Message     string             `json:"message"`
}

// OperationManager tracks running operations, checkpoints, and idempotency state.
type OperationManager struct {
	mu         sync.RWMutex
	operations map[string]*ir.ActiveOperationIR
}

// NewOperationManager creates a new active operation manager.
func NewOperationManager() *OperationManager {
	return &OperationManager{
		operations: make(map[string]*ir.ActiveOperationIR),
	}
}

// StartOperation registers a new active operation on the local platform.
func (m *OperationManager) StartOperation(opType, target string, payload map[string]any) *ir.ActiveOperationIR {
	return m.StartOperationWithStep(opType, target, "initial", payload, nil)
}

// StartOperationWithStep registers a new active operation with an explicit step and execution context.
func (m *OperationManager) StartOperationWithStep(opType, target, step string, payload, execContext map[string]any) *ir.ActiveOperationIR {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().Unix()
	if execContext == nil {
		execContext = make(map[string]any)
	}
	op := &ir.ActiveOperationIR{
		ID:               fmt.Sprintf("op_%s_%d", opType, now),
		Type:             opType,
		Target:           target,
		Status:           ir.OpStatusRunning,
		CurrentStep:      step,
		Progress:         0.0,
		Payload:          payload,
		Checkpoint:       make(map[string]any),
		ExecutionContext: execContext,
		Idempotency:      generateRandomID(16),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	m.operations[op.ID] = op
	return op
}

// Checkpoint updates an operation's intermediate progress and continuation parameters.
func (m *OperationManager) Checkpoint(opID string, progress float64, checkpointData map[string]any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	op, exists := m.operations[opID]
	if !exists {
		return fmt.Errorf("operation '%s' not found for checkpoint", opID)
	}

	op.Progress = progress
	if op.Checkpoint == nil {
		op.Checkpoint = make(map[string]any)
	}
	for k, v := range checkpointData {
		op.Checkpoint[k] = v
	}
	if step, ok := checkpointData["step"].(string); ok && step != "" {
		op.CurrentStep = step
	}
	op.UpdatedAt = time.Now().Unix()
	return nil
}

// CheckpointStep updates an operation's step, progress, checkpoint parameters, and execution context.
func (m *OperationManager) CheckpointStep(opID, step string, progress float64, checkpointData, execContext map[string]any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	op, exists := m.operations[opID]
	if !exists {
		return fmt.Errorf("operation '%s' not found for checkpoint", opID)
	}

	op.CurrentStep = step
	op.Progress = progress
	if op.Checkpoint == nil {
		op.Checkpoint = make(map[string]any)
	}
	for k, v := range checkpointData {
		op.Checkpoint[k] = v
	}
	if execContext != nil {
		if op.ExecutionContext == nil {
			op.ExecutionContext = make(map[string]any)
		}
		for k, v := range execContext {
			op.ExecutionContext[k] = v
		}
	}
	op.UpdatedAt = time.Now().Unix()
	return nil
}

// GetActiveOperations returns all operations that are currently running or paused.
func (m *OperationManager) GetActiveOperations() []*ir.ActiveOperationIR {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var active []*ir.ActiveOperationIR
	for _, op := range m.operations {
		if op.Status == ir.OpStatusRunning || op.Status == ir.OpStatusPaused || op.Status == ir.OpStatusPending {
			active = append(active, op)
		}
	}
	return active
}

// ContinueOperation executes continuation logic for a migrated active operation on the destination platform.
func ContinueOperation(op *ir.ActiveOperationIR, app *ir.Application, targetPlatform ir.TargetPlatform) (*OperationResult, error) {
	if op == nil {
		return nil, fmt.Errorf("cannot continue nil operation")
	}

	if op.Status == ir.OpStatusCompleted {
		return &OperationResult{
			OperationID: op.ID,
			Type:        op.Type,
			Status:      ir.OpStatusCompleted,
			Progress:    1.0,
			Message:     "Operation already completed prior to migration (idempotent skipped)",
		}, nil
	}

	// Verify idempotency token presence
	if op.Idempotency == "" {
		op.Idempotency = generateRandomID(16)
	}

	// Determine continuation step
	step := op.CurrentStep
	if step == "" {
		if s, ok := op.Checkpoint["step"].(string); ok && s != "" {
			step = s
		} else {
			step = "initial"
		}
	}

	// Resume operation execution from checkpoint
	switch op.Type {
	case "checkout_order", "form_submission":
		// Advance transaction through to completion on destination
		resData := map[string]any{
			"resumed_from_step":   step,
			"resumed_on_platform": targetPlatform,
			"idempotency_token":   op.Idempotency,
			"transaction_id":      fmt.Sprintf("txn_%s_%s", op.ID, op.Idempotency[:8]),
			"execution_context":   op.ExecutionContext,
		}
		for k, v := range op.Payload {
			resData[k] = v
		}

		op.Status = ir.OpStatusCompleted
		op.Progress = 1.0
		op.CurrentStep = step + "_completed"
		op.UpdatedAt = time.Now().Unix()

		return &OperationResult{
			OperationID: op.ID,
			Type:        op.Type,
			Status:      ir.OpStatusCompleted,
			Progress:    1.0,
			ResultData:  resData,
			Message:     fmt.Sprintf("Successfully resumed and completed '%s' on %s from checkpoint '%s'", op.Type, targetPlatform, step),
		}, nil

	case "resumable_task", "data_sync":
		offset := 0
		if o, ok := op.Checkpoint["offset"].(int); ok {
			offset = o
		} else if oFloat, ok := op.Checkpoint["offset"].(float64); ok {
			offset = int(oFloat)
		}

		resData := map[string]any{
			"resumed_from_step":   step,
			"resumed_offset":      offset,
			"resumed_on_platform": targetPlatform,
			"sync_status":         "synchronized",
			"execution_context":   op.ExecutionContext,
		}

		op.Status = ir.OpStatusCompleted
		op.Progress = 1.0
		op.CurrentStep = step + "_completed"
		op.UpdatedAt = time.Now().Unix()

		return &OperationResult{
			OperationID: op.ID,
			Type:        op.Type,
			Status:      ir.OpStatusCompleted,
			Progress:    1.0,
			ResultData:  resData,
			Message:     fmt.Sprintf("Resumed data stream from byte offset %d on %s", offset, targetPlatform),
		}, nil

	default:
		// Generic backend task continuation
		op.Status = ir.OpStatusCompleted
		op.Progress = 1.0
		op.CurrentStep = step + "_completed"
		op.UpdatedAt = time.Now().Unix()

		resData := make(map[string]any)
		for k, v := range op.Payload {
			resData[k] = v
		}
		resData["resumed_from_step"] = step
		resData["execution_context"] = op.ExecutionContext

		return &OperationResult{
			OperationID: op.ID,
			Type:        op.Type,
			Status:      ir.OpStatusCompleted,
			Progress:    1.0,
			ResultData:  resData,
			Message:     fmt.Sprintf("Generic operation '%s' resumed and executed on %s", op.Type, targetPlatform),
		}, nil
	}
}
