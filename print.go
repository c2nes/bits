package main

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

func formatTable(kvs ...string) string {
	if len(kvs)%2 != 0 {
		panic("mismatched kv pair")
	}
	var out []string
	for i := 0; i < len(kvs); i += 2 {
		k := kvs[i]
		v := kvs[i+1]
		out = append(out, fmt.Sprintf("%-7s %s", k, v))
	}
	return strings.Join(out, "\n")
}

func (s Num) String() string {
	jsonNumBytes, err := json.Marshal(s.val)
	var jsonNum string
	if err != nil {
		jsonNum = err.Error()
	} else {
		jsonNum = string(jsonNumBytes)
	}

	if f, ok := s.val.(float64); ok {
		bits := math.Float64bits(f)
		signBit := (bits >> 63) & 1
		expBits := int32((bits >> 52) & ((1 << 11) - 1))
		manBits := bits & ((1 << 52) - 1)

		sign := "+"
		if signBit == 1 {
			sign = "-"
		}
		var man string
		switch expBits {
		case 0:
			// Zero or subnormal
			if manBits == 0 {
				man = "0"
			} else {
				man = fmt.Sprintf("0b0.%052b", manBits)
			}
		case (1 << 11) - 1:
			// Inf or NaN
			if manBits == 0 {
				man = "Inf"
			} else {
				man = "NaN"
			}
		default:
			// Normal
			man = fmt.Sprintf("0b1.%052b", manBits)
		}
		man = strings.TrimRight(strings.TrimRight(man, "0"), ".")
		man = fmt.Sprintf("%s (%#013x)", man, manBits)
		return formatTable(
			"type", fmt.Sprintf("%T", s.val),
			"dec", fmt.Sprintf("%g", f),
			"hex", fmt.Sprintf("%x", f),
			"fixed", fmt.Sprintf("%.17e", f),
			"json", jsonNum,
			"bits", fmt.Sprintf("%#016x", bits),
			"", fmt.Sprintf("0b%01b %011b %052b", signBit, expBits, manBits),
			"", fmt.Sprintf("  %s %11d %52s", sign, expBits-1023, man),
		)
	}

	if f, ok := s.val.(float32); ok {
		bits := math.Float32bits(f)
		signBit := (bits >> 31) & 1
		expBits := int32((bits >> 23) & ((1 << 8) - 1))
		manBits := bits & ((1 << 23) - 1)

		sign := "+"
		if signBit == 1 {
			sign = "-"
		}
		var man string
		switch expBits {
		case 0:
			// Zero or subnormal
			if manBits == 0 {
				man = "0"
			} else {
				man = fmt.Sprintf("0b0.%023b", manBits)
			}
		case (1 << 8) - 1:
			// Inf or NaN
			if manBits == 0 {
				man = "Inf"
			} else {
				man = "NaN"
			}
		default:
			// Normal
			man = fmt.Sprintf("0b1.%023b", manBits)
		}
		man = strings.TrimRight(strings.TrimRight(man, "0"), ".")
		man = fmt.Sprintf("%s (%#06x)", man, manBits)
		return formatTable(
			"type", fmt.Sprintf("%T", s.val),
			"dec", fmt.Sprintf("%g", f),
			"hex", fmt.Sprintf("%x", f),
			"fixed", fmt.Sprintf("%.9e", f),
			"json", jsonNum,
			"bits", fmt.Sprintf("%#08x", bits),
			"", fmt.Sprintf("0b%01b %08b %023b", signBit, expBits, manBits),
			"", fmt.Sprintf("  %s %8d %23s", sign, expBits-127, man),
		)
	}

	if s.CanInt() {
		d := s.Int()
		return formatTable(
			"type", fmt.Sprintf("%T", s.val),
			"dec", fmt.Sprintf("%d", d),
			"hex", fmt.Sprintf("%#0*x", s.Bits()/4, d),
			"bin", fmt.Sprintf("%#0*b", s.Bits(), d),
		)
	}

	d := s.Uint()
	return formatTable(
		"type", fmt.Sprintf("%T", s.val),
		"dec", fmt.Sprintf("%d", d),
		"hex", fmt.Sprintf("%#0*x", s.Bits()/4, d),
		"bin", fmt.Sprintf("%#0*b", s.Bits(), d),
	)
}
