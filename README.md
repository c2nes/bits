# `bits`

RPN bit manipulation calculator.

## Installation

```
go install github.com/c2nes/bits@latest
```

## Usage

One shot,

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

As script interpreter,

```
$ cat example.bits
#!/usr/bin/env bits

# A simple example of adding two numbers
1 1 +

# We can then bit cast the value to a float32
bits->f32

$ ./example.bits
type    float32
dec     3e-45
hex     0x1p-148
fixed   2.802596929e-45
json    3e-45
bits    0x00000002
        0b0 00000000 00000000000000000000010
          +     -127        0b1.1 (0x000002)
```

## Commands

TODO
