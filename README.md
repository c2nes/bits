# `bits`

RPN bit manipulation calculator.

## Installation

```
go install github.com/c2nes/bits@latest
```

## Usage

From command line arguments,

```
$ bits 1 1 +
type    uint64
dec     2
hex     0x0000000000000002
bin     0b0000000000000000000000000000000000000000000000000000000000000010
```

From stdin,

```
$ echo '1 1 +' | bits
type    uint64
dec     2
hex     0x0000000000000002
bin     0b0000000000000000000000000000000000000000000000000000000000000010
```

Interactively,

```
$ bits
> 1
> 1
> +
> print
2 (uint64)
> 1 1 + print
2 (uint64)
```

As a script interpreter,

```
$ cat example.bits
#!/usr/bin/env bits

# Compute 2^30 as an untyped integer.
1 30 <<

# Cast to an unsigned 32 bit integer.
u32

# Interpret those bits as a float.
fbits

$ ./example.bits
type    float32
dec     2
hex     0x1p+01
fixed   2.000000000e+00
json    2
bits    0x40000000
        0b0 10000000 00000000000000000000000
          +        1          0b1 (0x000000)
```

## Number formats

`bits` supports signed and unsigned integers with widths of 8, 16, 32 and 64 bits as well as single and double precision floats (32 and 64 bits respectively).

Numbers may be input in decimal, hexadecimal, or binary. Examples,

<!-- higlight as c++ since it's close enough -->

```cpp
// Decimal integers
12345
-12345

// Decimal floats
123.45
0.5
.5
5.
5e10
5e-100

// Hex integers
0x1234
-0x1234

// Hex floats (exponent is decimal power of 2)
0x1.
0x0.5
0x1p40
-0x5.5p-2

// Binary integers
0b1001
-0b1001

// Binary floats (exponent is decimal power of 2)
0b10.01
-0b10.01
0b10.01p10
0b10.01p-5
```

## Constants

| Constant        | Value                      |
| --------------- | -------------------------- |
| `i64min`        | `-9223372036854775808`     |
| `i64max`        | `9223372036854775807`      |
| `u64min`        | `0`                        |
| `u64max`        | `18446744073709551615`     |
| `i32min`        | `-2147483648`              |
| `i32max`        | `2147483647`               |
| `u32min`        | `0`                        |
| `u32max`        | `4294967295`               |
| `i16min`        | `-32768`                   |
| `i16max`        | `32767`                    |
| `u16min`        | `0`                        |
| `u16max`        | `65535`                    |
| `i8min`         | `-128`                     |
| `i8max`         | `127`                      |
| `u8min`         | `0`                        |
| `u8max`         | `255`                      |
| `f64minnorm`    | `2.2250738585072014e-308`  |
| `f64minsubnorm` | `5e-324`                   |
| `f64min`        | `-1.7976931348623157e+308` |
| `f64max`        | `1.7976931348623157e+308`  |
| `f32minnorm`    | `1.1754944e-38`            |
| `f32minsubnorm` | `1e-45`                    |
| `f32min`        | `-3.4028235e+38`           |
| `f32max`        | `3.4028235e+38`            |

## Commands

| Command | Aliases         | Description                                                           |
| ------- | --------------- | --------------------------------------------------------------------- |
| `<<`    |                 | Left shift. For floats interpreted as multiplication by a power of 2. |
| `>>`    |                 | Right shift. For floats interpreted as division by a power of 2.      |
| `**`    |                 | Exponentation                                                         |
| `*`     |                 | Multiplication                                                        |
| `/`     |                 | Division                                                              |
| `-`     |                 | Subtraction                                                           |
| `+`     |                 | Addition                                                              |
| `!`     |                 | Negation.                                                             |
| `^`     |                 | Bitwise xor.                                                          |
| `\|`    |                 | Bitwise or.                                                           |
| `&`     |                 | Bitwise and.                                                          |
| `~`     |                 | Bitwise not.                                                          |
| `i8`    |                 | Convert to signed 8 bit integer.                                      |
| `i16`   |                 | Convert to signed 16 bit integer.                                     |
| `i32`   |                 | Convert to signed 32 bit integer.                                     |
| `i64`   |                 | Convert to signed 64 bit integer.                                     |
| `u8`    |                 | Convert to unsigned 8 bit integer.                                    |
| `u16`   |                 | Convert to unsigned 16 bit integer.                                   |
| `u32`   |                 | Convert to unsigned 32 bit integer.                                   |
| `u64`   |                 | Convert to unsigned 64 bit integer.                                   |
| `f32`   |                 | Convert to 32 bit float.                                              |
| `f64`   |                 | Convert to 64 bit float.                                              |
| `bits`  |                 | Convert input to bits.                                                |
| `fbits` | `floatfrombits` | Convert bit input to a float.                                         |
| `drop`  |                 | Drop the entry at the top of the stack.                               |
| `dup`   | `.`             | Duplicate the entry at the top of the stack.                          |
| `swap`  | `x`             | Swap the two elements at the top of the stack.                        |
| `print` | `p`             | Concisely print the value at the top of the stack.                    |
| `dump`  | `d`             | Verbosely print all values in the stack.                              |
| `list`  | `ls`, `l`       | Concisely print all values in the stack.                              |
