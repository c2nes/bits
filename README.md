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

## Commands

TODO
