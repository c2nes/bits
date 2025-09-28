package main

import (
	"fmt"
	"math"
)

type Num struct {
	val any
}

func (n Num) Bits() int {
	switch n.val.(type) {
	case int8, uint8:
		return 8
	case int16, uint16:
		return 16
	case int32, uint32, float32:
		return 32
	default:
		return 64
	}
}

func (n Num) WithBits(bits int) Num {
	if n.CanFloat() {
		switch bits {
		case 32:
			return Num{float32(n.Float())}
		case 64:
			return Num{n.Float()}
		default:
			panic("invalid float bits")
		}
	}
	if n.CanInt() {
		switch bits {
		case 8:
			return Num{int8(n.Int())}
		case 16:
			return Num{int16(n.Int())}
		case 32:
			return Num{int32(n.Int())}
		case 64:
			return Num{n.Int()}
		default:
			panic("invalid int bits")
		}
	}
	switch bits {
	case 8:
		return Num{uint8(n.Uint())}
	case 16:
		return Num{uint16(n.Uint())}
	case 32:
		return Num{uint32(n.Uint())}
	case 64:
		return Num{n.Uint()}
	default:
		panic("invalid uint bits")
	}
}

func (n Num) CanFloat() bool {
	switch n.val.(type) {
	case float32, float64:
		return true
	default:
		return false
	}
}

func (n Num) Float() float64 {
	switch val := n.val.(type) {
	case float32:
		return float64(val)
	case float64:
		return val
	default:
		panic("not a float")
	}
}

func (n Num) AsFloat() float64 {
	switch {
	case n.CanFloat():
		return n.Float()
	case n.CanInt():
		return float64(n.Int())
	default:
		return float64(n.Uint())
	}
}

func (n Num) CanInt() bool {
	switch n.val.(type) {
	case int8, int16, int32, int64:
		return true
	default:
		return false
	}
}

func (n Num) Int() int64 {
	switch val := n.val.(type) {
	case int8:
		return int64(val)
	case int16:
		return int64(val)
	case int32:
		return int64(val)
	case int64:
		return val
	default:
		panic("not an int")
	}
}

func (n Num) AsInt() int64 {
	switch {
	case n.CanFloat():
		return int64(n.Float())
	case n.CanInt():
		return n.Int()
	default:
		return int64(n.Uint())
	}
}

func (n Num) CanUint() bool {
	switch n.val.(type) {
	case uint8, uint16, uint32, uint64:
		return true
	default:
		return false
	}
}

func (n Num) Uint() uint64 {
	switch val := n.val.(type) {
	case uint8:
		return uint64(val)
	case uint16:
		return uint64(val)
	case uint32:
		return uint64(val)
	case uint64:
		return val
	default:
		panic(fmt.Errorf("not an uint: %T(%#v)", n.val, n.val))
	}
}

func (n Num) AsUint() uint64 {
	switch {
	case n.CanFloat():
		return uint64(n.Float())
	case n.CanInt():
		return uint64(n.Int())
	default:
		return n.Uint()
	}
}

func (n Num) AsBits() uint64 {
	switch val := n.val.(type) {
	case float32:
		return uint64(math.Float32bits(val))
	case float64:
		return math.Float64bits(n.AsFloat())
	default:
		return n.AsUint()
	}
}

func (n Num) OpI8() Num  { return Num{int8(n.AsInt())} }
func (n Num) OpI16() Num { return Num{int16(n.AsInt())} }
func (n Num) OpI32() Num { return Num{int32(n.AsInt())} }
func (n Num) OpI64() Num { return Num{int64(n.AsInt())} }
func (n Num) OpU8() Num  { return Num{uint8(n.AsUint())} }
func (n Num) OpU16() Num { return Num{uint16(n.AsUint())} }
func (n Num) OpU32() Num { return Num{uint32(n.AsUint())} }
func (n Num) OpU64() Num { return Num{uint64(n.AsUint())} }
func (n Num) OpF32() Num { return Num{float32(n.AsFloat())} }
func (n Num) OpF64() Num { return Num{float64(n.AsFloat())} }

type Integer64 interface{ int64 | uint64 }
type Num64 interface{ Integer64 | float64 }

type F64Binary func(x, y float64) float64
type I64Binary func(x, y int64) int64
type U64Binary func(x, y uint64) uint64

func minBits(nums ...Num) int {
	min := 64
	for _, num := range nums {
		bits := num.Bits()
		if bits < min {
			min = bits
		}
	}
	return min
}

func dispatchBinary(n, m Num, fnF64 F64Binary, fnI64 I64Binary, fnU64 U64Binary) Num {
	if n.CanFloat() || m.CanFloat() {
		return Num{fnF64(n.AsFloat(), m.AsFloat())}.WithBits(minBits(n, m))
	}
	if n.CanInt() || m.CanInt() {
		return Num{fnI64(n.AsInt(), m.AsInt())}.WithBits(minBits(n, m))
	}
	return Num{fnU64(n.AsUint(), m.AsUint())}.WithBits(minBits(n, m))
}

func add[N Num64](n, m N) N { return n + m }

func (n Num) OpAdd(m Num) Num {
	return dispatchBinary(n, m, add[float64], add[int64], add[uint64])
}

func sub[N Num64](n, m N) N { return n - m }

func (n Num) OpSub(m Num) Num {
	return dispatchBinary(n, m, sub[float64], sub[int64], sub[uint64])
}

func mul[N Num64](n, m N) N { return n * m }

func (n Num) OpMul(m Num) Num {
	return dispatchBinary(n, m, mul[float64], mul[int64], mul[uint64])
}

func div[N Num64](n, m N) N { return n / m }

func (n Num) OpDiv(m Num) Num {
	return dispatchBinary(n, m, div[float64], div[int64], div[uint64])
}

func expFloat(n, m float64) float64 { return math.Pow(n, m) }
func expInt[N Integer64](n, m N) N {
	if n == 1 || m == 0 {
		return 1
	}
	if m < 0 {
		return 0
	}
	acc := N(1)
	for ; m > 0; m-- {
		acc *= n
	}
	return acc
}

func (n Num) OpExp(m Num) Num {
	return dispatchBinary(n, m, expFloat, expInt[int64], expInt[uint64])
}

func (n Num) OpShl(m Num) Num {
	shift := int(m.AsInt())
	var val any
	if n.CanFloat() {
		val = math.Ldexp(n.Float(), shift)
	} else if n.CanInt() {
		if shift < 0 {
			val = n.Int() >> -shift
		} else {
			val = n.Int() << shift
		}
	} else {
		if shift < 0 {
			val = n.Uint() >> -shift
		} else {
			val = n.Uint() << shift
		}
	}
	return Num{val}.WithBits(n.Bits())
}

func (n Num) OpShr(m Num) Num {
	shift := int(m.AsInt())
	var val any
	if n.CanFloat() {
		val = math.Ldexp(n.Float(), -shift)
	} else if n.CanInt() {
		if shift < 0 {
			val = n.Int() << -shift
		} else {
			val = n.Int() >> shift
		}
	} else {
		if shift < 0 {
			val = n.Uint() << -shift
		} else {
			val = n.Uint() >> shift
		}
	}
	return Num{val}.WithBits(n.Bits())
}

func (n Num) OpNeg() Num {
	return n.OpMul(Num{int64(-1)})
}

func dispatchBitwiseBinary(n, m Num, op func(x, y uint64) uint64) Num {
	if n.CanFloat() || m.CanFloat() {
		x := n.AsBits()
		y := m.AsBits()
		out := op(x, y)
		nbits := minBits(n, m)
		if nbits == 64 {
			return Num{math.Float64frombits(out)}
		} else {
			return Num{math.Float32frombits(uint32(out))}
		}
	}
	if n.CanInt() || m.CanInt() {
		x := n.AsUint()
		y := n.AsUint()
		val := Num{op(x, y)}.AsInt()
		return Num{val}.WithBits(minBits(n, m))
	}
	x := n.AsUint()
	y := n.AsUint()
	val := Num{op(x, y)}
	return Num{val}.WithBits(minBits(n, m))
}

func dispatchBitwiseUnary(n Num, op func(x uint64) uint64) Num {
	if n.CanFloat() {
		x := math.Float64bits(n.Float())
		val := math.Float64frombits(op(x))
		return Num{val}.WithBits(n.Bits())
	}
	if n.CanInt() {
		x := n.AsUint()
		val := Num{op(x)}.AsInt()
		return Num{val}.WithBits(n.Bits())
	}
	return Num{op(n.Uint())}.WithBits(n.Bits())
}

func (n Num) OpXor(m Num) Num {
	return dispatchBitwiseBinary(n, m, func(x, y uint64) uint64 {
		return x ^ y
	})
}

func (n Num) OpOr(m Num) Num {
	return dispatchBitwiseBinary(n, m, func(x, y uint64) uint64 {
		return x | y
	})
}

func (n Num) OpAnd(m Num) Num {
	return dispatchBitwiseBinary(n, m, func(x, y uint64) uint64 {
		return x & y
	})
}

func (n Num) OpNot() Num {
	return dispatchBitwiseUnary(n, func(x uint64) uint64 {
		return ^x
	})
}
