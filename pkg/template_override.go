package pkg

import (
	"errors"
	"reflect"
)

var (
	errBadComparisonType = errors.New("invalid type for comparison")
	errBadComparison     = errors.New("incompatible types for comparison")
	errNoComparison      = errors.New("missing argument for comparison")
)

type kind int

const (
	invalidKind kind = iota
	boolKind
	complexKind
	intKind
	floatKind
	stringKind
	uintKind
	objectKind
	nilKind
)

func basicKind(v reflect.Value) (kind, error) {
	if !v.IsValid() {
		// Almost guaranteed this is a nil interface
		return nilKind, nil
	}

	switch v.Kind() {
	case reflect.Bool:
		return boolKind, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return intKind, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return uintKind, nil
	case reflect.Float32, reflect.Float64:
		return floatKind, nil
	case reflect.Complex64, reflect.Complex128:
		return complexKind, nil
	case reflect.String:
		return stringKind, nil
	case reflect.Array, reflect.Map, reflect.Slice, reflect.Struct:
		return objectKind, nil
	}
	return invalidKind, errBadComparisonType
}

// eq evaluates the comparison a == b || a == c || ...
func tplEq(arg1 reflect.Value, arg2 ...reflect.Value) (bool, error) {
	v1 := indirectInterface(arg1)
	k1, err := basicKind(v1)
	if err != nil {
		return false, err
	}
	if len(arg2) == 0 {
		return false, errNoComparison
	}
	for _, arg := range arg2 {
		v2 := indirectInterface(arg)
		k2, err := basicKind(v2)
		if err != nil {
			return false, err
		}
		truth := false
		if k1 != k2 {
			// Special case: Can compare integer values regardless of type's sign.
			switch {
			case k1 == intKind && k2 == uintKind:
				truth = v1.Int() >= 0 && uint64(v1.Int()) == v2.Uint()
			case k1 == uintKind && k2 == intKind:
				truth = v2.Int() >= 0 && v1.Uint() == uint64(v2.Int())
			case (k1 == intKind || k1 == uintKind) && k2 == floatKind:
				var value float64
				if k1 == intKind {
					value = float64(v1.Int())
				} else {
					value = float64(v1.Uint())
				}

				truth = value == v2.Float()
			case (k2 == intKind || k2 == uintKind) && k1 == floatKind:
				var value float64
				if k2 == intKind {
					value = float64(v2.Int())
				} else {
					value = float64(v2.Uint())
				}

				truth = value == v1.Float()
			case k1 == nilKind || k2 == nilKind:
				return false, nil
			default:
				return reflect.DeepEqual(v1.Interface(), v2.Interface()), nil
			}
		} else {
			switch k1 {
			case boolKind:
				truth = v1.Bool() == v2.Bool()
			case complexKind:
				truth = v1.Complex() == v2.Complex()
			case floatKind:
				truth = v1.Float() == v2.Float()
			case intKind:
				truth = v1.Int() == v2.Int()
			case stringKind:
				truth = v1.String() == v2.String()
			case uintKind:
				truth = v1.Uint() == v2.Uint()
			case nilKind:
				truth = true
			case objectKind:
				return reflect.DeepEqual(v1.Interface(), v2.Interface()), nil
			default:
				panic("invalid kind")
			}
		}
		if truth {
			return true, nil
		}
	}
	return false, nil
}

// ne evaluates the comparison a != b.
func tplNE(arg1, arg2 reflect.Value) (bool, error) {
	// != is the inverse of ==.
	equal, err := tplEq(arg1, arg2)
	return !equal, err
}

func getValueAndKind(arg reflect.Value) (reflect.Value, kind, error) {
	v1 := indirectInterface(arg)
	k1, err := basicKind(v1)
	return v1, k1, err
}

func ltReflect(v1 reflect.Value, k1 kind, v2 reflect.Value, k2 kind) (bool, error) {
	truth := false
	if k1 != k2 {
		// Special case: Can compare integer values regardless of type's sign.
		switch {
		case k1 == intKind && k2 == uintKind:
			truth = v1.Int() < 0 || uint64(v1.Int()) < v2.Uint()
		case k1 == uintKind && k2 == intKind:
			truth = v2.Int() >= 0 && v1.Uint() < uint64(v2.Int())
		case (k1 == intKind || k1 == uintKind) && k2 == floatKind:
			var value float64
			if k1 == intKind {
				value = float64(v1.Int())
			} else {
				value = float64(v1.Uint())
			}

			truth = value < v2.Float()
		case (k2 == intKind || k2 == uintKind) && k1 == floatKind:
			var value float64
			if k2 == intKind {
				value = float64(v2.Int())
			} else {
				value = float64(v2.Uint())
			}

			truth = v1.Float() < value
		default:
			return false, errBadComparison
		}
	} else {
		switch k1 {
		case boolKind, complexKind:
			return false, errBadComparisonType
		case floatKind:
			truth = v1.Float() < v2.Float()
		case intKind:
			truth = v1.Int() < v2.Int()
		case stringKind:
			truth = v1.String() < v2.String()
		case uintKind:
			truth = v1.Uint() < v2.Uint()
		default:
			panic("invalid kind")
		}
	}
	return truth, nil
}

// lt evaluates the comparison a < b.
func tplLT(arg1, arg2 reflect.Value) (bool, error) {
	v1, k1, err := getValueAndKind(arg1)
	if err != nil {
		return false, err
	}
	if k1 == nilKind || k1 == objectKind {
		return false, nil
	}
	v2, k2, err := getValueAndKind(arg2)
	if err != nil {
		return false, err
	}
	if k2 == nilKind || k2 == objectKind {
		return false, nil
	}

	return ltReflect(v1, k1, v2, k2)
}

// le evaluates the comparison <= b.
func tplLE(arg1, arg2 reflect.Value) (bool, error) {
	// <= is < or ==.
	lessThan, err := tplLT(arg1, arg2)
	if lessThan || err != nil {
		return lessThan, err
	}
	return tplEq(arg1, arg2)
}

// gt evaluates the comparison a > b.
func tplGT(arg1, arg2 reflect.Value) (bool, error) {
	v1, k1, err := getValueAndKind(arg1)
	if err != nil {
		return false, err
	}
	if k1 == nilKind || k1 == objectKind {
		return false, nil
	}
	v2, k2, err := getValueAndKind(arg2)
	if err != nil {
		return false, err
	}
	if k2 == nilKind || k2 == objectKind {
		return false, nil
	}

	lt, err := ltReflect(v1, k1, v2, k2)
	if lt || err != nil {
		return false, err
	}
	eq, err := tplEq(arg1, arg2)
	if err != nil {
		return false, err
	}
	return !eq, nil
}

// ge evaluates the comparison a >= b.
func tplGE(arg1, arg2 reflect.Value) (bool, error) {
	// >= is the inverse of <.
	greaterThan, err := tplGT(arg1, arg2)
	if greaterThan || err != nil {
		return greaterThan, err
	}
	return tplEq(arg1, arg2)
}

// indirectInterface returns the concrete value in an interface value,
// or else the zero reflect.Value.
// That is, if v represents the interface value x, the result is the same as reflect.ValueOf(x):
// the fact that x was an interface value is forgotten.
func indirectInterface(v reflect.Value) reflect.Value {
	if v.Kind() != reflect.Interface {
		return v
	}
	if v.IsNil() {
		return reflect.Value{}
	}
	return v.Elem()
}
