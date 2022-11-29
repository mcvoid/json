package json

import (
	"errors"
	"fmt"
	"strconv"
)

var (
	// A value is trying to be cast to an incorrect type.
	ErrType = errors.New("type error")
	// A problem occured while parsing the JSON
	ErrParse = errors.New("parse error")
)

// The type of a JSON value.
type Type int

// Possible JSON values
const (
	Null Type = iota
	Number
	Integer
	String
	Boolean
	Array
	Object
	numTypes
	typeUnknown Type = -1
)

var typeStrings = [numTypes]string{
	"<null>",
	"<number>",
	"<integer>",
	"<string>",
	"<boolean>",
	"<array>",
	"<object>",
}

// Returns a string representation of a JSON Type.
func (t Type) String() string {
	if t < 0 || t >= numTypes {
		return "<unknown>"
	}
	return typeStrings[t]
}

// A structured JSON value
type Value struct {
	jsonType     Type
	numberValue  float64
	integerValue int64
	stringValue  string
	booleanValue bool
	arrayValue   []*Value
	objectValue  []pair
}

type pair struct {
	key string
	val *Value
}

// Gets the type of the current value.
func (v *Value) Type() Type {
	if v.jsonType >= 0 && v.jsonType < numTypes {
		return v.jsonType
	}
	return typeUnknown
}

// Extracts a null value from the JSON. Returns ErrType if the value is not null, nil otherwise.
func (v *Value) AsNull() (struct{}, error) {
	if v.jsonType == Null {
		return struct{}{}, nil
	}
	return struct{}{}, fmt.Errorf("%w: value not null %v", ErrType, v)
}

// Extracts a number from the JSON. If the value is an integer, it is cast to a float64. If integer
// precision is needed, use AsInteger instead. Returns ErrType if the value is niether a number nor
// an integer. Returns nil otherwise.
func (v *Value) AsNumber() (float64, error) {
	if v.jsonType == Integer {
		return float64(v.integerValue), nil
	}
	if v.jsonType == Number {
		return v.numberValue, nil
	}
	return 0, fmt.Errorf("%w: value not a valid number %v", ErrType, v)
}

// Extracts an integer from the JSON. Will not convert decimal to integer. If decimal precision is
// needed, use AsNumber instead. Returns ErrType if the value is niether a number nor an integer.
// Returns nil otherwise.
func (v *Value) AsInteger() (int64, error) {
	if v.jsonType == Integer {
		return v.integerValue, nil
	}
	return 0, fmt.Errorf("%w: value not a valid integer %v", ErrType, v)
}

// Extracts a string value from the JSON. Returns ErrType if the value is not string, nil otherwise.
func (v *Value) AsString() (string, error) {
	if v.jsonType == String {
		return v.stringValue, nil
	}
	return "", fmt.Errorf("%w: value not a valid string %v", ErrType, v)
}

// Extracts a boolean value from the JSON. Returns ErrType if the value is not boolean, nil otherwise.
func (v *Value) AsBoolean() (bool, error) {
	if v.jsonType == Boolean {
		return v.booleanValue, nil
	}
	return false, fmt.Errorf("%w: value not a valid boolean %v", ErrType, v)
}

// Extracts an array value from the JSON. Returns ErrType if the value is not array, nil otherwise.
func (v *Value) AsArray() ([]*Value, error) {
	if v.jsonType == Array {
		return v.arrayValue, nil
	}
	return nil, fmt.Errorf("%w: value not a valid array %v", ErrType, v)
}

// Extracts an object value from the JSON. Returns ErrType if the value is not object, nil otherwise.
func (v *Value) AsObject() (map[string]*Value, error) {
	if v.jsonType == Object {
		m := map[string]*Value{}
		for _, pair := range v.objectValue {
			m[pair.key] = pair.val
		}
		return m, nil
	}
	return nil, fmt.Errorf("%w: value not a valid array %v", ErrType, v)
}

// Returns a string representation of the values. NOT valid JSON!
func (v *Value) String() string {
	switch v.jsonType {
	case Null:
		return "null"
	case Integer:
		return strconv.FormatInt(v.integerValue, 10)
	case Number:
		return strconv.FormatFloat(v.numberValue, 'f', -1, 64)
	case String:
		return strconv.Quote(v.stringValue)
	case Boolean:
		if v.booleanValue {
			return "true"
		}
		return "false"
	case Array:
		str := "["
		for i, val := range v.arrayValue {
			if i > 0 {
				str += ", "
			}
			str += val.String()
		}
		str += "]"
		return str
	case Object:
		str := "{"
		for i, pair := range v.objectValue {
			if i > 0 {
				str += ", "
			}
			str += strconv.Quote(pair.key)
			str += ": "
			str += pair.val.String()
		}
		str += "}"
		return str
	}
	return "<unknown>"
}

// Fluent inteface for accessing array members. Returns nil instead of error.
func (v *Value) Index(i int) *Value {
	if v.jsonType != Array {
		return &Value{}
	}

	if i < 0 || i >= len(v.arrayValue) {
		return &Value{}
	}

	return v.arrayValue[i]
}

// Fluent inteface for accessing object members. Returns nil instead of error.
func (v *Value) Key(k string) *Value {
	if v.jsonType != Object {
		return &Value{}
	}

	for _, p := range v.objectValue {
		if p.key == k {
			return p.val
		}
	}

	return &Value{}
}
