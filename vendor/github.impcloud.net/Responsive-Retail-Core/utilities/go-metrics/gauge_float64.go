package metrics

import "sync"

// GaugeFloat64 holds a float64 value that can be set arbitrarily.
type GaugeFloat64 interface {
	Snapshot() GaugeFloat64
	Update(float64)
	Value() float64
	IsSet() bool
	Clear()
}

// GetOrRegisterGaugeFloat64 returns an existing GaugeFloat64 or constructs and registers a
// new StandardGaugeFloat64.
func GetOrRegisterGaugeFloat64(name string, r Registry) GaugeFloat64 {
	if nil == r {
		r = DefaultRegistry
	}
	return r.GetOrRegister(name, NewGaugeFloat64()).(GaugeFloat64)
}

// NewGaugeFloat64 constructs a new StandardGaugeFloat64.
func NewGaugeFloat64() GaugeFloat64 {
	if UseNilMetrics {
		return NilGaugeFloat64{}
	}
	return &StandardGaugeFloat64{
		value: 0.0,
		isSet: false,
	}
}

// NewRegisteredGaugeFloat64 constructs and registers a new StandardGaugeFloat64.
func NewRegisteredGaugeFloat64(name string, r Registry) GaugeFloat64 {
	c := NewGaugeFloat64()
	if nil == r {
		r = DefaultRegistry
	}
	LogErrorIfAny(r.Register(name, c))
	return c
}

// NewFunctionalGaugeFloat64 constructs a new FunctionalGauge.
func NewFunctionalGaugeFloat64(f func() float64, i func() bool) GaugeFloat64 {
	if UseNilMetrics {
		return NilGaugeFloat64{}
	}
	return &FunctionalGaugeFloat64{
		value: f,
		isSet: i,
	}
}

// NewRegisteredFunctionalGaugeFloat64 constructs and registers a new StandardGauge.
func NewRegisteredFunctionalGaugeFloat64(name string, r Registry, f func() float64, i func() bool) GaugeFloat64 {
	c := NewFunctionalGaugeFloat64(f, i)
	if nil == r {
		r = DefaultRegistry
	}
	LogErrorIfAny(r.Register(name, c))
	return c
}

// GaugeFloat64Snapshot is a read-only copy of another GaugeFloat64.
type GaugeFloat64Snapshot struct {
	value float64
	isSet bool
}

// Snapshot returns the snapshot.
func (g GaugeFloat64Snapshot) Snapshot() GaugeFloat64 { return g }

// Update panics.
func (GaugeFloat64Snapshot) Update(float64) {
	panic("Update called on a GaugeFloat64Snapshot")
}

// Value returns the value at the time the snapshot was taken.
func (g GaugeFloat64Snapshot) Value() float64 {
	return g.value
}

// IsSet returns the isSet at the time the snapshot was taken.
func (g GaugeFloat64Snapshot) IsSet() bool {
	return g.isSet
}

// Clear is not supposed to call for GaugeFloat64Snapshot.
func (g GaugeFloat64Snapshot) Clear() {
	panic("Clear called on a GaugeFloat64Snapshot")
}

// NilGaugeFloat64 is a no-op Gauge.
type NilGaugeFloat64 struct{}

// Snapshot is a no-op.
func (NilGaugeFloat64) Snapshot() GaugeFloat64 { return NilGaugeFloat64{} }

// Update is a no-op.
func (NilGaugeFloat64) Update(v float64) {}

// Value is a no-op.
func (NilGaugeFloat64) Value() float64 { return 0.0 }

// IsSet is a no-op.
func (NilGaugeFloat64) IsSet() bool { return false }

// Clear is a no-op.
func (NilGaugeFloat64) Clear() {}

// StandardGaugeFloat64 is the standard implementation of a GaugeFloat64 and uses
// sync.Mutex to manage the struct values.
type StandardGaugeFloat64 struct {
	mutex sync.Mutex
	value float64
	isSet bool
}

// Snapshot returns a read-only copy of the gauge.
func (g *StandardGaugeFloat64) Snapshot() GaugeFloat64 {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	return GaugeFloat64Snapshot{g.value, g.isSet}
}

// Update updates the gauge's value.
func (g *StandardGaugeFloat64) Update(v float64) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	g.value = v
	g.isSet = true
}

// Value returns the gauge's current value.
func (g *StandardGaugeFloat64) Value() float64 {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	return g.value
}

// IsSet returns the gauge's isSet value.
func (g *StandardGaugeFloat64) IsSet() bool {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	return g.isSet
}

// Clear resets the gauge's value to the defaults
func (g *StandardGaugeFloat64) Clear() {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	g.value = 0.0
	g.isSet = false
}

// FunctionalGaugeFloat64 returns value from given function
type FunctionalGaugeFloat64 struct {
	value func() float64
	isSet func() bool
}

// Value returns the gauge's current value.
func (g FunctionalGaugeFloat64) Value() float64 {
	return g.value()
}

// IsSet returns the gauge's isSet value.
func (g FunctionalGaugeFloat64) IsSet() bool {
	return g.isSet()
}

// Snapshot returns the snapshot.
func (g FunctionalGaugeFloat64) Snapshot() GaugeFloat64 {
	return GaugeFloat64Snapshot{
		g.Value(),
		g.IsSet(),
	}
}

// Update panics.
func (FunctionalGaugeFloat64) Update(float64) {
	panic("Update called on a FunctionalGaugeFloat64")
}

// Clear is not supposed to call on FunctionalGaugeFloat64.
func (FunctionalGaugeFloat64) Clear() {
	panic("Clear called on a FunctionalGaugeFloat64")
}
