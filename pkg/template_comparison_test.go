package pkg

import (
	"fmt"
	"testing"
	"text/template"
)

type cmpTest struct {
	expr  string
	truth string
	ok    bool
}

var cmpTests = []cmpTest{
	{"eq true true", "true", true},
	{"eq true false", "false", true},
	{"eq 1+2i 1+2i", "true", true},
	{"eq 1+2i 1+3i", "false", true},
	{"eq 1.5 1.5", "true", true},
	{"eq 1.5 2.5", "false", true},
	{"eq 1 1", "true", true},
	{"eq 1 2", "false", true},
	{"eq `xy` `xy`", "true", true},
	{"eq `xy` `xyz`", "false", true},
	{"eq .Uthree .Uthree", "true", true},
	{"eq .Uthree .Ufour", "false", true},
	{"eq 3 4 5 6 3", "true", true},
	{"eq 3 4 5 6 7", "false", true},
	{"ne true true", "false", true},
	{"ne true false", "true", true},
	{"ne 1+2i 1+2i", "false", true},
	{"ne 1+2i 1+3i", "true", true},
	{"ne 1.5 1.5", "false", true},
	{"ne 1.5 2.5", "true", true},
	{"ne 1 1", "false", true},
	{"ne 1 2", "true", true},
	{"ne `xy` `xy`", "false", true},
	{"ne `xy` `xyz`", "true", true},
	{"ne .Uthree .Uthree", "false", true},
	{"ne .Uthree .Ufour", "true", true},
	{"lt 1.5 1.5", "false", true},
	{"lt 1.5 2.5", "true", true},
	{"lt 1 1", "false", true},
	{"lt 1 2", "true", true},
	{"lt `xy` `xy`", "false", true},
	{"lt `xy` `xyz`", "true", true},
	{"lt .Uthree .Uthree", "false", true},
	{"lt .Uthree .Ufour", "true", true},
	{"le 1.5 1.5", "true", true},
	{"le 1.5 2.5", "true", true},
	{"le 2.5 1.5", "false", true},
	{"le 1 1", "true", true},
	{"le 1 2", "true", true},
	{"le 2 1", "false", true},
	{"le `xy` `xy`", "true", true},
	{"le `xy` `xyz`", "true", true},
	{"le `xyz` `xy`", "false", true},
	{"le .Uthree .Uthree", "true", true},
	{"le .Uthree .Ufour", "true", true},
	{"le .Ufour .Uthree", "false", true},
	{"gt 1.5 1.5", "false", true},
	{"gt 1.5 2.5", "false", true},
	{"gt 1 1", "false", true},
	{"gt 2 1", "true", true},
	{"gt 1 2", "false", true},
	{"gt `xy` `xy`", "false", true},
	{"gt `xy` `xyz`", "false", true},
	{"gt .Uthree .Uthree", "false", true},
	{"gt .Uthree .Ufour", "false", true},
	{"gt .Ufour .Uthree", "true", true},
	{"ge 1.5 1.5", "true", true},
	{"ge 1.5 2.5", "false", true},
	{"ge 2.5 1.5", "true", true},
	{"ge 1 1", "true", true},
	{"ge 1 2", "false", true},
	{"ge 2 1", "true", true},
	{"ge `xy` `xy`", "true", true},
	{"ge `xy` `xyz`", "false", true},
	{"ge `xyz` `xy`", "true", true},
	{"ge .Uthree .Uthree", "true", true},
	{"ge .Uthree .Ufour", "false", true},
	{"ge .Ufour .Uthree", "true", true},
	// Mixing signed and unsigned integers.
	{"eq .Uthree .Three", "true", true},
	{"eq .Three .Uthree", "true", true},
	{"le .Uthree .Three", "true", true},
	{"le .Three .Uthree", "true", true},
	{"ge .Uthree .Three", "true", true},
	{"ge .Three .Uthree", "true", true},
	{"lt .Uthree .Three", "false", true},
	{"lt .Three .Uthree", "false", true},
	{"gt .Uthree .Three", "false", true},
	{"gt .Three .Uthree", "false", true},
	{"eq .Ufour .Three", "false", true},
	{"lt .Ufour .Three", "false", true},
	{"gt .Ufour .Three", "true", true},
	{"eq .NegOne .Uthree", "false", true},
	{"eq .Uthree .NegOne", "false", true},
	{"ne .NegOne .Uthree", "true", true},
	{"ne .Uthree .NegOne", "true", true},
	{"lt .NegOne .Uthree", "true", true},
	{"lt .Uthree .NegOne", "false", true},
	{"le .NegOne .Uthree", "true", true},
	{"le .Uthree .NegOne", "false", true},
	{"gt .NegOne .Uthree", "false", true},
	{"gt .Uthree .NegOne", "true", true},
	{"ge .NegOne .Uthree", "false", true},
	{"ge .Uthree .NegOne", "true", true},
	{"eq (index `x` 0) 'x'", "true", true}, // The example that triggered this rule.
	{"eq (index `x` 0) 'y'", "false", true},
	// Comparison between float and int
	{"eq 2 2.0", "true", true},
	{"eq 2 2.0000001", "false", true},
	{"gt 2.0000001 2", "true", true},
	{"ge 2.000000 2", "true", true},
	{"lt 2 2.0001", "true", true},
	{"eq `xy` 1", "false", true}, // Different types.
	{"lt 2 2.0001", "true", true},
	{"lt 5 nil", "false", true},
	{"gt 5 nil", "false", true},
	{"ge 5 nil", "false", true},
	{"le 5 nil", "false", true},
	// Errors
	{"lt true true", "", false}, // Unordered types.
	{"lt 1+0i 1+0i", "", false}, // Unordered types.
}

func TestComparison(t *testing.T) {
	var cmpStruct = struct {
		Uthree, Ufour uint
		NegOne, Three int
	}{
		3,
		4,
		-1,
		3,
	}
	for _, test := range cmpTests {
		text := fmt.Sprintf("{{if %s}}true{{else}}false{{end}}", test.expr)
		tmpl, err := template.New("test").Funcs(GetTPLFuncsMap()).Parse(text)
		if err != nil {
			t.Fatalf("%q: %s", test.expr, err)
		}

		output, err := ExecuteTextTemplate(tmpl, &cmpStruct)
		if test.ok && err != nil {
			t.Errorf("%s errored incorrectly: %s", test.expr, err)
			continue
		}
		if !test.ok && err == nil {
			t.Errorf("%s did not error", test.expr)
			continue
		}
		if output != test.truth {
			t.Errorf("%s: want %s; got %s", test.expr, test.truth, output)
		}
	}
}

type cmpTestComplex struct {
	expr    string
	truth   string
	ok      bool
	payload map[string]interface{}
}

var cmpTestsComplex = []cmpTestComplex{
	{"eq .sliceA .sliceB", "true", true, map[string]interface{}{
		"sliceA": []string{"a", "b", "c"},
		"sliceB": []string{"a", "b", "c"},
	}},
	{"eq .sliceA .sliceB", "false", true, map[string]interface{}{
		"sliceA": []string{"a", "b", "c"},
		"sliceB": []string{"a", "b", "d"},
	}},
	// Array and slice cannot be compared with deep equal
	{"eq .sliceA .arrayA", "false", true, map[string]interface{}{
		"sliceA": []string{"a", "b", "c"},
		"arrayA": [3]string{"a", "b", "c"},
	}},
	{"eq .mapA .mapB", "true", true, map[string]interface{}{
		"mapA": map[string]interface{}{
			"name":     "Hello",
			"lastname": "BB",
		},
		"mapB": map[string]interface{}{
			"name":     "Hello",
			"lastname": "BB",
		},
	}},
	{"eq .mapA .mapB", "false", true, map[string]interface{}{
		"mapA": map[string]interface{}{
			"name":     "Hello",
			"lastname": "BB",
		},
		"mapB": map[string]interface{}{
			"name":     "Hello",
			"lastname": "BB2",
		},
	}},
	{"eq .structA .structB", "true", true, map[string]interface{}{
		"structA": struct {
			Name string
		}{
			Name: "hello",
		},
		"structB": struct {
			Name string
		}{
			Name: "hello",
		},
	}},
	{"eq .structA .structB", "false", true, map[string]interface{}{
		"structA": struct {
			Name string
		}{
			Name: "hello",
		},
		"structB": struct {
			Name     string
			Lastname string
		}{
			Name: "hello",
		},
	}},
	{"eq .structA .structB", "false", true, map[string]interface{}{
		"structA": struct {
			Name string
		}{
			Name: "hello",
		},
		"structB": struct {
			Name string
		}{
			Name: "hello2",
		},
	}},
}

func TestComparisonComplex(t *testing.T) {

	for idx, test := range cmpTestsComplex {
		text := fmt.Sprintf("{{if %s}}true{{else}}false{{end}}", test.expr)
		tmpl, err := template.New("test").Funcs(GetTPLFuncsMap()).Parse(text)
		if err != nil {
			t.Fatalf("[%d] %q: %s", idx, test.expr, err)
		}

		output, err := ExecuteTextTemplate(tmpl, test.payload)
		if test.ok && err != nil {
			t.Errorf("[%d] %s errored incorrectly: %s", idx, test.expr, err)
			continue
		}
		if !test.ok && err == nil {
			t.Errorf("[%d] %s did not error", idx, test.expr)
			continue
		}
		if output != test.truth {
			t.Errorf("[%d] %s: want %s; got %s", idx, test.expr, test.truth, output)
		}
	}
}
