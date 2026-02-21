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
var reHexNumber = regexp.MustCompile(`(?i)^[+-]?0x[0-9a-f]+(\.[0-9a-f]*)?(p[+-]?\d+)?`)
var reBinNumber = regexp.MustCompile(`(?i)^[+-]?0b[01]+(\.[01]*)?(p[+-]?\d+)?`)
var reComment = regexp.MustCompile(`(?m)^(#|//).*?$`)

var reSetVar = regexp.MustCompile(`^=\w+`)
var reUseVar = regexp.MustCompile(`^\$\w+`)
var reExecVar = regexp.MustCompile(`^@\w+`)

type OpSetVar string
type OpUseVar string
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
	OpSucc
	OpPred
	OpUlp
	OpDump
	OpPrint
	OpList
	OpDup
	OpSwap
	OpDrop
	// Block {...}
	OpStartBlock
	OpEndBlock
	OpExecute
	OpExecuteEq
	OpExecuteLt
	OpExecuteGt
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
	{"neg", OpNeg},
	{"!", OpNeg},
	{"i64min", Num{int64(math.MinInt64), true}},
	{"i64max", Num{int64(math.MaxInt64), true}},
	{"u64min", Num{uint64(0), true}},
	{"u64max", Num{uint64(math.MaxUint64), true}},
	{"i32min", Num{int32(math.MinInt32), true}},
	{"i32max", Num{int32(math.MaxInt32), true}},
	{"u32min", Num{uint32(0), true}},
	{"u32max", Num{uint32(math.MaxUint32), true}},
	{"i16min", Num{int16(math.MinInt16), true}},
	{"i16max", Num{int16(math.MaxInt16), true}},
	{"u16min", Num{uint16(0), true}},
	{"u16max", Num{uint16(math.MaxUint16), true}},
	{"i8max", Num{int8(math.MaxInt8), true}},
	{"i8min", Num{int8(math.MinInt8), true}},
	{"u8min", Num{uint8(0), true}},
	{"u8max", Num{uint8(math.MaxUint8), true}},
	{"f64minnorm", Num{float64(0x1p-1022), true}},
	{"f64minsubnorm", Num{float64(math.SmallestNonzeroFloat64), true}},
	{"f64min", Num{float64(-math.MaxFloat64), true}},
	{"f64max", Num{float64(math.MaxFloat64), true}},
	{"f32minnorm", Num{float32(0x1p-126), true}},
	{"f32minsubnorm", Num{float32(math.SmallestNonzeroFloat32), true}},
	{"f32min", Num{float32(-math.MaxFloat32), true}},
	{"f32max", Num{float32(math.MaxFloat32), true}},
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
	{"succ", OpSucc},
	{"after", OpSucc},
	{"next", OpSucc},
	{"pred", OpPred},
	{"before", OpPred},
	{"prev", OpPred},
	{"ulp", OpUlp},
	{"f32", OpF32},
	{"f64", OpF64},
	{"drop", OpDrop},
	{"dup", OpDup},
	{".", OpDup},
	{"swap", OpSwap},
	{"%", OpSwap},
	{"x", OpSwap},
	{"print", OpPrint}, // Concisely print the top of the stack
	{"p", OpPrint},
	{"dump", OpDump}, // Verbosely print the entire stack
	{"d", OpDump},
	{"list", OpList}, // Concisely print the entire stack
	{"ls", OpList},
	{"l", OpList},
	{"{", OpStartBlock},
	{"}", OpEndBlock},
	// Conditional executional
	{"@=", OpExecuteEq},
	{"@<", OpExecuteLt},
	{"@>", OpExecuteGt},
	// Unconditional execution
	{"call", OpExecute},
	{"@", OpExecute},
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

	if !float {
		if neg {
			return strconv.ParseInt(s, 0, 64)
		} else {
			return strconv.ParseUint(s, 0, 64)
		}
	}

	idxWhole := 2
	if neg {
		idxWhole++
	}

	var exp int64
	idxExp := strings.IndexAny(s, "pP")
	if idxExp >= 0 {
		var err error
		exp, err = strconv.ParseInt(s[idxExp+1:], 10, 11)
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

	if !float {
		if neg {
			return strconv.ParseInt(s, 0, 64)
		} else {
			return strconv.ParseUint(s, 0, 64)
		}
	}

	idxWhole := 2
	if neg {
		idxWhole++
	}

	var exp int64
	idxExp := strings.IndexAny(s, "pP")
	if idxExp >= 0 {
		var err error
		exp, err = strconv.ParseInt(s[idxExp+1:], 10, 11)
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

	setVar := reSetVar.FindString(script)
	if setVar != "" {
		return OpSetVar(setVar[1:]), script[len(setVar):], nil
	}

	useVar := reUseVar.FindString(script)
	if useVar != "" {
		return OpUseVar(useVar[1:]), script[len(useVar):], nil
	}

	execVar := reExecVar.FindString(script)
	if execVar != "" {
		return []any{OpUseVar(execVar[1:]), OpExecute}, script[len(execVar):], nil
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
	tokens := []any{}
	for script != "" {
		var token any
		var err error
		token, script, err = popToken(script)
		if err != nil {
			return nil, err
		}
		if token != "" {
			if multi, ok := token.([]any); ok {
				tokens = append(tokens, multi...)
			} else {
				tokens = append(tokens, token)
			}

		}
	}
	return tokens, nil
}

type Block struct {
	body   []any
	closed bool
}

func (b *Block) String() string {
	return "{...}"
}

type Stack struct {
	s []any
}

func (s *Stack) Pop() any {
	n := s.s[len(s.s)-1]
	s.s = s.s[:len(s.s)-1]
	return n
}

func (s *Stack) PopNum() Num {
	_ = s.Top().(Num)
	return s.Pop().(Num)
}

func (s *Stack) PopBlock() *Block {
	_ = s.Top().(*Block)
	return s.Pop().(*Block)
}

func (s *Stack) Push(v any) {
	s.s = append(s.s, v)
}

func (s *Stack) Len() int {
	return len(s.s)
}

func (s *Stack) Empty() bool {
	return s.Len() == 0
}

func (s *Stack) Top() any {
	return s.s[s.Len()-1]
}

func (s *Stack) Print() string {
	if s.Empty() {
		return "(empty)"
	}
	top := s.Top()
	if v, ok := top.(Num); ok {
		top = v.val
	}
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
	for i, n := range s.s {
		if num, ok := n.(Num); ok {
			n = num.val
		}
		out = append(out, fmt.Sprintf("%*d: %v (%T)", w, s.Len()-i-1, n, n))
	}
	return strings.Join(out, "\n")
}

func (s *Stack) Dump() string {
	if s.Empty() {
		return "(empty)"
	}
	var out []string
	w := s.maxIndexWidth()
	for i, n := range s.s {
		if i > 0 {
			out = append(out, "")
			out = append(out, strings.Repeat("-", 79))
		}
		lines := strings.Split(fmt.Sprint(n), "\n")
		out = append(out, fmt.Sprintf("%*d: %s", w, s.Len()-i-1, lines[0]))
		for _, line := range lines[1:] {
			out = append(out, fmt.Sprintf("%*s  %s", w, "", line))
		}
	}
	return strings.Join(out, "\n")
}

type Frame struct {
	code []any
	pos  int
	vars map[string]any
}

type CallStack struct {
	frames []*Frame
}

func (s *CallStack) PushFrame(code []any) {
	if len(s.frames) > 1 {
		top := s.frames[len(s.frames)-1]
		if top.pos == len(top.code) {
			// Tail call optimization. Re-use current top frame.
			top.pos = 0
			top.code = code
			return
		}
	}
	s.frames = append(s.frames, &Frame{code, 0, nil})
}

func (s *CallStack) Return() {
	s.frames = s.frames[:len(s.frames)-1]
}

func (s *CallStack) Empty() bool {
	for _, frame := range s.frames {
		if frame.pos < len(frame.code) {
			return false
		}
	}
	return true
}

func (s *CallStack) NextToken() any {
	for {
		top := s.frames[len(s.frames)-1]
		if top.pos == len(top.code) {
			s.frames = s.frames[:len(s.frames)-1]
			continue
		}

		tok := top.code[top.pos]
		top.pos++
		return tok
	}
}

func (s *CallStack) SetVar(name string, value any) {
	top := s.frames[len(s.frames)-1]
	if top.vars == nil {
		top.vars = make(map[string]any)
	}
	top.vars[name] = value
}

func (s *CallStack) GetVar(name string) any {
	for i := len(s.frames) - 1; i >= 0; i-- {
		if val, ok := s.frames[i].vars[name]; ok {
			return val
		}
	}
	if (name == "_" || name == "this") && len(s.frames) > 1 {
		top := s.frames[len(s.frames)-1]
		return &Block{top.code, true}
	}
	panic(fmt.Sprintf("no such var: %q", name))
}

func run(stack *Stack, globals map[string]any, input func() (string, error)) (skipOutput bool, err error) {
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

		callStack := &CallStack{[]*Frame{{tokens, 0, globals}}}

		for !callStack.Empty() {
			tok := callStack.NextToken()
			printed := false

			// Wrap numbers
			switch v := tok.(type) {
			case int8, int16, int32, int64,
				uint8, uint16, uint32, uint64,
				float32, float64:
				tok = Num{v, false}
			}

			if !stack.Empty() {
				if block, ok := stack.Top().(*Block); ok && !block.closed {
					if tok == OpStartBlock {
						stack.Push(&Block{})
					} else if tok == OpEndBlock {
						block.closed = true
						// Push a frame with the compiled block just simply to
						// handle the case of a containing block still being open.
						callStack.PushFrame([]any{stack.Pop()})
					} else {
						block.body = append(block.body, tok)
					}
					continue
				}
			}

			switch v := tok.(type) {
			case Num, *Block:
				stack.Push(v)
			case Op:
				switch v {
				// Arithmetic
				case OpAdd:
					x := stack.PopNum()
					y := stack.PopNum()
					stack.Push(y.OpAdd(x))
				case OpSub:
					x := stack.PopNum()
					y := stack.PopNum()
					stack.Push(y.OpSub(x))
				case OpMul:
					x := stack.PopNum()
					y := stack.PopNum()
					stack.Push(y.OpMul(x))
				case OpDiv:
					x := stack.PopNum()
					y := stack.PopNum()
					stack.Push(y.OpDiv(x))
				case OpExp:
					x := stack.PopNum()
					y := stack.PopNum()
					stack.Push(y.OpExp(x))
				case OpShl:
					x := stack.PopNum()
					y := stack.PopNum()
					stack.Push(y.OpShl(x))
				case OpShr:
					x := stack.PopNum()
					y := stack.PopNum()
					stack.Push(y.OpShr(x))
				case OpNeg:
					x := stack.PopNum()
					stack.Push(x.OpNeg())
				// Bitwise operations
				case OpXor:
					x := stack.PopNum()
					y := stack.PopNum()
					stack.Push(y.OpXor(x))
				case OpAnd:
					x := stack.PopNum()
					y := stack.PopNum()
					stack.Push(y.OpAnd(x))
				case OpOr:
					x := stack.PopNum()
					y := stack.PopNum()
					stack.Push(y.OpOr(x))
				case OpNot:
					x := stack.PopNum()
					stack.Push(x.OpNot())
				// Conversions
				case OpI8:
					stack.Push(stack.PopNum().OpI8())
				case OpI16:
					stack.Push(stack.PopNum().OpI16())
				case OpI32:
					stack.Push(stack.PopNum().OpI32())
				case OpI64:
					stack.Push(stack.PopNum().OpI64())
				case OpU8:
					stack.Push(stack.PopNum().OpU8())
				case OpU16:
					stack.Push(stack.PopNum().OpU16())
				case OpU32:
					stack.Push(stack.PopNum().OpU32())
				case OpU64:
					stack.Push(stack.PopNum().OpU64())
				case OpF32:
					stack.Push(stack.PopNum().OpF32())
				case OpF64:
					stack.Push(stack.PopNum().OpF64())
				// Float to/from bits
				case OpBits:
					x := stack.PopNum()
					stack.Push(x.OpBits())
				case OpFloatFromBits:
					x := stack.PopNum()
					stack.Push(x.OpFloatFromBits())
				case OpSucc:
					x := stack.PopNum()
					stack.Push(x.OpSucc())
				case OpPred:
					x := stack.PopNum()
					stack.Push(x.OpPred())
				case OpUlp:
					x := stack.PopNum()
					stack.Push(x.OpUlp())
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
						stack.PopNum()
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
				case OpStartBlock:
					stack.Push(&Block{})
				case OpExecute:
					callStack.PushFrame(stack.PopBlock().body)
				case OpExecuteEq:
					block := stack.PopBlock()
					if num, ok := stack.Top().(Num); ok && num.AsFloat() == 0 {
						callStack.PushFrame(block.body)
					}
				case OpExecuteLt:
					block := stack.PopBlock()
					if num, ok := stack.Top().(Num); ok && num.AsFloat() < 0 {
						callStack.PushFrame(block.body)
					}
				case OpExecuteGt:
					block := stack.PopBlock()
					if num, ok := stack.Top().(Num); ok && num.AsFloat() > 0 {
						callStack.PushFrame(block.body)
					}
				}
			case OpSetVar:
				callStack.SetVar(string(v), stack.Pop())
			case OpUseVar:
				stack.Push(callStack.GetVar(string(v)))
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
	} else if *useArgs || len(args) > 0 {
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
	vars := make(map[string]any)
	var skipOutput bool
	var err error
	for {
		skipOutput, err = run(&stack, vars, input)
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
