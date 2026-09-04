package modules

import (
	"context"
	"errors"
	"fmt"
)

// Lifecycle is implemented by modules that own background work: workers,
// pollers, refreshers or long lived connections that must be started after
// every module has been provisioned and stopped before the process exits.
//
// The contract is:
//
//   - Start is called once, after all modules are loaded, in module load
//     order. It must return promptly; long running work belongs in a
//     goroutine the module owns. A non-nil error aborts startup.
//   - Stop is called once, in reverse load order, and must be idempotent and
//     safe to call even if Start was never reached.
//
// Provision configures a module, Start activates it, Stop deactivates it and
// Cleanup (see CleanerUpper) releases the resources. A module may implement
// any subset.
type Lifecycle interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// StartModules starts every loaded module that implements Lifecycle, in the
// order the modules were loaded.
//
// If a module fails to start, the modules already started are stopped again
// in reverse order and the error is returned, so the caller never serves a
// half configured process. The stop errors, if any, are joined to the
// original failure but the original failure is always the first cause.
func (mm *ModuleManager) StartModules(ctx context.Context) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	var started []ModuleID
	for _, id := range mm.order {
		instance, ok := mm.instances[id]
		if !ok {
			continue
		}
		lc, ok := instance.(Lifecycle)
		if !ok {
			continue
		}
		if mm.started[id] {
			continue
		}
		mm.logf("Starting module", "module_id", id)
		if err := lc.Start(ctx); err != nil {
			startErr := fmt.Errorf("start failed for %s: %w", id, err)
			if rollbackErr := mm.stopLocked(ctx, started); rollbackErr != nil {
				return errors.Join(startErr, fmt.Errorf("rollback: %w", rollbackErr))
			}
			return startErr
		}
		mm.started[id] = true
		started = append(started, id)
	}
	return nil
}

// StopModules stops every started module in reverse load order. Every module
// is given the chance to stop even when an earlier one fails; the errors are
// joined. Stopping is idempotent: a module that was never started, or that
// was already stopped, is skipped.
func (mm *ModuleManager) StopModules(ctx context.Context) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	return mm.stopLocked(ctx, mm.startedOrderLocked())
}

// startedOrderLocked returns the ids of the currently started modules in load
// order. mm.mu must be held.
func (mm *ModuleManager) startedOrderLocked() []ModuleID {
	ids := make([]ModuleID, 0, len(mm.started))
	for _, id := range mm.order {
		if mm.started[id] {
			ids = append(ids, id)
		}
	}
	return ids
}

// stopLocked stops the given modules in reverse of the order given. mm.mu
// must be held.
func (mm *ModuleManager) stopLocked(ctx context.Context, ids []ModuleID) error {
	var errs []error
	for i := len(ids) - 1; i >= 0; i-- {
		id := ids[i]
		if !mm.started[id] {
			continue
		}
		instance, ok := mm.instances[id]
		if !ok {
			delete(mm.started, id)
			continue
		}
		lc, ok := instance.(Lifecycle)
		if !ok {
			delete(mm.started, id)
			continue
		}
		mm.logf("Stopping module", "module_id", id)
		if err := lc.Stop(ctx); err != nil {
			errs = append(errs, fmt.Errorf("stop failed for %s: %w", id, err))
		}
		delete(mm.started, id)
	}
	return errors.Join(errs...)
}
