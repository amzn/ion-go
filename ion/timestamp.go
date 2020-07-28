package ion

import (
	"fmt"
	"strings"
	"time"
)

// TimestampPrecision is for tracking the precision of a timestamp
type TimestampPrecision uint8

// Possible TimestampPrecision values
const (
	TimestampNoPrecision TimestampPrecision = iota
	TimestampPrecisionYear
	TimestampPrecisionMonth
	TimestampPrecisionDay
	TimestampPrecisionMinute
	TimestampPrecisionSecond
	TimestampPrecisionNanosecond
)

func (tp TimestampPrecision) String() string {
	switch tp {
	case TimestampNoPrecision:
		return "<no precision>"
	case TimestampPrecisionYear:
		return "Year"
	case TimestampPrecisionMonth:
		return "Month"
	case TimestampPrecisionDay:
		return "Day"
	case TimestampPrecisionMinute:
		return "Minute"
	case TimestampPrecisionSecond:
		return "Second"
	case TimestampPrecisionNanosecond:
		return "Nanosecond"
	default:
		return fmt.Sprintf("<unknown precision %v>", uint8(tp))
	}
}

// Layout returns a suitable format string to be used in time.Parse() or time.Format().
func (tp TimestampPrecision) Layout(kind TimezoneKind, precisionUnits uint8) string {
	switch tp {
	case TimestampPrecisionYear:
		return "2006T"
	case TimestampPrecisionMonth:
		return "2006-01T"
	case TimestampPrecisionDay:
		return "2006-01-02T"
	case TimestampPrecisionMinute:
		if kind == TimezoneUnspecified {
			return "2006-01-02T15:04-07:00"
		}
		return "2006-01-02T15:04Z07:00"
	case TimestampPrecisionSecond:
		if kind == TimezoneUnspecified {
			return "2006-01-02T15:04:05-07:00"
		}
		return "2006-01-02T15:04:05Z07:00"
	case TimestampPrecisionNanosecond:
		formatStr := "2006-01-02T15:04:05"

		if precisionUnits > 9 {
			precisionUnits = 9
		}

		if precisionUnits > 0 {
			formatStr += "."
			for i := uint8(0); i < precisionUnits; i++ {
				formatStr += "9"
			}
		}

		if kind == TimezoneUnspecified {
			formatStr += "-07:00"
		} else {
			formatStr += "Z07:00"
		}

		return formatStr
	}

	return time.RFC3339Nano
}

// TimezoneKind tracks the type of timezone.
type TimezoneKind uint8

const (
	// TimezoneUnspecified is for timestamps without a timezone such as dates with no time component (ie. Year/Month/Day precision).
	// Negative zero offsets (ie. yyyy-mm-ddThh:mm-00:00) are also considered Unspecified.
	TimezoneUnspecified TimezoneKind = iota

	// TimezoneUTC is for UTC timestamps and they are usually denoted with a trailing 'Z' (ie. yyyy-mm-ddThh:mmZ).
	// Timestamps with a positive zero offset (ie. yyyy-mm-ddThh:mm+00:00) are also considered UTC.
	TimezoneUTC

	// TimezoneLocal is for timestamps that have a non-zero offset from UTC (ie. 2001-02-03T04:05+08:30, 2009-05-18T16:20-04:00).
	TimezoneLocal
)

// Timestamp struct
type Timestamp struct {
	dateTime             time.Time
	precision            TimestampPrecision
	kind                 TimezoneKind
	numFractionalSeconds uint8
}

// NewDateTimestamp constructor meant for timestamps that only have a date portion (ie. no time portion).
func NewDateTimestamp(dateTime time.Time, precision TimestampPrecision) Timestamp {
	return Timestamp{dateTime, precision, TimezoneUnspecified, 0}
}

// NewTimestamp constructor
func NewTimestamp(dateTime time.Time, precision TimestampPrecision, kind TimezoneKind) Timestamp {
	if precision <= TimestampPrecisionDay {
		// Timestamps with Year, Month, or Day precision necessarily have TimezoneUnspecified timezone.
		kind = TimezoneUnspecified
	}
	return Timestamp{dateTime, precision, kind, 0}
}

// NewTimestampWithFractionalSeconds constructor
func NewTimestampWithFractionalSeconds(dateTime time.Time, precision TimestampPrecision, kind TimezoneKind, fractionPrecision uint8) Timestamp {
	if fractionPrecision > 9 {
		// 9 is the max precision supported
		fractionPrecision = 9
	}
	return Timestamp{dateTime, precision, kind, fractionPrecision}
}

// NewTimestampFromStr constructor
func NewTimestampFromStr(dateStr string, precision TimestampPrecision, kind TimezoneKind) (Timestamp, error) {
	// Count number of fractional seconds units.
	fractionUnits := uint8(0)
	if precision >= TimestampPrecisionNanosecond {
		pointIdx := strings.LastIndex(dateStr, ".")
		if pointIdx != -1 {
			nonZeroFraction := false

			idx := pointIdx + 1
			for idx < len(dateStr) && isDigit(int(dateStr[idx])) {
				if dateStr[idx] != '0' {
					nonZeroFraction = true
				}
				fractionUnits++
				idx++
			}

			if idx == len(dateStr) {
				return Timestamp{time.Time{}, TimestampNoPrecision, TimezoneUnspecified, 0},
					fmt.Errorf("ion: invalid date string '%v'", dateStr)
			}

			// We do not want to include trailing zeros for a non-zero fraction (ie. .1234000 -> .1234)
			// So we adjust fractionUnits accordingly.
			if nonZeroFraction {
				idx--
				for idx > pointIdx && dateStr[idx] == '0' {
					fractionUnits--
					idx--
				}
			}
		}
	}

	dateTime, err := time.Parse(precision.Layout(kind, fractionUnits), dateStr)
	if err != nil {
		return Timestamp{time.Time{}, TimestampNoPrecision, TimezoneUnspecified, 0}, err
	}

	return NewTimestampWithFractionalSeconds(dateTime, precision, kind, fractionUnits), nil
}

func emptyTimestamp() Timestamp {
	return Timestamp{time.Time{}, TimestampNoPrecision, TimezoneUnspecified, 0}
}

func invalidTimestamp(val string) (Timestamp, error) {
	return emptyTimestamp(), fmt.Errorf("ion: invalid timestamp: %v", val)
}

func tryCreateDateTimestamp(year, month, day int, precision TimestampPrecision) (Timestamp, error) {
	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)

	// time.Date converts 2000-01-32 input to 2000-02-01
	if year != date.Year() || time.Month(month) != date.Month() || day != date.Day() {
		return emptyTimestamp(), fmt.Errorf("ion: invalid timestamp")
	}

	return NewDateTimestamp(date, precision), nil
}

func tryCreateTimestamp(ts []int, nsecs int, overflow bool, offset, sign int64, precision TimestampPrecision, fractionPrecision uint8) (Timestamp, error) {
	date := time.Date(ts[0], time.Month(ts[1]), ts[2], ts[3], ts[4], ts[5], nsecs, time.UTC)
	// time.Date converts 2000-01-32 input to 2000-02-01
	if ts[0] != date.Year() || time.Month(ts[1]) != date.Month() || ts[2] != date.Day() {
		return emptyTimestamp(), fmt.Errorf("ion: invalid timestamp")
	}

	if precision <= TimestampPrecisionDay {
		return NewDateTimestamp(date, precision), nil
	}

	if overflow {
		date = date.Add(time.Second)
	}

	if offset == 0 {
		if sign == -1 {
			// Negative zero timezone offset is Unspecified
			return NewTimestampWithFractionalSeconds(date, precision, TimezoneUnspecified, fractionPrecision), nil
		}

		// Positive zero timezone offset is UTC
		return NewTimestampWithFractionalSeconds(date, precision, TimezoneUTC, fractionPrecision), nil
	}

	date = date.In(time.FixedZone("fixed", int(offset)*60))

	// Non-zero offset is Local
	return NewTimestampWithFractionalSeconds(date, precision, TimezoneLocal, fractionPrecision), nil
}

// Format returns a formatted Timestamp string.
func (ts *Timestamp) Format() string {
	layout := ts.precision.Layout(ts.kind, ts.numFractionalSeconds)
	format := ts.dateTime.Format(layout)

	// The above time.Format() does not produce the format we want in some scenarios.
	// So we may need to make some adjustments.

	// Add back removed trailing zeros from fractional seconds (ie. ".000")
	if ts.precision >= TimestampPrecisionNanosecond && ts.dateTime.Nanosecond() == 0 && ts.numFractionalSeconds > 0 {
		// Find the position of 'T'
		tIndex := strings.Index(format, "T")
		if tIndex == -1 {
			tIndex = strings.Index(format, "t")
			if tIndex == -1 {
				return format
			}
		}

		index := strings.LastIndex(format, "Z")
		if index == -1 || index < tIndex {
			index = strings.LastIndex(format, "+")
			if index == -1 || index < tIndex {
				index = strings.LastIndex(format, "-")
			}
		}

		// This position better be right of 'T'
		if index != -1 && tIndex < index {
			zeros := "."
			for i := uint8(0); i < ts.numFractionalSeconds; i++ {
				zeros += "0"
			}

			format = format[0:index] + zeros + format[index:]
		}
	}

	// A timestamp with time precision (ie. Minute/Second/Nanosecond) and Unspecified timezone
	// should have a "-00:00" offset but time.Format() is returning a "+00:00" offset.
	if ts.precision >= TimestampPrecisionMinute && ts.kind == TimezoneUnspecified {
		index := strings.LastIndex(format, "+00:00")
		if index != -1 {
			format = format[0:index] + "-00:00"
		}
	}

	return format
}

// Equal figures out if two timestamps are equal for each component.
func (ts *Timestamp) Equal(ts1 Timestamp) bool {
	_, offset := ts.dateTime.Zone()
	_, offset1 := ts1.dateTime.Zone()

	return ts.dateTime.Equal(ts1.dateTime) &&
		offset == offset1 &&
		ts.precision == ts1.precision &&
		ts.kind == ts1.kind &&
		ts.numFractionalSeconds == ts1.numFractionalSeconds
}

// SetLocation sets the location for the internal time object.
func (ts *Timestamp) SetLocation(loc *time.Location) {
	ts.dateTime = ts.dateTime.In(loc)
}

// TruncatedNanoseconds returns nanoseconds with trailing zeros removed (ie. 123456000 gets truncated to 123456).
func (ts *Timestamp) TruncatedNanoseconds() int {
	nsecs := ts.dateTime.Nanosecond()
	for i := uint8(0); i < (9-ts.numFractionalSeconds) && nsecs > 0 && (nsecs%10) == 0; i++ {
		nsecs /= 10
	}

	return nsecs
}
