package pig

import (
	"fmt"
	"strings"
	"time"

	"github.com/wyw14/solo-036/internal/model"
)

const (
	// sealMaxOversize bounds how far a seal may exceed the line bore and still
	// pass flanges and bends; below the bore it cannot seal at all.
	sealMaxOversize = 1.05
	// bodyMaxRatio keeps the pig body clear of the tightest bore on the line.
	bodyMaxRatio = 0.95
)

// LineLimit carries the route restrictions a pig is evaluated against.
type LineLimit struct {
	RouteID     string
	Diameter    float64
	MaxPressure float64
}

// Evaluate checks a pig specification against the line restrictions and
// returns a conclusion whose checks explain the verdict one by one.
func Evaluate(spec model.PigSpec, line LineLimit, targetPressure float64, at time.Time) model.Compatibility {
	checks := make([]model.CompatibilityCheck, 0, 4)

	sealLow := line.Diameter
	sealHigh := line.Diameter * sealMaxOversize
	for _, seal := range spec.SealComponents() {
		pass := seal.Diameter >= sealLow && seal.Diameter <= sealHigh
		detail := fmt.Sprintf("seal %s diameter %.1f mm within %.1f-%.1f mm oversize of line bore %.1f mm", seal.ID, seal.Diameter, sealLow, sealHigh, line.Diameter)
		if !pass {
			detail = fmt.Sprintf("seal %s diameter %.1f mm outside %.1f-%.1f mm oversize of line bore %.1f mm", seal.ID, seal.Diameter, sealLow, sealHigh, line.Diameter)
		}
		checks = append(checks, model.CompatibilityCheck{Name: "seal-diameter/" + seal.ID, Pass: pass, Detail: detail})
	}

	bodyLimit := line.Diameter * bodyMaxRatio
	bodyPass := spec.BodyDiameter <= bodyLimit
	bodyDetail := fmt.Sprintf("body diameter %.1f mm within clearance limit %.1f mm (%.0f%% of bore %.1f mm)", spec.BodyDiameter, bodyLimit, bodyMaxRatio*100, line.Diameter)
	if !bodyPass {
		bodyDetail = fmt.Sprintf("body diameter %.1f mm exceeds clearance limit %.1f mm (%.0f%% of bore %.1f mm)", spec.BodyDiameter, bodyLimit, bodyMaxRatio*100, line.Diameter)
	}
	checks = append(checks, model.CompatibilityCheck{Name: "body-clearance", Pass: bodyPass, Detail: bodyDetail})

	pressurePass := spec.MaxPressure >= targetPressure
	pressureDetail := fmt.Sprintf("pig rating %.2f MPa covers target pressure %.2f MPa", spec.MaxPressure, targetPressure)
	if !pressurePass {
		pressureDetail = fmt.Sprintf("pig rating %.2f MPa below target pressure %.2f MPa", spec.MaxPressure, targetPressure)
	}
	checks = append(checks, model.CompatibilityCheck{Name: "pig-pressure", Pass: pressurePass, Detail: pressureDetail})

	linePass := targetPressure <= line.MaxPressure
	lineDetail := fmt.Sprintf("target pressure %.2f MPa within line limit %.2f MPa", targetPressure, line.MaxPressure)
	if !linePass {
		lineDetail = fmt.Sprintf("target pressure %.2f MPa exceeds line limit %.2f MPa", targetPressure, line.MaxPressure)
	}
	checks = append(checks, model.CompatibilityCheck{Name: "route-pressure", Pass: linePass, Detail: lineDetail})

	compatible := true
	for _, check := range checks {
		if !check.Pass {
			compatible = false
			break
		}
	}
	return model.Compatibility{
		PigID:          spec.ID,
		PigRevision:    spec.Revision,
		RouteID:        line.RouteID,
		TargetPressure: targetPressure,
		Compatible:     compatible,
		Checks:         checks,
		EvaluatedAt:    at,
	}
}

func Explain(conclusion model.Compatibility) string {
	verdict := "incompatible"
	if conclusion.Compatible {
		verdict = "compatible"
	}
	parts := make([]string, 0, len(conclusion.Checks))
	for _, check := range conclusion.Checks {
		parts = append(parts, check.Name)
	}
	return fmt.Sprintf("pig %s revision %d on route %s: %s (%s)", conclusion.PigID, conclusion.PigRevision, conclusion.RouteID, verdict, strings.Join(parts, ", "))
}
