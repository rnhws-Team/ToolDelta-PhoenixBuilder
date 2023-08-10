package mcstructure

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

func stringifyNBTInterface(input interface{}) (string, error) {
	switch reflect.TypeOf(input).Kind() {
	case reflect.Uint8:
		return fmt.Sprintf("%db", int(input.(byte))), nil
		// byte
	case reflect.Int16:
		return fmt.Sprintf("%ds", input.(int16)), nil
		// short
	case reflect.Int32:
		return fmt.Sprintf("%d", input.(int32)), nil
		// int
	case reflect.Int64:
		return fmt.Sprintf("%dl", input.(int64)), nil
		// long
	case reflect.Float32:
		return fmt.Sprintf("%sf", strconv.FormatFloat(float64(input.(float32)), 'f', -1, 32)), nil
		// float
	case reflect.Float64:
		return fmt.Sprintf("%sf", strconv.FormatFloat(float64(input.(float64)), 'f', -1, 32)), nil
		// double
	case reflect.Array:
		ans := []string{}
		value := reflect.ValueOf(input)
		// prepare
		switch reflect.TypeOf(input).Elem().Kind() {
		case reflect.Uint8:
			for i := 0; i < value.Len(); i++ {
				ans = append(ans, fmt.Sprintf("%db", int(value.Index(i).Interface().(byte))))
			}
			return fmt.Sprintf("[B; %s]", strings.Join(ans, ", ")), nil
			// byte_array
		case reflect.Int32:
			for i := 0; i < value.Len(); i++ {
				ans = append(ans, fmt.Sprintf("%d", value.Index(i).Interface().(int32)))
			}
			return fmt.Sprintf("[I; %s]", strings.Join(ans, ", ")), nil
			// int_array
		case reflect.Int64:
			for i := 0; i < value.Len(); i++ {
				ans = append(ans, fmt.Sprintf("%dl", value.Index(i).Interface().(int64)))
			}
			return fmt.Sprintf("[L; %s]", strings.Join(ans, ", ")), nil
			// long_array
		}
		// byte_array, int_array, long_array
	case reflect.String:
		return fmt.Sprintf("%#v", input.(string)), nil
		// string
	case reflect.Slice:
		value := input.([]interface{})
		list, err := ConvertListToString(value)
		if err != nil {
			return "", fmt.Errorf("stringifyNBTInterface: Failed in %#v", value)
		}
		return list, nil
		// list
	case reflect.Map:
		value := input.(map[string]interface{})
		compound, err := ConvertCompoundToString(value)
		if err != nil {
			return "", fmt.Errorf("stringifyNBTInterface: Failed in %#v", value)
		}
		return compound, nil
		// compound
	}
	return "", fmt.Errorf("stringifyNBTInterface: Failed because of unknown type of the target data, occurred in %#v", input)
}

func ConvertCompoundToString(input map[string]interface{}) (string, error) {
	ans := make([]string, 0)
	for key, value := range input {
		if value == nil {
			return "", fmt.Errorf("ConvertCompoundToString: Crashed in input[\"%v\"]; errorLogs = value is nil; input = %#v", key, input)
		}
		got, err := stringifyNBTInterface(value)
		if err != nil {
			return "", fmt.Errorf("ConvertCompoundToString: Crashed in input[\"%v\"]; errorLogs = %v; input = %#v", key, err, input)
		}
		ans = append(ans, fmt.Sprintf("%#v: %s", key, got))
	}
	return fmt.Sprintf("{%s}", strings.Join(ans, ", ")), nil
}

func ConvertListToString(input []interface{}) (string, error) {
	ans := make([]string, 0)
	for key, value := range input {
		if value == nil {
			return "", fmt.Errorf("ConvertListToString: Crashed in input[\"%v\"]; errorLogs = value is nil; input = %#v", key, input)
		}
		got, err := stringifyNBTInterface(value)
		if err != nil {
			return "", fmt.Errorf("ConvertListToString: Crashed in input[\"%v\"]; errorLogs = %v; input = %#v", key, err, input)
		}
		ans = append(ans, got)
	}
	return fmt.Sprintf("[%s]", strings.Join(ans, ", ")), nil
}
