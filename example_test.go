package json_test

import (
	"testing"

	"github.com/mcvoid/json"
)

func TestUsage(t *testing.T) {
	// use one of the ParseXXX functions to get a JSON value from text.
	// You can pass in strings, []byte, or io.Reader.
	val, err := json.ParseString(`
	{
		"null": null,
		"integer": 5,
		"number": 5.0,
		"boolean": true,
		"array": [null, 5, 5.0, true],
		"object": {}
	}
	`)
	if err != nil {
		t.Error("Can't parse json... somehow.")
	}

	// to inspect the type, use the Type method.
	if val.Type() != json.Object {
		t.Error("JSON object is wrong type!")
	}

	// Objects can be extracted as maps of values
	m, _ := val.AsObject()
	if m["null"].Type() != json.Null {
		t.Error("JSON null is wrong type!")
	}

	// We differentiate integers and numbers, but integers count as numbers, too.
	// Integer is mainly there for large whole numbers that float64 might
	// not have the precision for.
	i, _ := m["integer"].AsNumber()
	n, _ := m["number"].AsNumber()
	if i != n {
		t.Error("It works this time, but this isn't the best way to check for floating point equivalency, btw")
	}

	// Arrays are represented as slices of JSON values.
	a, _ := m["array"].AsArray()

	// Booleans are bools.
	b, _ := a[3].AsBoolean()
	if !b {
		t.Error("true... isn't?")
	}

	// And that's all there is to it. Enjoy!
}
