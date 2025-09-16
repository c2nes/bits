package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var reDecNumber = regexp.MustCompile(`(?i)^[+-]?\d+(\.\d+)?(e[+-]?\d+)?`)
var reHexNumber = regexp.MustCompile(`(?i)^[+-]?0x[0-9a-f]+(\.[0-9a-f]+)?(p[+-]?\d+)?`)
var reBinNumber = regexp.MustCompile(`(?i)^[+-]?0b[01]+(\.[01]+)?(p[+-]?\d+)?`)

type Op int

const (
	OpShl Op = iota
	OpShr
	OpExp
	OpMul
	OpDiv
	OpSub
	OpAdd
	OpXor
	OpOr
	OpAnd
	OpNot
	OpI8
	OpI16
	OpI32
	OpI64
	OpU8
	OpU16
	OpU32
	OpU64
	OpF32
	OpF64
	OpF32Bits
	OpF64Bits
	OpBitsF32
	OpBitsF64
)

var tokenMap = []struct {
	s string
	v any
}{
	// Operators are tested in order. If one operator is the prefix of
	// another, the longer operator should come first (e.g. "**" should
	// come before "*").
	{"<<", OpShl},
	{">>", OpShr},
	{"**", OpExp},
	{"*", OpMul},
	{"/", OpDiv},
	{"-", OpSub},
	{"+", OpAdd},
	{"^", OpXor},
	{"|", OpOr},
	{"&", OpAnd},
	{"~", OpNot},
	{"i64min", int64(math.MinInt64)},
	{"i64max", int64(math.MaxInt64)},
	{"u64min", uint64(0)},
	{"u64max", uint64(math.MaxUint64)},
	{"i32min", int32(math.MinInt32)},
	{"i32max", int32(math.MaxInt32)},
	{"u32min", uint32(0)},
	{"u32max", uint32(math.MaxUint32)},
	{"i16min", int16(math.MinInt16)},
	{"i16max", int16(math.MaxInt16)},
	{"u16min", uint16(0)},
	{"u16max", uint16(math.MaxUint16)},
	{"i8min", int8(math.MinInt8)},
	{"i8max", int8(math.MaxInt8)},
	{"u8min", uint8(0)},
	{"u8max", uint8(math.MaxUint8)},
	{"f64min", float64(-math.MaxFloat64)},
	{"f64max", float64(math.MaxFloat64)},
	{"f64smallest", float64(math.SmallestNonzeroFloat64)},
	{"f32min", float32(-math.MaxFloat32)},
	{"f32max", float32(math.MaxFloat32)},
	{"f32smallest", float32(math.SmallestNonzeroFloat32)},
	{"i8", OpI8},
	{"i16", OpI16},
	{"i32", OpI32},
	{"i64", OpI64},
	{"u8", OpU8},
	{"u16", OpU16},
	{"u32", OpU32},
	{"u64", OpU64},
	{"f32tobits", OpF32Bits},
	{"f32->bits", OpF32Bits},
	{"f64tobits", OpF64Bits},
	{"f64->bits", OpF64Bits},
	{"f32frombits", OpBitsF32},
	{"bits->f32", OpBitsF32},
	{"f64frombits", OpBitsF64},
	{"bits->f64", OpBitsF64},
	{"f32", OpF32},
	{"f64", OpF64},
}

func parseDec(s string) (any, error) {
	float := strings.ContainsAny(s, ".eE")
	if float {
		return strconv.ParseFloat(s, 64)
	}
	neg := strings.HasPrefix(s, "-")
	if neg {
		return strconv.ParseInt(s, 10, 64)
	}
	return strconv.ParseUint(s, 10, 64)
}

func parseHex(s string) (any, error) {
	neg := strings.HasPrefix(s, "-")
	float := strings.ContainsAny(s, ".pP")
	idxWhole := 2

	if !float {
		if neg {
			return strconv.ParseInt(s[idxWhole:], 16, 64)
		} else {
			return strconv.ParseUint(s[idxWhole:], 16, 64)
		}
	}

	if neg {
		idxWhole++
	}

	var exp int64
	idxExp := strings.IndexAny(s, "pP")
	if idxExp >= 0 {
		var err error
		exp, err = strconv.ParseInt(s[idxExp+1:], 10, 10)
		if err != nil {
			return nil, err
		}
		s = s[:idxExp]
	}

	var strWhole, strFrac string
	idxFrac := strings.Index(s, ".")
	if idxFrac >= 0 {
		strWhole = s[idxWhole:idxFrac]
		strFrac = s[idxFrac+1:]
	} else {
		strWhole = s[idxWhole:]
		strFrac = ""
	}

	strDigits := strWhole + strFrac
	mantissa, err := strconv.ParseUint(strDigits, 16, 64)
	if err != nil {
		return nil, err
	}
	exp -= int64(len(strFrac) * 4)
	sign := 1.0
	if neg {
		sign = -1.0
	}
	return math.Copysign(math.Ldexp(float64(mantissa), int(exp)), sign), nil
}

func parseBin(s string) (any, error) {
	neg := strings.HasPrefix(s, "-")
	float := strings.ContainsAny(s, ".pP")
	idxWhole := 2

	if !float {
		if neg {
			return strconv.ParseInt(s[idxWhole:], 2, 64)
		} else {
			return strconv.ParseUint(s[idxWhole:], 2, 64)
		}
	}

	if neg {
		idxWhole++
	}

	var exp int64
	idxExp := strings.IndexAny(s, "pP")
	if idxExp >= 0 {
		var err error
		exp, err = strconv.ParseInt(s[idxExp+1:], 10, 10)
		if err != nil {
			return nil, err
		}
		s = s[:idxExp]
	}

	var strWhole, strFrac string
	idxFrac := strings.Index(s, ".")
	if idxFrac >= 0 {
		strWhole = s[idxWhole:idxFrac]
		strFrac = s[idxFrac+1:]
	} else {
		strWhole = s[idxWhole:]
		strFrac = ""
	}

	strDigits := strWhole + strFrac
	mantissa, err := strconv.ParseUint(strDigits, 2, 64)
	if err != nil {
		return nil, err
	}
	exp -= int64(len(strFrac))
	sign := 1.0
	if neg {
		sign = -1.0
	}
	return math.Copysign(math.Ldexp(float64(mantissa), int(exp)), sign), nil
}

func popToken(script string) (any, string, error) {
	script = strings.TrimSpace(script)
	if script == "" {
		return "", "", nil
	}

	num := reHexNumber.FindString(script)
	if num != "" {
		val, err := parseHex(num)
		return val, script[len(num):], err
	}

	num = reBinNumber.FindString(script)
	if num != "" {
		val, err := parseBin(num)
		return val, script[len(num):], err
	}

	num = reDecNumber.FindString(script)
	if num != "" {
		val, err := parseDec(num)
		return val, script[len(num):], err
	}

	for _, e := range tokenMap {
		if strings.HasPrefix(script, e.s) {
			return e.v, script[len(e.s):], nil
		}
	}

	return "", "", errors.New("no token found")
}

func tokenize(script string) ([]any, error) {
	var tokens []any
	for script != "" {
		var token any
		var err error
		token, script, err = popToken(script)
		if err != nil {
			return nil, err
		}
		if token != "" {
			tokens = append(tokens, token)
		}
	}
	return tokens, nil
}

type Stack struct {
	numbers []Num
}

func (s *Stack) Pop() Num {
	n := s.numbers[len(s.numbers)-1]
	s.numbers = s.numbers[:len(s.numbers)-1]
	return n
}

func (s *Stack) Push(n Num) {
	s.numbers = append(s.numbers, n)
}

func main() {
	var flags flag.FlagSet
	if flags.Parse(os.Args[1:]) != nil {
		os.Exit(1)
	}

	script := strings.Join(flags.Args(), " ")
	tokens, err := tokenize(script)
	if err != nil {
		log.Fatal(err)
	}

	var stack Stack
	for _, tok := range tokens {
		switch v := tok.(type) {
		case int8, int16, int32, int64,
			uint8, uint16, uint32, uint64,
			float32, float64:
			stack.Push(Num{v})
		case Op:
			switch v {
			// Arithmetic
			case OpAdd:
				x := stack.Pop()
				y := stack.Pop()
				stack.Push(y.OpAdd(x))
			case OpSub:
				x := stack.Pop()
				y := stack.Pop()
				stack.Push(y.OpSub(x))
			case OpMul:
				x := stack.Pop()
				y := stack.Pop()
				stack.Push(y.OpMul(x))
			case OpDiv:
				x := stack.Pop()
				y := stack.Pop()
				stack.Push(y.OpDiv(x))
			case OpExp:
				x := stack.Pop()
				y := stack.Pop()
				stack.Push(y.OpExp(x))
			case OpShl:
				x := stack.Pop()
				y := stack.Pop()
				stack.Push(y.OpShl(x))
			case OpShr:
				x := stack.Pop()
				y := stack.Pop()
				stack.Push(y.OpShr(x))
			// Bitwise operations
			case OpXor:
				x := stack.Pop()
				y := stack.Pop()
				stack.Push(y.OpXor(x))
			case OpAnd:
				x := stack.Pop()
				y := stack.Pop()
				stack.Push(y.OpAnd(x))
			case OpOr:
				x := stack.Pop()
				y := stack.Pop()
				stack.Push(y.OpOr(x))
			case OpNot:
				x := stack.Pop()
				stack.Push(x.OpNot())
			// Conversions
			case OpI8:
				stack.Push(stack.Pop().OpI8())
			case OpI16:
				stack.Push(stack.Pop().OpI16())
			case OpI32:
				stack.Push(stack.Pop().OpI32())
			case OpI64:
				stack.Push(stack.Pop().OpI64())
			case OpU8:
				stack.Push(stack.Pop().OpU8())
			case OpU16:
				stack.Push(stack.Pop().OpU16())
			case OpU32:
				stack.Push(stack.Pop().OpU32())
			case OpU64:
				stack.Push(stack.Pop().OpU64())
			case OpF32:
				stack.Push(stack.Pop().OpF32())
			case OpF64:
				stack.Push(stack.Pop().OpF64())
			// Float to/from bits
			case OpBitsF32:
				x := stack.Pop()
				stack.Push(Num{math.Float32frombits(uint32(x.AsUint()))})
			case OpBitsF64:
				x := stack.Pop()
				stack.Push(Num{math.Float64frombits(x.AsUint())})
			case OpF32Bits:
				x := stack.Pop()
				stack.Push(Num{math.Float32bits(float32(x.AsFloat()))})
			case OpF64Bits:
				x := stack.Pop()
				stack.Push(Num{math.Float64bits(x.AsFloat())})
			}
		}
	}
	for i, s := range stack.numbers {
		if i > 0 {
			fmt.Println(strings.Repeat("-", 76))
		}
		if s.CanFloat() {
			if s.Bits() == 64 {
				f := s.Float()
				fmt.Printf("%%g\t%g\n", f)
				fmt.Printf("%%.17f\t%.17f\n", f)
				j, err := json.Marshal(f)
				if err != nil {
					fmt.Printf("json\t%v\n", err)
				} else {
					fmt.Printf("json\t%s\n", string(j))
				}
				bits := math.Float64bits(f)
				fmt.Printf("bits\t%#016x\n", bits)
				signBit := (bits >> 63) & 1
				expBits := int32((bits >> 52) & ((1 << 11) - 1))
				manBits := bits & ((1 << 52) - 1)
				fmt.Printf("    \t0b%01b %011b %052b\n",
					signBit, expBits, manBits)
				sign := "+"
				if signBit == 1 {
					sign = "-"
				}
				man := fmt.Sprintf("0b1.%b", manBits)
				if manBits != 0 {
					man = strings.TrimRight(man, "0")
				}
				man = fmt.Sprintf("%s (%#013x)", man, manBits)
				fmt.Printf("    \t  %s %11d %52s \n",
					sign, expBits-1023, man)
			} else {
				f := float32(s.Float())
				fmt.Printf("%%g\t%g\n", f)
				fmt.Printf("%%.9f\t%.17f\n", f)
				j, err := json.Marshal(f)
				if err != nil {
					fmt.Printf("json\t%v\n", err)
				} else {
					fmt.Printf("json\t%s\n", string(j))
				}
				bits := math.Float32bits(f)
				fmt.Printf("bits\t%#08x\n", bits)
				signBit := (bits >> 31) & 1
				expBits := int32((bits >> 23) & ((1 << 8) - 1))
				manBits := bits & ((1 << 23) - 1)
				fmt.Printf("    \t0b%01b %08b %023b\n",
					signBit, expBits, manBits)
				sign := "+"
				if signBit == 1 {
					sign = "-"
				}
				man := fmt.Sprintf("0b1.%b", manBits)
				if manBits != 0 {
					man = strings.TrimRight(man, "0")
				}
				man = fmt.Sprintf("%s (%#06x)", man, manBits)
				fmt.Printf("    \t  %s %8d %23s \n",
					sign, expBits-127, man)
			}
		} else if s.CanInt() {
			d := s.Int()
			fmt.Printf("dec\t%d\n", d)
			fmt.Printf("hex\t%#x\n", d)
			fmt.Printf("bin\t%#b\n", d)
		} else {
			d := s.Uint()
			fmt.Printf("dec\t%d\n", d)
			fmt.Printf("hex\t%#x\n", d)
			fmt.Printf("bin\t%#b\n", d)
		}
	}
}
