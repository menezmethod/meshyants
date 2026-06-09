// Package crdt implements the three allowed CRDT families per AllowNaiveCRDT:
// pheromone_summary, capability_summary, and sensor_aggregate (docs/v1/04-security-resilience-and-consensus.md).
package crdt

import (
	"encoding/json"
	"sort"
	"time"
)

// --- Pheromone Summary (Last-Write-Wins per record ID) ---

// PheromoneSummary uses last-write-wins to merge pheromone records.
type PheromoneSummary struct {
	Records map[string]PheromoneEntry // keyed by record ID
}

type PheromoneEntry struct {
	Kind      string    // PheromoneKind string
	Subject   string
	Issuer    string
	IssuedAt  time.Time
	Strength  float64
}

func NewPheromoneSummary() *PheromoneSummary {
	return &PheromoneSummary{Records: make(map[string]PheromoneEntry)}
}

// Merge returns a new summary with the latest record per ID winning.
func (m *PheromoneSummary) Merge(other any) *PheromoneSummary {
	out := &PheromoneSummary{Records: make(map[string]PheromoneEntry)}
	for k, v := range m.Records {
		out.Records[k] = v
	}
	if o, ok := other.(*PheromoneSummary); ok {
		for k, v := range o.Records {
			if existing, ok := out.Records[k]; !ok || v.IssuedAt.After(existing.IssuedAt) {
				out.Records[k] = v
			}
		}
	}
	return out
}

func (m *PheromoneSummary) Set(kind, subject, issuer string, issuedAt time.Time, strength float64) {
	m.Records[subject+"_"+issuer] = PheromoneEntry{
		Kind: kind, Subject: subject, Issuer: issuer, IssuedAt: issuedAt, Strength: strength,
	}
}

func (m *PheromoneSummary) SizeBytes() int {
	b, _ := json.Marshal(m.Records)
	return len(b)
}

// --- Capability Summary (Set Union) ---

// CapabilitySummary merges capability advertisements via set union.
type CapabilitySummary struct {
	Tiers           map[string]bool // RuntimeTier names
	Accelerators    map[string]bool
	TransportProfs  map[string]bool
	QueueDepth      uint32
}

func NewCapabilitySummary() *CapabilitySummary {
	return &CapabilitySummary{
		Tiers:          make(map[string]bool),
		Accelerators:   make(map[string]bool),
		TransportProfs: make(map[string]bool),
	}
}

// Merge unions all sets and takes the max queue depth.
func (m *CapabilitySummary) Merge(other any) *CapabilitySummary {
	out := &CapabilitySummary{
		Tiers:           make(map[string]bool),
		Accelerators:    make(map[string]bool),
		TransportProfs: make(map[string]bool),
	}
	for k, v := range m.Tiers {
		out.Tiers[k] = v
	}
	for k, v := range m.Accelerators {
		out.Accelerators[k] = v
	}
	for k, v := range m.TransportProfs {
		out.TransportProfs[k] = v
	}
	out.QueueDepth = m.QueueDepth
	if o, ok := other.(*CapabilitySummary); ok {
		for k, v := range o.Tiers {
			out.Tiers[k] = v
		}
		for k, v := range o.Accelerators {
			out.Accelerators[k] = v
		}
		for k, v := range o.TransportProfs {
			out.TransportProfs[k] = v
		}
		if o.QueueDepth > out.QueueDepth {
			out.QueueDepth = o.QueueDepth
		}
	}
	return out
}

func (m *CapabilitySummary) AddTier(t string)         { m.Tiers[t] = true }
func (m *CapabilitySummary) AddAccelerator(a string)   { m.Accelerators[a] = true }
func (m *CapabilitySummary) AddTransportProfile(p string) { m.TransportProfs[p] = true }

// --- Sensor Aggregate (Last-Write-Wins per key) ---

// SensorAggregate merges sensor readings with last-write-wins per key.
type SensorAggregate struct {
	Values map[string]SensorValue // keyed by metric name
}

type SensorValue struct {
	Val   float64
	Unit  string
	At    time.Time
}

func NewSensorAggregate() *SensorAggregate {
	return &SensorAggregate{Values: make(map[string]SensorValue)}
}

// Merge takes the latest value per key by timestamp.
func (m *SensorAggregate) Merge(other any) *SensorAggregate {
	out := &SensorAggregate{Values: make(map[string]SensorValue)}
	for k, v := range m.Values {
		out.Values[k] = v
	}
	if o, ok := other.(*SensorAggregate); ok {
		for k, v := range o.Values {
			if existing, ok := out.Values[k]; !ok || v.At.After(existing.At) {
				out.Values[k] = v
			}
		}
	}
	return out
}

func (m *SensorAggregate) Set(key string, val float64, unit string, at time.Time) {
	m.Values[key] = SensorValue{Val: val, Unit: unit, At: at}
}

func (m *SensorAggregate) Keys() []string {
	keys := make([]string, 0, len(m.Values))
	for k := range m.Values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// --- Registry for permitted CRDT families ---

// Family returns the appropriate CRDT instance for the given family name.
// Returns nil if the family is not in the allowed set.
func Family(familyName string) interface{} {
	switch familyName {
	case "pheromone_summary":
		return NewPheromoneSummary()
	case "capability_summary":
		return NewCapabilitySummary()
	case "sensor_aggregate":
		return NewSensorAggregate()
	default:
		return nil
	}
}

// Merge is implemented by each CRDT type.
type Merge interface {
	Merge(other any) any
}
