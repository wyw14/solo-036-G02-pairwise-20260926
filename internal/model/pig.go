package model

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type ComponentKind string

const (
	ComponentSeal    ComponentKind = "seal"
	ComponentGuide   ComponentKind = "guide"
	ComponentGauging ComponentKind = "gauging"
)

type Pig struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	BodyDiameter float64   `json:"body_diameter"`
	BodyLength   float64   `json:"body_length"`
	MaxPressure  float64   `json:"max_pressure"`
	Generation   int64     `json:"generation"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type PigComponent struct {
	ID             string        `json:"id"`
	PigID          string        `json:"pig_id"`
	Kind           ComponentKind `json:"kind"`
	Name           string        `json:"name"`
	Material       string        `json:"material"`
	Diameter       float64       `json:"diameter"`
	PressureRating float64       `json:"pressure_rating"`
	InstalledAt    time.Time     `json:"installed_at"`
}

type LineConstraint struct {
	RouteID     string   `json:"route_id"`
	Stations    []string `json:"stations"`
	Segments    []string `json:"segments"`
	MaxPressure float64  `json:"max_pressure"`
}

type FitCheck struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

type FitAssessment struct {
	ID         string     `json:"id"`
	PigID      string     `json:"pig_id"`
	RouteID    string     `json:"route_id"`
	Generation int64      `json:"generation"`
	Fit        bool       `json:"fit"`
	Checks     []FitCheck `json:"checks"`
	AssessedAt time.Time  `json:"assessed_at"`
	Superseded bool       `json:"superseded"`
}

func (p Pig) Validate() error {
	if strings.TrimSpace(p.ID) == "" {
		return errors.New("pig id is required")
	}
	if strings.TrimSpace(p.Name) == "" {
		return errors.New("pig name is required")
	}
	if p.BodyDiameter <= 0 {
		return errors.New("pig body diameter must be positive")
	}
	if p.BodyLength <= 0 {
		return errors.New("pig body length must be positive")
	}
	if p.MaxPressure <= 0 {
		return errors.New("pig max pressure must be positive")
	}
	if p.Generation <= 0 {
		return errors.New("pig generation must be positive")
	}
	return nil
}

func (c PigComponent) Validate() error {
	if strings.TrimSpace(c.ID) == "" || strings.TrimSpace(c.PigID) == "" {
		return errors.New("component identity is incomplete")
	}
	if strings.TrimSpace(c.Name) == "" {
		return errors.New("component name is required")
	}
	switch c.Kind {
	case ComponentSeal, ComponentGuide, ComponentGauging:
	default:
		return fmt.Errorf("invalid component kind %q", c.Kind)
	}
	if c.Diameter <= 0 {
		return errors.New("component diameter must be positive")
	}
	if c.PressureRating <= 0 {
		return errors.New("component pressure rating must be positive")
	}
	return nil
}

func (a FitAssessment) Validate() error {
	if strings.TrimSpace(a.ID) == "" || strings.TrimSpace(a.PigID) == "" || strings.TrimSpace(a.RouteID) == "" {
		return errors.New("assessment identity is incomplete")
	}
	if a.Generation <= 0 {
		return errors.New("assessment generation must be positive")
	}
	if len(a.Checks) == 0 {
		return errors.New("assessment must record at least one check")
	}
	for _, check := range a.Checks {
		if strings.TrimSpace(check.Name) == "" {
			return errors.New("assessment check name is required")
		}
		if !check.Passed && strings.TrimSpace(check.Detail) == "" {
			return fmt.Errorf("failed check %s must explain why", check.Name)
		}
	}
	return nil
}
