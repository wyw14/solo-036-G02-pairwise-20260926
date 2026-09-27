package pig

import (
	"fmt"
	"time"

	"github.com/wyw14/solo-036/internal/model"
)

func Assess(p model.Pig, components []model.PigComponent, line model.LineConstraint) model.FitAssessment {
	checks := []model.FitCheck{
		checkDimensions(p),
		checkSealsPresent(components),
		checkSealDiameter(p, components),
		checkBodyPressure(p, line),
		checkSealPressure(components, line),
	}
	fit := true
	for _, check := range checks {
		if !check.Passed {
			fit = false
			break
		}
	}
	return model.FitAssessment{
		ID:         model.NewRevision(),
		PigID:      p.ID,
		RouteID:    line.RouteID,
		Generation: p.Generation,
		Fit:        fit,
		Checks:     checks,
		AssessedAt: time.Now().UTC(),
	}
}

func checkDimensions(p model.Pig) model.FitCheck {
	check := model.FitCheck{Name: "dimensions"}
	if p.BodyDiameter <= 0 || p.BodyLength <= 0 {
		check.Detail = fmt.Sprintf("body dimensions incomplete: diameter %.2f mm, length %.2f mm", p.BodyDiameter, p.BodyLength)
		return check
	}
	check.Passed = true
	check.Detail = fmt.Sprintf("body diameter %.2f mm, length %.2f mm recorded", p.BodyDiameter, p.BodyLength)
	return check
}

func checkSealsPresent(components []model.PigComponent) model.FitCheck {
	check := model.FitCheck{Name: "seals_present"}
	seals := sealComponents(components)
	if len(seals) == 0 {
		check.Detail = "no seal component installed"
		return check
	}
	check.Passed = true
	check.Detail = fmt.Sprintf("%d seal component(s) installed", len(seals))
	return check
}

func checkSealDiameter(p model.Pig, components []model.PigComponent) model.FitCheck {
	check := model.FitCheck{Name: "seal_diameter"}
	seals := sealComponents(components)
	if len(seals) == 0 {
		check.Detail = "no seal component installed"
		return check
	}
	weakest := seals[0]
	for _, seal := range seals[1:] {
		if seal.Diameter < weakest.Diameter {
			weakest = seal
		}
	}
	if weakest.Diameter < p.BodyDiameter {
		check.Detail = fmt.Sprintf("seal %s diameter %.2f mm is smaller than body diameter %.2f mm", weakest.Name, weakest.Diameter, p.BodyDiameter)
		return check
	}
	check.Passed = true
	check.Detail = fmt.Sprintf("smallest seal diameter %.2f mm covers body diameter %.2f mm", weakest.Diameter, p.BodyDiameter)
	return check
}

func checkBodyPressure(p model.Pig, line model.LineConstraint) model.FitCheck {
	check := model.FitCheck{Name: "body_pressure"}
	if p.MaxPressure < line.MaxPressure {
		check.Detail = fmt.Sprintf("pig max pressure %.2f is below line ceiling %.2f", p.MaxPressure, line.MaxPressure)
		return check
	}
	check.Passed = true
	check.Detail = fmt.Sprintf("pig max pressure %.2f covers line ceiling %.2f", p.MaxPressure, line.MaxPressure)
	return check
}

func checkSealPressure(components []model.PigComponent, line model.LineConstraint) model.FitCheck {
	check := model.FitCheck{Name: "seal_pressure"}
	seals := sealComponents(components)
	if len(seals) == 0 {
		check.Detail = "no seal component installed"
		return check
	}
	weakest := seals[0]
	for _, seal := range seals[1:] {
		if seal.PressureRating < weakest.PressureRating {
			weakest = seal
		}
	}
	if weakest.PressureRating < line.MaxPressure {
		check.Detail = fmt.Sprintf("seal %s rating %.2f is below line ceiling %.2f", weakest.Name, weakest.PressureRating, line.MaxPressure)
		return check
	}
	check.Passed = true
	check.Detail = fmt.Sprintf("lowest seal rating %.2f covers line ceiling %.2f", weakest.PressureRating, line.MaxPressure)
	return check
}

func sealComponents(components []model.PigComponent) []model.PigComponent {
	seals := make([]model.PigComponent, 0, len(components))
	for _, component := range components {
		if component.Kind == model.ComponentSeal {
			seals = append(seals, component)
		}
	}
	return seals
}
