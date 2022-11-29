package json

import (
	"fmt"
	"testing"
)

func TestTypeStrings(t *testing.T) {
	for _, test := range []struct {
		input    Type
		expected string
	}{
		{Null, typeStrings[Null]},
		{Array, typeStrings[Array]},
		{Object, typeStrings[Object]},
		{Boolean, typeStrings[Boolean]},
		{Integer, typeStrings[Integer]},
		{Number, typeStrings[Number]},
		{String, typeStrings[String]},
		{numTypes, "<unknown>"},
		{1000, "<unknown>"},
		{-1, "<unknown>"},
	} {
		t.Run(fmt.Sprintf("%v", test.input), func(t *testing.T) {
			actual := test.input.String()
			if test.expected != actual {
				t.Errorf("expected %v got %v", test.expected, actual)
			}
		})
	}
}

func TestType(t *testing.T) {
	for _, test := range []struct {
		input    Value
		expected Type
	}{
		{Value{jsonType: Null}, Null},
		{Value{jsonType: Array}, Array},
		{Value{jsonType: Object}, Object},
		{Value{jsonType: Boolean}, Boolean},
		{Value{jsonType: Integer}, Integer},
		{Value{jsonType: Number}, Number},
		{Value{jsonType: String}, String},
		{Value{jsonType: numTypes}, typeUnknown},
		{Value{jsonType: 1000}, typeUnknown},
		{Value{jsonType: -1}, typeUnknown},
	} {
		t.Run(fmt.Sprintf("%v", test.input), func(t *testing.T) {
			actual := test.input.Type()
			if test.expected != actual {
				t.Errorf("expected %v got %v", test.expected, actual)
			}
		})
	}
}

func TestAsNull(t *testing.T) {
	val := Value{}
	if _, err := val.AsNull(); err != nil {
		t.Errorf("expected no error got %v", err)
	}
	val = Value{jsonType: Boolean, booleanValue: true}
	if _, err := val.AsNull(); err == nil {
		t.Errorf("expected error got none")
	}
}

func TestAsNumber(t *testing.T) {
	val := Value{jsonType: Number, numberValue: 5}
	num, err := val.AsNumber()
	if err != nil {
		t.Errorf("expected no error got %v", err)
	}
	if num != 5 {
		t.Errorf("expected%v got %v", 5, num)
	}

	val = Value{jsonType: Integer, integerValue: 5}
	num, err = val.AsNumber()
	if err != nil {
		t.Errorf("expected no error got %v", err)
	}
	if num != 5 {
		t.Errorf("expected%v got %v", 5, num)
	}

	val = Value{jsonType: Boolean, booleanValue: true}
	_, err = val.AsNumber()
	if err == nil {
		t.Errorf("expected error got none")
	}
}

func TestAsInteger(t *testing.T) {
	val := Value{jsonType: Integer, integerValue: 5}
	num, err := val.AsInteger()
	if err != nil {
		t.Errorf("expected no error got %v", err)
	}
	if num != 5 {
		t.Errorf("expected%v got %v", 5, num)
	}

	val = Value{jsonType: Boolean, booleanValue: true}
	_, err = val.AsInteger()
	if err == nil {
		t.Errorf("expected error got none")
	}
}

func TestAsString(t *testing.T) {
	val := Value{jsonType: String, stringValue: "5"}
	num, err := val.AsString()
	if err != nil {
		t.Errorf("expected no error got %v", err)
	}
	if num != "5" {
		t.Errorf("expected%v got %v", "5", num)
	}

	val = Value{jsonType: Boolean, booleanValue: true}
	_, err = val.AsString()
	if err == nil {
		t.Errorf("expected error got none")
	}
}

func TestAsBoolean(t *testing.T) {
	val := Value{jsonType: Boolean, booleanValue: true}
	b, err := val.AsBoolean()
	if err != nil {
		t.Errorf("expected no error got %v", err)
	}
	if b != true {
		t.Errorf("expected%v got %v", true, b)
	}

	val = Value{}
	_, err = val.AsBoolean()
	if err == nil {
		t.Errorf("expected error got none")
	}
}

func TestAsArray(t *testing.T) {
	val := Value{jsonType: Array, arrayValue: []*Value{{}}}
	a, err := val.AsArray()
	if err != nil {
		t.Errorf("expected no error got %v", err)
	}
	if !equals(a[0], &Value{}) {
		t.Errorf("expected%v got %v", &Value{}, a[0])
	}

	val = Value{}
	_, err = val.AsArray()
	if err == nil {
		t.Errorf("expected error got none")
	}
}

func TestAsObject(t *testing.T) {
	val := Value{jsonType: Object, objectValue: []pair{{"a", &Value{}}}}
	o, err := val.AsObject()
	if err != nil {
		t.Errorf("expected no error got %v", err)
	}
	if !equals(o["a"], &Value{}) {
		t.Errorf("expected%v got %v", &Value{}, o["a"])
	}

	val = Value{}
	_, err = val.AsObject()
	if err == nil {
		t.Errorf("expected error got none")
	}
}

func TestString(t *testing.T) {
	for _, test := range []struct {
		input    Value
		expected string
	}{
		{Value{}, "null"},
		{Value{jsonType: Integer, integerValue: -5}, `-5`},
		{Value{jsonType: Number, numberValue: -5}, `-5`},
		{Value{jsonType: Number, numberValue: -5.1}, `-5.1`},
		{Value{jsonType: Number, numberValue: -5.12}, `-5.12`},
		{Value{jsonType: String, stringValue: "-5.12"}, `"-5.12"`},
		{Value{jsonType: Boolean, booleanValue: true}, `true`},
		{Value{jsonType: Boolean, booleanValue: false}, `false`},
		{Value{jsonType: Array, arrayValue: []*Value{
			{},
			{jsonType: Integer, integerValue: -5},
			{jsonType: String, stringValue: "-5.12"},
			{jsonType: Boolean, booleanValue: true},
		}}, `[null, -5, "-5.12", true]`},
		{Value{jsonType: Object, objectValue: []pair{
			{"a", &Value{}},
			{"b", &Value{jsonType: Integer, integerValue: -5}},
			{"c", &Value{jsonType: String, stringValue: "-5.12"}},
			{"d", &Value{jsonType: Boolean, booleanValue: true}},
		}}, `{"a": null, "b": -5, "c": "-5.12", "d": true}`},
		{Value{jsonType: numTypes, integerValue: -5}, `<unknown>`},
	} {
		t.Run(fmt.Sprintf("%v", test.input), func(t *testing.T) {
			actual := test.input.String()
			if test.expected != actual {
				t.Errorf("expected %v got %v", test.expected, actual)
			}
		})
	}
}

func TestIndex(t *testing.T) {
	val, err := ParseString(`[[[true, false]]]`)

	if err != nil {
		t.Errorf("expected no error got %v", err)
	}
	for _, test := range []struct {
		actual   *Value
		expected *Value
	}{
		{
			val.Index(0).Index(0).Index(0),
			&Value{jsonType: Boolean, booleanValue: true},
		},
		{
			val.Index(0).Index(0).Index(1),
			&Value{jsonType: Boolean, booleanValue: false},
		},
		{
			val.Index(0).Index(0).Index(2),
			&Value{},
		},
		{
			val.Index(0).Index(1).Index(2),
			&Value{},
		},
		{
			val.Index(-1).Index(1).Index(2),
			&Value{},
		},
	} {
		t.Run(fmt.Sprintf("%v", test.actual), func(t *testing.T) {
			if !equals(test.actual, test.expected) {
				t.Errorf("expected %v\ngot %v", test.expected, test.actual)
			}
		})
	}
}

func TestKey(t *testing.T) {
	val, err := ParseString(`{"a": {"b": {"c": true, "d":false}}}`)

	if err != nil {
		t.Errorf("expected no error got %v", err)
	}
	for _, test := range []struct {
		actual   *Value
		expected *Value
	}{
		{
			val.Key("a").Key("b").Key("c"),
			&Value{jsonType: Boolean, booleanValue: true},
		},
		{
			val.Key("a").Key("b").Key("d"),
			&Value{jsonType: Boolean, booleanValue: false},
		},
		{
			val.Key("a").Key("b").Key("e"),
			&Value{},
		},
		{
			val.Key("a").Key("e").Key("d"),
			&Value{},
		},
		{
			val.Key("e").Key("b").Key("d"),
			&Value{},
		},
	} {
		t.Run(fmt.Sprintf("%v", test.actual), func(t *testing.T) {
			if !equals(test.actual, test.expected) {
				t.Errorf("expected %v\ngot %v", test.expected, test.actual)
			}
		})
	}
}
