package metrics

import (
	"sync"
)

// Gauge holds an int64 value that can be set arbitrarily.
type Gauge interface {
	Snapshot() Gauge
	Update(int64)
	Value() int64
	IsSet() bool
	Clear()
}

// GetOrRegisterGauge returns an existing Gauge or constructs and registers a
// new StandardGauge.
func GetOrRegisterGauge(name string, r Registry) Gauge {
	if nil == r {
		r = DefaultRegistry
	}
	return r.GetOrRegister(name, NewGauge).(Gauge)
}

// NewGauge constructs a new StandardGauge.
func NewGauge() Gauge {
	if UseNilMetrics {
		return NilGauge{}
	}
	return &StandardGauge{
		value: 0,
		isSet: false,
	}
}

// NewRegisteredGauge constructs and registers a new StandardGauge.
func NewRegisteredGauge(name string, r Registry) Gauge {
	c := NewGauge()
	if nil == r {
		r = DefaultRegistry
	}
	LogErrorIfAny(r.Register(name, c))
	return c
}

// NewFunctionalGauge constructs a new FunctionalGauge.
func NewFunctionalGauge(f func() int64, i func() bool) Gauge {
	if UseNilMetrics {
		return NilGauge{}
	}
	return &FunctionalGauge{
		value: f,
		isSet: i,
	}
}

// NewRegisteredFunctionalGauge constructs and registers a new StandardGauge.
func NewRegisteredFunctionalGauge(name string, r Registry, f func() int64, i func() bool) Gauge {
	c := NewFunctionalGauge(f, i)
	if nil == r {
		r = DefaultRegistry
	}
	LogErrorIfAny(r.Register(name, c))
	return c
}

// GaugeSnapshot is a read-only copy of another Gauge.
type GaugeSnapshot struct {
	value int64
	isSet bool
}

// Snapshot returns the snapshot.
func (g GaugeSnapshot) Snapshot() Gauge { return g }

// Update panics.
func (GaugeSnapshot) Update(int64) {
	panic("Update called on a GaugeSnapshot")
}

// Value returns the value at the time the snapshot was taken.
func (g GaugeSnapshot) Value() int64 {
	return g.value
}

// IsSet returns whether gauge snapshot is set
func (g GaugeSnapshot) IsSet() bool {
	return g.isSet
}

// Clear is not supposed to call for GaugeSnapshot
func (g GaugeSnapshot) Clear() {
	panic("Clear called on a GaugeSnapshot")
}

// NilGauge is a no-op Gauge.
type NilGauge struct{}

// Snapshot is a no-op.
func (NilGauge) Snapshot() Gauge { return NilGauge{} }

// Update is a no-op.
func (NilGauge) Update(v int64) {}

// Value is a no-op.
func (NilGauge) Value() int64 { return 0 }

// IsSet is a no-op.
func (NilGauge) IsSet() bool { return false }

// Clear is a no-op.
func (NilGauge) Clear() {}

// StandardGauge is the standard implementation of a Gauge and uses the
// sync.Mutex to manage the struct values.
type StandardGauge struct {
	mutex sync.Mutex
	value int64
	isSet bool
}

// Snapshot returns a read-only copy of the gauge.
func (g *StandardGauge) Snapshot() Gauge {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	return GaugeSnapshot{g.value, g.isSet}
}

// Update updates the gauge's value.
func (g *StandardGauge) Update(v int64) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	g.value = v
	g.isSet = true
}

// Value returns the gauge's current value.
func (g *StandardGauge) Value() int64 {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	return g.value
}

// IsSet returns whether standard gauge is set
func (g *StandardGauge) IsSet() bool {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	return g.isSet
}

// Clear reset the standard gauge to its default state
func (g *StandardGauge) Clear() {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	g.value = 0
	g.isSet = false
}

// FunctionalGauge returns value from given function
type FunctionalGauge struct {
	value func() int64
	isSet func() bool
}

// Value returns the gauge's current value.
func (g FunctionalGauge) Value() int64 {
	return g.value()
}

// IsSet returns whether functional gauge is set
func (g FunctionalGauge) IsSet() bool {
	return g.isSet()
}

// Snapshot returns the snapshot.
func (g FunctionalGauge) Snapshot() Gauge {
	return GaugeSnapshot{
		g.Value(),
		g.IsSet(),
	}
}

// Update panics.
func (FunctionalGauge) Update(int64) {
	panic("Update called on a FunctionalGauge")
}

// Clear is not supposed to call for FunctionalGauge
func (FunctionalGauge) Clear() {
	panic("Clear called on a FunctionalGauge")
}
