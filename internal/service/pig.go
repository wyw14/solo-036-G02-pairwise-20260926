package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/wyw14/solo-036/internal/model"
	"github.com/wyw14/solo-036/internal/pig"
)

func (r *Runtime) RegisterPig(spec model.PigSpec) (model.PigSpec, error) {
	registered, err := r.Pigs.Register(spec)
	if err != nil {
		return model.PigSpec{}, err
	}
	if err := r.appendEvent(model.Event{
		Kind:    "pig_registered",
		Payload: registered.ID,
	}); err != nil {
		return model.PigSpec{}, err
	}
	return registered, nil
}

func (r *Runtime) ListPigs() []model.PigSpec { return r.Pigs.List() }

func (r *Runtime) Pig(id string) (model.PigSpec, error) {
	spec, ok := r.Pigs.Get(id)
	if !ok {
		return model.PigSpec{}, fmt.Errorf("pig %s not registered", id)
	}
	return spec, nil
}

func (r *Runtime) ReplacePigComponent(pigID, componentID string, replacement model.PigComponent) (model.PigSpec, error) {
	spec, err := r.Pigs.ReplaceComponent(pigID, componentID, replacement)
	if err != nil {
		return model.PigSpec{}, err
	}
	if err := r.appendEvent(model.Event{
		Kind:    "pig_component_replaced",
		Epoch:   spec.Revision,
		Payload: fmt.Sprintf("%s/%s", pigID, componentID),
	}); err != nil {
		return model.PigSpec{}, err
	}
	return spec, nil
}

func (r *Runtime) EvaluatePigCompatibility(pigID, routeID string, targetPressure float64) (model.Compatibility, error) {
	if targetPressure <= 0 {
		return model.Compatibility{}, errors.New("target pressure must be positive")
	}
	spec, err := r.Pig(pigID)
	if err != nil {
		return model.Compatibility{}, err
	}
	control := r.ensureControl()
	routeState, ok := control.Routes[routeID]
	if !ok {
		return model.Compatibility{}, fmt.Errorf("unknown route %s", routeID)
	}
	limits, err := control.Network.Limits(routeState.FromStation, routeState.ToStation)
	if err != nil {
		return model.Compatibility{}, err
	}
	line := pig.LineLimit{RouteID: routeID, Diameter: limits.Diameter, MaxPressure: limits.Pressure}
	for _, stationID := range []string{routeState.FromStation, routeState.ToStation} {
		if station, ok := control.Stations[stationID]; ok && station.PressureLimit < line.MaxPressure {
			line.MaxPressure = station.PressureLimit
		}
	}
	conclusion := pig.Evaluate(spec, line, targetPressure, time.Now().UTC())
	r.Pigs.RecordCompatibility(conclusion)
	if err := r.appendEvent(model.Event{
		Kind:    "pig_compatibility",
		Epoch:   spec.Revision,
		Payload: pig.Explain(conclusion),
	}); err != nil {
		return model.Compatibility{}, err
	}
	return conclusion, nil
}

func (r *Runtime) PigCompatibility(pigID string) ([]model.Compatibility, error) {
	if _, err := r.Pig(pigID); err != nil {
		return nil, err
	}
	return r.Pigs.Conclusions(pigID), nil
}
