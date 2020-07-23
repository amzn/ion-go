package ion

import (
	"fmt"
	"time"
)

// TimestampPrecision is for tracking the precision of a timestamp
type TimestampPrecision uint8

// Possible TimestampPrecision values
const (
	NoPrecision TimestampPrecision = iota
	Year
	Month
	Day
	Minute
	Second
	Nanosecond
)

func (tp TimestampPrecision) String() string {
	switch tp {
	case NoPrecision:
		return "<no precision>"
	case Year:
		return "Year"
	case Month:
		return "Month"
	case Day:
		return "Day"
	case Minute:
		return "Minute"
	case Second:
		return "Second"
	case Nanosecond:
		return "Nanosecond"
	default:
		return fmt.Sprintf("<unknown precision %v>", uint8(tp))
	}
}

func (tp TimestampPrecision) formatString(hasOffset bool) string {
	switch tp {
	case NoPrecision:
		return ""
	case Year:
		return "2006T"
	case Month:
		return "2006-01T"
	case Day:
		return "2006-01-02T"
	case Minute:
		if hasOffset {
			return "2006-01-02T15:04Z07:00"
		}
		return "2006-01-02T15:04Z"
	case Second:
		if hasOffset {
			return "2006-01-02T15:04:05Z07:00"
		}
		return "2006-01-02T15:04:05Z"
	case Nanosecond:
		if hasOffset {
			return "2006-01-02T15:04:05.999999999Z07:00"
		}
		return "2006-01-02T15:04:05.999999999Z"
	}

	return time.RFC3339Nano
}

// TimestampKind is for tracking the kind of timestamp
type TimestampKind uint8

// Possible TimestampPrecision values
const (
	Unspecified TimestampKind = iota
	UTC
	Local
)

// Timestamp struct
type Timestamp struct {
	DateTime  time.Time
	precision TimestampPrecision
	hasOffset bool
	kind      TimestampKind
}

// NewSimpleTimestamp constructor
func NewSimpleTimestamp(dateTime time.Time, precision TimestampPrecision) Timestamp {
	return Timestamp{dateTime, precision, false, Unspecified}
}

// NewTimestamp constructor
func NewTimestamp(dateTime time.Time, precision TimestampPrecision, hasOffset bool, kind TimestampKind) Timestamp {
	if precision <= Day {
		// offset does not apply to Timestamps with Year, Month, or Day precision
		return Timestamp{dateTime, precision, false, kind}
	}
	return Timestamp{dateTime, precision, hasOffset, kind}
}

// NewTimestampFromStr constructor
func NewTimestampFromStr(dateStr string, precision TimestampPrecision, hasOffset bool, kind TimestampKind) (Timestamp, error) {
	dateTime, err := time.Parse(precision.formatString(hasOffset), dateStr)
	if err != nil {
		return Timestamp{time.Time{}, NoPrecision, false, Unspecified}, err
	}

	return NewTimestamp(dateTime, precision, hasOffset, kind), nil
}

func emptyTimestamp() Timestamp {
	return Timestamp{time.Time{}, NoPrecision, false, Unspecified}
}

// Format returns a formatted Timestamp string.
func (ts *Timestamp) Format() string {
	return ts.DateTime.Format(ts.precision.formatString(ts.hasOffset))
}

// Equal figures out if two timestamps are equal for each component.
func (ts *Timestamp) Equal(ts1 Timestamp) bool {
	return ts.DateTime.Equal(ts1.DateTime) &&
		ts.precision == ts1.precision &&
		ts.hasOffset == ts1.hasOffset &&
		ts.kind == ts1.kind
}

// Equivalent figures out if two timestamps have equal DateTime, precision, and kind.
// eg. "2004-12-11T12:10" and "2004-12-11T12:10+00:00" are considered equivalent to each other
// even though one has an offset and the other does not.
func (ts *Timestamp) Equivalent(ts1 Timestamp) bool {
	return ts.DateTime.Equal(ts1.DateTime) &&
		ts.precision == ts1.precision &&
		ts.kind == ts1.kind
}

// SetLocation sets the location for the internal time object.
func (ts *Timestamp) SetLocation(loc *time.Location) {
	ts.DateTime = ts.DateTime.In(loc)
}
