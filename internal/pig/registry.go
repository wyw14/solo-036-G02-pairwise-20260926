package pig

import (
	"fmt"
	"sort"
	"time"

	"github.com/wyw14/solo-036/internal/model"
)

func (r *Registry) Register(name string, diameter, length, maxPressure float64) (model.Pig, error) {
	now := time.Now().UTC()
	candidate := model.Pig{
		ID:           model.NewRevision(),
		Name:         name,
		BodyDiameter: diameter,
		BodyLength:   length,
		MaxPressure:  maxPressure,
		Generation:   1,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := candidate.Validate(); err != nil {
		return model.Pig{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pigs[candidate.ID] = candidate
	return candidate, nil
}

func (r *Registry) UpdateSpec(id, name string, diameter, length, maxPressure float64) (model.Pig, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.pigs[id]
	if !ok {
		return model.Pig{}, fmt.Errorf("pig %s not found", id)
	}
	current.Name = name
	current.BodyDiameter = diameter
	current.BodyLength = length
	current.MaxPressure = maxPressure
	current.Generation++
	current.UpdatedAt = time.Now().UTC()
	if err := current.Validate(); err != nil {
		return model.Pig{}, err
	}
	r.pigs[id] = current
	r.supersedeLocked(id)
	return current, nil
}

func (r *Registry) Install(pigID string, kind model.ComponentKind, name, material string, diameter, rating float64) (model.PigComponent, error) {
	candidate := model.PigComponent{
		ID:             model.NewRevision(),
		PigID:          pigID,
		Kind:           kind,
		Name:           name,
		Material:       material,
		Diameter:       diameter,
		PressureRating: rating,
		InstalledAt:    time.Now().UTC(),
	}
	if err := candidate.Validate(); err != nil {
		return model.PigComponent{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.pigs[pigID]; !ok {
		return model.PigComponent{}, fmt.Errorf("pig %s not found", pigID)
	}
	if r.components[pigID] == nil {
		r.components[pigID] = make(map[string]model.PigComponent)
	}
	r.components[pigID][candidate.ID] = candidate
	r.bumpLocked(pigID)
	return candidate, nil
}

func (r *Registry) Replace(pigID, componentID string, kind model.ComponentKind, name, material string, diameter, rating float64) (model.PigComponent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.pigs[pigID]; !ok {
		return model.PigComponent{}, fmt.Errorf("pig %s not found", pigID)
	}
	installed := r.components[pigID]
	if _, ok := installed[componentID]; !ok {
		return model.PigComponent{}, fmt.Errorf("component %s not found on pig %s", componentID, pigID)
	}
	replacement := model.PigComponent{
		ID:             model.NewRevision(),
		PigID:          pigID,
		Kind:           kind,
		Name:           name,
		Material:       material,
		Diameter:       diameter,
		PressureRating: rating,
		InstalledAt:    time.Now().UTC(),
	}
	if err := replacement.Validate(); err != nil {
		return model.PigComponent{}, err
	}
	delete(installed, componentID)
	installed[replacement.ID] = replacement
	r.bumpLocked(pigID)
	return replacement, nil
}

func (r *Registry) Remove(pigID, componentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.pigs[pigID]; !ok {
		return fmt.Errorf("pig %s not found", pigID)
	}
	installed := r.components[pigID]
	if _, ok := installed[componentID]; !ok {
		return fmt.Errorf("component %s not found on pig %s", componentID, pigID)
	}
	delete(installed, componentID)
	r.bumpLocked(pigID)
	return nil
}

func (r *Registry) Record(assessment model.FitAssessment) error {
	if err := assessment.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.pigs[assessment.PigID]; !ok {
		return fmt.Errorf("pig %s not found", assessment.PigID)
	}
	r.assessments[assessment.PigID] = append(r.assessments[assessment.PigID], assessment)
	return nil
}

func (r *Registry) Get(id string) (model.Pig, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.pigs[id]
	return p, ok
}

func (r *Registry) Components(pigID string) []model.PigComponent {
	r.mu.RLock()
	defer r.mu.RUnlock()
	installed := r.components[pigID]
	out := make([]model.PigComponent, 0, len(installed))
	for _, component := range installed {
		out = append(out, component)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].InstalledAt.Equal(out[j].InstalledAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].InstalledAt.Before(out[j].InstalledAt)
	})
	return out
}

func (r *Registry) Assessments(pigID string) []model.FitAssessment {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := append([]model.FitAssessment(nil), r.assessments[pigID]...)
	sort.Slice(out, func(i, j int) bool { return out[i].AssessedAt.Before(out[j].AssessedAt) })
	return out
}

func (r *Registry) Current(pigID, routeID string) (model.FitAssessment, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.pigs[pigID]
	if !ok {
		return model.FitAssessment{}, false
	}
	var current model.FitAssessment
	found := false
	for _, assessment := range r.assessments[pigID] {
		if assessment.RouteID != routeID || assessment.Superseded || assessment.Generation != p.Generation {
			continue
		}
		if !found || assessment.AssessedAt.After(current.AssessedAt) {
			current = assessment
			found = true
		}
	}
	return current, found
}

func (r *Registry) Snapshot() ([]model.Pig, []model.PigComponent, []model.FitAssessment) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	pigs := make([]model.Pig, 0, len(r.pigs))
	for _, p := range r.pigs {
		pigs = append(pigs, p)
	}
	sort.Slice(pigs, func(i, j int) bool { return pigs[i].ID < pigs[j].ID })
	components := make([]model.PigComponent, 0)
	for _, installed := range r.components {
		for _, component := range installed {
			components = append(components, component)
		}
	}
	sort.Slice(components, func(i, j int) bool { return components[i].ID < components[j].ID })
	assessments := make([]model.FitAssessment, 0)
	for _, history := range r.assessments {
		assessments = append(assessments, history...)
	}
	sort.Slice(assessments, func(i, j int) bool {
		if assessments[i].AssessedAt.Equal(assessments[j].AssessedAt) {
			return assessments[i].ID < assessments[j].ID
		}
		return assessments[i].AssessedAt.Before(assessments[j].AssessedAt)
	})
	return pigs, components, assessments
}

func Restore(registry *Registry, pigs []model.Pig, components []model.PigComponent, assessments []model.FitAssessment) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	for _, p := range pigs {
		if p.Validate() == nil {
			registry.pigs[p.ID] = p
		}
	}
	for _, component := range components {
		if component.Validate() != nil {
			continue
		}
		if _, ok := registry.pigs[component.PigID]; !ok {
			continue
		}
		if registry.components[component.PigID] == nil {
			registry.components[component.PigID] = make(map[string]model.PigComponent)
		}
		registry.components[component.PigID][component.ID] = component
	}
	for _, assessment := range assessments {
		if assessment.Validate() != nil {
			continue
		}
		if _, ok := registry.pigs[assessment.PigID]; !ok {
			continue
		}
		registry.assessments[assessment.PigID] = append(registry.assessments[assessment.PigID], assessment)
	}
}

func (r *Registry) bumpLocked(pigID string) {
	p := r.pigs[pigID]
	p.Generation++
	p.UpdatedAt = time.Now().UTC()
	r.pigs[pigID] = p
	r.supersedeLocked(pigID)
}

func (r *Registry) supersedeLocked(pigID string) {
	history := r.assessments[pigID]
	for i := range history {
		history[i].Superseded = true
	}
	r.assessments[pigID] = history
}
