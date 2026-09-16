package state

import (
	"fmt"
	"strings"
	"time"

	"pogonian/pkg/ir"
)

// ConflictResolution records a single field-level merge decision.
type ConflictResolution struct {
	Field        string `json:"field"`
	WinningValue any    `json:"winning_value"`
	Source       string `json:"source"` // "local", "remote"
	Reason       string `json:"reason"`
}

// ReconcileStateConflicts resolves discrepancies between two snapshots using field-level logical clocks.
func ReconcileStateConflicts(local, remote *ir.StateSnapshot) (*ir.StateSnapshot, []ConflictResolution) {
	if local == nil {
		return remote, nil
	}
	if remote == nil {
		return local, nil
	}

	mergedData := make(map[string]any)
	mergedVector := make(map[string]int64)
	var resolutions []ConflictResolution

	// Copy all local keys first
	for k, v := range local.StateData {
		mergedData[k] = v
		if local.StateVector != nil {
			mergedVector[k] = local.StateVector[k]
		} else {
			mergedVector[k] = local.Timestamp
		}
	}

	// Compare with remote keys
	for k, remoteVal := range remote.StateData {
		localVal, localExists := mergedData[k]
		if !localExists {
			// New field from remote
			mergedData[k] = remoteVal
			if remote.StateVector != nil {
				mergedVector[k] = remote.StateVector[k]
			} else {
				mergedVector[k] = remote.Timestamp
			}
			resolutions = append(resolutions, ConflictResolution{
				Field:        k,
				WinningValue: remoteVal,
				Source:       "remote",
				Reason:       "Non-conflicting field introduced by remote device",
			})
			continue
		}

		// Field exists in both: compare logical timestamps
		localClock := mergedVector[k]
		var remoteClock int64
		if remote.StateVector != nil {
			remoteClock = remote.StateVector[k]
		} else {
			remoteClock = remote.Timestamp
		}

		if remoteClock > localClock {
			// Remote is strictly newer
			mergedData[k] = remoteVal
			mergedVector[k] = remoteClock
			resolutions = append(resolutions, ConflictResolution{
				Field:        k,
				WinningValue: remoteVal,
				Source:       "remote",
				Reason:       fmt.Sprintf("Remote vector clock %d exceeds local clock %d", remoteClock, localClock),
			})
		} else if localClock > remoteClock {
			// Local is strictly newer
			resolutions = append(resolutions, ConflictResolution{
				Field:        k,
				WinningValue: localVal,
				Source:       "local",
				Reason:       fmt.Sprintf("Local vector clock %d exceeds remote clock %d", localClock, remoteClock),
			})
		} else {
			// Tied logical clocks: deterministic tie-breaker (lexicographical snapshot ID)
			if strings.Compare(remote.SnapshotID, local.SnapshotID) > 0 {
				mergedData[k] = remoteVal
				resolutions = append(resolutions, ConflictResolution{
					Field:        k,
					WinningValue: remoteVal,
					Source:       "remote",
					Reason:       "Equal logical clocks: remote won deterministic tie-break",
				})
			} else {
				resolutions = append(resolutions, ConflictResolution{
					Field:        k,
					WinningValue: localVal,
					Source:       "local",
					Reason:       "Equal logical clocks: local won deterministic tie-break",
				})
			}
		}
	}

	// Union active operations, deduplicating by ID
	opMap := make(map[string]*ir.ActiveOperationIR)
	for _, op := range local.ActiveOperations {
		opMap[op.ID] = op
	}
	for _, op := range remote.ActiveOperations {
		if existing, ok := opMap[op.ID]; ok {
			// Take the one with higher progress or later update
			if op.UpdatedAt >= existing.UpdatedAt {
				opMap[op.ID] = op
			}
		} else {
			opMap[op.ID] = op
		}
	}

	var mergedOps []*ir.ActiveOperationIR
	for _, op := range opMap {
		mergedOps = append(mergedOps, op)
	}

	newVersionClock := local.VersionClock
	if remote.VersionClock > newVersionClock {
		newVersionClock = remote.VersionClock
	}
	newVersionClock++

	activeRoute := local.ActiveRoute
	if remote.Timestamp > local.Timestamp && remote.ActiveRoute != "" {
		activeRoute = remote.ActiveRoute
	}

	mergedSnapshot := &ir.StateSnapshot{
		SnapshotID:       generateRandomID(16),
		Timestamp:        time.Now().Unix(),
		AppVersion:       local.AppVersion,
		OriginPlatform:   local.OriginPlatform,
		ActiveRoute:      activeRoute,
		StateData:        mergedData,
		ActiveOperations: mergedOps,
		MountedResource:  local.MountedResource,
		VersionClock:     newVersionClock,
		StateVector:      mergedVector,
	}

	chk, _ := mergedSnapshot.ComputeChecksum()
	mergedSnapshot.SchemaChecksum = chk

	return mergedSnapshot, resolutions
}
