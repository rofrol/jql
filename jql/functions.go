package jql

import (
	"fmt"
	"reflect"
	"strings"
)

var Functions = map[string]func(ts ...Expression) (Expression, error){
	"elem":    NewElement,
	"keys":    NewKeys,
	"id":      NewIdentity,
	"array":   NewArray,
	"object":  NewObject,
	"pipe":    NewPipe,
	"sprintf": NewSprintf,
	"join":    NewJoin,
	"filter":  NewFilter,
	"eq":      NewEqual,
	"lt":      NewLessThan,
	"gt":      NewGreaterThan,
	"range":   NewRange,
}

type Constant struct {
	value interface{}
}

func NewConstant(value interface{}) Expression {
	return &Constant{value: value}
}

func (s Constant) Get(input interface{}) (interface{}, error) {
	return s.value, nil
}

type Element struct {
	Positions       Expression
	ValueExpression Expression
}

func NewElement(ts ...Expression) (Expression, error) {
	switch len(ts) {
	case 1:
		return Element{
			Positions:       ts[0],
			ValueExpression: Identity{},
		}, nil
	case 2:
		return Element{
			Positions:       ts[0],
			ValueExpression: ts[1],
		}, nil
	default:
		return nil, fmt.Errorf("invalid argument count to elem function: %v", len(ts))
	}
}

func (t Element) Get(arg interface{}) (interface{}, error) {
	positions, err := t.Positions.Get(arg)
	if err != nil {
		return nil, fmt.Errorf("couldn't get positions to get: %w", err)
	}

	switch typed := arg.(type) {
	case []interface{}:
		switch indices := positions.(type) {
		case []interface{}:
			outArray := make([]interface{}, len(indices))
			for i := range indices {
				index, ok := indices[i].(int)
				if !ok {
					return nil, fmt.Errorf("position in array should be integer, is %v of type %v", indices[i], reflect.TypeOf(indices[i]))
				}

				if len(typed) <= index {
					return nil, fmt.Errorf("index %d out of bounds in array of length %d", indices[i], len(typed))
				}
				value := typed[index]

				outArray[i], err = t.ValueExpression.Get(value)
				if err != nil {
					return nil, fmt.Errorf("couldn't get transformed value for index %s with value %v: %w", indices[i], value, err)
				}
			}

			return outArray, nil

		case map[string]interface{}:
			outObject := make(map[string]interface{}, len(indices))
			for k, i := range indices {
				index, ok := i.(int)
				if !ok {
					return nil, fmt.Errorf("position in array should be integer, is %v of type %v", i, reflect.TypeOf(i))
				}

				if len(typed) <= index {
					return nil, nil
				}
				value := typed[index]

				outObject[k], err = t.ValueExpression.Get(value)
				if err != nil {
					return nil, fmt.Errorf("couldn't get transformed value for index %s with value %v: %w", index, value, err)
				}
			}

			return outObject, nil

		case int:
			if len(typed) <= indices {
				return nil, nil
			}

			out, err := t.ValueExpression.Get(typed[indices])
			if err != nil {
				return nil, fmt.Errorf("couldn't get transformed value for index %s with value %v: %w", indices, typed[indices], err)
			}
			return out, nil

		default:
			return nil, fmt.Errorf("invalid positions for array: %v, should be array of integers or integer", positions)
		}

	case map[string]interface{}:
		switch fields := positions.(type) {
		case []interface{}:
			outArray := make([]interface{}, 0, len(fields))
			for i := range fields {
				field, ok := fields[i].(string)
				if !ok {
					return nil, fmt.Errorf("position in object should be string, is %v of type %v", fields[i], reflect.TypeOf(fields[i]))
				}

				value, ok := typed[field]
				if !ok {
					return nil, nil
				}

				expressionValue, err := t.ValueExpression.Get(value)
				if err != nil {
					return nil, fmt.Errorf("couldn't get transformed value for field %s with value %v: %w", fields[i], value, err)
				}
				outArray = append(outArray, expressionValue)
			}

			return outArray, nil

		case map[string]interface{}:
			outObject := make(map[string]interface{}, len(fields))
			for k := range fields {
				field, ok := fields[k].(string)
				if !ok {
					return nil, fmt.Errorf("position in object should be string, is %v of type %v", fields[k], reflect.TypeOf(fields[k]))
				}

				value, ok := typed[field]
				if !ok {
					return nil, nil
				}

				expressionValue, err := t.ValueExpression.Get(value)
				if err != nil {
					return nil, fmt.Errorf("couldn't get transformed value for field %s with value %v: %w", field, value, err)
				}
				outObject[k] = expressionValue
			}

			return outObject, nil

		case string:
			valueExpressionArgument, ok := typed[fields]
			if !ok {
				return nil, nil
			}

			out, err := t.ValueExpression.Get(valueExpressionArgument)
			if err != nil {
				return nil, fmt.Errorf("couldn't get transformed value for field %s with value %v: %w", fields, typed[fields], err)
			}
			return out, nil

		default:
			return nil, fmt.Errorf("invalid fields for object: %v, should be array of strings or string", positions)
		}

	default:
		return nil, fmt.Errorf("can only use element on array or object, used on: %s", reflect.TypeOf(arg))
	}
}

type Keys struct {
}

func NewKeys(ts ...Expression) (Expression, error) {
	if len(ts) != 0 {
		return nil, fmt.Errorf("expected no arguments to keys function, got %d arguments", len(ts))
	}

	return Keys{}, nil
}

func (s Keys) Get(arg interface{}) (interface{}, error) {
	switch typed := arg.(type) {
	case []interface{}:
		outIndices := make([]interface{}, len(typed))
		for i := range typed {
			outIndices[i] = i
		}

		return outIndices, nil

	case map[string]interface{}:
		outFields := make([]interface{}, 0, len(typed))
		for field := range typed {
			outFields = append(outFields, field)
		}

		return outFields, nil

	default:
		return nil, fmt.Errorf("can only use keys on array or object, used on: %s", reflect.TypeOf(arg))
	}
}

type Identity struct {
}

func NewIdentity(ts ...Expression) (Expression, error) {
	if len(ts) != 0 {
		return nil, fmt.Errorf("expected no arguments to id function, got %d arguments", len(ts))
	}

	return Identity{}, nil
}

func (t Identity) Get(arg interface{}) (interface{}, error) {
	return arg, nil
}

type Array struct {
	Values []Expression
}

func NewArray(ts ...Expression) (Expression, error) {
	return Array{Values: ts}, nil
}

func (t Array) Get(arg interface{}) (interface{}, error) {
	outArray := make([]interface{}, len(t.Values))
	for i := range t.Values {
		var err error
		outArray[i], err = t.Values[i].Get(arg)
		if err != nil {
			return nil, fmt.Errorf("couldn't construct array index %d: %w", i, err)
		}
	}

	return outArray, nil
}

type Object struct {
	Keys   []Expression
	Values []Expression
}

func NewObject(ts ...Expression) (Expression, error) {
	if len(ts)%2 != 0 {
		return nil, fmt.Errorf("object function should contain an even argument count (you need a value for each key), got argument count: %v", len(ts))
	}
	keys := make([]Expression, len(ts)/2)
	values := make([]Expression, len(ts)/2)
	for i := range ts {
		if i%2 == 0 {
			keys[i/2] = ts[i]
		} else {
			values[i/2] = ts[i]
		}
	}
	return Object{Keys: keys, Values: values}, nil
}

func (t Object) Get(arg interface{}) (interface{}, error) {
	outObject := make(map[string]interface{}, len(t.Values))
	for i, keyExpression := range t.Keys {
		keyValue, err := keyExpression.Get(arg)
		if err != nil {
			return nil, fmt.Errorf("couldn't get key out of key expression with index %d: %w", i, err)
		}

		key, ok := keyValue.(string)
		if !ok {
			return nil, fmt.Errorf("got object key %v of type %s at position %d, must be string", keyValue, reflect.TypeOf(keyValue), i)
		}

		outObject[key], err = t.Values[i].Get(arg)
		if err != nil {
			return nil, fmt.Errorf("couldn't construct object field %s at index %d: %w", key, i, err)
		}
	}

	return outObject, nil
}

type Pipe struct {
	Expressions []Expression
}

func NewPipe(ts ...Expression) (Expression, error) {
	if len(ts) == 0 {
		return nil, fmt.Errorf("pipe function needs at least one argument")
	}
	return Pipe{Expressions: ts}, nil
}

func (t Pipe) Get(arg interface{}) (interface{}, error) {
	object := arg
	for i := range t.Expressions {
		var err error
		object, err = t.Expressions[i].Get(object)
		if err != nil {
			return nil, fmt.Errorf("error in pipe subexpression with index %d: %w", i, err)
		}
	}
	return object, nil
}

type Sprintf struct {
	Format      Expression
	Expressions []Expression
}

func NewSprintf(ts ...Expression) (Expression, error) {
	if len(ts) == 0 {
		return nil, fmt.Errorf("sprintf function needs at least one argument")
	}
	return Sprintf{Format: ts[0], Expressions: ts[1:]}, nil
}

func (t Sprintf) Get(arg interface{}) (interface{}, error) {
	formatValue, err := t.Format.Get(arg)
	if err != nil {
		return nil, fmt.Errorf("couldn't evaluate sprintf format argument: %w", err)
	}
	format, ok := formatValue.(string)
	if !ok {
		return nil, fmt.Errorf("sprintf format argument should be string, is %v of type %s", formatValue, reflect.TypeOf(formatValue))
	}

	values := make([]interface{}, len(t.Expressions))
	for i := range t.Expressions {
		var err error
		values[i], err = t.Expressions[i].Get(arg)
		if err != nil {
			return nil, fmt.Errorf("couldn't evaluate sprintf argument with index %d: %w", i, err)
		}
	}
	return fmt.Sprintf(format, values...), nil
}

type Join struct {
	Strings   Expression
	Separator Expression
}

func NewJoin(ts ...Expression) (Expression, error) {
	switch len(ts) {
	case 1:
		return Join{
			Strings:   ts[0],
			Separator: &Constant{value: ""},
		}, nil
	case 2:
		return Join{
			Strings:   ts[0],
			Separator: ts[1],
		}, nil
	default:
		return nil, fmt.Errorf("invalid argument count to join function: %v", len(ts))
	}
}

func (t Join) Get(arg interface{}) (interface{}, error) {
	separatorValue, err := t.Separator.Get(arg)
	if err != nil {
		return nil, fmt.Errorf("couldn't evaluate join separator argument: %w", err)
	}
	separator, ok := separatorValue.(string)
	if !ok {
		return nil, fmt.Errorf("join separator argument should be string, is %v of type %s", separatorValue, reflect.TypeOf(separatorValue))
	}

	argsValue, err := t.Strings.Get(arg)
	if err != nil {
		return nil, fmt.Errorf("couldn't evaluate join strings argument: %w", err)
	}
	args, ok := argsValue.([]interface{})
	if !ok {
		return fmt.Sprint(argsValue), nil
	}

	stringArgs := make([]string, len(args))
	for i := range args {
		stringArgs[i] = fmt.Sprint(args[i])
	}

	return strings.Join(stringArgs, separator), nil
}

func IsTruthy(value interface{}) bool {
	if value == nil {
		return false
	}
	if b, ok := value.(bool); ok && !b {
		return false
	}
	return true
}

type Filter struct {
	Predicate  Expression
	Expression Expression
}

func NewFilter(ts ...Expression) (Expression, error) {
	switch len(ts) {
	case 1:
		return Filter{
			Predicate:  ts[0],
			Expression: &Identity{},
		}, nil
	case 2:
		return Filter{
			Predicate:  ts[0],
			Expression: ts[1],
		}, nil
	default:
		return nil, fmt.Errorf("invalid argument count to filter function: %v", len(ts))
	}
}

func (t Filter) Get(arg interface{}) (interface{}, error) {
	args, ok := arg.([]interface{})
	if !ok {
		return nil, fmt.Errorf("filter expects an array, received %v of type %s", arg, reflect.TypeOf(arg))
	}

	out := make([]interface{}, 0, len(args))

	for i := range args {
		predicateValue, err := t.Predicate.Get(args[i])
		if err != nil {
			return nil, fmt.Errorf("couldn't evaluate filter predicate for array index %d with expression value %v: %w", i, args[i], err)
		}

		if IsTruthy(predicateValue) {
			out = append(out, args[i])
		}
	}

	return out, nil
}

type Equal struct {
	Left  Expression
	Right Expression
}

func NewEqual(ts ...Expression) (Expression, error) {
	if len(ts) != 2 {
		return nil, fmt.Errorf("invalid argument count to equal function: %v", len(ts))
	}

	return &Equal{
		Left:  ts[0],
		Right: ts[1],
	}, nil
}

func (t Equal) Get(arg interface{}) (interface{}, error) {
	leftValue, err := t.Left.Get(arg)
	if err != nil {
		return nil, fmt.Errorf("couldn't evaluate equal function left expression: %w", err)
	}
	rightValue, err := t.Right.Get(arg)
	if err != nil {
		return nil, fmt.Errorf("couldn't evaluate equal function right expression: %w", err)
	}

	return reflect.DeepEqual(leftValue, rightValue), nil
}

type LessThan struct {
	Left  Expression
	Right Expression
}

func NewLessThan(ts ...Expression) (Expression, error) {
	if len(ts) != 2 {
		return nil, fmt.Errorf("invalid argument count to lt function: %v", len(ts))
	}

	return &LessThan{
		Left:  ts[0],
		Right: ts[1],
	}, nil
}

func (t LessThan) Get(arg interface{}) (interface{}, error) {
	leftValue, err := t.Left.Get(arg)
	if err != nil {
		return nil, fmt.Errorf("couldn't evaluate lt function left expression: %w", err)
	}
	rightValue, err := t.Right.Get(arg)
	if err != nil {
		return nil, fmt.Errorf("couldn't evaluate lt function right expression: %w", err)
	}

	switch left := leftValue.(type) {
	case string:
		right, ok := rightValue.(string)
		if !ok {
			return false, nil
		}
		return left < right, nil

	case int:
		right, ok := rightValue.(int)
		if !ok {
			return false, nil
		}
		return left < right, nil

	case float64:
		right, ok := rightValue.(float64)
		if !ok {
			return false, nil
		}
		return left < right, nil

	default:
		return false, nil
	}
}

type GreaterThan struct {
	Left  Expression
	Right Expression
}

func NewGreaterThan(ts ...Expression) (Expression, error) {
	if len(ts) != 2 {
		return nil, fmt.Errorf("invalid argument count to gt function: %v", len(ts))
	}

	return &GreaterThan{
		Left:  ts[0],
		Right: ts[1],
	}, nil
}

func (t GreaterThan) Get(arg interface{}) (interface{}, error) {
	leftValue, err := t.Left.Get(arg)
	if err != nil {
		return nil, fmt.Errorf("couldn't evaluate gt function left expression: %w", err)
	}
	rightValue, err := t.Right.Get(arg)
	if err != nil {
		return nil, fmt.Errorf("couldn't evaluate gt function right expression: %w", err)
	}

	switch left := leftValue.(type) {
	case string:
		right, ok := rightValue.(string)
		if !ok {
			return false, nil
		}
		return left > right, nil

	case int:
		right, ok := rightValue.(int)
		if !ok {
			return false, nil
		}
		return left > right, nil

	case float64:
		right, ok := rightValue.(float64)
		if !ok {
			return false, nil
		}
		return left > right, nil

	default:
		return false, nil
	}
}

type Range struct {
	Begin Expression
	End   Expression
}

func NewRange(ts ...Expression) (Expression, error) {
	switch len(ts) {
	case 1:
		return &Range{
			Begin: &Constant{value: 0},
			End:   ts[0],
		}, nil
	case 2:
		return &Range{
			Begin: ts[0],
			End:   ts[1],
		}, nil
	default:
		return nil, fmt.Errorf("invalid argument count to range function: %v", len(ts))
	}
}

func (t Range) Get(arg interface{}) (interface{}, error) {
	beginValue, err := t.Begin.Get(arg)
	if err != nil {
		return nil, fmt.Errorf("couldn't evaluate range function begin expression: %w", err)
	}
	begin, ok := beginValue.(int)
	if !ok {
		return nil, fmt.Errorf("range expected begin argument of type int, got %v of type %s", beginValue, reflect.TypeOf(beginValue))
	}

	endValue, err := t.End.Get(arg)
	if err != nil {
		return nil, fmt.Errorf("couldn't evaluate range function end expression: %w", err)
	}
	end, ok := endValue.(int)
	if !ok {
		return nil, fmt.Errorf("range expected end argument of type int, got %v of type %s", endValue, reflect.TypeOf(endValue))
	}

	out := make([]interface{}, end-begin)
	for i := range out {
		out[i] = begin + i
	}

	return out, nil
}
