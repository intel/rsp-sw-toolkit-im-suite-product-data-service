package metrics

import (
	"sync"
	"time"
)

// GaugeCollection holds a collection of int64 values and timestamps that can be accumulated.
type GaugeCollection interface {
	Snapshot() GaugeCollection
	Add(int64)
	Readings() []GaugeReading
	IsSet() bool
	Clear()
}

type GaugeReading struct {
	Reading     int64
	Time        time.Time
}

// GetOrRegisterGaugeCollection returns an existing Gauge or constructs and registers a
// new StandardGaugeCollection.
func GetOrRegisterGaugeCollection(name string, r Registry) GaugeCollection {
	if nil == r {
		r = DefaultRegistry
	}
	return r.GetOrRegister(name, NewGaugeCollection).(GaugeCollection)
}

// NewGaugeCollection constructs a new StandardGaugeCollection.
func NewGaugeCollection() GaugeCollection {
	if UseNilMetrics {
		return NilGaugeCollection{}
	}
	return &StandardGaugeCollection{}
}

// NewRegisteredGaugeCollection constructs and registers a new StandardGaugeCollection.
func NewRegisteredGaugeCollection(name string, r Registry) GaugeCollection {
	c := NewGaugeCollection()
	if nil == r {
		r = DefaultRegistry
	}
	LogErrorIfAny(r.Register(name, c))
	return c
}

// GaugeCollectionSnapshot is a read-only copy of another GaugeCollection.
type GaugeCollectionSnapshot struct {
	readings []GaugeReading
}

// Snapshot returns the snapshot.
func (g GaugeCollectionSnapshot) Snapshot() GaugeCollection { return g }

// Add panics. Suppose to be read-only
func (GaugeCollectionSnapshot) Add(int64) {
	panic("Add called on a GaugeCollectionSnapshot")
}

// Readings returns the collect of readings at the time the snapshot was taken.
func (g GaugeCollectionSnapshot) Readings() []GaugeReading {
	return g.readings
}

// IsSet returns true if the collection has any values
func (g GaugeCollectionSnapshot) IsSet() bool {
	return len(g.readings) > 0
}

func (g GaugeCollectionSnapshot) Clear() {
	panic("Clear called on a GaugeSnapshot")
}

// NilGauge is a no-op Gauge.
type NilGaugeCollection struct{}

// Snapshot is a no-op.
func (NilGaugeCollection) Snapshot() GaugeCollection { return NilGaugeCollection{} }

// Add is a no-op.
func (NilGaugeCollection) Add(v int64) {}

// Readings is a no-op.
func (NilGaugeCollection) Readings() []GaugeReading { return nil }

// IsSet is a no-op.
func (NilGaugeCollection) IsSet() bool { return false}

// Clear is a no-op.
func (NilGaugeCollection) Clear() { }

// StandardGaugeCollection is the standard implementation of a GaugeCollection and uses the
// sync.Mutex to manage the struct values.
type StandardGaugeCollection struct {
	mutex sync.Mutex
	readings []GaugeReading
}

// Snapshot returns a read-only copy of the GaugeCollection.
func (g *StandardGaugeCollection) Snapshot() GaugeCollection {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	readings :=	make([]GaugeReading, len(g.readings))
	copy(readings,g.readings)
	return GaugeCollectionSnapshot {readings}
}

// Add Adds a reading to the collection.
func (g *StandardGaugeCollection) Add(reading int64) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	gaugeReading := GaugeReading{ Reading: reading, Time: time.Now()}
	g.readings = append(g.readings, gaugeReading)
}

// Readings returns a copy of the collection of readings.
func (g *StandardGaugeCollection) Readings() []GaugeReading {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	readings :=	make([]GaugeReading, len(g.readings))
	copy(readings,g.readings)
	return readings
}

// IsSet returns true if the collection has any values
func (g *StandardGaugeCollection) IsSet() bool {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	return len(g.readings) > 0
}

func (g *StandardGaugeCollection) Clear() {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	g.readings = []GaugeReading{}
}

