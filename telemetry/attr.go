package telemetry

// Attr is one structured key/value detail attached to a log event, span or
// measurement.
//
// Values are restricted by construction to string, int64, float64 or bool, so
// an Attr cannot hold an id-sized object or personal data by accident. Keep
// values low-cardinality. Attr is immutable and comparable. The zero Attr is
// invalid (see Valid) and is dropped or reported by recording doubles.
type Attr struct {
	key string
	val any
}

// AttrErrorMessage is the key used by Err, matching the OpenTelemetry
// exception.message attribute.
const AttrErrorMessage = "exception.message"

// String returns a string attribute.
func String(key, value string) Attr { return Attr{key, value} }

// Int returns an integer attribute, stored as int64.
func Int(key string, value int) Attr { return Attr{key, int64(value)} }

// Int64 returns an int64 attribute.
func Int64(key string, value int64) Attr { return Attr{key, value} }

// Float returns a float64 attribute.
func Float(key string, value float64) Attr { return Attr{key, value} }

// Bool returns a boolean attribute.
func Bool(key string, value bool) Attr { return Attr{key, value} }

// Err returns the attribute that reports err's message under AttrErrorMessage,
// for the rare log event that reports a handled failure. It returns the
// invalid zero Attr when err is nil.
func Err(err error) Attr {
	if err == nil {
		return Attr{}
	}
	return String(AttrErrorMessage, err.Error())
}

// Key returns the attribute key.
func (a Attr) Key() string { return a.key }

// Value returns the attribute value: a string, int64, float64 or bool, or nil
// for the zero Attr.
func (a Attr) Value() any { return a.val }

// Valid reports whether a has a non-empty key and a value.
func (a Attr) Valid() bool { return a.key != "" && a.val != nil }
