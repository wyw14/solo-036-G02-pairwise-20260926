package pig

import (
	"fmt"
	"sort"
	"time"

	"github.com/wyw14/solo-036/internal/model"
)

func (r *Registry) Register(spec model.PigSpec) (model.PigSpec, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.pigs[spec.ID]; exists {
		return model.PigSpec{}, fmt.Errorf("pig %s already registered", spec.ID)
	}
	now := time.Now().UTC()
	for i := range spec.Components {
		if spec.Components[i].ID == "" {
			spec.Components[i].ID = fmt.Sprintf("component-%d", i+1)
		}
		if spec.Components[i].InstalledAt.IsZero() {
			spec.Components[i].InstalledAt = now
		}
	}
	spec.Revision = 1
	spec.CreatedAt = now
	spec.UpdatedAt = now
	if err := spec.Validate(); err != nil {
		return model.PigSpec{}, err
	}
	r.pigs[spec.ID] = spec
	return spec, nil
}

func (r *Registry) Get(id string) (model.PigSpec, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	spec, ok := r.pigs[id]
	return spec, ok
}

func (r *Registry) List() []model.PigSpec {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]model.PigSpec, 0, len(r.pigs))
	for _, spec := range r.pigs {
		out = append(out, spec)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// ReplaceComponent swaps one installed component and bumps the pig revision.
// Every compatibility conclusion recorded against the older revision is
// invalidated, because it was derived from a configuration that no longer exists.
func (r *Registry) ReplaceComponent(pigID, componentID string, replacement model.PigComponent) (model.PigSpec, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	spec, ok := r.pigs[pigID]
	if !ok {
		return model.PigSpec{}, fmt.Errorf("pig %s not registered", pigID)
	}
	updated := false
	for i, component := range spec.Components {
		if component.ID != componentID {
			continue
		}
		replacement.ID = component.ID
		if replacement.Kind == "" {
			replacement.Kind = component.Kind
		}
		replacement.InstalledAt = time.Now().UTC()
		if err := replacement.Validate(); err != nil {
			return model.PigSpec{}, err
		}
		spec.Components[i] = replacement
		updated = true
		break
	}
	if !updated {
		return model.PigSpec{}, fmt.Errorf("component %s not found on pig %s", componentID, pigID)
	}
	spec.Revision++
	spec.UpdatedAt = time.Now().UTC()
	r.pigs[pigID] = spec
	for key, conclusion := range r.conclusions {
		if conclusion.PigID == pigID {
			delete(r.conclusions, key)
		}
	}
	return spec, nil
}

func (r *Registry) RecordCompatibility(conclusion model.Compatibility) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.conclusions[conclusionKey(conclusion.PigID, conclusion.RouteID)] = conclusion
}

// Conclusions returns the stored conclusions for a pig, flagging any whose
// revision no longer matches the pig's current configuration as stale.
func (r *Registry) Conclusions(pigID string) []model.Compatibility {
	r.mu.Lock()
	defer r.mu.Unlock()
	spec, registered := r.pigs[pigID]
	out := make([]model.Compatibility, 0)
	for _, conclusion := range r.conclusions {
		if conclusion.PigID != pigID {
			continue
		}
		if !registered || conclusion.PigRevision != spec.Revision {
			conclusion.Stale = true
		}
		out = append(out, conclusion)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].EvaluatedAt.Before(out[j].EvaluatedAt) })
	return out
}

func (r *Registry) Snapshot() ([]model.PigSpec, []model.Compatibility) {
	r.mu.Lock()
	defer r.mu.Unlock()
	pigs := make([]model.PigSpec, 0, len(r.pigs))
	for _, spec := range r.pigs {
		pigs = append(pigs, spec)
	}
	sort.Slice(pigs, func(i, j int) bool { return pigs[i].ID < pigs[j].ID })
	conclusions := make([]model.Compatibility, 0, len(r.conclusions))
	for _, conclusion := range r.conclusions {
		conclusions = append(conclusions, conclusion)
	}
	sort.Slice(conclusions, func(i, j int) bool {
		if conclusions[i].PigID == conclusions[j].PigID {
			return conclusions[i].RouteID < conclusions[j].RouteID
		}
		return conclusions[i].PigID < conclusions[j].PigID
	})
	return pigs, conclusions
}

func Restore(r *Registry, pigs []model.PigSpec, conclusions []model.Compatibility) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pigs = make(map[string]model.PigSpec, len(pigs))
	for _, spec := range pigs {
		r.pigs[spec.ID] = spec
	}
	r.conclusions = make(map[string]model.Compatibility, len(conclusions))
	for _, conclusion := range conclusions {
		r.conclusions[conclusionKey(conclusion.PigID, conclusion.RouteID)] = conclusion
	}
}
