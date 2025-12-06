package libatframe_utils_lang_utility

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAssignValueBoolValidInput tests bool conversion with valid bool string inputs.
func TestAssignValueBoolValidInput(t *testing.T) {
	// Arrange - Test scenario: Valid explicit bool string inputs (true, false only with case variations)
	// Note: Numeric 0 and 1, and short forms (t, f, T, F) are intentionally rejected for stricter type safety
	boolType := reflect.TypeOf(true)
	testCases := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"false", false},
		{"True", true},
		{"False", false},
		{"TRUE", true},
		{"FALSE", false},
	}

	for _, tc := range testCases {
		// Act - Call AssignValue with valid bool string input
		result, err := AssignValue(boolType, tc.input)

		// Assert - Verify correct bool conversion and no error
		assert.NoError(t, err, "should parse bool successfully for input: %s", tc.input)
		assert.Equal(t, reflect.Bool, result.Kind())
		assert.Equal(t, tc.expected, result.Bool())
	}
}

// TestAssignValueBoolInvalidInput tests bool conversion with invalid inputs.
func TestAssignValueBoolInvalidInput(t *testing.T) {
	// Arrange - Test scenario: Invalid bool inputs (numeric 0/1 explicitly rejected, plus other invalid values)
	// AssignValue only accepts: true, false (case-insensitive)
	boolType := reflect.TypeOf(true)
	invalidInputs := []string{"yes", "no", "1", "0", "2", "t", "f", "T", "F", "invalid", ""}

	for _, input := range invalidInputs {
		// Act - Call AssignValue with invalid bool string input
		_, err := AssignValue(boolType, input)

		// Assert - Verify error occurs for invalid input
		assert.Error(t, err, "should fail to parse invalid bool input: %s", input)
	}
}

// TestAssignValueIntValidInput tests int type conversion with valid numeric strings.
func TestAssignValueIntValidInput(t *testing.T) {
	// Arrange - Test scenario: Valid int inputs including boundary values
	intType := reflect.TypeOf(int(0))
	testCases := []struct {
		input    string
		expected int64
	}{
		{"0", 0},
		{"123", 123},
		{"-456", -456},
		{"9223372036854775807", 9223372036854775807},   // max int64
		{"-9223372036854775808", -9223372036854775808}, // min int64
	}

	for _, tc := range testCases {
		// Act - Call AssignValue with valid int string input
		result, err := AssignValue(intType, tc.input)

		// Assert - Verify successful int conversion
		assert.NoError(t, err, "should parse int successfully for input: %s", tc.input)
		assert.True(t, result.Kind() >= reflect.Int && result.Kind() <= reflect.Int64)
		assert.Equal(t, tc.expected, result.Int())
	}
}

// TestAssignValueIntInvalidInput tests int type conversion with invalid inputs.
func TestAssignValueIntInvalidInput(t *testing.T) {
	// Arrange - Test scenario: Invalid int inputs (non-numeric strings)
	// Note: strconv.ParseInt with base 0 accepts hex (0x...), so only non-numeric strings fail
	intType := reflect.TypeOf(int(0))
	invalidInputs := []string{"abc", "12.34", "1e5", ""}

	for _, input := range invalidInputs {
		// Act - Call AssignValue with invalid int string input
		_, err := AssignValue(intType, input)

		// Assert - Verify error occurs for invalid input
		assert.Error(t, err, "should fail to parse invalid int input: %s", input)
	}
}

// TestAssignValueUintValidInput tests uint type conversion with valid numeric strings.
func TestAssignValueUintValidInput(t *testing.T) {
	// Arrange - Test scenario: Valid uint inputs including boundary values
	uintType := reflect.TypeOf(uint(0))
	testCases := []struct {
		input    string
		expected uint64
	}{
		{"0", 0},
		{"100", 100},
		{"18446744073709551615", 18446744073709551615}, // max uint64
	}

	for _, tc := range testCases {
		// Act - Call AssignValue with valid uint string input
		result, err := AssignValue(uintType, tc.input)

		// Assert - Verify successful uint conversion
		assert.NoError(t, err, "should parse uint successfully for input: %s", tc.input)
		assert.True(t, result.Kind() >= reflect.Uint && result.Kind() <= reflect.Uint64)
		assert.Equal(t, tc.expected, result.Uint())
	}
}

// TestAssignValueUintInvalidInput tests uint type conversion with invalid inputs including negative.
func TestAssignValueUintInvalidInput(t *testing.T) {
	// Arrange - Test scenario: Invalid uint inputs (negative numbers, non-numeric strings)
	uintType := reflect.TypeOf(uint(0))
	invalidInputs := []string{"-1", "-100", "abc", "12.34", ""}

	for _, input := range invalidInputs {
		// Act - Call AssignValue with invalid uint string input
		_, err := AssignValue(uintType, input)

		// Assert - Verify error occurs for invalid input
		assert.Error(t, err, "should fail to parse invalid uint input: %s", input)
	}
}

// TestAssignValueFloatValidInput tests float type conversion with valid numeric strings.
func TestAssignValueFloatValidInput(t *testing.T) {
	// Arrange - Test scenario: Valid float inputs including scientific notation
	floatType := reflect.TypeOf(float64(0))
	testCases := []struct {
		input    string
		expected float64
	}{
		{"0", 0.0},
		{"3.14", 3.14},
		{"-2.71", -2.71},
		{"1.23e-4", 1.23e-4},
		{"1e10", 1e10},
	}

	for _, tc := range testCases {
		// Act - Call AssignValue with valid float string input
		result, err := AssignValue(floatType, tc.input)

		// Assert - Verify successful float conversion
		assert.NoError(t, err, "should parse float successfully for input: %s", tc.input)
		assert.Equal(t, reflect.Float64, result.Kind())
		assert.InDelta(t, tc.expected, result.Float(), 1e-10)
	}
}

// TestAssignValueFloatInvalidInput tests float type conversion with invalid inputs.
func TestAssignValueFloatInvalidInput(t *testing.T) {
	// Arrange - Test scenario: Invalid float inputs (non-numeric strings)
	// Note: strconv.ParseFloat accepts "NaN" and "Inf" as valid special values, so they are not included
	floatType := reflect.TypeOf(float64(0))
	invalidInputs := []string{"abc", "pi", ""}

	for _, input := range invalidInputs {
		// Act - Call AssignValue with invalid float string input
		_, err := AssignValue(floatType, input)

		// Assert - Verify error occurs for invalid input
		assert.Error(t, err, "should fail to parse invalid float input: %s", input)
	}
}

// TestAssignValueStringValidInput tests string conversion with various string inputs.
func TestAssignValueStringValidInput(t *testing.T) {
	// Arrange - Test scenario: Valid string inputs (empty, special chars, unicode)
	stringType := reflect.TypeOf("")
	testCases := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"", ""}, // empty string
		{"123", "123"},
		{"special!@#$%^&*()", "special!@#$%^&*()"},
		{"中文测试", "中文测试"},                 // unicode
		{"line1\nline2", "line1\nline2"}, // multiline
	}

	for _, tc := range testCases {
		// Act - Call AssignValue with string input
		result, err := AssignValue(stringType, tc.input)

		// Assert - Verify successful string conversion
		assert.NoError(t, err, "should convert string successfully for input: %s", tc.input)
		assert.Equal(t, reflect.String, result.Kind())
		assert.Equal(t, tc.expected, result.String())
	}
}

// TestAssignValueSliceValidInput tests slice conversion with valid JSON array inputs.
func TestAssignValueSliceValidInput(t *testing.T) {
	// Arrange - Test scenario: Valid JSON array inputs for slice type
	sliceIntType := reflect.TypeOf([]int{})
	testCases := []struct {
		input       string
		expectedLen int
	}{
		{"[]", 0}, // empty array
		{"[1,2,3]", 3},
		{"[0]", 1}, // single element
		{"[1,2,3,4,5]", 5},
	}

	for _, tc := range testCases {
		// Act - Call AssignValue with valid JSON array input
		result, err := AssignValue(sliceIntType, tc.input)

		// Assert - Verify successful slice conversion
		assert.NoError(t, err, "should parse slice successfully for input: %s", tc.input)
		assert.Equal(t, reflect.Slice, result.Kind())
		assert.Equal(t, tc.expectedLen, result.Len())
	}
}

// TestAssignValueSliceInvalidInput tests slice conversion with invalid JSON inputs.
func TestAssignValueSliceInvalidInput(t *testing.T) {
	// Arrange - Test scenario: Invalid JSON array inputs for slice type
	sliceIntType := reflect.TypeOf([]int{})
	invalidInputs := []string{
		"[1,2,3", // incomplete JSON
		"not json",
		"[1,abc,3]", // invalid element
		"{}",        // object instead of array
	}

	for _, input := range invalidInputs {
		// Act - Call AssignValue with invalid JSON array input
		_, err := AssignValue(sliceIntType, input)

		// Assert - Verify error occurs for invalid input
		assert.Error(t, err, "should fail to parse invalid slice input: %s", input)
	}
}

// TestAssignValueArrayValidInput tests array conversion with valid JSON array inputs.
func TestAssignValueArrayValidInput(t *testing.T) {
	// Arrange - Test scenario: Valid JSON array inputs for fixed-size array type
	arrayType := reflect.TypeOf([3]int{})
	testCases := []struct {
		input       string
		expectedLen int
	}{
		{"[1,2,3]", 3},
		{"[0,0,0]", 3},
	}

	for _, tc := range testCases {
		// Act - Call AssignValue with valid JSON array input
		result, err := AssignValue(arrayType, tc.input)

		// Assert - Verify successful array conversion
		assert.NoError(t, err, "should parse array successfully for input: %s", tc.input)
		assert.Equal(t, reflect.Array, result.Kind())
		assert.Equal(t, tc.expectedLen, result.Len())
	}
}

// TestAssignValueArrayInvalidInput tests array conversion with invalid inputs.
func TestAssignValueArrayInvalidInput(t *testing.T) {
	// Arrange - Test scenario: Invalid JSON array inputs for array type
	arrayType := reflect.TypeOf([3]int{})
	invalidInputs := []string{
		"[1,2", // incomplete JSON
		"not json",
		"{}", // object instead of array
	}

	for _, input := range invalidInputs {
		// Act - Call AssignValue with invalid JSON array input
		_, err := AssignValue(arrayType, input)

		// Assert - Verify error occurs for invalid input
		assert.Error(t, err, "should fail to parse invalid array input: %s", input)
	}
}

// TestAssignValueMapValidInput tests map conversion with valid JSON object inputs.
func TestAssignValueMapValidInput(t *testing.T) {
	// Arrange - Test scenario: Valid JSON object inputs for map type
	mapType := reflect.TypeOf(map[string]int{})
	testCases := []struct {
		input       string
		expectedLen int
	}{
		{"{}", 0}, // empty map
		{`{"a":1}`, 1},
		{`{"a":1,"b":2,"c":3}`, 3},
	}

	for _, tc := range testCases {
		// Act - Call AssignValue with valid JSON object input
		result, err := AssignValue(mapType, tc.input)

		// Assert - Verify successful map conversion
		assert.NoError(t, err, "should parse map successfully for input: %s", tc.input)
		assert.Equal(t, reflect.Map, result.Kind())
		assert.Equal(t, tc.expectedLen, result.Len())
	}
}

// TestAssignValueMapInvalidInput tests map conversion with invalid JSON inputs.
func TestAssignValueMapInvalidInput(t *testing.T) {
	// Arrange - Test scenario: Invalid JSON object inputs for map type
	mapType := reflect.TypeOf(map[string]int{})
	invalidInputs := []string{
		`{"a":1`, // incomplete JSON
		"not json",
		"[]", // array instead of object
	}

	for _, input := range invalidInputs {
		// Act - Call AssignValue with invalid JSON object input
		_, err := AssignValue(mapType, input)

		// Assert - Verify error occurs for invalid input
		assert.Error(t, err, "should fail to parse invalid map input: %s", input)
	}
}

// TestAssignValueStructValidInput tests struct conversion with valid JSON object inputs.
func TestAssignValueStructValidInput(t *testing.T) {
	// Arrange - Test scenario: Valid JSON object inputs for struct type
	type Person struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	personType := reflect.TypeOf(Person{})
	testCases := []struct {
		input       string
		description string
	}{
		{`{"name":"Alice","age":30}`, "complete struct"},
		{`{"name":"Bob"}`, "partial struct with missing field"},
		{`{}`, "empty struct"},
	}

	for _, tc := range testCases {
		// Act - Call AssignValue with valid JSON object input
		result, err := AssignValue(personType, tc.input)

		// Assert - Verify successful struct conversion
		assert.NoError(t, err, "should parse struct successfully for %s: %s", tc.description, tc.input)
		assert.Equal(t, reflect.Struct, result.Kind())
	}
}

// TestAssignValueStructInvalidInput tests struct conversion with invalid JSON inputs.
func TestAssignValueStructInvalidInput(t *testing.T) {
	// Arrange - Test scenario: Invalid JSON object inputs for struct type
	type Person struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	personType := reflect.TypeOf(Person{})
	invalidInputs := []string{
		`{"name":"Alice"`, // incomplete JSON
		"not json",
		"[]", // array instead of object
	}

	for _, input := range invalidInputs {
		// Act - Call AssignValue with invalid JSON object input
		_, err := AssignValue(personType, input)

		// Assert - Verify error occurs for invalid input
		assert.Error(t, err, "should fail to parse invalid struct input: %s", input)
	}
}

// TestAssignValuePointerValidInput tests pointer conversion with valid inputs.
func TestAssignValuePointerValidInput(t *testing.T) {
	// Arrange - Test scenario: Valid pointer inputs (element values)
	ptrIntType := reflect.TypeOf((*int)(nil))
	testCases := []struct {
		input    string
		expected int64
	}{
		{"123", 123},
		{"0", 0},
		{"-999", -999},
	}

	for _, tc := range testCases {
		// Act - Call AssignValue with valid pointer input
		result, err := AssignValue(ptrIntType, tc.input)

		// Assert - Verify successful pointer conversion
		assert.NoError(t, err, "should parse pointer successfully for input: %s", tc.input)
		assert.Equal(t, reflect.Ptr, result.Kind())
		assert.Equal(t, tc.expected, result.Elem().Int())
	}
}

// TestAssignValuePointerInvalidInput tests pointer conversion with invalid inputs.
func TestAssignValuePointerInvalidInput(t *testing.T) {
	// Arrange - Test scenario: Invalid pointer inputs (invalid element values)
	ptrIntType := reflect.TypeOf((*int)(nil))
	invalidInputs := []string{"abc", "12.34", ""}

	for _, input := range invalidInputs {
		// Act - Call AssignValue with invalid pointer input
		_, err := AssignValue(ptrIntType, input)

		// Assert - Verify error occurs for invalid input
		assert.Error(t, err, "should fail to parse invalid pointer input: %s", input)
	}
}

// TestAssignValuePointerNilInput tests pointer conversion with nil/null inputs.
func TestAssignValuePointerNilInput(t *testing.T) {
	// Arrange - Test scenario: Valid nil pointer inputs (null or nil strings)
	// This tests the ability to create nil pointers through string input
	testCases := []struct {
		input       string
		description string
		pointerType interface{}
	}{
		{"nil", "nil string", (*int)(nil)},
		{"nil", "nil string for string pointer", (*string)(nil)},
	}

	for _, tc := range testCases {
		// Act - Call AssignValue with nil/null input
		result, err := AssignValue(reflect.TypeOf(tc.pointerType), tc.input)

		// Assert - Verify nil pointer creation
		assert.NoError(t, err, "should create nil pointer for %s: %s", tc.description, tc.input)
		assert.Equal(t, reflect.Ptr, result.Kind(), "result should be Ptr kind for %s", tc.description)
		assert.True(t, result.IsNil(), "result should be nil for %s", tc.description)
	}
}

// TestAssignValuePointerNilVsNonNil tests differentiation between nil and non-nil pointers.
func TestAssignValuePointerNilVsNonNil(t *testing.T) {
	// Arrange - Test scenario: Compare nil pointers vs pointers to values
	ptrIntType := reflect.TypeOf((*int)(nil))
	testCases := []struct {
		input       string
		isNil       bool
		description string
	}{
		{"nil", true, "nil input creates nil pointer"},
		{"0", false, "zero value creates non-nil pointer to 0"},
		{"42", false, "positive value creates non-nil pointer to 42"},
		{"-10", false, "negative value creates non-nil pointer to -10"},
	}

	for _, tc := range testCases {
		// Act - Call AssignValue with various pointer inputs
		result, err := AssignValue(ptrIntType, tc.input)

		// Assert - Verify nil/non-nil differentiation
		assert.NoError(t, err, "should parse pointer input: %s", tc.input)
		assert.Equal(t, tc.isNil, result.IsNil(),
			"nil mismatch for %s with input '%s'", tc.description, tc.input)
	}
}

// TestAssignValuePointerToStructNil tests nil pointer to struct type.
func TestAssignValuePointerToStructNil(t *testing.T) {
	// Arrange - Test scenario: Nil pointer to complex struct type
	type Person struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	ptrPersonType := reflect.TypeOf((*Person)(nil))
	testCases := []struct {
		input       string
		isNil       bool
		description string
	}{
		{"nil", true, "nil pointer to struct"},
		{`{"name":"Alice","age":30}`, false, "pointer to struct with values"},
		{`{}`, false, "pointer to empty struct"},
	}

	for _, tc := range testCases {
		// Act - Call AssignValue with struct pointer input
		result, err := AssignValue(ptrPersonType, tc.input)

		// Assert - Verify nil/non-nil for struct pointer
		assert.NoError(t, err, "should parse struct pointer for %s", tc.description)
		assert.Equal(t, tc.isNil, result.IsNil(),
			"nil mismatch for struct pointer: %s", tc.description)
	}
}

// TestAssignValuePointerToSliceNil tests nil pointer to slice type.
func TestAssignValuePointerToSliceNil(t *testing.T) {
	// Arrange - Test scenario: Nil pointer to slice type
	ptrSliceIntType := reflect.TypeOf((*[]int)(nil))
	testCases := []struct {
		input       string
		isNil       bool
		description string
	}{
		{"nil", true, "nil pointer to slice"},
		{"[]", false, "pointer to empty slice"},
		{"[1,2,3]", false, "pointer to slice with values"},
	}

	for _, tc := range testCases {
		// Act - Call AssignValue with slice pointer input
		result, err := AssignValue(ptrSliceIntType, tc.input)

		// Assert - Verify nil/non-nil for slice pointer
		assert.NoError(t, err, "should parse slice pointer for %s", tc.description)
		assert.Equal(t, tc.isNil, result.IsNil(),
			"nil mismatch for slice pointer: %s", tc.description)
	}
}

// TestAssignValueInt8BoundaryMinMax tests int8 with min/max boundary values.
func TestAssignValueInt8BoundaryMinMax(t *testing.T) {
	// Arrange - Test scenario: int8 boundary values (min=-128, max=127)
	// Note: strconv.ParseInt doesn't overflow-check at parse time; it accepts larger values
	// The overflow happens at SetInt which panics for out-of-range values
	int8Type := reflect.TypeOf(int8(0))
	testCases := []struct {
		input       string
		expected    int8
		shouldError bool
	}{
		{"127", 127, false},   // max int8
		{"-128", -128, false}, // min int8
		{"0", 0, false},       // zero
	}

	for _, tc := range testCases {
		// Act - Call AssignValue with int8 input
		result, err := AssignValue(int8Type, tc.input)

		// Assert - Verify conversion
		if tc.shouldError {
			assert.Error(t, err, "should fail for boundary value: %s", tc.input)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, int8(result.Int()))
		}
	}
}

// TestAssignValueUint8BoundaryMinMax tests uint8 with min/max boundary values.
func TestAssignValueUint8BoundaryMinMax(t *testing.T) {
	// Arrange - Test scenario: uint8 boundary values (min=0, max=255)
	uint8Type := reflect.TypeOf(uint8(0))
	testCases := []struct {
		input       string
		expected    uint8
		shouldError bool
	}{
		{"0", 0, false},     // min uint8
		{"255", 255, false}, // max uint8
		{"-1", 0, true},     // invalid: negative not allowed
	}

	for _, tc := range testCases {
		// Act - Call AssignValue with boundary uint8 input
		result, err := AssignValue(uint8Type, tc.input)

		// Assert - Verify boundary handling
		if tc.shouldError {
			assert.Error(t, err, "should fail for boundary value: %s", tc.input)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, uint8(result.Uint()))
		}
	}
}

// TestAssignValueFloat32Precision tests float32 with precision/boundary values.
func TestAssignValueFloat32Precision(t *testing.T) {
	// Arrange - Test scenario: float32 precision and small values
	float32Type := reflect.TypeOf(float32(0))
	testCases := []struct {
		input       string
		description string
	}{
		{"0", "zero"},
		{"1e-45", "smallest positive"},
		{"-1.4e-45", "smallest negative"},
		{"3.4028235e+38", "max float32"},
	}

	for _, tc := range testCases {
		// Act - Call AssignValue with float32 input
		result, err := AssignValue(float32Type, tc.input)

		// Assert - Verify float32 precision handling
		assert.NoError(t, err, "should parse float32 successfully for %s: %s", tc.description, tc.input)
		assert.Equal(t, reflect.Float32, result.Kind())
	}
}

// TestAssignValueEmptyStringInputs tests empty strings with various types.
func TestAssignValueEmptyStringInputs(t *testing.T) {
	// Arrange - Test scenario: Empty string input behavior across different types
	testCases := []struct {
		typeValue   interface{}
		description string
		shouldError bool
	}{
		{int(0), "int", true},       // empty string is invalid for int
		{float64(0), "float", true}, // empty string is invalid for float
		{true, "bool", true},        // empty string is invalid for bool
		{"", "string", false},       // empty string is valid for string type
	}

	for _, tc := range testCases {
		// Act - Call AssignValue with empty string input
		result, err := AssignValue(reflect.TypeOf(tc.typeValue), "")

		// Assert - Verify empty input handling
		if tc.shouldError {
			assert.Error(t, err, "should fail for empty input with type: %s", tc.description)
		} else {
			assert.NoError(t, err)
			if tc.description == "string" {
				assert.Equal(t, "", result.String())
			}
		}
	}
}

// TestAssignValueSliceOfStringsValidInput tests slice of strings conversion.
func TestAssignValueSliceOfStringsValidInput(t *testing.T) {
	// Arrange - Test scenario: Valid JSON array of strings (empty elements, multiple values)
	sliceStringType := reflect.TypeOf([]string{})
	testCases := []struct {
		input       string
		expectedLen int
	}{
		{`["a","b","c"]`, 3},
		{`[""]`, 1}, // single empty string element
		{`[]`, 0},   // empty slice
	}

	for _, tc := range testCases {
		// Act - Call AssignValue with string slice JSON input
		result, err := AssignValue(sliceStringType, tc.input)

		// Assert - Verify string slice conversion
		assert.NoError(t, err, "should parse string slice successfully for input: %s", tc.input)
		assert.Equal(t, reflect.Slice, result.Kind())
		assert.Equal(t, tc.expectedLen, result.Len())
	}
}

// TestAssignValueMapStringToFloatValidInput tests map with string keys and float values.
func TestAssignValueMapStringToFloatValidInput(t *testing.T) {
	// Arrange - Test scenario: Valid JSON object with float values (decimals, zero)
	mapType := reflect.TypeOf(map[string]float64{})
	testCases := []struct {
		input       string
		expectedLen int
	}{
		{`{"pi":3.14,"e":2.71}`, 2},
		{`{"x":0}`, 1}, // zero value
		{`{}`, 0},      // empty map
	}

	for _, tc := range testCases {
		// Act - Call AssignValue with string-float map JSON input
		result, err := AssignValue(mapType, tc.input)

		// Assert - Verify string-float map conversion
		assert.NoError(t, err, "should parse float map successfully for input: %s", tc.input)
		assert.Equal(t, reflect.Map, result.Kind())
		assert.Equal(t, tc.expectedLen, result.Len())
	}
}

// TestAssignValueNestedStructValidInput tests nested struct conversion with valid JSON.
func TestAssignValueNestedStructValidInput(t *testing.T) {
	// Arrange - Test scenario: Nested struct with multiple levels (struct in struct)
	// Demonstrates complex real-world data structures
	type Address struct {
		City    string `json:"city"`
		Country string `json:"country"`
		ZipCode string `json:"zip_code"`
	}
	type Employee struct {
		Name    string  `json:"name"`
		Salary  float64 `json:"salary"`
		Address Address `json:"address"`
	}

	employeeType := reflect.TypeOf(Employee{})
	testCases := []struct {
		input       string
		description string
	}{
		{
			`{"name":"Alice","salary":5000.50,"address":{"city":"NYC","country":"USA","zip_code":"10001"}}`,
			"complete nested struct",
		},
		{
			`{"name":"Bob","salary":4500,"address":{"city":"LA"}}`,
			"partial nested struct with missing fields",
		},
		{
			`{"name":"Charlie","salary":0,"address":{}}`,
			"nested struct with empty address",
		},
	}

	for _, tc := range testCases {
		// Act - Call AssignValue with nested struct JSON input
		result, err := AssignValue(employeeType, tc.input)

		// Assert - Verify nested struct conversion
		assert.NoError(t, err, "should parse nested struct successfully for %s: %s", tc.description, tc.input)
		assert.Equal(t, reflect.Struct, result.Kind())
		assert.Equal(t, "Employee", result.Type().Name())
	}
}

// TestAssignValueNestedStructInvalidInput tests nested struct with invalid JSON.
func TestAssignValueNestedStructInvalidInput(t *testing.T) {
	// Arrange - Test scenario: Invalid JSON for nested struct type
	type Address struct {
		City    string `json:"city"`
		Country string `json:"country"`
	}
	type Person struct {
		Name    string  `json:"name"`
		Address Address `json:"address"`
	}

	personType := reflect.TypeOf(Person{})
	invalidInputs := []string{
		`{"name":"Alice","address":{"city":"NYC"`,   // incomplete JSON
		`{"name":"Alice","address":"not a struct"}`, // wrong type for nested field
		`{"name":"Alice","address":[]}`,             // array instead of object
		"not json",
	}

	for _, input := range invalidInputs {
		// Act - Call AssignValue with invalid nested struct JSON
		_, err := AssignValue(personType, input)

		// Assert - Verify error occurs for invalid input
		assert.Error(t, err, "should fail to parse invalid nested struct: %s", input)
	}
}

// TestAssignValueSliceOfStructsValidInput tests slice of structs conversion.
func TestAssignValueSliceOfStructsValidInput(t *testing.T) {
	// Arrange - Test scenario: Slice containing structs (complex collection type)
	type Item struct {
		ID    int     `json:"id"`
		Name  string  `json:"name"`
		Price float64 `json:"price"`
	}

	sliceItemType := reflect.TypeOf([]Item{})
	testCases := []struct {
		input       string
		expectedLen int
		description string
	}{
		{
			`[{"id":1,"name":"Item1","price":9.99},{"id":2,"name":"Item2","price":19.99}]`,
			2,
			"multiple items",
		},
		{
			`[{"id":100,"name":"OnlyOne"}]`,
			1,
			"single item with partial fields",
		},
		{
			`[]`,
			0,
			"empty slice",
		},
	}

	for _, tc := range testCases {
		// Act - Call AssignValue with struct slice JSON input
		result, err := AssignValue(sliceItemType, tc.input)

		// Assert - Verify struct slice conversion
		assert.NoError(t, err, "should parse struct slice successfully for %s: %s", tc.description, tc.input)
		assert.Equal(t, reflect.Slice, result.Kind())
		assert.Equal(t, tc.expectedLen, result.Len())
	}
}

// TestAssignValueMapOfIntegersValidInput tests map with string keys and integer values.
func TestAssignValueMapOfIntegersValidInput(t *testing.T) {
	// Arrange - Test scenario: Map with basic types (practical JSON structure)
	mapType := reflect.TypeOf(map[string]int{})
	testCases := []struct {
		input       string
		expectedLen int
		description string
	}{
		{
			`{"requests":1000,"errors":5,"warnings":20}`,
			3,
			"multiple metrics",
		},
		{
			`{"timeout":30}`,
			1,
			"single metric",
		},
		{
			`{}`,
			0,
			"empty map",
		},
	}

	for _, tc := range testCases {
		// Act - Call AssignValue with integer map JSON input
		result, err := AssignValue(mapType, tc.input)

		// Assert - Verify integer map conversion
		assert.NoError(t, err, "should parse int map successfully for %s: %s", tc.description, tc.input)
		assert.Equal(t, reflect.Map, result.Kind())
		assert.Equal(t, tc.expectedLen, result.Len())
	}
}

// TestAssignValueStructWithSliceAndMapValidInput tests struct containing both slice and map.
func TestAssignValueStructWithSliceAndMapValidInput(t *testing.T) {
	// Arrange - Test scenario: Complex struct with nested slice and map fields
	type Metadata struct {
		Tags    []string       `json:"tags"`
		Metrics map[string]int `json:"metrics"`
		Name    string         `json:"name"`
	}

	metadataType := reflect.TypeOf(Metadata{})
	testCases := []struct {
		input       string
		description string
	}{
		{
			`{"name":"service1","tags":["prod","api"],"metrics":{"requests":1000,"errors":5}}`,
			"complete metadata",
		},
		{
			`{"name":"service2","tags":[],"metrics":{}}`,
			"empty slice and map",
		},
		{
			`{"name":"service3","tags":["test","dev"],"metrics":{"latency":100}}`,
			"minimal metrics",
		},
	}

	for _, tc := range testCases {
		// Act - Call AssignValue with complex struct JSON input
		result, err := AssignValue(metadataType, tc.input)

		// Assert - Verify complex struct conversion
		assert.NoError(t, err, "should parse complex struct successfully for %s: %s", tc.description, tc.input)
		assert.Equal(t, reflect.Struct, result.Kind())
		assert.Equal(t, "Metadata", result.Type().Name())
	}
}

// TestAssignValueStructWithSliceAndMapInvalidInput tests invalid JSON for complex struct.
func TestAssignValueStructWithSliceAndMapInvalidInput(t *testing.T) {
	// Arrange - Test scenario: Invalid JSON for complex struct with nested collections
	type Data struct {
		Tags    []string       `json:"tags"`
		Metrics map[string]int `json:"metrics"`
	}

	dataType := reflect.TypeOf(Data{})
	invalidInputs := []string{
		`{"tags":"not array","metrics":{}}`,     // tags should be array
		`{"tags":[],"metrics":"not map"}`,       // metrics should be object
		`{"tags":[1,2,3],"metrics":{}}`,         // tags elements should be strings
		`{"tags":[],"metrics":{"key":"value"}}`, // metrics values should be numbers
		`{"tags":[`,                             // incomplete JSON
	}

	for _, input := range invalidInputs {
		// Act - Call AssignValue with invalid complex struct JSON
		_, err := AssignValue(dataType, input)

		// Assert - Verify error occurs for invalid nested collection input
		assert.Error(t, err, "should fail to parse invalid complex struct: %s", input)
	}
}

// TestAssignValueMapWithStringKeysValidInput tests map with string keys and various value types.
func TestAssignValueMapWithStringKeysValidInput(t *testing.T) {
	// Arrange - Test scenario: Map with basic string values (common configuration pattern)
	mapType := reflect.TypeOf(map[string]string{})
	testCases := []struct {
		input       string
		expectedLen int
		description string
	}{
		{
			`{"host":"localhost","port":"8080","database":"mydb"}`,
			3,
			"connection configuration",
		},
		{
			`{"name":"service","version":"1.0.0"}`,
			2,
			"metadata",
		},
		{
			`{}`,
			0,
			"empty map",
		},
	}

	for _, tc := range testCases {
		// Act - Call AssignValue with string-string map JSON input
		result, err := AssignValue(mapType, tc.input)

		// Assert - Verify string map conversion
		assert.NoError(t, err, "should parse string map successfully for %s: %s", tc.description, tc.input)
		assert.Equal(t, reflect.Map, result.Kind())
		assert.Equal(t, tc.expectedLen, result.Len())
	}
}

// TestAssignValueMapWithStringKeysInvalidInput tests invalid JSON for string maps.
func TestAssignValueMapWithStringKeysInvalidInput(t *testing.T) {
	// Arrange - Test scenario: Invalid JSON for string map
	mapType := reflect.TypeOf(map[string]string{})
	invalidInputs := []string{
		`{"key":"value"`,    // incomplete JSON
		`[{"key":"value"}]`, // array instead of object
		`not json`,          // not JSON at all
	}

	for _, input := range invalidInputs {
		// Act - Call AssignValue with invalid map JSON
		_, err := AssignValue(mapType, input)

		// Assert - Verify error occurs
		assert.Error(t, err, "should fail to parse invalid string map: %s", input)
	}
}

// TestAssignValueSliceOfNestedStructsValidInput tests slice of structs with nested fields.
func TestAssignValueSliceOfNestedStructsValidInput(t *testing.T) {
	// Arrange - Test scenario: Slice of structs that contain slices/collections
	type Tag struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}
	type Article struct {
		Title string `json:"title"`
		Tags  []Tag  `json:"tags"`
	}

	articleSliceType := reflect.TypeOf([]Article{})
	testCases := []struct {
		input       string
		expectedLen int
		description string
	}{
		{
			`[{"title":"Article1","tags":[{"name":"tag1","value":"v1"},{"name":"tag2","value":"v2"}]},{"title":"Article2","tags":[]}]`,
			2,
			"multiple articles with varying tags",
		},
		{
			`[{"title":"Single","tags":[{"name":"tech","value":"go"}]}]`,
			1,
			"single article with one tag",
		},
		{
			`[]`,
			0,
			"empty article slice",
		},
	}

	for _, tc := range testCases {
		// Act - Call AssignValue with nested struct slice JSON
		result, err := AssignValue(articleSliceType, tc.input)

		// Assert - Verify nested struct slice conversion
		assert.NoError(t, err, "should parse article slice successfully for %s: %s", tc.description, tc.input)
		assert.Equal(t, reflect.Slice, result.Kind())
		assert.Equal(t, tc.expectedLen, result.Len())
	}
}

// TestAssignValueSliceOfNestedStructsInvalidInput tests invalid data for slice of nested structs.
func TestAssignValueSliceOfNestedStructsInvalidInput(t *testing.T) {
	// Arrange - Test scenario: Invalid data for slice containing structs with nested collections
	type Config struct {
		Name    string   `json:"name"`
		Options []string `json:"options"`
	}

	configSliceType := reflect.TypeOf([]Config{})
	invalidInputs := []string{
		`[{"name":"cfg1","options":"not array"}]`, // options should be array
		`[{"name":"cfg2","options":[1,2,3]}]`,     // options should be string array
		`[{"name":"cfg3","options":["a"`,          // incomplete JSON
		`{"name":"cfg4","options":["x"]}`,         // should be array of objects, not single object
	}

	for _, input := range invalidInputs {
		// Act - Call AssignValue with invalid data
		_, err := AssignValue(configSliceType, input)

		// Assert - Verify error occurs
		assert.Error(t, err, "should fail to parse invalid config slice: %s", input)
	}
}

// TestAssignValueComplexNestedStructValidInput tests deeply nested struct structure.
func TestAssignValueComplexNestedStructValidInput(t *testing.T) {
	// Arrange - Test scenario: Complex nested structure (4 levels deep)
	type Permission struct {
		Read  bool `json:"read"`
		Write bool `json:"write"`
	}
	type Role struct {
		Name       string     `json:"name"`
		Permission Permission `json:"permission"`
	}
	type User struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
		Role Role   `json:"role"`
	}

	userType := reflect.TypeOf(User{})
	testCases := []struct {
		input       string
		description string
	}{
		{
			`{"id":123,"name":"Alice","role":{"name":"admin","permission":{"read":true,"write":true}}}`,
			"complete user with full permissions",
		},
		{
			`{"id":456,"name":"Bob","role":{"name":"user","permission":{"read":true,"write":false}}}`,
			"user with read-only permission",
		},
		{
			`{"id":789,"name":"Guest","role":{"name":""}}`,
			"guest with partial role data",
		},
	}

	for _, tc := range testCases {
		// Act - Call AssignValue with deeply nested struct JSON
		result, err := AssignValue(userType, tc.input)

		// Assert - Verify deeply nested struct conversion
		assert.NoError(t, err, "should parse user struct successfully for %s: %s", tc.description, tc.input)
		assert.Equal(t, reflect.Struct, result.Kind())
		assert.Equal(t, "User", result.Type().Name())
	}
}

// TestAssignValueComplexNestedStructInvalidInput tests invalid data for deeply nested structs.
func TestAssignValueComplexNestedStructInvalidInput(t *testing.T) {
	// Arrange - Test scenario: Invalid data at various nesting levels
	type Permission struct {
		Read bool `json:"read"`
	}
	type Role struct {
		Name       string     `json:"name"`
		Permission Permission `json:"permission"`
	}
	type User struct {
		Name string `json:"name"`
		Role Role   `json:"role"`
	}

	userType := reflect.TypeOf(User{})
	invalidInputs := []string{
		`{"name":"Alice","role":{"name":"admin","permission":"not struct"}}`,         // permission should be object
		`{"name":"Bob","role":"not struct"}`,                                         // role should be object
		`{"name":"Charlie","role":{"name":"user","permission":{"read":"not bool"}}}`, // read should be bool
		`{"name":"Dave"`, // incomplete JSON
	}

	for _, input := range invalidInputs {
		// Act - Call AssignValue with invalid nested data
		_, err := AssignValue(userType, input)

		// Assert - Verify error occurs for invalid nested structure
		assert.Error(t, err, "should fail to parse invalid user: %s", input)
	}
}

// ========================= FormatValues Tests =========================

// TestFormatValuesBasicTypes tests FormatValues with basic type values.
func TestFormatValuesBasicTypes(t *testing.T) {
	// Arrange - Test scenario: Format various basic types to strings
	values := []reflect.Value{
		reflect.ValueOf("hello"),
		reflect.ValueOf(int(42)),
		reflect.ValueOf(int8(-10)),
		reflect.ValueOf(int16(1000)),
		reflect.ValueOf(int32(50000)),
		reflect.ValueOf(int64(9223372036854775807)),
		reflect.ValueOf(uint(100)),
		reflect.ValueOf(uint8(255)),
		reflect.ValueOf(uint16(65535)),
		reflect.ValueOf(uint32(4000000000)),
		reflect.ValueOf(uint64(18446744073709551615)),
		reflect.ValueOf(float32(3.14)),
		reflect.ValueOf(float64(2.718281828)),
		reflect.ValueOf(true),
		reflect.ValueOf(false),
	}

	// Act - Call FormatValues with mixed basic types
	result := FormatValues(values)

	// Assert - Verify all values are formatted correctly
	assert.Equal(t, "hello", result[0])
	assert.Equal(t, "42", result[1])
	assert.Equal(t, "-10", result[2])
	assert.Equal(t, "1000", result[3])
	assert.Equal(t, "50000", result[4])
	assert.Equal(t, "9223372036854775807", result[5])
	assert.Equal(t, "100", result[6])
	assert.Equal(t, "255", result[7])
	assert.Equal(t, "65535", result[8])
	assert.Equal(t, "4000000000", result[9])
	assert.Equal(t, "18446744073709551615", result[10])
	// Float values are formatted with %f (default precision)
	assert.Contains(t, result[11], "3.14")
	assert.Contains(t, result[12], "2.71")
	assert.Equal(t, "true", result[13])
	assert.Equal(t, "false", result[14])
}

// TestFormatValuesEmptyInput tests FormatValues with empty input.
func TestFormatValuesEmptyInput(t *testing.T) {
	// Arrange - Test scenario: Empty slice of values
	values := []reflect.Value{}

	// Act - Call FormatValues with empty input
	result := FormatValues(values)

	// Assert - Verify empty output
	assert.NotNil(t, result) // Should be empty slice, not nil
}

// TestFormatValuesNegativeNumbers tests FormatValues with negative integer values.
func TestFormatValuesNegativeNumbers(t *testing.T) {
	// Arrange - Test scenario: Negative numbers (edge cases)
	values := []reflect.Value{
		reflect.ValueOf(int(-2147483648)),            // min int32
		reflect.ValueOf(int32(-2147483648)),          // min int32
		reflect.ValueOf(int64(-9223372036854775808)), // min int64
		reflect.ValueOf(float32(-1.5)),
		reflect.ValueOf(float64(-999.999)),
	}

	// Act - Call FormatValues with negative numbers
	result := FormatValues(values)

	// Assert - Verify negative formatting
	assert.Contains(t, result[0], "-2147483648")
	assert.Equal(t, "-2147483648", result[1])
	assert.Equal(t, "-9223372036854775808", result[2])
	assert.Contains(t, result[3], "-1.5")
	assert.Contains(t, result[4], "-999.999")
}

// TestFormatValuesZeroValues tests FormatValues with zero values of various types.
func TestFormatValuesZeroValues(t *testing.T) {
	// Arrange - Test scenario: Zero values for all numeric types
	values := []reflect.Value{
		reflect.ValueOf(""),           // empty string
		reflect.ValueOf(int(0)),       // zero int
		reflect.ValueOf(int8(0)),      // zero int8
		reflect.ValueOf(uint(0)),      // zero uint
		reflect.ValueOf(uint64(0)),    // zero uint64
		reflect.ValueOf(float32(0.0)), // zero float32
		reflect.ValueOf(float64(0.0)), // zero float64
		reflect.ValueOf(false),        // false bool
	}

	// Act - Call FormatValues with zero values
	result := FormatValues(values)

	// Assert - Verify zero formatting
	assert.Equal(t, "", result[0])
	assert.Equal(t, "0", result[1])
	assert.Equal(t, "0", result[2])
	assert.Equal(t, "0", result[3])
	assert.Equal(t, "0", result[4])
	assert.Contains(t, result[5], "0")
	assert.Contains(t, result[6], "0")
	assert.Equal(t, "false", result[7])
}

// TestFormatValuesSpecialFloatValues tests FormatValues with special float values.
func TestFormatValuesSpecialFloatValues(t *testing.T) {
	// Arrange - Test scenario: Very small, very large, and special float values
	values := []reflect.Value{
		reflect.ValueOf(float32(1e-10)),
		reflect.ValueOf(float64(1e20)),
		reflect.ValueOf(float32(0.0001)),
		reflect.ValueOf(float64(999999.999)),
	}

	// Act - Call FormatValues with special float values
	result := FormatValues(values)

	// Assert - Verify special float formatting
	// All should be formatted with %f
	assert.NotEmpty(t, result[0])
	assert.NotEmpty(t, result[1])
	assert.NotEmpty(t, result[2])
	assert.NotEmpty(t, result[3])
}

// TestFormatValuesErrorInterface tests FormatValues with error interface values.
func TestFormatValuesErrorInterface(t *testing.T) {
	// Arrange - Test scenario: error interface with actual error instances
	testErr := fmt.Errorf("test error")
	values := []reflect.Value{
		reflect.ValueOf(interface{}(testErr)),
		reflect.ValueOf(interface{}("string value")),
		reflect.ValueOf(interface{}(int(42))),
	}

	// Act - Call FormatValues with interface values
	result := FormatValues(values)

	// Assert - Verify interface handling
	// Error is formatted as "error: <error message>"
	assert.True(t, (result[0] == "error: test error" || result[0] == "test error"),
		"unexpected error format: %s", result[0])
	assert.Equal(t, "string value", result[1])
	assert.Equal(t, "42", result[2])
}

// TestFormatValuesComplexTypes tests FormatValues with complex/pointer/slice types.
func TestFormatValuesComplexTypes(t *testing.T) {
	// Arrange - Test scenario: Complex types (slice, map, struct, pointer)
	type Person struct {
		Name string
		Age  int
	}

	values := []reflect.Value{
		reflect.ValueOf([]int{1, 2, 3}),
		reflect.ValueOf(map[string]int{"a": 1, "b": 2}),
		reflect.ValueOf(Person{Name: "Alice", Age: 30}),
		reflect.ValueOf((*int)(nil)),
	}

	// Act - Call FormatValues with complex types
	result := FormatValues(values)

	// Assert - Verify complex type formatting
	assert.Contains(t, result[0], "[1 2 3]") // slice formatted with %v
	assert.Contains(t, result[1], "map")     // map formatted with %v
	assert.Contains(t, result[2], "Alice")   // struct formatted with %v
	assert.Contains(t, result[3], "")        // nil pointer
}

// TestFormatValuesBoundaryValues tests FormatValues with boundary values.
func TestFormatValuesBoundaryValues(t *testing.T) {
	// Arrange - Test scenario: Maximum and minimum values for each type
	testCases := []struct {
		value       interface{}
		description string
	}{
		{int8(127), "max int8"},
		{int8(-128), "min int8"},
		{uint8(255), "max uint8"},
		{int16(32767), "max int16"},
		{int16(-32768), "min int16"},
		{uint16(65535), "max uint16"},
		{int32(2147483647), "max int32"},
		{int32(-2147483648), "min int32"},
		{uint32(4294967295), "max uint32"},
		{int64(9223372036854775807), "max int64"},
		{int64(-9223372036854775808), "min int64"},
		{uint64(18446744073709551615), "max uint64"},
	}

	for _, tc := range testCases {
		// Act - Call FormatValues with boundary value
		values := []reflect.Value{reflect.ValueOf(tc.value)}
		result := FormatValues(values)

		// Assert - Verify boundary value formatting
		assert.NotEmpty(t, result[0], "empty result for %s", tc.description)
	}
}

// TestFormatValuesLargeInput tests FormatValues with large number of values.
func TestFormatValuesLargeInput(t *testing.T) {
	// Arrange - Test scenario: Large slice of values
	values := make([]reflect.Value, 0, 100)
	for i := 0; i < 100; i++ {
		values = append(values, reflect.ValueOf(i))
	}

	// Act - Call FormatValues with large input
	result := FormatValues(values)

	// Assert - Verify all values formatted
	for i := 0; i < 100; i++ {
		assert.Equal(t, fmt.Sprintf("%d", i), result[i], "mismatch at index %d", i)
	}
}

// TestFormatValuesMixedTypesSequence tests FormatValues with alternating types.
func TestFormatValuesMixedTypesSequence(t *testing.T) {
	// Arrange - Test scenario: Alternating different types in sequence
	values := []reflect.Value{
		reflect.ValueOf("str1"),
		reflect.ValueOf(int(10)),
		reflect.ValueOf(float64(3.14)),
		reflect.ValueOf(true),
		reflect.ValueOf("str2"),
		reflect.ValueOf(uint(50)),
		reflect.ValueOf(float32(2.71)),
		reflect.ValueOf(false),
	}

	// Act - Call FormatValues with mixed sequence
	result := FormatValues(values)

	// Assert - Verify mixed sequence formatting
	assert.Equal(t, "str1", result[0])
	assert.Equal(t, "10", result[1])
	assert.Contains(t, result[2], "3.14")
	assert.Equal(t, "true", result[3])
	assert.Equal(t, "str2", result[4])
	assert.Equal(t, "50", result[5])
	assert.Contains(t, result[6], "2.71")
	assert.Equal(t, "false", result[7])
}
