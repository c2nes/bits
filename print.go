package main

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

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
		man := fmt.Sprintf("0b1.%b", manBits)
		if manBits != 0 {
			man = strings.TrimRight(man, "0")
		}
		man = fmt.Sprintf("%s (%#013x)", man, manBits)

		return fmt.Sprintf(""+
			"type\t%T\n"+
			"dec\t%g\n"+
			"fixed\t%.17e\n"+
			"json\t%v\n"+
			"bits\t%#016x\n"+
			"    \t0b%01b %011b %052b\n"+
			"    \t  %s %11d %52s \n",
			s.val, f, f, jsonNum, bits,
			signBit, expBits, manBits,
			sign, expBits-1023, man,
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
		man := fmt.Sprintf("0b1.%b", manBits)
		if manBits != 0 {
			man = strings.TrimRight(man, "0")
		}
		man = fmt.Sprintf("%s (%#06x)", man, manBits)

		return fmt.Sprintf(""+
			"type\t%T\n"+
			"dec\t%g\n"+
			"fixed\t%.9e\n"+
			"json\t%v\n"+
			"bits\t%#08x\n"+
			"    \t0b%01b %08b %023b\n"+
			"    \t  %s %8d %23s \n",
			s.val, f, f, jsonNum, bits,
			signBit, expBits, manBits,
			sign, expBits-127, man,
		)
	}

	if s.CanInt() {
		d := s.Int()
		return fmt.Sprintf(""+
			"type\t%T\n"+
			"dec\t%d\n"+
			"hex\t%#0*x\n"+
			"bin\t%#0*b\n",
			s.val, d, s.Bits()/4, d, s.Bits(), d,
		)
	}

	d := s.Uint()
	return fmt.Sprintf(""+
		"type\t%T\n"+
		"dec\t%d\n"+
		"hex\t%#0*x\n"+
		"bin\t%#0*b\n",
		s.val, d, s.Bits()/4, d, s.Bits(), d,
	)
}
