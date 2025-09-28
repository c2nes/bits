package main

import (
	"io"
	"math"
	"os"
	"reflect"
	"strings"
	"testing"
)

// floatEquals compares two floats for approximate equality.
func floatEquals(a, b float64) bool {
	const tolerance = 1e-9
	return math.Abs(a-b) < tolerance
}

func TestRun(t *testing.T) {
	testCases := []struct {
		name          string
		script        string
		expectedStack []any
		expectedErr   string
	}{
		// Basic Arithmetic
		{"add", "2 3 +", []any{uint64(5)}, ""},
		{"subtract", "5 2 -", []any{uint64(3)}, ""},
		{"multiply", "3 4 *", []any{uint64(12)}, ""},
		{"divide", "10 2 /", []any{uint64(5)}, ""},
		{"exponent", "2 8 **", []any{uint64(256)}, ""},

		// Number Bases
		{"hex add", "0x10 0x20 +", []any{uint64(0x30)}, ""},
		{"bin add", "0b10 0b11 +", []any{uint64(5)}, ""},
		{"hex float", "0x1.999999999999ap-4", []any{float64(0.1)}, ""},
		{"bin float", "0b1.1001100110011001100110011001101p-4", []any{float64(0.1)}, ""},

		// Floating Point & Mixed
		{"float add", "1.5 2.5 +", []any{float64(4.0)}, ""},
		{"mixed add", "10 1.5 +", []any{float64(11.5)}, ""},

		// Stack Manipulation
		{"dup", "1 2 dup", []any{uint64(1), uint64(2), uint64(2)}, ""},
		{"dup alias", "1 2 .", []any{uint64(1), uint64(2), uint64(2)}, ""},
		{"swap", "1 2 swap", []any{uint64(2), uint64(1)}, ""},
		{"swap alias", "1 2 x", []any{uint64(2), uint64(1)}, ""},
		{"drop", "1 2 drop", []any{uint64(1)}, ""},

		// Type Conversions
		{"i32 conv", "3.14 i32", []any{int32(3)}, ""},
		{"u8 conv", "255 u8", []any{uint8(255)}, ""},
		{"f64 conv", "10 i32 f64", []any{float64(10)}, ""},

		// Bitwise Operations
		{"and", "0x0f 0xf0 &", []any{uint64(0)}, ""},
		{"or", "0x0f 0xf0 |", []any{uint64(0xff)}, ""},
		{"xor", "0x55 0xff ^", []any{uint64(0xaa)}, ""},
		{"not", "0xffffffffffffffff ~", []any{uint64(0)}, ""},
		{"shl", "1 8 <<", []any{uint64(256)}, ""},
		{"shr", "256 4 >>", []any{uint64(16)}, ""},

		// Unary Operations
		{"negate", "10 neg", []any{int64(-10)}, ""},
		{"negate alias", "10 !", []any{int64(-10)}, ""},

		// Float/Bits Conversion
		{"bits", "1.0 f64 bits", []any{uint64(0x3ff0000000000000)}, ""},
		{"fbits", "0x3ff0000000000000 fbits", []any{float64(1.0)}, ""},

		// Comments
		{"comment", "1 2 + # comment", []any{uint64(3)}, ""},
		{"line comment", "// ignore this\n" + "5 5 +", []any{uint64(10)}, ""},

		// Edge Cases & Errors
		{"stack underflow", "1 +", nil, "runtime error: index out of range"},
		{"syntax error", "1 foo", nil, `syntax error at "foo"`},
		{"division by zero", "1 0 /", nil, "runtime error: integer divide by zero"},
		{"float division by zero", "1.0 0.0 /", []any{math.Inf(1)}, ""},
		{"uint8 overflow", "255 u8 1 +", []any{uint8(0)}, ""},
		{"int8 max", "i8max", []any{int8(math.MaxInt8)}, ""},
		{"int32 overflow", "i32max 1 +", []any{int32(math.MinInt32)}, ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var stack Stack
			input := stringInput(tc.script)
			_, err := run(&stack, input)

			if tc.expectedErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, but got none", tc.expectedErr)
				}
				if !strings.Contains(err.Error(), tc.expectedErr) {
					t.Fatalf("expected error to contain %q, but got %q", tc.expectedErr, err.Error())
				}
				return
			}

			if err != nil && err != io.EOF {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(tc.expectedStack) != stack.Len() {
				t.Fatalf("expected stack length %d, but got %d. Stack: %s", len(tc.expectedStack), stack.Len(), stack.List())
			}

			for i, expected := range tc.expectedStack {
				got := stack.At(i).val
				if fexp, ok := expected.(float64); ok {
					if fgot, ok2 := got.(float64); ok2 {
						if math.IsNaN(fexp) {
							if !math.IsNaN(fgot) {
								t.Errorf("stack item %d: expected NaN, but got %v", i, fgot)
							}
						} else if math.IsInf(fexp, 0) {
							if fexp != fgot {
								t.Errorf("stack item %d: expected %v, but got %v", i, fexp, fgot)
							}
						} else if !floatEquals(fexp, fgot) {
							t.Errorf("stack item %d: expected %v, but got %v", i, fexp, fgot)
						}
					} else {
						t.Errorf("stack item %d: expected type float64, but got %T", i, got)
					}
				} else if !reflect.DeepEqual(expected, got) {
					t.Errorf("stack item %d: expected %v (%T), but got %v (%T)", i, expected, expected, got, got)
				}
			}
		})
	}
}

func TestTokenize(t *testing.T) {
	testCases := []struct {
		name           string
		script         string
		expectedTokens []any
		expectedErr    string
	}{
		{
			name:           "basic with comment",
			script:         "1 2 + 0x10 // comment\n-3.14",
			expectedTokens: []any{uint64(1), uint64(2), OpAdd, uint64(16), float64(-3.14)},
		},
		{
			name:           "all operators",
			script:         "<< >> ** * / - + ^ | & ~ !",
			expectedTokens: []any{OpShl, OpShr, OpExp, OpMul, OpDiv, OpSub, OpAdd, OpXor, OpOr, OpAnd, OpNot, OpNeg},
		},
		{
			name:        "syntax error",
			script:      "1 2 $",
			expectedErr: `syntax error at "$"`,
		},
		{
			name:           "empty script",
			script:         "",
			expectedTokens: []any{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tokens, err := tokenize(tc.script)

			if tc.expectedErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, but got none", tc.expectedErr)
				}
				if !strings.Contains(err.Error(), tc.expectedErr) {
					t.Fatalf("expected error to contain %q, but got %q", tc.expectedErr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(tc.expectedTokens, tokens) {
				t.Errorf("expected tokens %v (%T), but got %v (%T)", tc.expectedTokens, tc.expectedTokens, tokens, tokens)
			}
		})
	}
}

func TestParseHexFloat(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected float64
	}{
		{"simple", "0x1p0", 1.0},
		{"fraction", "0x0.8p0", 0.5},
		{"exponent", "0x1p10", 1024.0},
		{"negative exponent", "0x1p-1", 0.5},
		{"complex", "0x1.8p1", 3.0},
		{"negative", "-0x1p0", -1.0},
		{"from test", "0x1.999999999999ap-4", 0.1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			val, err := parseHex(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			fval, ok := val.(float64)
			if !ok {
				t.Fatalf("expected float64, but got %T", val)
			}
			if !floatEquals(tc.expected, fval) {
				t.Fatalf("expected %f, but got %f", tc.expected, fval)
			}
		})
	}
}

func TestSanitizeArgs(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	testCases := []struct {
		name     string
		args     []string
		expected []string
	}{
		{"no args", []string{"bits"}, []string{"bits"}},
		{"flag only", []string{"bits", "-q"}, []string{"bits", "-q"}},
		{"number", []string{"bits", "1"}, []string{"bits", "--", "1"}},
		{"negative number", []string{"bits", "-1"}, []string{"bits", "--", "-1"}},
		{"flag then number", []string{"bits", "-q", "1"}, []string{"bits", "-q", "--", "1"}},
		{"number then flag", []string{"bits", "1", "-q"}, []string{"bits", "--", "1", "-q"}},
		{"-- already present", []string{"bits", "--", "1"}, []string{"bits", "--", "1"}},
		{"-- with flag", []string{"bits", "-q", "--", "1"}, []string{"bits", "-q", "--", "1"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			os.Args = tc.args
			sanitizeArgs()
			if !reflect.DeepEqual(os.Args, tc.expected) {
				t.Errorf("expected args %v, but got %v", tc.expected, os.Args)
			}
		})
	}
}
