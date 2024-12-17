package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

// An Alist maps symbols (strings) to their corresponding values (interfaces).
type Alist map[string]interface{}

// globalAlist is the global environment that stores variables and function definitions.
var globalAlist Alist = make(Alist)

// isNil checks if the given value is considered NIL in Lisp.
// In Lisp, NIL represents both the empty list and the boolean false.
func isNil(x interface{}) bool {
	return x == nil || (isSymbol(x, "NIL"))
}

// isSymbol checks if x is a symbol matching the given string (case-insensitive).
func isSymbol(x interface{}, s string) bool {
	if str, ok := x.(string); ok && strings.ToUpper(str) == strings.ToUpper(s) {
		return true
	}
	return false
}

// toLispString converts a Go value to its Lisp string representation.
func toLispString(obj interface{}) string {
	switch v := obj.(type) {
	case nil:
		return "NIL"
	case string:
		if isSymbol(v, "T") {
			return "T"
		}
		if isSymbol(v, "NIL") {
			return "NIL"
		}
		return v
	case int:
		return fmt.Sprintf("%d", v)
	case []interface{}:
		var parts []string
		for _, e := range v {
			parts = append(parts, toLispString(e))
		}
		return "(" + strings.Join(parts, " ") + ")"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// myEval evaluates a Lisp expression within a given alist (environment).
func myEval(expr interface{}, alist Alist) interface{} {
	switch v := expr.(type) {
	case string, int:
		// If the expression is an atom (symbol or number), evaluate it accordingly.
		return myEvalAtom(v, alist)
	case []interface{}:
		if len(v) == 0 {
			return nil
		}
		// The first element is expected to be a function name or a special form.
		fnSym, ok := v[0].(string)
		if !ok {
			panic("Invalid function: must be a symbol")
		}
		// Apply the function to the remaining elements of the list.
		return myApply(fnSym, v[1:], alist)
	default:
		// For other types, return the expression as is.
		return expr
	}
}

// myEvalAtom evaluates an atomic expression (symbol or number) within the given alist.
func myEvalAtom(atom interface{}, alist Alist) interface{} {
	switch v := atom.(type) {
	case int:
		// Numbers evaluate to themselves.
		return v
	case string:
		// Handle special symbols "T" and "NIL".
		up := strings.ToUpper(v)
		if up == "T" {
			return "T"
		}
		if up == "NIL" {
			return nil
		}
		// Look up the symbol in the local alist.
		if val, ok := alist[v]; ok {
			return val
		} else if val, ok := globalAlist[v]; ok {
			// If not found locally, look in the global alist.
			return val
		}
		// If the symbol is not bound, return it as is.
		return v
	default:
		// Return the atom as is for other types.
		return atom
	}
}

// myEvalList evaluates a list of expressions in sequence and returns the last result.
func myEvalList(exprs []interface{}, alist Alist) interface{} {
	var result interface{}
	for i, expr := range exprs {
		val := myEval(expr, alist)
		if i == len(exprs)-1 {
			result = val
		}
	}
	return result
}

// equalp checks if two Lisp values are equal, considering case-insensitivity for symbols.
func equalp(x, y interface{}) bool {
	switch xv := x.(type) {
	case nil:
		return y == nil
	case string:
		if yv, ok := y.(string); ok {
			return strings.ToUpper(xv) == strings.ToUpper(yv)
		}
		return false
	case int:
		if yv, ok := y.(int); ok {
			return xv == yv
		}
		return false
	case []interface{}:
		yv, ok := y.([]interface{})
		if !ok {
			return false
		}
		if len(xv) != len(yv) {
			return false
		}
		for i := range xv {
			if !equalp(xv[i], yv[i]) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// bindFormals binds formal parameters to actual arguments in a new alist (environment).
func bindFormals(formals []interface{}, actuals []interface{}, alist Alist) Alist {
	if len(formals) != len(actuals) {
		panic("Lambda argument count mismatch")
	}
	newAlist := make(Alist)
	// Arguments are already evaluated before myApplyLambda is called.
	for i, f := range formals {
		sym, ok := f.(string)
		if !ok {
			panic("Formal parameters must be symbols")
		}
		newAlist[sym] = actuals[i]
	}
	// Inherit existing bindings from the parent alist if not overridden.
	for k, v := range alist {
		if _, found := newAlist[k]; !found {
			newAlist[k] = v
		}
	}
	return newAlist
}

// myApplyLambda applies a lambda function to arguments within an alist.
func myApplyLambda(fnBody []interface{}, args []interface{}, alist Alist) interface{} {
	if len(fnBody) < 2 {
		panic("Invalid lambda function definition")
	}
	// The first element of fnBody is the list of formal parameters.
	formals, ok := fnBody[0].([]interface{})
	if !ok {
		panic("Invalid lambda formals")
	}
	// The rest of fnBody constitutes the function body.
	body := fnBody[1:]
	// Create a new alist by binding formals to args.
	newAlist := bindFormals(formals, args, alist)
	// Evaluate the function body in the new alist.
	return myEvalList(body, newAlist)
}

// myEvalSetq evaluates a setq expression, assigning a value to a variable in the global alist.
func myEvalSetq(varName string, val interface{}) interface{} {
	evaluated := myEval(val, globalAlist)
	globalAlist[varName] = evaluated
	return evaluated
}

// myEvalDefun evaluates a defun expression, defining a new function in the global alist.
func myEvalDefun(args []interface{}) interface{} {
	if len(args) < 3 {
		panic("defun: must have (defun fname (args...) body...)")
	}
	// Extract the function name.
	fname, ok := args[0].(string)
	if !ok {
		panic("defun: first argument must be a symbol")
	}
	// Extract the list of formal parameters.
	formals, ok := args[1].([]interface{})
	if !ok {
		panic("defun: second argument must be a list of formals")
	}
	// The rest of the arguments constitute the function body.
	body := args[2:]
	// Function definition: [formals, body...]
	fnDef := append([]interface{}{formals}, body...)
	// Store the function definition in the global alist.
	globalAlist[fname] = fnDef
	return fname
}

// myEvalCond evaluates a cond expression, which is a series of condition-action clauses.
func myEvalCond(clauses []interface{}, alist Alist) interface{} {
	for _, c := range clauses {
		clauseList, ok := c.([]interface{})
		if !ok || len(clauseList) == 0 {
			panic("cond: each clause must be a non-empty list")
		}
		// Evaluate the condition of the current clause.
		condition := myEval(clauseList[0], alist)
		if !isNil(condition) {
			// If the condition is true (not NIL), evaluate and return the body.
			return myEvalList(clauseList[1:], alist)
		}
	}
	// If no conditions are true, return NIL.
	return nil
}

// toList ensures that the argument is a list, wrapping it in a list if necessary.
func toList(x interface{}) []interface{} {
	if x == nil {
		return []interface{}{}
	}
	if l, ok := x.([]interface{}); ok {
		return l
	}
	return []interface{}{x}
}

// appendHelp recursively appends two lists.
// Note: This function is defined but not used in the current implementation.
func appendHelp(x, y []interface{}) []interface{} {
	if len(x) == 0 {
		return y
	}
	return append([]interface{}{x[0]}, appendHelp(x[1:], y)...)
}

// boolToT converts a boolean value to Lisp's "T" or NIL.
func boolToT(b bool) interface{} {
	if b {
		return "T"
	}
	return nil
}

// myApply applies a function symbol to arguments within an alist.
// It handles special forms and built-in functions.
func myApply(fnSym string, args []interface{}, alist Alist) interface{} {
	up := strings.ToUpper(fnSym)

	// Handle special forms that have unique evaluation rules.
	switch up {
	case "QUOTE":
		if len(args) != 1 {
			panic("quote expects exactly one argument")
		}
		return args[0] // No evaluation for quote.
	case "COND":
		return myEvalCond(args, alist)
	case "DEFUN":
		return myEvalDefun(args)
	case "SETQ":
		if len(args) != 2 {
			panic("setq expects 2 arguments")
		}
		varName, ok := args[0].(string)
		if !ok {
			panic("setq: first argument must be a symbol")
		}
		return myEvalSetq(varName, args[1])
	case "EVAL":
		if len(args) != 1 {
			panic("eval expects 1 argument")
		}
		val := myEval(args[0], alist)
		return myEval(val, alist)
	case "APPLY":
		if len(args) != 2 {
			panic("apply expects exactly 2 arguments")
		}
		fnVal := myEval(args[0], alist)
		fnName, ok := fnVal.(string)
		if !ok {
			panic("apply expects a function symbol as first arg")
		}
		argVal := myEval(args[1], alist)
		argList := toList(argVal)
		return myApplyAtom(fnName, argList, alist, false)
	case "AND":
		// Evaluate each argument; if any is NIL, return NIL.
		for _, a := range args {
			val := myEval(a, alist)
			if isNil(val) {
				return nil
			}
		}
		return "T"
	case "OR":
		// Evaluate each argument; if any is not NIL, return "T".
		for _, a := range args {
			val := myEval(a, alist)
			if !isNil(val) {
				return "T"
			}
		}
		return nil
	case "NOT":
		if len(args) != 1 {
			panic("not expects 1 argument")
		}
		val := myEval(args[0], alist)
		if isNil(val) {
			return "T"
		}
		return nil
	case "LET*":
		// Handle let* form: sequentially binds variables.
		if len(args) < 2 {
			panic("let* expects at least ((var val)...) and a body")
		}
		bindings, ok := args[0].([]interface{})
		if !ok {
			panic("let*: first argument must be a list of bindings")
		}
		localAlist := make(Alist)
		// Inherit from the current alist.
		for k, v := range alist {
			localAlist[k] = v
		}
		// Process each binding sequentially.
		for _, b := range bindings {
			pair, ok := b.([]interface{})
			if !ok || len(pair) != 2 {
				panic("let*: each binding must be a pair (var val)")
			}
			varName, ok := pair[0].(string)
			if !ok {
				panic("let*: variable name must be a symbol")
			}
			val := myEval(pair[1], localAlist)
			localAlist[varName] = val
		}
		body := args[1:]
		return myEvalList(body, localAlist)
	case "IF":
		// Handle the if special form.
		if len(args) < 2 || len(args) > 3 {
			panic("if expects (if condition then [else])")
		}
		condition := myEval(args[0], alist)
		if !isNil(condition) {
			return myEval(args[1], alist)
		} else {
			if len(args) == 3 {
				return myEval(args[2], alist)
			}
			return nil
		}
	case "LET":
		// Handle let form: binds variables in parallel.
		if len(args) < 2 {
			panic("let expects ((var val)...) and a body")
		}
		bindings, ok := args[0].([]interface{})
		if !ok {
			panic("let: first argument must be a list of bindings")
		}

		// Evaluate all values first for parallel binding.
		localAlist := make(Alist)
		for k, v := range alist {
			localAlist[k] = v
		}

		varNames := []string{}
		varVals := []interface{}{}

		for _, b := range bindings {
			pair, ok := b.([]interface{})
			if !ok || len(pair) != 2 {
				panic("let: each binding must be (var val)")
			}
			varName, ok := pair[0].(string)
			if !ok {
				panic("let: variable name must be a symbol")
			}
			varNames = append(varNames, varName)
			// Evaluate the value in the original alist for parallel binding.
			val := myEval(pair[1], alist)
			varVals = append(varVals, val)
		}

		// Now bind all variables at once.
		for i, varName := range varNames {
			localAlist[varName] = varVals[i]
		}

		body := args[1:]
		return myEvalList(body, localAlist)
	default:
		// Handle normal functions or built-in functions.
		evaledArgs := make([]interface{}, len(args))
		for i, a := range args {
			evaledArgs[i] = myEval(a, alist)
		}
		return myApplyAtom(fnSym, evaledArgs, alist, true)
	}
}

// myApplyAtom applies built-in functions or user-defined functions to arguments.
func myApplyAtom(fnSym string, args []interface{}, alist Alist, fullyEvaluated bool) interface{} {
	up := strings.ToUpper(fnSym)
	switch up {
	case "CAR":
		// Return the first element of a list.
		if len(args) != 1 {
			panic("car expects 1 argument")
		}
		if args[0] == nil {
			return nil
		}
		lst, ok := args[0].([]interface{})
		if !ok || len(lst) == 0 {
			return nil
		}
		return lst[0]
	case "CDR":
		// Return the rest of the list after the first element.
		if len(args) != 1 {
			panic("cdr expects 1 argument")
		}
		if args[0] == nil {
			return nil
		}
		lst, ok := args[0].([]interface{})
		if !ok || len(lst) < 1 {
			return nil
		}
		if len(lst) == 1 {
			return nil
		}
		return lst[1:]
	case "CONS":
		// Construct a new list by prepending an element.
		if len(args) != 2 {
			panic("cons expects 2 arguments")
		}
		if args[1] == nil {
			return []interface{}{args[0]}
		}
		if lst, ok := args[1].([]interface{}); ok {
			return append([]interface{}{args[0]}, lst...)
		}
		// If the second argument is not a list, return a dotted pair.
		return []interface{}{args[0], args[1]}
	case "EQ":
		// Check if two symbols or numbers are the same.
		if len(args) != 2 {
			panic("eq expects 2 arguments")
		}
		x := args[0]
		y := args[1]
		if isNil(x) && isNil(y) {
			return "T"
		}
		switch xv := x.(type) {
		case string:
			if yv, ok := y.(string); ok && strings.ToUpper(xv) == strings.ToUpper(yv) {
				return "T"
			}
			return nil
		case int:
			if yv, ok := y.(int); ok && xv == yv {
				return "T"
			}
			return nil
		}
		return nil
	case "EQUAL":
		// Check if two values are structurally equal.
		if len(args) != 2 {
			panic("equal expects 2 arguments")
		}
		if equalp(args[0], args[1]) {
			return "T"
		}
		return nil
	case "ATOM":
		// Check if the argument is an atom (not a list).
		if len(args) != 1 {
			panic("atom expects 1 argument")
		}
		if _, ok := args[0].([]interface{}); ok {
			return nil
		}
		return "T"
	case "NULL":
		// Check if the argument is NIL.
		if len(args) != 1 {
			panic("null expects 1 argument")
		}
		if isNil(args[0]) {
			return "T"
		}
		return nil
	case "LISTP":
		// Check if the argument is a list.
		if len(args) != 1 {
			panic("listp expects 1 argument")
		}
		_, ok := args[0].([]interface{})
		return boolToT(ok)
	case "SYMBOLP":
		// Check if the argument is a symbol.
		if len(args) != 1 {
			panic("symbolp expects 1 argument")
		}
		_, isStr := args[0].(string)
		return boolToT(isStr)
	case "STRINGP":
		// Check if the argument is a string.
		if len(args) != 1 {
			panic("stringp expects 1 argument")
		}
		_, isStr := args[0].(string)
		return boolToT(isStr)
	case "NUMBERP":
		// Check if the argument is a number.
		if len(args) != 1 {
			panic("numberp expects 1 argument")
		}
		_, isNum := args[0].(int)
		return boolToT(isNum)
	case "PRINT":
		// Print the argument to the console.
		if len(args) != 1 {
			panic("print expects 1 argument")
		}
		fmt.Println(toLispString(args[0]))
		return args[0]
	case "+":
		// Addition of numbers.
		sum := 0
		for _, a := range args {
			num, ok := a.(int)
			if !ok {
				panic("+ expects integers")
			}
			sum += num
		}
		return sum
	case "-":
		// Subtraction of numbers.
		if len(args) < 1 {
			panic("- expects at least one argument")
		}
		first, ok := args[0].(int)
		if !ok {
			panic("- expects integers")
		}
		if len(args) == 1 {
			// Unary negation.
			return -first
		}
		result := first
		for _, a := range args[1:] {
			num, ok := a.(int)
			if !ok {
				panic("- expects integers")
			}
			result -= num
		}
		return result
	case "*":
		// Multiplication of numbers.
		prod := 1
		for _, a := range args {
			num, ok := a.(int)
			if !ok {
				panic("* expects integers")
			}
			prod *= num
		}
		return prod
	case "/":
		// Division of numbers.
		if len(args) < 2 {
			panic("/ expects at least two arguments")
		}
		first, ok := args[0].(int)
		if !ok {
			panic("/ expects integers")
		}
		result := first
		for _, a := range args[1:] {
			num, ok := a.(int)
			if !ok {
				panic("/ expects integers")
			}
			if num == 0 {
				panic("division by zero")
			}
			result = result / num
		}
		return result
	case "<":
		// Less than comparison.
		if len(args) != 2 {
			panic("< expects exactly two arguments")
		}
		x, okx := args[0].(int)
		y, oky := args[1].(int)
		if !okx || !oky {
			panic("< expects integers")
		}
		return boolToT(x < y)
	case ">":
		// Greater than comparison.
		if len(args) != 2 {
			panic("> expects exactly two arguments")
		}
		x, okx := args[0].(int)
		y, oky := args[1].(int)
		if !okx || !oky {
			panic("> expects integers")
		}
		return boolToT(x > y)
	case "1+":
		// Increment a number by one.
		if len(args) != 1 {
			panic("1+ expects one argument")
		}
		n, ok := args[0].(int)
		if !ok {
			panic("1+ expects an integer")
		}
		return n + 1
	case "1-":
		// Decrement a number by one.
		if len(args) != 1 {
			panic("1- expects one argument")
		}
		n, ok := args[0].(int)
		if !ok {
			panic("1- expects an integer")
		}
		return n - 1
	case "MOD":
		// Modulus operation.
		if len(args) != 2 {
			panic("mod expects exactly 2 arguments")
		}
		x, okx := args[0].(int)
		y, oky := args[1].(int)
		if !okx || !oky {
			panic("mod expects integers")
		}
		if y == 0 {
			panic("mod by zero")
		}
		return x % y
	case "FLOOR":
		// Floor function: either floor a number or perform floor division.
		if len(args) == 1 {
			// Single argument: floor the number.
			switch vv := args[0].(type) {
			case int:
				return vv
			case string:
				f, err := strconv.ParseFloat(vv, 64)
				if err != nil {
					panic("floor expects a number")
				}
				return int(math.Floor(f))
			default:
				panic("floor expects a number")
			}
		} else if len(args) == 2 {
			// Two arguments: floor division.
			x, okx := args[0].(int)
			y, oky := args[1].(int)
			if !okx || !oky {
				panic("floor expects integers when given two arguments")
			}
			if y == 0 {
				panic("division by zero")
			}
			return x / y
		} else {
			panic("floor expects one or two arguments")
		}
	case "=":
		// Equality comparison for numbers.
		if len(args) != 2 {
			panic("= expects exactly 2 arguments")
		}
		x, okx := args[0].(int)
		y, oky := args[1].(int)
		if !okx || !oky {
			panic("= expects integers")
		}
		if x == y {
			return "T"
		}
		return nil
	case "LIST":
		// Create a list from the provided arguments.
		return args
	case "ZEROP":
		// Check if a number is zero.
		if len(args) != 1 {
			panic("zerop expects 1 argument")
		}
		n, ok := args[0].(int)
		if !ok {
			panic("zerop expects an integer")
		}
		return boolToT(n == 0)
	case "ELEM":
		// Check if the first argument is an element of the second argument (a list).
		if len(args) != 2 {
			panic("elem expects 2 arguments")
		}
		item := args[0]
		lst, ok := args[1].([]interface{})
		if !ok {
			return nil
		}
		for _, v := range lst {
			if equalp(v, item) {
				return "T"
			}
		}
		return nil
	case "LAMBDA":
		// Return the lambda expression as a closure.
		return args
	case "IF":
		// Handle the if special form (duplicated handling, can be removed if not needed).
		if len(args) < 2 || len(args) > 3 {
			panic("if expects (if condition then [else])")
		}
		condition := myEval(args[0], alist)
		if !isNil(condition) {
			return myEval(args[1], alist)
		} else {
			if len(args) == 3 {
				return myEval(args[2], alist)
			}
			return nil
		}
	default:
		// Handle user-defined functions.
		fnDef, ok := globalAlist[fnSym]
		if !ok {
			panic("Unknown function: " + fnSym)
		}
		lambdaBody, ok := fnDef.([]interface{})
		if !ok {
			panic("Invalid function definition for: " + fnSym)
		}
		// Apply the user-defined lambda function.
		return myApplyLambda(lambdaBody, args, globalAlist)
	}
}

// tokenize splits the input string into Lisp tokens.
func tokenize(input string) []string {
	input = strings.TrimSpace(input)
	var tokens []string
	var token strings.Builder
	for i := 0; i < len(input); i++ {
		ch := input[i]
		switch ch {
		case '(':
			tokens = appendToken(tokens, token)
			token.Reset()
			tokens = append(tokens, "(")
		case ')':
			tokens = appendToken(tokens, token)
			token.Reset()
			tokens = append(tokens, ")")
		case '\'':
			tokens = appendToken(tokens, token)
			token.Reset()
			tokens = append(tokens, "'")
		case ' ':
			if token.Len() > 0 {
				tokens = appendToken(tokens, token)
				token.Reset()
			}
		default:
			token.WriteByte(ch)
		}
	}
	tokens = appendToken(tokens, token)
	return tokens
}

// appendToken appends the current token to tokens if it's not empty.
func appendToken(tokens []string, token strings.Builder) []string {
	if token.Len() > 0 {
		tokens = append(tokens, token.String())
	}
	return tokens
}

// parser struct keeps track of the list of tokens and the current position.
type parser struct {
	tokens []string
	pos    int
}

// next returns the next token and advances the position.
func (p *parser) next() string {
	if p.pos >= len(p.tokens) {
		return ""
	}
	t := p.tokens[p.pos]
	p.pos++
	return t
}

// peek returns the next token without advancing the position.
func (p *parser) peek() string {
	if p.pos >= len(p.tokens) {
		return ""
	}
	return p.tokens[p.pos]
}

// parseSExpression parses a single S-expression from the parser.
func parseSExpression(p *parser) interface{} {
	if p.pos >= len(p.tokens) {
		return nil
	}
	t := p.next()
	switch t {
	case "'":
		// Handle quoted expressions by converting 'expr to (quote expr).
		expr := parseSExpression(p)
		return []interface{}{"quote", expr}
	case "(":
		// Parse a list until the corresponding closing parenthesis.
		var lst []interface{}
		for {
			if p.pos >= len(p.tokens) {
				panic("unmatched parenthesis")
			}
			if p.peek() == ")" {
				p.next()
				break
			}
			lst = append(lst, parseSExpression(p))
		}
		return lst
	case ")":
		// Unexpected closing parenthesis.
		panic("unexpected )")
	default:
		// Try to parse the token as an integer; if it fails, treat it as a symbol.
		if num, err := strconv.Atoi(t); err == nil {
			return num
		}
		return t
	}
}

// readSExpression tokenizes and parses the input string into an S-expression.
func readSExpression(input string) interface{} {
	tokens := tokenize(input)
	if len(tokens) == 0 {
		return nil
	}
	p := &parser{tokens: tokens}
	expr := parseSExpression(p)
	if p.pos != len(p.tokens) {
		panic("extra tokens after parse")
	}
	return expr
}

func myTop() {
	fmt.Println("Simple LISP Interpreter in Go (Using MY-EVAL)")
	fmt.Println("Type 'exit' to quit.")

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		// Read input from the user.
		line, err := reader.ReadString('\n')
		if err != nil {
			panic(err)
		}
		line = strings.TrimSpace(line)
		if line == "exit" {
			break
		}
		if line == "" {
			continue
		}
		// Parse the input into an S-expression.
		expr := readSExpression(line)
		// Evaluate the S-expression.
		result := myEval(expr, globalAlist)
		// Print the result of the evaluation.
		fmt.Println(toLispString(result))
	}
}

// main function starts the REPL.
func main() {
	myTop()
}
