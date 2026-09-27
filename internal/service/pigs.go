package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/wyw14/solo-036/internal/model"
	"github.com/wyw14/solo-036/internal/pig"
)

type PigSpecRequest struct {
	Name         string  `json:"name"`
	BodyDiameter float64 `json:"body_diameter"`
	BodyLength   float64 `json:"body_length"`
	MaxPressure  float64 `json:"max_pressure"`
}

type PigComponentRequest struct {
	Kind           string  `json:"kind"`
	Name           string  `json:"name"`
	Material       string  `json:"material"`
	Diameter       float64 `json:"diameter"`
	PressureRating float64 `json:"pressure_rating"`
}

type PigLedgerEntry struct {
	Pig         model.Pig             `json:"pig"`
	Components  []model.PigComponent  `json:"components"`
	Assessments []model.FitAssessment `json:"assessments"`
}

func (r *Runtime) RegisterPig(request PigSpecRequest) (model.Pig, error) {
	registered, err := r.Pigs.Register(request.Name, request.BodyDiameter, request.BodyLength, request.MaxPressure)
	if err != nil {
		return model.Pig{}, err
	}
	r.bumpRevision()
	if err := r.appendEvent(model.Event{Kind: "pig_registered", Payload: registered.ID, At: time.Now().UTC()}); err != nil {
		return model.Pig{}, fmt.Errorf("journal: %w", err)
	}
	return registered, nil
}

func (r *Runtime) UpdatePigSpec(pigID string, request PigSpecRequest) (model.Pig, error) {
	updated, err := r.Pigs.UpdateSpec(pigID, request.Name, request.BodyDiameter, request.BodyLength, request.MaxPressure)
	if err != nil {
		return model.Pig{}, err
	}
	r.bumpRevision()
	if err := r.appendEvent(model.Event{Kind: "pig_spec_updated", Payload: fmt.Sprintf("%s@%d", updated.ID, updated.Generation), At: time.Now().UTC()}); err != nil {
		return model.Pig{}, fmt.Errorf("journal: %w", err)
	}
	return updated, nil
}

func (r *Runtime) InstallPigComponent(pigID string, request PigComponentRequest) (model.PigComponent, error) {
	installed, err := r.Pigs.Install(pigID, model.ComponentKind(request.Kind), request.Name, request.Material, request.Diameter, request.PressureRating)
	if err != nil {
		return model.PigComponent{}, err
	}
	r.bumpRevision()
	if err := r.appendEvent(model.Event{Kind: "pig_component_installed", Payload: fmt.Sprintf("%s/%s", pigID, installed.ID), At: time.Now().UTC()}); err != nil {
		return model.PigComponent{}, fmt.Errorf("journal: %w", err)
	}
	return installed, nil
}

func (r *Runtime) ReplacePigComponent(pigID, componentID string, request PigComponentRequest) (model.PigComponent, error) {
	replacement, err := r.Pigs.Replace(pigID, componentID, model.ComponentKind(request.Kind), request.Name, request.Material, request.Diameter, request.PressureRating)
	if err != nil {
		return model.PigComponent{}, err
	}
	r.bumpRevision()
	if err := r.appendEvent(model.Event{Kind: "pig_component_replaced", Payload: fmt.Sprintf("%s/%s>%s", pigID, componentID, replacement.ID), At: time.Now().UTC()}); err != nil {
		return model.PigComponent{}, fmt.Errorf("journal: %w", err)
	}
	return replacement, nil
}

func (r *Runtime) RemovePigComponent(pigID, componentID string) error {
	if err := r.Pigs.Remove(pigID, componentID); err != nil {
		return err
	}
	r.bumpRevision()
	if err := r.appendEvent(model.Event{Kind: "pig_component_removed", Payload: fmt.Sprintf("%s/%s", pigID, componentID), At: time.Now().UTC()}); err != nil {
		return fmt.Errorf("journal: %w", err)
	}
	return nil
}

func (r *Runtime) AssessPig(pigID, routeID string) (model.FitAssessment, error) {
	selected, ok := r.Pigs.Get(pigID)
	if !ok {
		return model.FitAssessment{}, fmt.Errorf("pig %s not found", pigID)
	}
	control := r.ensureControl()
	routeState, ok := control.Routes[routeID]
	if !ok {
		return model.FitAssessment{}, fmt.Errorf("unknown route %s", routeID)
	}
	line, err := control.Network.Constraint(routeState.FromStation, routeState.ToStation)
	if err != nil {
		return model.FitAssessment{}, err
	}
	line.RouteID = routeID
	assessment := pig.Assess(selected, r.Pigs.Components(pigID), line)
	if err := r.Pigs.Record(assessment); err != nil {
		return model.FitAssessment{}, err
	}
	r.bumpRevision()
	if err := r.appendEvent(model.Event{Kind: "pig_assessed", Payload: fmt.Sprintf("%s/%s/%t", assessment.PigID, assessment.RouteID, assessment.Fit), At: time.Now().UTC()}); err != nil {
		return model.FitAssessment{}, fmt.Errorf("journal: %w", err)
	}
	return assessment, nil
}

func (r *Runtime) PigLedger() []PigLedgerEntry {
	pigs, _, _ := r.Pigs.Snapshot()
	entries := make([]PigLedgerEntry, 0, len(pigs))
	for _, p := range pigs {
		entries = append(entries, PigLedgerEntry{
			Pig:         p,
			Components:  r.Pigs.Components(p.ID),
			Assessments: r.Pigs.Assessments(p.ID),
		})
	}
	return entries
}

func (r *Runtime) PigLedgerEntry(pigID string) (PigLedgerEntry, error) {
	selected, ok := r.Pigs.Get(pigID)
	if !ok {
		return PigLedgerEntry{}, fmt.Errorf("pig %s not found", pigID)
	}
	return PigLedgerEntry{
		Pig:         selected,
		Components:  r.Pigs.Components(pigID),
		Assessments: r.Pigs.Assessments(pigID),
	}, nil
}

func (r *Runtime) PigAssessments(pigID string) ([]model.FitAssessment, error) {
	if _, ok := r.Pigs.Get(pigID); !ok {
		return nil, fmt.Errorf("pig %s not found", pigID)
	}
	return r.Pigs.Assessments(pigID), nil
}

func (r *Runtime) bumpRevision() {
	r.ensureControl()
	r.mu.Lock()
	r.control.Revision = model.NewRevision()
	r.mu.Unlock()
}

func failedCheckDetails(assessment model.FitAssessment) string {
	details := make([]string, 0, len(assessment.Checks))
	for _, check := range assessment.Checks {
		if !check.Passed {
			details = append(details, check.Name+": "+check.Detail)
		}
	}
	return strings.Join(details, "; ")
}
