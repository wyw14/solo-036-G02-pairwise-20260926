package pig

import (
	"sync"

	"github.com/wyw14/solo-036/internal/model"
)

type Registry struct {
	mu          sync.Mutex
	pigs        map[string]model.PigSpec
	conclusions map[string]model.Compatibility
}

func NewRegistry() *Registry {
	return &Registry{pigs: make(map[string]model.PigSpec), conclusions: make(map[string]model.Compatibility)}
}

func conclusionKey(pigID, routeID string) string { return pigID + "/" + routeID }
