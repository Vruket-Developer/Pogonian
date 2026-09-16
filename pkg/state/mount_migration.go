package state

import (
	"fmt"
	"strings"

	"pogonian/pkg/ir"
)

// MountIncompatibilityError describes a missing architectural relationship preventing migration.
type MountIncompatibilityError struct {
	Resource   string
	RoutePath  string
	MissingDep string
	Reason     string
}

func (e *MountIncompatibilityError) Error() string {
	return fmt.Sprintf("mount-aware migration failed for resource '%s' on route '%s': missing %s (%s)",
		e.Resource, e.RoutePath, e.MissingDep, e.Reason)
}

func findTargetMount(targetApp *ir.Application, activeRoute string, resourceName string) *ir.MountIR {
	for _, m := range targetApp.Mounts {
		if matchRoute(m.RoutePath, activeRoute) {
			return m
		}
	}
	if m := targetApp.FindMountByRoute(activeRoute); m != nil {
		return m
	}
	if resourceName != "" {
		return targetApp.FindMountByResource(resourceName)
	}
	return nil
}

// PreflightMountCheck validates that the destination application architecture supports the mounted resource.
func PreflightMountCheck(snapshot *ir.StateSnapshot, targetApp *ir.Application, targetPlatform ir.TargetPlatform) error {
	if snapshot == nil || targetApp == nil {
		return fmt.Errorf("snapshot and target application must not be nil")
	}

	if snapshot.ActiveRoute == "" {
		return nil
	}

	resName := ""
	if snapshot.MountedResource != nil {
		resName = snapshot.MountedResource.Resource
	}

	targetMount := findTargetMount(targetApp, snapshot.ActiveRoute, resName)
	originMount := snapshot.MountedResource

	if originMount != nil && originMount.Resource != "" {
		if targetMount == nil {
			return &MountIncompatibilityError{
				Resource:   originMount.Resource,
				RoutePath:  snapshot.ActiveRoute,
				MissingDep: fmt.Sprintf("mount contract for resource '%s'", originMount.Resource),
				Reason:     fmt.Sprintf("destination platform '%s' does not declare an authoritative mount for this resource", targetPlatform),
			}
		}

		// Check if destination declares the entity
		if originMount.Entity != "" {
			var entFound bool
			for _, ent := range targetApp.Entities {
				if strings.EqualFold(ent.Name, originMount.Entity) || strings.EqualFold(ent.TableName, originMount.Table) {
					entFound = true
					break
				}
			}
			if !entFound {
				return &MountIncompatibilityError{
					Resource:   originMount.Resource,
					RoutePath:  snapshot.ActiveRoute,
					MissingDep: fmt.Sprintf("entity '%s'", originMount.Entity),
					Reason:     fmt.Sprintf("destination platform '%s' does not declare entity or table required by active resource", targetPlatform),
				}
			}
		}

		// Verify repository binding if required
		if originMount.Repository != "" {
			var repoFound bool
			for _, r := range targetApp.Repositories {
				if strings.EqualFold(r.Name, originMount.Repository) || strings.EqualFold(r.TargetTable, originMount.Table) {
					repoFound = true
					break
				}
			}
			if !repoFound && len(targetApp.Repositories) > 0 {
				return &MountIncompatibilityError{
					Resource:   originMount.Resource,
					RoutePath:  snapshot.ActiveRoute,
					MissingDep: fmt.Sprintf("repository '%s'", originMount.Repository),
					Reason:     "destination backend does not declare the required repository for data persistence",
				}
			}
		}

		// Verify service binding if required
		if originMount.Service != "" && len(targetApp.Services) > 0 {
			var svcFound bool
			for _, s := range targetApp.Services {
				if strings.EqualFold(s.Name, originMount.Service) {
					svcFound = true
					break
				}
			}
			if !svcFound {
				return &MountIncompatibilityError{
					Resource:   originMount.Resource,
					RoutePath:  snapshot.ActiveRoute,
					MissingDep: fmt.Sprintf("service '%s'", originMount.Service),
					Reason:     "destination backend does not declare the required service for business logic execution",
				}
			}
		}
	}

	return nil
}

// ReconstructMountedResource reconnects the active route to its authoritative architectural data binding on the destination device.
// If the destination does not declare an authoritative mount satisfying the contract, it rejects migration.
func ReconstructMountedResource(snapshot *ir.StateSnapshot, targetApp *ir.Application) (*ir.MountIR, error) {
	if snapshot.ActiveRoute == "" {
		return nil, nil
	}

	resName := ""
	if snapshot.MountedResource != nil {
		resName = snapshot.MountedResource.Resource
	}

	mount := findTargetMount(targetApp, snapshot.ActiveRoute, resName)

	// Strict Authoritative Check: If origin had an active mounted resource, the destination MUST satisfy the mount contract!
	if snapshot.MountedResource != nil && snapshot.MountedResource.Resource != "" {
		if mount == nil {
			return nil, fmt.Errorf("mount-aware migration failed: destination application does not declare an authoritative mount contract for resource '%s' on route '%s'",
				snapshot.MountedResource.Resource, snapshot.ActiveRoute)
		}

		// Verify Entity compatibility
		if snapshot.MountedResource.Entity != "" && mount.Entity != "" &&
			!strings.EqualFold(mount.Entity, snapshot.MountedResource.Entity) {
			return nil, fmt.Errorf("mount contract mismatch: resource '%s' requires entity '%s' but destination declares '%s'",
				snapshot.MountedResource.Resource, snapshot.MountedResource.Entity, mount.Entity)
		}
	}

	return mount, nil
}
