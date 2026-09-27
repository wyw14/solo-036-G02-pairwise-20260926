package route

import (
	"container/heap"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/wyw14/solo-036/internal/model"
)

type Segment struct {
	ID              string  `json:"id"`
	From            string  `json:"from"`
	To              string  `json:"to"`
	Length          float64 `json:"length"`
	MaximumPressure float64 `json:"maximum_pressure"`
	Enabled         bool    `json:"enabled"`
}

type Network struct {
	stations map[string]model.Station
	segments map[string]Segment
	outgoing map[string][]string
}

type Path struct {
	Stations        []string `json:"stations"`
	Segments        []string `json:"segments"`
	Length          float64  `json:"length"`
	MaximumPressure float64  `json:"maximum_pressure"`
}

type pathCandidate struct {
	station  string
	distance float64
	pressure float64
	stations []string
	segments []string
	index    int
}

type candidateHeap []*pathCandidate

func (h candidateHeap) Len() int           { return len(h) }
func (h candidateHeap) Less(i, j int) bool { return h[i].distance < h[j].distance }
func (h candidateHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i]; h[i].index = i; h[j].index = j }
func (h *candidateHeap) Push(value any) {
	item := value.(*pathCandidate)
	item.index = len(*h)
	*h = append(*h, item)
}
func (h *candidateHeap) Pop() any {
	old := *h
	count := len(old)
	item := old[count-1]
	old[count-1] = nil
	item.index = -1
	*h = old[:count-1]
	return item
}

func NewNetwork(stations []model.Station, segments []Segment) (*Network, error) {
	network := &Network{stations: make(map[string]model.Station), segments: make(map[string]Segment), outgoing: make(map[string][]string)}
	for _, station := range stations {
		if err := station.Validate(); err != nil {
			return nil, err
		}
		if _, exists := network.stations[station.ID]; exists {
			return nil, fmt.Errorf("duplicate station %s", station.ID)
		}
		network.stations[station.ID] = station
	}
	for _, segment := range segments {
		if err := network.addSegment(segment); err != nil {
			return nil, err
		}
	}
	if err := network.Validate(); err != nil {
		return nil, err
	}
	return network, nil
}

func (n *Network) addSegment(segment Segment) error {
	if segment.ID == "" || segment.From == "" || segment.To == "" {
		return errors.New("segment identity is incomplete")
	}
	if _, exists := n.segments[segment.ID]; exists {
		return fmt.Errorf("duplicate segment %s", segment.ID)
	}
	if _, ok := n.stations[segment.From]; !ok {
		return fmt.Errorf("segment %s has unknown source station", segment.ID)
	}
	if _, ok := n.stations[segment.To]; !ok {
		return fmt.Errorf("segment %s has unknown destination station", segment.ID)
	}
	if segment.From == segment.To {
		return fmt.Errorf("segment %s loops to its source", segment.ID)
	}
	if segment.Length <= 0 {
		return fmt.Errorf("segment %s length must be positive", segment.ID)
	}
	if segment.MaximumPressure <= 0 {
		return fmt.Errorf("segment %s pressure limit must be positive", segment.ID)
	}
	n.segments[segment.ID] = segment
	n.outgoing[segment.From] = append(n.outgoing[segment.From], segment.ID)
	sort.Strings(n.outgoing[segment.From])
	return nil
}

func (n *Network) Validate() error {
	if len(n.stations) < 2 {
		return errors.New("network needs at least two stations")
	}
	if len(n.segments) == 0 {
		return errors.New("network needs at least one segment")
	}
	launchers := 0
	receivers := 0
	for _, station := range n.stations {
		if station.Role == model.StationLauncher {
			launchers++
		}
		if station.Role == model.StationReceiver {
			receivers++
		}
	}
	if launchers == 0 || receivers == 0 {
		return errors.New("network needs a launcher and receiver")
	}
	for _, segment := range n.segments {
		from := n.stations[segment.From]
		to := n.stations[segment.To]
		if from.Kilometer >= to.Kilometer {
			return fmt.Errorf("segment %s does not advance downstream", segment.ID)
		}
		limit := from.PressureLimit
		if to.PressureLimit < limit {
			limit = to.PressureLimit
		}
		if segment.MaximumPressure > limit {
			return fmt.Errorf("segment %s limit exceeds adjacent station limit", segment.ID)
		}
	}
	return nil
}

func (n *Network) Stations() []model.Station {
	out := make([]model.Station, 0, len(n.stations))
	for _, station := range n.stations {
		out = append(out, station)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kilometer == out[j].Kilometer {
			return out[i].ID < out[j].ID
		}
		return out[i].Kilometer < out[j].Kilometer
	})
	return out
}

func (n *Network) Segments() []Segment {
	out := make([]Segment, 0, len(n.segments))
	for _, segment := range n.segments {
		out = append(out, segment)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (n *Network) Path(from, to string, requestedPressure float64) (Path, error) {
	if _, ok := n.stations[from]; !ok {
		return Path{}, fmt.Errorf("unknown source station %s", from)
	}
	if _, ok := n.stations[to]; !ok {
		return Path{}, fmt.Errorf("unknown destination station %s", to)
	}
	if requestedPressure <= 0 {
		return Path{}, errors.New("requested pressure must be positive")
	}
	queue := candidateHeap{}
	heap.Init(&queue)
	heap.Push(&queue, &pathCandidate{station: from, distance: 0, pressure: requestedPressure, stations: []string{from}, segments: []string{}})
	best := make(map[string]float64)
	for queue.Len() > 0 {
		candidate := heap.Pop(&queue).(*pathCandidate)
		if distance, ok := best[candidate.station]; ok && distance <= candidate.distance {
			continue
		}
		best[candidate.station] = candidate.distance
		if candidate.station == to {
			return Path{Stations: candidate.stations, Segments: candidate.segments, Length: candidate.distance, MaximumPressure: candidate.pressure}, nil
		}
		for _, segmentID := range n.outgoing[candidate.station] {
			segment := n.segments[segmentID]
			if !segment.Enabled || requestedPressure > segment.MaximumPressure {
				continue
			}
			pressure := candidate.pressure
			if segment.MaximumPressure < pressure {
				pressure = segment.MaximumPressure
			}
			heap.Push(&queue, &pathCandidate{station: segment.To, distance: candidate.distance + segment.Length, pressure: pressure, stations: appendCopy(candidate.stations, segment.To), segments: appendCopy(candidate.segments, segment.ID)})
		}
	}
	return Path{}, fmt.Errorf("no enabled route from %s to %s at pressure %.2f", from, to, requestedPressure)
}

func (n *Network) Constraint(from, to string) (model.LineConstraint, error) {
	fromStation, ok := n.stations[from]
	if !ok {
		return model.LineConstraint{}, fmt.Errorf("unknown source station %s", from)
	}
	toStation, ok := n.stations[to]
	if !ok {
		return model.LineConstraint{}, fmt.Errorf("unknown destination station %s", to)
	}
	path, err := n.Path(from, to, minimumPressure(fromStation.PressureLimit, toStation.PressureLimit))
	if err != nil {
		return model.LineConstraint{}, err
	}
	ceiling := math.Inf(1)
	for _, stationID := range path.Stations {
		if limit := n.stations[stationID].PressureLimit; limit < ceiling {
			ceiling = limit
		}
	}
	for _, segmentID := range path.Segments {
		if limit := n.segments[segmentID].MaximumPressure; limit < ceiling {
			ceiling = limit
		}
	}
	return model.LineConstraint{
		Stations:    path.Stations,
		Segments:    path.Segments,
		MaxPressure: ceiling,
	}, nil
}

func appendCopy(values []string, value string) []string {
	out := make([]string, len(values), len(values)+1)
	copy(out, values)
	return append(out, value)
}

func (n *Network) RouteStates() []model.RouteState {
	launchers := make([]model.Station, 0)
	receivers := make([]model.Station, 0)
	for _, station := range n.Stations() {
		if station.Role == model.StationLauncher {
			launchers = append(launchers, station)
		}
		if station.Role == model.StationReceiver {
			receivers = append(receivers, station)
		}
	}
	states := make([]model.RouteState, 0)
	for _, launcher := range launchers {
		for _, receiver := range receivers {
			path, err := n.Path(launcher.ID, receiver.ID, minimumPressure(launcher.PressureLimit, receiver.PressureLimit))
			if err != nil || len(path.Segments) == 0 {
				continue
			}
			states = append(states, model.RouteState{ID: "main-" + launcher.ID + "-" + receiver.ID, FromStation: launcher.ID, ToStation: receiver.ID, UpdatedAt: time.Now().UTC()})
		}
	}
	return states
}

func minimumPressure(left, right float64) float64 {
	if left < right {
		return left
	}
	return right
}

func DefaultNetwork() *Network {
	stations := []model.Station{
		{ID: "station-a", Name: "North Launcher", Role: model.StationLauncher, Kilometer: 0, PressureLimit: 12.5, Enabled: true},
		{ID: "station-b", Name: "Central Block Valve", Role: model.StationIntermediate, Kilometer: 86.4, PressureLimit: 11.8, Enabled: true},
		{ID: "station-c", Name: "South Receiver", Role: model.StationReceiver, Kilometer: 173.2, PressureLimit: 11.2, Enabled: true},
	}
	segments := []Segment{
		{ID: "segment-a-b", From: "station-a", To: "station-b", Length: 86.4, MaximumPressure: 11.8, Enabled: true},
		{ID: "segment-b-c", From: "station-b", To: "station-c", Length: 86.8, MaximumPressure: 11.2, Enabled: true},
	}
	network, err := NewNetwork(stations, segments)
	if err != nil {
		panic(err)
	}
	return network
}
