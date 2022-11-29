# A Simple JSON reader

This library is for dynamically inspecting parsed JSON values rather than for data binding or serialization.

```
import "github.com/mcvoid/json"

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

```

Uses a table-driven parser based on Doug Crockford's `json-c` JSON Checker C library / command line utility. This should
lead to a fairly fast parsing time, though that's not guaranteed and isn't even a design goal.

See `example_test.go` for usage examples.

### License

Copyright (c) 2022 Sean Wolcott

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

Based on `json-checker` by (c) 2016 Douglas Crockford, no license information specified.
