package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/chzyer/readline"
	"golang.org/x/term"
)

var reDecNumber = regexp.MustCompile(`(?i)^[+-]?(\d+(\.\d*)?|\.\d+?)(e[+-]?\d+)?`)
var reHexNumber = regexp.MustCompile(`(?i)^[+-]?0x[0-9a-f]+(\.[0-9a-f]+)?(p[+-]?\d+)?`)
var reBinNumber = regexp.MustCompile(`(?i)^[+-]?0b[01]+(\.[01]+)?(p[+-]?\d+)?`)
var reComment = regexp.MustCompile(`(?m)^#.*?$`)

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
	OpNeg
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
	OpBits
	OpFloatFromBits
	OpDump
	OpPrint
	OpList
	OpDup
	OpSwap
	OpDrop
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
	{"!", OpNeg},
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
	{"bits", OpBits},
	{"fbits", OpFloatFromBits},
	{"floatfrombits", OpFloatFromBits},
	{"f32", OpF32},
	{"f64", OpF64},
	{"drop", OpDrop},
	{"dup", OpDup},
	{".", OpDup},
	{"swap", OpSwap},
	{"x", OpSwap},
	{"print", OpPrint}, // Concisely print the top of the stack
	{"p", OpPrint},
	{"dump", OpDump}, // Verbosely print the entire stack
	{"d", OpDump},
	{"list", OpList}, // Concisely print the entire stack
	{"ls", OpList},
	{"l", OpList},
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

	comment := reComment.FindString(script)
	if comment != "" {
		return "", script[len(comment):], nil
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

	snippet := script
	if len(snippet) > 20 {
		snippet = snippet[:20] + "..."
	}
	return "", "", fmt.Errorf("syntax error at %q", snippet)
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

func (s *Stack) Len() int {
	return len(s.numbers)
}

func (s *Stack) Empty() bool {
	return s.Len() == 0
}

func (s *Stack) Top() Num {
	return s.numbers[s.Len()-1]
}

func (s *Stack) At(i int) Num {
	return s.numbers[i]
}

func (s *Stack) Print() string {
	if s.Empty() {
		return "(empty)"
	}
	top := s.Top().val
	return fmt.Sprintf("%v (%T)", top, top)
}

func (s *Stack) maxIndexWidth() int {
	width := 1
	for maxIndex := s.Len() - 1; maxIndex >= 10; maxIndex /= 10 {
		width++
	}
	return width
}

func (s *Stack) List() string {
	if s.Empty() {
		return "(empty)"
	}
	var out []string
	w := s.maxIndexWidth()
	for i, n := range s.numbers {
		out = append(out, fmt.Sprintf("%*d: %v (%T)", w, s.Len()-i-1, n.val, n.val))
	}
	return strings.Join(out, "\n")
}

func (s *Stack) Dump() string {
	if s.Empty() {
		return "(empty)"
	}
	var out []string
	w := s.maxIndexWidth()
	for i, n := range s.numbers {
		if i > 0 {
			out = append(out, "")
			out = append(out, strings.Repeat("-", 79))
		}
		lines := strings.Split(n.String(), "\n")
		out = append(out, fmt.Sprintf("%*d: %s", w, s.Len()-i-1, lines[0]))
		for _, line := range lines[1:] {
			out = append(out, fmt.Sprintf("%*s  %s", w, "", line))
		}
	}
	return strings.Join(out, "\n")
}

func run(stack *Stack, input func() (string, error)) (skipOutput bool, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	skipOutput = false

	for {
		var src string
		src, err = input()
		if err != nil {
			if err == io.EOF {
				err = nil
				return
			}
			return
		}

		var tokens []any
		tokens, err = tokenize(src)
		if err != nil {
			return
		}

		for _, tok := range tokens {
			printed := false
			switch v := tok.(type) {
			case int8, int16, int32, int64,
				uint8, uint16, uint32, uint64,
				float32, float64:
				stack.Push(Num{v, false})
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
				case OpNeg:
					x := stack.Pop()
					stack.Push(x.OpNeg())
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
				case OpBits:
					x := stack.Pop()
					stack.Push(x.OpBits())
				case OpFloatFromBits:
					x := stack.Pop()
					stack.Push(x.OpFloatFromBits())
				// Printing
				case OpPrint:
					fmt.Println(stack.Print())
					printed = true
				case OpList:
					fmt.Println(stack.List())
					printed = true
				case OpDump:
					fmt.Println(stack.Dump())
					printed = true
				// Stack manipulation
				case OpDrop:
					if stack.Empty() {
						fmt.Println("(empty)")
					} else {
						stack.Pop()
					}
				case OpSwap:
					x := stack.Pop()
					y := stack.Pop()
					stack.Push(x)
					stack.Push(y)
				case OpDup:
					x := stack.Pop()
					stack.Push(x)
					stack.Push(x)
				}
			}
			skipOutput = printed
		}
	}
}

func sanitizeArgs() {
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if arg == "--" {
			return
		}
		// Add "--" before the first non-flag
		if len(arg) == 0 ||
			arg[0] != '-' ||
			// negative numbers
			(len(arg) > 1 && arg[1] >= '0' && arg[1] <= '9') {
			var patched []string
			patched = append(patched, os.Args[:i]...)
			patched = append(patched, "--")
			patched = append(patched, os.Args[i:]...)
			os.Args = patched
			return
		}
	}
}

func fileInput(fns ...string) (input func() (string, error), cleanup func()) {
	var f *os.File
	var sc *bufio.Scanner
	input = func() (string, error) {
		for {
			if sc != nil {
				if sc.Scan() {
					return sc.Text(), nil
				}
				if err := sc.Err(); err != nil {
					return "", err
				}
			}
			f.Close()
			f = nil
			if len(fns) == 0 {
				return "", io.EOF
			}
			var err error
			fn := fns[0]
			fns = fns[1:]
			f, err = os.Open(fn)
			if err != nil {
				return "", err
			}
			sc = bufio.NewScanner(f)
		}
	}
	cleanup = func() {
		if f != nil {
			f.Close()
		}
	}
	return
}

func stringInput(script string) func() (string, error) {
	return func() (string, error) {
		if script == "" {
			return "", io.EOF
		}
		s := script
		script = ""
		return s, nil
	}
}

func fileExists(fn string) bool {
	_, err := os.Stat(fn)
	return err == nil
}

func main() {
	sanitizeArgs()
	useFile := flag.Bool("f", false, `read input from a file`)
	useArgs := flag.Bool("c", false, `use command line arguments as input`)
	quiet := flag.Bool("q", false, `skip automatic dumping of the stack on exit`)
	flag.Parse()

	var input func() (string, error)
	args := flag.Args()
	continueOnError := false
	if *useFile || (!*useArgs && len(args) == 1 && fileExists(args[0])) {
		var cleanup func()
		input, cleanup = fileInput(args...)
		defer cleanup()
	} else if *useArgs || len(os.Args) > 1 {
		input = stringInput(strings.Join(args, " "))
	} else if term.IsTerminal(int(os.Stdin.Fd())) {
		historyFile, err := HistoryFile()
		if err != nil {
			historyFile = ""
			log.Printf("warn: %v", err)
		}
		rl, err := readline.NewEx(&readline.Config{
			Prompt:      "> ",
			HistoryFile: historyFile,
		})
		if err != nil {
			log.Fatal(err)
		}
		defer rl.Close()
		rl.CaptureExitSignal()
		input = rl.Readline
		continueOnError = true
	} else {
		scan := bufio.NewScanner(os.Stdin)
		input = func() (string, error) {
			if scan.Scan() {
				return scan.Text(), nil
			}
			if scan.Err() == nil {
				return "", io.EOF
			}
			return "", scan.Err()
		}
	}

	var stack Stack
	var skipOutput bool
	var err error
	for {
		skipOutput, err = run(&stack, input)
		if err == nil || err == io.EOF {
			break
		}
		if continueOnError {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		} else {
			log.Fatalf("error: %v\n", err)
		}
	}
	if !*quiet && !skipOutput {
		if stack.Len() == 1 {
			fmt.Println(stack.Top())
		} else {
			fmt.Println(stack.Dump())
		}
	}
}
