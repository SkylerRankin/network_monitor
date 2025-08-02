package optional

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

type Opt[T any] struct {
	value    T
	hasValue bool
}

var errMissingValue = errors.New("cannot get missing value")

func New[T any](value T) Opt[T] {
	return Opt[T]{
		value:    value,
		hasValue: true,
	}
}

func Empty[T any]() Opt[T] {
	return Opt[T]{
		hasValue: false,
	}
}

func (o *Opt[T]) Has() bool {
	return o.hasValue
}

func (o *Opt[T]) Get() (T, error) {
	if o.Has() {
		return o.value, nil
	} else {
		return o.value, errMissingValue
	}
}

func (o *Opt[T]) Else(e T) T {
	if o.Has() {
		return o.value
	} else {
		return e
	}
}

// Implements the Scanner interface in order to scan values from SQLite row
func (o *Opt[T]) Scan(src any) error {
	var v sql.Null[T]
	if err := v.Scan(src); err != nil {
		return err
	}

	if v.Valid {
		*o = Opt[T]{
			value:    v.V,
			hasValue: true,
		}
	} else {
		*o = Opt[T]{
			hasValue: false,
		}
	}

	return nil
}

// Implements the Valuer interface in order to write to SQLite
func (o *Opt[T]) Value() (driver.Value, error) {
	if o.hasValue {
		return driver.DefaultParameterConverter.ConvertValue(o.value)
	} else {
		return nil, nil
	}
}

// Implements the Marshaler interface for JSON marshalling
func (o *Opt[T]) MarshalJSON() ([]byte, error) {
	if o.hasValue {
		valueJson, err := json.Marshal(o.value)
		if err != nil {
			return nil, err
		}
		return valueJson, nil
	} else {
		return []byte("null"), nil
	}
}

func (o *Opt[T]) String() string {
	if o.hasValue {
		return fmt.Sprintf("%v", o.value)
	} else {
		return "empty"
	}
}
