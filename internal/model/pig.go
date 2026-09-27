package model

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type ComponentKind string

const (
	ComponentSeal  ComponentKind = "seal"
	ComponentGuide ComponentKind = "guide"
	ComponentBody  ComponentKind = "body"
)

type PigComponent struct {
	ID          string        `json:"id"`
	Kind        ComponentKind `json:"kind"`
	Diameter    float64       `json:"diameter"`
	Material    string        `json:"material"`
	InstalledAt time.Time     `json:"installed_at"`
}

type PigSpec struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	BodyDiameter float64        `json:"body_diameter"`
	Length       float64        `json:"length"`
	MaxPressure  float64        `json:"max_pressure"`
	Components   []PigComponent `json:"components"`
	Revision     int64          `json:"revision"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

type CompatibilityCheck struct {
	Name   string `json:"name"`
	Pass   bool   `json:"pass"`
	Detail string `json:"detail"`
}

type Compatibility struct {
	PigID          string               `json:"pig_id"`
	PigRevision    int64                `json:"pig_revision"`
	RouteID        string               `json:"route_id"`
	TargetPressure float64              `json:"target_pressure"`
	Compatible     bool                 `json:"compatible"`
	Checks         []CompatibilityCheck `json:"checks"`
	EvaluatedAt    time.Time            `json:"evaluated_at"`
	Stale          bool                 `json:"stale"`
}

func (c PigComponent) Validate() error {
	if strings.TrimSpace(c.ID) == "" {
		return errors.New("component id is required")
	}
	switch c.Kind {
	case ComponentSeal, ComponentGuide, ComponentBody:
	default:
		return fmt.Errorf("invalid component kind %q", c.Kind)
	}
	if c.Diameter <= 0 {
		return errors.New("component diameter must be positive")
	}
	if strings.TrimSpace(c.Material) == "" {
		return errors.New("component material is required")
	}
	return nil
}

func (p PigSpec) Validate() error {
	if strings.TrimSpace(p.ID) == "" {
		return errors.New("pig id is required")
	}
	if strings.TrimSpace(p.Name) == "" {
		return errors.New("pig name is required")
	}
	if p.BodyDiameter <= 0 {
		return errors.New("pig body diameter must be positive")
	}
	if p.Length <= 0 {
		return errors.New("pig length must be positive")
	}
	if p.MaxPressure <= 0 {
		return errors.New("pig max pressure must be positive")
	}
	if p.Revision <= 0 {
		return errors.New("pig revision must be positive")
	}
	seen := make(map[string]bool, len(p.Components))
	seals := 0
	for _, component := range p.Components {
		if err := component.Validate(); err != nil {
			return fmt.Errorf("component %s: %w", component.ID, err)
		}
		if seen[component.ID] {
			return fmt.Errorf("duplicate component %s", component.ID)
		}
		seen[component.ID] = true
		if component.Kind == ComponentSeal {
			seals++
		}
	}
	if seals == 0 {
		return errors.New("pig needs at least one seal component")
	}
	return nil
}

func (p PigSpec) SealComponents() []PigComponent {
	out := make([]PigComponent, 0, len(p.Components))
	for _, component := range p.Components {
		if component.Kind == ComponentSeal {
			out = append(out, component)
		}
	}
	return out
}
