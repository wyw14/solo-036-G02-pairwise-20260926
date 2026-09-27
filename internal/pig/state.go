package pig

import (
	"github.com/wyw14/solo-036/internal/model"
	"sync"
)

type Registry struct {
	mu          sync.RWMutex
	pigs        map[string]model.Pig
	components  map[string]map[string]model.PigComponent
	assessments map[string][]model.FitAssessment
}

func NewRegistry() *Registry {
	return &Registry{
		pigs:        make(map[string]model.Pig),
		components:  make(map[string]map[string]model.PigComponent),
		assessments: make(map[string][]model.FitAssessment),
	}
}
