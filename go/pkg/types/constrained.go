package types

import (
	"encoding/json"
	"fmt"
	"math"
	"time"
)

// Score is a float64 constrained to [0.0, 1.0].
type Score struct{ value float64 }

// NewScore creates a Score. Returns an error if v is outside [0.0, 1.0].
func NewScore(v float64) (Score, error) {
	if math.IsNaN(v) || v < 0.0 || v > 1.0 {
		return Score{}, &OutOfRangeError{Field: "Score", Value: v, Min: 0.0, Max: 1.0}
	}
	return Score{value: v}, nil
}

// MustScore creates a Score. Panics if v is outside [0.0, 1.0].
func MustScore(v float64) Score {
	s, err := NewScore(v)
	if err != nil {
		panic(err)
	}
	return s
}

// Value returns the underlying float64.
func (s Score) Value() float64 { return s.value }

// String returns the string representation.
func (s Score) String() string { return fmt.Sprintf("%g", s.value) }

// MarshalJSON marshals to the underlying float64.
func (s Score) MarshalJSON() ([]byte, error) { return json.Marshal(s.value) }

// UnmarshalJSON unmarshals from a float64, validating the range.
func (s *Score) UnmarshalJSON(b []byte) error {
	var v float64
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	score, err := NewScore(v)
	if err != nil {
		return err
	}
	*s = score
	return nil
}

// Weight is a float64 constrained to [-1.0, 1.0].
type Weight struct{ value float64 }

// NewWeight creates a Weight. Returns an error if v is outside [-1.0, 1.0].
func NewWeight(v float64) (Weight, error) {
	if math.IsNaN(v) || v < -1.0 || v > 1.0 {
		return Weight{}, &OutOfRangeError{Field: "Weight", Value: v, Min: -1.0, Max: 1.0}
	}
	return Weight{value: v}, nil
}

// MustWeight creates a Weight. Panics if v is outside [-1.0, 1.0].
func MustWeight(v float64) Weight {
	w, err := NewWeight(v)
	if err != nil {
		panic(err)
	}
	return w
}

// Value returns the underlying float64.
func (w Weight) Value() float64 { return w.value }

// String returns the string representation.
func (w Weight) String() string { return fmt.Sprintf("%g", w.value) }

// MarshalJSON marshals to the underlying float64.
func (w Weight) MarshalJSON() ([]byte, error) { return json.Marshal(w.value) }

// UnmarshalJSON unmarshals from a float64, validating the range.
func (w *Weight) UnmarshalJSON(b []byte) error {
	var v float64
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	weight, err := NewWeight(v)
	if err != nil {
		return err
	}
	*w = weight
	return nil
}

// Activation is a float64 constrained to [0.0, 1.0].
type Activation struct{ value float64 }

// NewActivation creates an Activation. Returns an error if v is outside [0.0, 1.0].
func NewActivation(v float64) (Activation, error) {
	if math.IsNaN(v) || v < 0.0 || v > 1.0 {
		return Activation{}, &OutOfRangeError{Field: "Activation", Value: v, Min: 0.0, Max: 1.0}
	}
	return Activation{value: v}, nil
}

// MustActivation creates an Activation. Panics if v is outside [0.0, 1.0].
func MustActivation(v float64) Activation {
	a, err := NewActivation(v)
	if err != nil {
		panic(err)
	}
	return a
}

// Value returns the underlying float64.
func (a Activation) Value() float64 { return a.value }

// String returns the string representation.
func (a Activation) String() string { return fmt.Sprintf("%g", a.value) }

// MarshalJSON marshals to the underlying float64.
func (a Activation) MarshalJSON() ([]byte, error) { return json.Marshal(a.value) }

// UnmarshalJSON unmarshals from a float64, validating the range.
func (a *Activation) UnmarshalJSON(b []byte) error {
	var v float64
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	act, err := NewActivation(v)
	if err != nil {
		return err
	}
	*a = act
	return nil
}

// Layer is an int constrained to [0, 13].
type Layer struct{ value int }

// NewLayer creates a Layer. Returns an error if v is outside [0, 13].
func NewLayer(v int) (Layer, error) {
	if v < 0 || v > 13 {
		return Layer{}, &IntOutOfRangeError{Field: "Layer", Value: v, Min: 0, Max: 13}
	}
	return Layer{value: v}, nil
}

// MustLayer creates a Layer. Panics if v is outside [0, 13].
func MustLayer(v int) Layer {
	l, err := NewLayer(v)
	if err != nil {
		panic(err)
	}
	return l
}

// Value returns the underlying int.
func (l Layer) Value() int { return l.value }

// String returns the string representation.
func (l Layer) String() string { return fmt.Sprintf("%d", l.value) }

// MarshalJSON marshals to the underlying int.
func (l Layer) MarshalJSON() ([]byte, error) { return json.Marshal(l.value) }

// UnmarshalJSON unmarshals from an int, validating the range.
func (l *Layer) UnmarshalJSON(b []byte) error {
	var v int
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	layer, err := NewLayer(v)
	if err != nil {
		return err
	}
	*l = layer
	return nil
}

// Cadence is an int constrained to [1, ∞).
type Cadence struct{ value int }

// NewCadence creates a Cadence. Returns an error if v is less than 1.
func NewCadence(v int) (Cadence, error) {
	if v < 1 {
		return Cadence{}, &IntOutOfRangeError{Field: "Cadence", Value: v, Min: 1, Max: math.MaxInt}
	}
	return Cadence{value: v}, nil
}

// MustCadence creates a Cadence. Panics if v is less than 1.
func MustCadence(v int) Cadence {
	c, err := NewCadence(v)
	if err != nil {
		panic(err)
	}
	return c
}

// Value returns the underlying int.
func (c Cadence) Value() int { return c.value }

// String returns the string representation.
func (c Cadence) String() string { return fmt.Sprintf("%d", c.value) }

// MarshalJSON marshals to the underlying int.
func (c Cadence) MarshalJSON() ([]byte, error) { return json.Marshal(c.value) }

// UnmarshalJSON unmarshals from an int, validating the range.
func (c *Cadence) UnmarshalJSON(b []byte) error {
	var v int
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	cad, err := NewCadence(v)
	if err != nil {
		return err
	}
	*c = cad
	return nil
}

// Tick is an int constrained to [0, ∞).
type Tick struct{ value int }

// NewTick creates a Tick. Returns an error if v is negative.
func NewTick(v int) (Tick, error) {
	if v < 0 {
		return Tick{}, &IntOutOfRangeError{Field: "Tick", Value: v, Min: 0, Max: math.MaxInt}
	}
	return Tick{value: v}, nil
}

// MustTick creates a Tick. Panics if v is negative.
func MustTick(v int) Tick {
	t, err := NewTick(v)
	if err != nil {
		panic(err)
	}
	return t
}

// Value returns the underlying int.
func (t Tick) Value() int { return t.value }

// String returns the string representation.
func (t Tick) String() string { return fmt.Sprintf("%d", t.value) }

// MarshalJSON marshals to the underlying int.
func (t Tick) MarshalJSON() ([]byte, error) { return json.Marshal(t.value) }

// UnmarshalJSON unmarshals from an int, validating the range.
func (t *Tick) UnmarshalJSON(b []byte) error {
	var v int
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	tick, err := NewTick(v)
	if err != nil {
		return err
	}
	*t = tick
	return nil
}

// Duration is an int64 of nanoseconds constrained to [0, ∞).
type Duration struct{ value int64 }

// NewDuration creates a Duration. Returns an error if v is negative.
func NewDuration(v int64) (Duration, error) {
	if v < 0 {
		return Duration{}, &OutOfRangeError{Field: "Duration", Value: float64(v), Min: 0, Max: math.MaxFloat64}
	}
	return Duration{value: v}, nil
}

// MustDuration creates a Duration. Panics if v is negative.
func MustDuration(v int64) Duration {
	d, err := NewDuration(v)
	if err != nil {
		panic(err)
	}
	return d
}

// Value returns the underlying int64 nanoseconds.
func (d Duration) Value() int64 { return d.value }

// String returns the string representation.
func (d Duration) String() string { return fmt.Sprintf("%dns", d.value) }

// MarshalJSON marshals to the underlying int64.
func (d Duration) MarshalJSON() ([]byte, error) { return json.Marshal(d.value) }

// UnmarshalJSON unmarshals from an int64, validating the range.
func (d *Duration) UnmarshalJSON(b []byte) error {
	var v int64
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	dur, err := NewDuration(v)
	if err != nil {
		return err
	}
	*d = dur
	return nil
}

// FieldPath is a dot-separated path into a data structure.
// Each segment must match [a-zA-Z_][a-zA-Z0-9_]*.
type FieldPath struct{ value string }

// NewFieldPath creates a FieldPath. Returns an error if the path is empty
// or contains invalid segments.
func NewFieldPath(v string) (FieldPath, error) {
	if v == "" {
		return FieldPath{}, &EmptyRequiredError{Field: "FieldPath"}
	}
	for i, start := 0, 0; i <= len(v); i++ {
		if i == len(v) || v[i] == '.' {
			seg := v[start:i]
			if !isValidFieldSegment(seg) {
				return FieldPath{}, &InvalidFormatError{
					Field:    "FieldPath",
					Value:    v,
					Expected: "dot-separated segments matching [a-zA-Z_][a-zA-Z0-9_]*",
				}
			}
			start = i + 1
		}
	}
	return FieldPath{value: v}, nil
}

// MustFieldPath creates a FieldPath. Panics on invalid input.
func MustFieldPath(v string) FieldPath {
	fp, err := NewFieldPath(v)
	if err != nil {
		panic(err)
	}
	return fp
}

// Value returns the underlying string.
func (fp FieldPath) Value() string { return fp.value }

// String returns the string representation.
func (fp FieldPath) String() string { return fp.value }

// Timestamp wraps time.Time for domain timestamps. Always UTC, nanosecond precision.
type Timestamp struct{ value time.Time }

// NewTimestamp creates a Timestamp. Normalizes to UTC.
func NewTimestamp(t time.Time) Timestamp {
	return Timestamp{value: t.UTC()}
}

// Now returns the current time as a Timestamp.
func Now() Timestamp {
	return Timestamp{value: time.Now().UTC()}
}

// ZeroTimestamp returns the zero-value timestamp.
func ZeroTimestamp() Timestamp {
	return Timestamp{}
}

// Value returns the underlying time.Time (UTC).
func (t Timestamp) Value() time.Time { return t.value }

// UnixNano returns nanoseconds since epoch.
func (t Timestamp) UnixNano() int64 { return t.value.UnixNano() }

// IsZero returns true if this is a zero timestamp.
func (t Timestamp) IsZero() bool { return t.value.IsZero() }

// String returns the RFC3339Nano representation.
func (t Timestamp) String() string { return t.value.Format(time.RFC3339Nano) }

// MarshalJSON marshals the timestamp as an RFC3339Nano string.
func (t Timestamp) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.value.Format(time.RFC3339Nano))
}

// UnmarshalJSON unmarshals an RFC3339Nano string.
func (t *Timestamp) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	parsed, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return err
	}
	t.value = parsed.UTC()
	return nil
}

func isValidFieldSegment(seg string) bool {
	if len(seg) == 0 {
		return false
	}
	c := seg[0]
	if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_') {
		return false
	}
	for i := 1; i < len(seg); i++ {
		c = seg[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return true
}
