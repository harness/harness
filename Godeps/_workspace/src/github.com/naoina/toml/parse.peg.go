package toml

import (
	"fmt"
	"math"
	"sort"
	"strconv"
)

const end_symbol rune = 4

/* The rule types inferred from the grammar are below. */
type pegRule uint8

const (
	ruleUnknown pegRule = iota
	ruleTOML
	ruleExpression
	rulenewline
	rulews
	rulewsnl
	rulecomment
	rulekeyval
	rulekey
	rulebareKey
	rulequotedKey
	ruleval
	ruletable
	rulestdTable
	rulearrayTable
	ruleinlineTable
	ruleinlineTableKeyValues
	ruletableKey
	ruletableKeySep
	ruleinlineTableValSep
	ruleinteger
	ruleint
	rulefloat
	rulefrac
	ruleexp
	rulestring
	rulebasicString
	rulebasicChar
	ruleescaped
	rulebasicUnescaped
	ruleescape
	rulemlBasicString
	rulemlBasicBody
	ruleliteralString
	ruleliteralChar
	rulemlLiteralString
	rulemlLiteralBody
	rulemlLiteralChar
	rulehexdigit
	rulehexQuad
	ruleboolean
	ruledateFullYear
	ruledateMonth
	ruledateMDay
	ruletimeHour
	ruletimeMinute
	ruletimeSecond
	ruletimeSecfrac
	ruletimeNumoffset
	ruletimeOffset
	rulepartialTime
	rulefullDate
	rulefullTime
	ruledatetime
	ruledigit
	ruledigitDual
	ruledigitQuad
	rulearray
	rulearrayValues
	rulearraySep
	ruleAction0
	rulePegText
	ruleAction1
	ruleAction2
	ruleAction3
	ruleAction4
	ruleAction5
	ruleAction6
	ruleAction7
	ruleAction8
	ruleAction9
	ruleAction10
	ruleAction11
	ruleAction12
	ruleAction13
	ruleAction14
	ruleAction15
	ruleAction16
	ruleAction17
	ruleAction18
	ruleAction19
	ruleAction20
	ruleAction21
	ruleAction22
	ruleAction23

	rulePre_
	rule_In_
	rule_Suf
)

var rul3s = [...]string{
	"Unknown",
	"TOML",
	"Expression",
	"newline",
	"ws",
	"wsnl",
	"comment",
	"keyval",
	"key",
	"bareKey",
	"quotedKey",
	"val",
	"table",
	"stdTable",
	"arrayTable",
	"inlineTable",
	"inlineTableKeyValues",
	"tableKey",
	"tableKeySep",
	"inlineTableValSep",
	"integer",
	"int",
	"float",
	"frac",
	"exp",
	"string",
	"basicString",
	"basicChar",
	"escaped",
	"basicUnescaped",
	"escape",
	"mlBasicString",
	"mlBasicBody",
	"literalString",
	"literalChar",
	"mlLiteralString",
	"mlLiteralBody",
	"mlLiteralChar",
	"hexdigit",
	"hexQuad",
	"boolean",
	"dateFullYear",
	"dateMonth",
	"dateMDay",
	"timeHour",
	"timeMinute",
	"timeSecond",
	"timeSecfrac",
	"timeNumoffset",
	"timeOffset",
	"partialTime",
	"fullDate",
	"fullTime",
	"datetime",
	"digit",
	"digitDual",
	"digitQuad",
	"array",
	"arrayValues",
	"arraySep",
	"Action0",
	"PegText",
	"Action1",
	"Action2",
	"Action3",
	"Action4",
	"Action5",
	"Action6",
	"Action7",
	"Action8",
	"Action9",
	"Action10",
	"Action11",
	"Action12",
	"Action13",
	"Action14",
	"Action15",
	"Action16",
	"Action17",
	"Action18",
	"Action19",
	"Action20",
	"Action21",
	"Action22",
	"Action23",

	"Pre_",
	"_In_",
	"_Suf",
}

type tokenTree interface {
	Print()
	PrintSyntax()
	PrintSyntaxTree(buffer string)
	Add(rule pegRule, begin, end, next, depth int)
	Expand(index int) tokenTree
	Tokens() <-chan token32
	AST() *node32
	Error() []token32
	trim(length int)
}

type node32 struct {
	token32
	up, next *node32
}

func (node *node32) print(depth int, buffer string) {
	for node != nil {
		for c := 0; c < depth; c++ {
			fmt.Printf(" ")
		}
		fmt.Printf("\x1B[34m%v\x1B[m %v\n", rul3s[node.pegRule], strconv.Quote(string(([]rune(buffer)[node.begin:node.end]))))
		if node.up != nil {
			node.up.print(depth+1, buffer)
		}
		node = node.next
	}
}

func (ast *node32) Print(buffer string) {
	ast.print(0, buffer)
}

type element struct {
	node *node32
	down *element
}

/* ${@} bit structure for abstract syntax tree */
type token16 struct {
	pegRule
	begin, end, next int16
}

func (t *token16) isZero() bool {
	return t.pegRule == ruleUnknown && t.begin == 0 && t.end == 0 && t.next == 0
}

func (t *token16) isParentOf(u token16) bool {
	return t.begin <= u.begin && t.end >= u.end && t.next > u.next
}

func (t *token16) getToken32() token32 {
	return token32{pegRule: t.pegRule, begin: int32(t.begin), end: int32(t.end), next: int32(t.next)}
}

func (t *token16) String() string {
	return fmt.Sprintf("\x1B[34m%v\x1B[m %v %v %v", rul3s[t.pegRule], t.begin, t.end, t.next)
}

type tokens16 struct {
	tree    []token16
	ordered [][]token16
}

func (t *tokens16) trim(length int) {
	t.tree = t.tree[0:length]
}

func (t *tokens16) Print() {
	for _, token := range t.tree {
		fmt.Println(token.String())
	}
}

func (t *tokens16) Order() [][]token16 {
	if t.ordered != nil {
		return t.ordered
	}

	depths := make([]int16, 1, math.MaxInt16)
	for i, token := range t.tree {
		if token.pegRule == ruleUnknown {
			t.tree = t.tree[:i]
			break
		}
		depth := int(token.next)
		if length := len(depths); depth >= length {
			depths = depths[:depth+1]
		}
		depths[depth]++
	}
	depths = append(depths, 0)

	ordered, pool := make([][]token16, len(depths)), make([]token16, len(t.tree)+len(depths))
	for i, depth := range depths {
		depth++
		ordered[i], pool, depths[i] = pool[:depth], pool[depth:], 0
	}

	for i, token := range t.tree {
		depth := token.next
		token.next = int16(i)
		ordered[depth][depths[depth]] = token
		depths[depth]++
	}
	t.ordered = ordered
	return ordered
}

type state16 struct {
	token16
	depths []int16
	leaf   bool
}

func (t *tokens16) AST() *node32 {
	tokens := t.Tokens()
	stack := &element{node: &node32{token32: <-tokens}}
	for token := range tokens {
		if token.begin == token.end {
			continue
		}
		node := &node32{token32: token}
		for stack != nil && stack.node.begin >= token.begin && stack.node.end <= token.end {
			stack.node.next = node.up
			node.up = stack.node
			stack = stack.down
		}
		stack = &element{node: node, down: stack}
	}
	return stack.node
}

func (t *tokens16) PreOrder() (<-chan state16, [][]token16) {
	s, ordered := make(chan state16, 6), t.Order()
	go func() {
		var states [8]state16
		for i, _ := range states {
			states[i].depths = make([]int16, len(ordered))
		}
		depths, state, depth := make([]int16, len(ordered)), 0, 1
		write := func(t token16, leaf bool) {
			S := states[state]
			state, S.pegRule, S.begin, S.end, S.next, S.leaf = (state+1)%8, t.pegRule, t.begin, t.end, int16(depth), leaf
			copy(S.depths, depths)
			s <- S
		}

		states[state].token16 = ordered[0][0]
		depths[0]++
		state++
		a, b := ordered[depth-1][depths[depth-1]-1], ordered[depth][depths[depth]]
	depthFirstSearch:
		for {
			for {
				if i := depths[depth]; i > 0 {
					if c, j := ordered[depth][i-1], depths[depth-1]; a.isParentOf(c) &&
						(j < 2 || !ordered[depth-1][j-2].isParentOf(c)) {
						if c.end != b.begin {
							write(token16{pegRule: rule_In_, begin: c.end, end: b.begin}, true)
						}
						break
					}
				}

				if a.begin < b.begin {
					write(token16{pegRule: rulePre_, begin: a.begin, end: b.begin}, true)
				}
				break
			}

			next := depth + 1
			if c := ordered[next][depths[next]]; c.pegRule != ruleUnknown && b.isParentOf(c) {
				write(b, false)
				depths[depth]++
				depth, a, b = next, b, c
				continue
			}

			write(b, true)
			depths[depth]++
			c, parent := ordered[depth][depths[depth]], true
			for {
				if c.pegRule != ruleUnknown && a.isParentOf(c) {
					b = c
					continue depthFirstSearch
				} else if parent && b.end != a.end {
					write(token16{pegRule: rule_Suf, begin: b.end, end: a.end}, true)
				}

				depth--
				if depth > 0 {
					a, b, c = ordered[depth-1][depths[depth-1]-1], a, ordered[depth][depths[depth]]
					parent = a.isParentOf(b)
					continue
				}

				break depthFirstSearch
			}
		}

		close(s)
	}()
	return s, ordered
}

func (t *tokens16) PrintSyntax() {
	tokens, ordered := t.PreOrder()
	max := -1
	for token := range tokens {
		if !token.leaf {
			fmt.Printf("%v", token.begin)
			for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
				fmt.Printf(" \x1B[36m%v\x1B[m", rul3s[ordered[i][depths[i]-1].pegRule])
			}
			fmt.Printf(" \x1B[36m%v\x1B[m\n", rul3s[token.pegRule])
		} else if token.begin == token.end {
			fmt.Printf("%v", token.begin)
			for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
				fmt.Printf(" \x1B[31m%v\x1B[m", rul3s[ordered[i][depths[i]-1].pegRule])
			}
			fmt.Printf(" \x1B[31m%v\x1B[m\n", rul3s[token.pegRule])
		} else {
			for c, end := token.begin, token.end; c < end; c++ {
				if i := int(c); max+1 < i {
					for j := max; j < i; j++ {
						fmt.Printf("skip %v %v\n", j, token.String())
					}
					max = i
				} else if i := int(c); i <= max {
					for j := i; j <= max; j++ {
						fmt.Printf("dupe %v %v\n", j, token.String())
					}
				} else {
					max = int(c)
				}
				fmt.Printf("%v", c)
				for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
					fmt.Printf(" \x1B[34m%v\x1B[m", rul3s[ordered[i][depths[i]-1].pegRule])
				}
				fmt.Printf(" \x1B[34m%v\x1B[m\n", rul3s[token.pegRule])
			}
			fmt.Printf("\n")
		}
	}
}

func (t *tokens16) PrintSyntaxTree(buffer string) {
	tokens, _ := t.PreOrder()
	for token := range tokens {
		for c := 0; c < int(token.next); c++ {
			fmt.Printf(" ")
		}
		fmt.Printf("\x1B[34m%v\x1B[m %v\n", rul3s[token.pegRule], strconv.Quote(string(([]rune(buffer)[token.begin:token.end]))))
	}
}

func (t *tokens16) Add(rule pegRule, begin, end, depth, index int) {
	t.tree[index] = token16{pegRule: rule, begin: int16(begin), end: int16(end), next: int16(depth)}
}

func (t *tokens16) Tokens() <-chan token32 {
	s := make(chan token32, 16)
	go func() {
		for _, v := range t.tree {
			s <- v.getToken32()
		}
		close(s)
	}()
	return s
}

func (t *tokens16) Error() []token32 {
	ordered := t.Order()
	length := len(ordered)
	tokens, length := make([]token32, length), length-1
	for i, _ := range tokens {
		o := ordered[length-i]
		if len(o) > 1 {
			tokens[i] = o[len(o)-2].getToken32()
		}
	}
	return tokens
}

/* ${@} bit structure for abstract syntax tree */
type token32 struct {
	pegRule
	begin, end, next int32
}

func (t *token32) isZero() bool {
	return t.pegRule == ruleUnknown && t.begin == 0 && t.end == 0 && t.next == 0
}

func (t *token32) isParentOf(u token32) bool {
	return t.begin <= u.begin && t.end >= u.end && t.next > u.next
}

func (t *token32) getToken32() token32 {
	return token32{pegRule: t.pegRule, begin: int32(t.begin), end: int32(t.end), next: int32(t.next)}
}

func (t *token32) String() string {
	return fmt.Sprintf("\x1B[34m%v\x1B[m %v %v %v", rul3s[t.pegRule], t.begin, t.end, t.next)
}

type tokens32 struct {
	tree    []token32
	ordered [][]token32
}

func (t *tokens32) trim(length int) {
	t.tree = t.tree[0:length]
}

func (t *tokens32) Print() {
	for _, token := range t.tree {
		fmt.Println(token.String())
	}
}

func (t *tokens32) Order() [][]token32 {
	if t.ordered != nil {
		return t.ordered
	}

	depths := make([]int32, 1, math.MaxInt16)
	for i, token := range t.tree {
		if token.pegRule == ruleUnknown {
			t.tree = t.tree[:i]
			break
		}
		depth := int(token.next)
		if length := len(depths); depth >= length {
			depths = depths[:depth+1]
		}
		depths[depth]++
	}
	depths = append(depths, 0)

	ordered, pool := make([][]token32, len(depths)), make([]token32, len(t.tree)+len(depths))
	for i, depth := range depths {
		depth++
		ordered[i], pool, depths[i] = pool[:depth], pool[depth:], 0
	}

	for i, token := range t.tree {
		depth := token.next
		token.next = int32(i)
		ordered[depth][depths[depth]] = token
		depths[depth]++
	}
	t.ordered = ordered
	return ordered
}

type state32 struct {
	token32
	depths []int32
	leaf   bool
}

func (t *tokens32) AST() *node32 {
	tokens := t.Tokens()
	stack := &element{node: &node32{token32: <-tokens}}
	for token := range tokens {
		if token.begin == token.end {
			continue
		}
		node := &node32{token32: token}
		for stack != nil && stack.node.begin >= token.begin && stack.node.end <= token.end {
			stack.node.next = node.up
			node.up = stack.node
			stack = stack.down
		}
		stack = &element{node: node, down: stack}
	}
	return stack.node
}

func (t *tokens32) PreOrder() (<-chan state32, [][]token32) {
	s, ordered := make(chan state32, 6), t.Order()
	go func() {
		var states [8]state32
		for i, _ := range states {
			states[i].depths = make([]int32, len(ordered))
		}
		depths, state, depth := make([]int32, len(ordered)), 0, 1
		write := func(t token32, leaf bool) {
			S := states[state]
			state, S.pegRule, S.begin, S.end, S.next, S.leaf = (state+1)%8, t.pegRule, t.begin, t.end, int32(depth), leaf
			copy(S.depths, depths)
			s <- S
		}

		states[state].token32 = ordered[0][0]
		depths[0]++
		state++
		a, b := ordered[depth-1][depths[depth-1]-1], ordered[depth][depths[depth]]
	depthFirstSearch:
		for {
			for {
				if i := depths[depth]; i > 0 {
					if c, j := ordered[depth][i-1], depths[depth-1]; a.isParentOf(c) &&
						(j < 2 || !ordered[depth-1][j-2].isParentOf(c)) {
						if c.end != b.begin {
							write(token32{pegRule: rule_In_, begin: c.end, end: b.begin}, true)
						}
						break
					}
				}

				if a.begin < b.begin {
					write(token32{pegRule: rulePre_, begin: a.begin, end: b.begin}, true)
				}
				break
			}

			next := depth + 1
			if c := ordered[next][depths[next]]; c.pegRule != ruleUnknown && b.isParentOf(c) {
				write(b, false)
				depths[depth]++
				depth, a, b = next, b, c
				continue
			}

			write(b, true)
			depths[depth]++
			c, parent := ordered[depth][depths[depth]], true
			for {
				if c.pegRule != ruleUnknown && a.isParentOf(c) {
					b = c
					continue depthFirstSearch
				} else if parent && b.end != a.end {
					write(token32{pegRule: rule_Suf, begin: b.end, end: a.end}, true)
				}

				depth--
				if depth > 0 {
					a, b, c = ordered[depth-1][depths[depth-1]-1], a, ordered[depth][depths[depth]]
					parent = a.isParentOf(b)
					continue
				}

				break depthFirstSearch
			}
		}

		close(s)
	}()
	return s, ordered
}

func (t *tokens32) PrintSyntax() {
	tokens, ordered := t.PreOrder()
	max := -1
	for token := range tokens {
		if !token.leaf {
			fmt.Printf("%v", token.begin)
			for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
				fmt.Printf(" \x1B[36m%v\x1B[m", rul3s[ordered[i][depths[i]-1].pegRule])
			}
			fmt.Printf(" \x1B[36m%v\x1B[m\n", rul3s[token.pegRule])
		} else if token.begin == token.end {
			fmt.Printf("%v", token.begin)
			for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
				fmt.Printf(" \x1B[31m%v\x1B[m", rul3s[ordered[i][depths[i]-1].pegRule])
			}
			fmt.Printf(" \x1B[31m%v\x1B[m\n", rul3s[token.pegRule])
		} else {
			for c, end := token.begin, token.end; c < end; c++ {
				if i := int(c); max+1 < i {
					for j := max; j < i; j++ {
						fmt.Printf("skip %v %v\n", j, token.String())
					}
					max = i
				} else if i := int(c); i <= max {
					for j := i; j <= max; j++ {
						fmt.Printf("dupe %v %v\n", j, token.String())
					}
				} else {
					max = int(c)
				}
				fmt.Printf("%v", c)
				for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
					fmt.Printf(" \x1B[34m%v\x1B[m", rul3s[ordered[i][depths[i]-1].pegRule])
				}
				fmt.Printf(" \x1B[34m%v\x1B[m\n", rul3s[token.pegRule])
			}
			fmt.Printf("\n")
		}
	}
}

func (t *tokens32) PrintSyntaxTree(buffer string) {
	tokens, _ := t.PreOrder()
	for token := range tokens {
		for c := 0; c < int(token.next); c++ {
			fmt.Printf(" ")
		}
		fmt.Printf("\x1B[34m%v\x1B[m %v\n", rul3s[token.pegRule], strconv.Quote(string(([]rune(buffer)[token.begin:token.end]))))
	}
}

func (t *tokens32) Add(rule pegRule, begin, end, depth, index int) {
	t.tree[index] = token32{pegRule: rule, begin: int32(begin), end: int32(end), next: int32(depth)}
}

func (t *tokens32) Tokens() <-chan token32 {
	s := make(chan token32, 16)
	go func() {
		for _, v := range t.tree {
			s <- v.getToken32()
		}
		close(s)
	}()
	return s
}

func (t *tokens32) Error() []token32 {
	ordered := t.Order()
	length := len(ordered)
	tokens, length := make([]token32, length), length-1
	for i, _ := range tokens {
		o := ordered[length-i]
		if len(o) > 1 {
			tokens[i] = o[len(o)-2].getToken32()
		}
	}
	return tokens
}

func (t *tokens16) Expand(index int) tokenTree {
	tree := t.tree
	if index >= len(tree) {
		expanded := make([]token32, 2*len(tree))
		for i, v := range tree {
			expanded[i] = v.getToken32()
		}
		return &tokens32{tree: expanded}
	}
	return nil
}

func (t *tokens32) Expand(index int) tokenTree {
	tree := t.tree
	if index >= len(tree) {
		expanded := make([]token32, 2*len(tree))
		copy(expanded, tree)
		t.tree = expanded
	}
	return nil
}

type tomlParser struct {
	toml

	Buffer string
	buffer []rune
	rules  [85]func() bool
	Parse  func(rule ...int) error
	Reset  func()
	tokenTree
}

type textPosition struct {
	line, symbol int
}

type textPositionMap map[int]textPosition

func translatePositions(buffer string, positions []int) textPositionMap {
	length, translations, j, line, symbol := len(positions), make(textPositionMap, len(positions)), 0, 1, 0
	sort.Ints(positions)

search:
	for i, c := range buffer[0:] {
		if c == '\n' {
			line, symbol = line+1, 0
		} else {
			symbol++
		}
		if i == positions[j] {
			translations[positions[j]] = textPosition{line, symbol}
			for j++; j < length; j++ {
				if i != positions[j] {
					continue search
				}
			}
			break search
		}
	}

	return translations
}

type parseError struct {
	p *tomlParser
}

func (e *parseError) Error() string {
	tokens, error := e.p.tokenTree.Error(), "\n"
	positions, p := make([]int, 2*len(tokens)), 0
	for _, token := range tokens {
		positions[p], p = int(token.begin), p+1
		positions[p], p = int(token.end), p+1
	}
	translations := translatePositions(e.p.Buffer, positions)
	for _, token := range tokens {
		begin, end := int(token.begin), int(token.end)
		error += fmt.Sprintf("parse error near \x1B[34m%v\x1B[m (line %v symbol %v - line %v symbol %v):\n%v\n",
			rul3s[token.pegRule],
			translations[begin].line, translations[begin].symbol,
			translations[end].line, translations[end].symbol,
			/*strconv.Quote(*/ e.p.Buffer[begin:end] /*)*/)
	}

	return error
}

func (p *tomlParser) PrintSyntaxTree() {
	p.tokenTree.PrintSyntaxTree(p.Buffer)
}

func (p *tomlParser) Highlighter() {
	p.tokenTree.PrintSyntax()
}

func (p *tomlParser) Execute() {
	buffer, begin, end := p.Buffer, 0, 0
	for token := range p.tokenTree.Tokens() {
		switch token.pegRule {

		case rulePegText:
			begin, end = int(token.begin), int(token.end)

		case ruleAction0:
			_ = buffer
		case ruleAction1:
			p.SetTableString(begin, end)
		case ruleAction2:
			p.AddLineCount(end - begin)
		case ruleAction3:
			p.AddLineCount(end - begin)
		case ruleAction4:
			p.AddKeyValue()
		case ruleAction5:
			p.SetKey(p.buffer, begin, end)
		case ruleAction6:
			p.SetKey(p.buffer, begin, end)
		case ruleAction7:
			p.SetTime(begin, end)
		case ruleAction8:
			p.SetFloat64(begin, end)
		case ruleAction9:
			p.SetInt64(begin, end)
		case ruleAction10:
			p.SetString(begin, end)
		case ruleAction11:
			p.SetBool(begin, end)
		case ruleAction12:
			p.SetArray(begin, end)
		case ruleAction13:
			p.SetTable(p.buffer, begin, end)
		case ruleAction14:
			p.SetArrayTable(p.buffer, begin, end)
		case ruleAction15:
			p.StartInlineTable()
		case ruleAction16:
			p.EndInlineTable()
		case ruleAction17:
			p.SetBasicString(p.buffer, begin, end)
		case ruleAction18:
			p.SetMultilineString()
		case ruleAction19:
			p.AddMultilineBasicBody(p.buffer, begin, end)
		case ruleAction20:
			p.SetLiteralString(p.buffer, begin, end)
		case ruleAction21:
			p.SetMultilineLiteralString(p.buffer, begin, end)
		case ruleAction22:
			p.StartArray()
		case ruleAction23:
			p.AddArrayVal()

		}
	}
	_, _, _ = buffer, begin, end
}

func (p *tomlParser) Init() {
	p.buffer = []rune(p.Buffer)
	if len(p.buffer) == 0 || p.buffer[len(p.buffer)-1] != end_symbol {
		p.buffer = append(p.buffer, end_symbol)
	}

	var tree tokenTree = &tokens16{tree: make([]token16, math.MaxInt16)}
	position, depth, tokenIndex, buffer, _rules := 0, 0, 0, p.buffer, p.rules

	p.Parse = func(rule ...int) error {
		r := 1
		if len(rule) > 0 {
			r = rule[0]
		}
		matches := p.rules[r]()
		p.tokenTree = tree
		if matches {
			p.tokenTree.trim(tokenIndex)
			return nil
		}
		return &parseError{p}
	}

	p.Reset = func() {
		position, tokenIndex, depth = 0, 0, 0
	}

	add := func(rule pegRule, begin int) {
		if t := tree.Expand(tokenIndex); t != nil {
			tree = t
		}
		tree.Add(rule, begin, position, depth, tokenIndex)
		tokenIndex++
	}

	matchDot := func() bool {
		if buffer[position] != end_symbol {
			position++
			return true
		}
		return false
	}

	/*matchChar := func(c byte) bool {
		if buffer[position] == c {
			position++
			return true
		}
		return false
	}*/

	/*matchRange := func(lower byte, upper byte) bool {
		if c := buffer[position]; c >= lower && c <= upper {
			position++
			return true
		}
		return false
	}*/

	_rules = [...]func() bool{
		nil,
		/* 0 TOML <- <(Expression (newline Expression)* newline? !. Action0)> */
		func() bool {
			position0, tokenIndex0, depth0 := position, tokenIndex, depth
			{
				position1 := position
				depth++
				if !_rules[ruleExpression]() {
					goto l0
				}
			l2:
				{
					position3, tokenIndex3, depth3 := position, tokenIndex, depth
					if !_rules[rulenewline]() {
						goto l3
					}
					if !_rules[ruleExpression]() {
						goto l3
					}
					goto l2
				l3:
					position, tokenIndex, depth = position3, tokenIndex3, depth3
				}
				{
					position4, tokenIndex4, depth4 := position, tokenIndex, depth
					if !_rules[rulenewline]() {
						goto l4
					}
					goto l5
				l4:
					position, tokenIndex, depth = position4, tokenIndex4, depth4
				}
			l5:
				{
					position6, tokenIndex6, depth6 := position, tokenIndex, depth
					if !matchDot() {
						goto l6
					}
					goto l0
				l6:
					position, tokenIndex, depth = position6, tokenIndex6, depth6
				}
				{
					add(ruleAction0, position)
				}
				depth--
				add(ruleTOML, position1)
			}
			return true
		l0:
			position, tokenIndex, depth = position0, tokenIndex0, depth0
			return false
		},
		/* 1 Expression <- <((<(ws table ws comment? (wsnl keyval ws comment?)*)> Action1) / (ws keyval ws comment?) / (ws comment?) / ws)> */
		func() bool {
			position8, tokenIndex8, depth8 := position, tokenIndex, depth
			{
				position9 := position
				depth++
				{
					position10, tokenIndex10, depth10 := position, tokenIndex, depth
					{
						position12 := position
						depth++
						if !_rules[rulews]() {
							goto l11
						}
						{
							position13 := position
							depth++
							{
								position14, tokenIndex14, depth14 := position, tokenIndex, depth
								{
									position16 := position
									depth++
									if buffer[position] != rune('[') {
										goto l15
									}
									position++
									if !_rules[rulews]() {
										goto l15
									}
									{
										position17 := position
										depth++
										if !_rules[ruletableKey]() {
											goto l15
										}
										depth--
										add(rulePegText, position17)
									}
									if !_rules[rulews]() {
										goto l15
									}
									if buffer[position] != rune(']') {
										goto l15
									}
									position++
									{
										add(ruleAction13, position)
									}
									depth--
									add(rulestdTable, position16)
								}
								goto l14
							l15:
								position, tokenIndex, depth = position14, tokenIndex14, depth14
								{
									position19 := position
									depth++
									if buffer[position] != rune('[') {
										goto l11
									}
									position++
									if buffer[position] != rune('[') {
										goto l11
									}
									position++
									if !_rules[rulews]() {
										goto l11
									}
									{
										position20 := position
										depth++
										if !_rules[ruletableKey]() {
											goto l11
										}
										depth--
										add(rulePegText, position20)
									}
									if !_rules[rulews]() {
										goto l11
									}
									if buffer[position] != rune(']') {
										goto l11
									}
									position++
									if buffer[position] != rune(']') {
										goto l11
									}
									position++
									{
										add(ruleAction14, position)
									}
									depth--
									add(rulearrayTable, position19)
								}
							}
						l14:
							depth--
							add(ruletable, position13)
						}
						if !_rules[rulews]() {
							goto l11
						}
						{
							position22, tokenIndex22, depth22 := position, tokenIndex, depth
							if !_rules[rulecomment]() {
								goto l22
							}
							goto l23
						l22:
							position, tokenIndex, depth = position22, tokenIndex22, depth22
						}
					l23:
					l24:
						{
							position25, tokenIndex25, depth25 := position, tokenIndex, depth
							if !_rules[rulewsnl]() {
								goto l25
							}
							if !_rules[rulekeyval]() {
								goto l25
							}
							if !_rules[rulews]() {
								goto l25
							}
							{
								position26, tokenIndex26, depth26 := position, tokenIndex, depth
								if !_rules[rulecomment]() {
									goto l26
								}
								goto l27
							l26:
								position, tokenIndex, depth = position26, tokenIndex26, depth26
							}
						l27:
							goto l24
						l25:
							position, tokenIndex, depth = position25, tokenIndex25, depth25
						}
						depth--
						add(rulePegText, position12)
					}
					{
						add(ruleAction1, position)
					}
					goto l10
				l11:
					position, tokenIndex, depth = position10, tokenIndex10, depth10
					if !_rules[rulews]() {
						goto l29
					}
					if !_rules[rulekeyval]() {
						goto l29
					}
					if !_rules[rulews]() {
						goto l29
					}
					{
						position30, tokenIndex30, depth30 := position, tokenIndex, depth
						if !_rules[rulecomment]() {
							goto l30
						}
						goto l31
					l30:
						position, tokenIndex, depth = position30, tokenIndex30, depth30
					}
				l31:
					goto l10
				l29:
					position, tokenIndex, depth = position10, tokenIndex10, depth10
					if !_rules[rulews]() {
						goto l32
					}
					{
						position33, tokenIndex33, depth33 := position, tokenIndex, depth
						if !_rules[rulecomment]() {
							goto l33
						}
						goto l34
					l33:
						position, tokenIndex, depth = position33, tokenIndex33, depth33
					}
				l34:
					goto l10
				l32:
					position, tokenIndex, depth = position10, tokenIndex10, depth10
					if !_rules[rulews]() {
						goto l8
					}
				}
			l10:
				depth--
				add(ruleExpression, position9)
			}
			return true
		l8:
			position, tokenIndex, depth = position8, tokenIndex8, depth8
			return false
		},
		/* 2 newline <- <(<('\r' / '\n')+> Action2)> */
		func() bool {
			position35, tokenIndex35, depth35 := position, tokenIndex, depth
			{
				position36 := position
				depth++
				{
					position37 := position
					depth++
					{
						position40, tokenIndex40, depth40 := position, tokenIndex, depth
						if buffer[position] != rune('\r') {
							goto l41
						}
						position++
						goto l40
					l41:
						position, tokenIndex, depth = position40, tokenIndex40, depth40
						if buffer[position] != rune('\n') {
							goto l35
						}
						position++
					}
				l40:
				l38:
					{
						position39, tokenIndex39, depth39 := position, tokenIndex, depth
						{
							position42, tokenIndex42, depth42 := position, tokenIndex, depth
							if buffer[position] != rune('\r') {
								goto l43
							}
							position++
							goto l42
						l43:
							position, tokenIndex, depth = position42, tokenIndex42, depth42
							if buffer[position] != rune('\n') {
								goto l39
							}
							position++
						}
					l42:
						goto l38
					l39:
						position, tokenIndex, depth = position39, tokenIndex39, depth39
					}
					depth--
					add(rulePegText, position37)
				}
				{
					add(ruleAction2, position)
				}
				depth--
				add(rulenewline, position36)
			}
			return true
		l35:
			position, tokenIndex, depth = position35, tokenIndex35, depth35
			return false
		},
		/* 3 ws <- <(' ' / '\t')*> */
		func() bool {
			{
				position46 := position
				depth++
			l47:
				{
					position48, tokenIndex48, depth48 := position, tokenIndex, depth
					{
						position49, tokenIndex49, depth49 := position, tokenIndex, depth
						if buffer[position] != rune(' ') {
							goto l50
						}
						position++
						goto l49
					l50:
						position, tokenIndex, depth = position49, tokenIndex49, depth49
						if buffer[position] != rune('\t') {
							goto l48
						}
						position++
					}
				l49:
					goto l47
				l48:
					position, tokenIndex, depth = position48, tokenIndex48, depth48
				}
				depth--
				add(rulews, position46)
			}
			return true
		},
		/* 4 wsnl <- <((&('\t') '\t') | (&(' ') ' ') | (&('\n' | '\r') (<('\r' / '\n')> Action3)))*> */
		func() bool {
			{
				position52 := position
				depth++
			l53:
				{
					position54, tokenIndex54, depth54 := position, tokenIndex, depth
					{
						switch buffer[position] {
						case '\t':
							if buffer[position] != rune('\t') {
								goto l54
							}
							position++
							break
						case ' ':
							if buffer[position] != rune(' ') {
								goto l54
							}
							position++
							break
						default:
							{
								position56 := position
								depth++
								{
									position57, tokenIndex57, depth57 := position, tokenIndex, depth
									if buffer[position] != rune('\r') {
										goto l58
									}
									position++
									goto l57
								l58:
									position, tokenIndex, depth = position57, tokenIndex57, depth57
									if buffer[position] != rune('\n') {
										goto l54
									}
									position++
								}
							l57:
								depth--
								add(rulePegText, position56)
							}
							{
								add(ruleAction3, position)
							}
							break
						}
					}

					goto l53
				l54:
					position, tokenIndex, depth = position54, tokenIndex54, depth54
				}
				depth--
				add(rulewsnl, position52)
			}
			return true
		},
		/* 5 comment <- <('#' <('\t' / [ -􏿿])*>)> */
		func() bool {
			position60, tokenIndex60, depth60 := position, tokenIndex, depth
			{
				position61 := position
				depth++
				if buffer[position] != rune('#') {
					goto l60
				}
				position++
				{
					position62 := position
					depth++
				l63:
					{
						position64, tokenIndex64, depth64 := position, tokenIndex, depth
						{
							position65, tokenIndex65, depth65 := position, tokenIndex, depth
							if buffer[position] != rune('\t') {
								goto l66
							}
							position++
							goto l65
						l66:
							position, tokenIndex, depth = position65, tokenIndex65, depth65
							if c := buffer[position]; c < rune(' ') || c > rune('\U0010ffff') {
								goto l64
							}
							position++
						}
					l65:
						goto l63
					l64:
						position, tokenIndex, depth = position64, tokenIndex64, depth64
					}
					depth--
					add(rulePegText, position62)
				}
				depth--
				add(rulecomment, position61)
			}
			return true
		l60:
			position, tokenIndex, depth = position60, tokenIndex60, depth60
			return false
		},
		/* 6 keyval <- <(key ws '=' ws val Action4)> */
		func() bool {
			position67, tokenIndex67, depth67 := position, tokenIndex, depth
			{
				position68 := position
				depth++
				if !_rules[rulekey]() {
					goto l67
				}
				if !_rules[rulews]() {
					goto l67
				}
				if buffer[position] != rune('=') {
					goto l67
				}
				position++
				if !_rules[rulews]() {
					goto l67
				}
				if !_rules[ruleval]() {
					goto l67
				}
				{
					add(ruleAction4, position)
				}
				depth--
				add(rulekeyval, position68)
			}
			return true
		l67:
			position, tokenIndex, depth = position67, tokenIndex67, depth67
			return false
		},
		/* 7 key <- <(bareKey / quotedKey)> */
		func() bool {
			position70, tokenIndex70, depth70 := position, tokenIndex, depth
			{
				position71 := position
				depth++
				{
					position72, tokenIndex72, depth72 := position, tokenIndex, depth
					{
						position74 := position
						depth++
						{
							position75 := position
							depth++
							{
								switch buffer[position] {
								case '_':
									if buffer[position] != rune('_') {
										goto l73
									}
									position++
									break
								case '-':
									if buffer[position] != rune('-') {
										goto l73
									}
									position++
									break
								case 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z':
									if c := buffer[position]; c < rune('a') || c > rune('z') {
										goto l73
									}
									position++
									break
								case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
									if c := buffer[position]; c < rune('0') || c > rune('9') {
										goto l73
									}
									position++
									break
								default:
									if c := buffer[position]; c < rune('A') || c > rune('Z') {
										goto l73
									}
									position++
									break
								}
							}

						l76:
							{
								position77, tokenIndex77, depth77 := position, tokenIndex, depth
								{
									switch buffer[position] {
									case '_':
										if buffer[position] != rune('_') {
											goto l77
										}
										position++
										break
									case '-':
										if buffer[position] != rune('-') {
											goto l77
										}
										position++
										break
									case 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z':
										if c := buffer[position]; c < rune('a') || c > rune('z') {
											goto l77
										}
										position++
										break
									case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
										if c := buffer[position]; c < rune('0') || c > rune('9') {
											goto l77
										}
										position++
										break
									default:
										if c := buffer[position]; c < rune('A') || c > rune('Z') {
											goto l77
										}
										position++
										break
									}
								}

								goto l76
							l77:
								position, tokenIndex, depth = position77, tokenIndex77, depth77
							}
							depth--
							add(rulePegText, position75)
						}
						{
							add(ruleAction5, position)
						}
						depth--
						add(rulebareKey, position74)
					}
					goto l72
				l73:
					position, tokenIndex, depth = position72, tokenIndex72, depth72
					{
						position81 := position
						depth++
						if buffer[position] != rune('"') {
							goto l70
						}
						position++
						{
							position82 := position
							depth++
							if !_rules[rulebasicChar]() {
								goto l70
							}
						l83:
							{
								position84, tokenIndex84, depth84 := position, tokenIndex, depth
								if !_rules[rulebasicChar]() {
									goto l84
								}
								goto l83
							l84:
								position, tokenIndex, depth = position84, tokenIndex84, depth84
							}
							depth--
							add(rulePegText, position82)
						}
						if buffer[position] != rune('"') {
							goto l70
						}
						position++
						{
							add(ruleAction6, position)
						}
						depth--
						add(rulequotedKey, position81)
					}
				}
			l72:
				depth--
				add(rulekey, position71)
			}
			return true
		l70:
			position, tokenIndex, depth = position70, tokenIndex70, depth70
			return false
		},
		/* 8 bareKey <- <(<((&('_') '_') | (&('-') '-') | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]) | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]))+> Action5)> */
		nil,
		/* 9 quotedKey <- <('"' <basicChar+> '"' Action6)> */
		nil,
		/* 10 val <- <((<datetime> Action7) / (<float> Action8) / ((&('{') inlineTable) | (&('[') (<array> Action12)) | (&('f' | 't') (<boolean> Action11)) | (&('"' | '\'') (<string> Action10)) | (&('+' | '-' | '0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') (<integer> Action9))))> */
		func() bool {
			position88, tokenIndex88, depth88 := position, tokenIndex, depth
			{
				position89 := position
				depth++
				{
					position90, tokenIndex90, depth90 := position, tokenIndex, depth
					{
						position92 := position
						depth++
						{
							position93 := position
							depth++
							{
								position94 := position
								depth++
								{
									position95 := position
									depth++
									{
										position96 := position
										depth++
										if !_rules[ruledigitDual]() {
											goto l91
										}
										if !_rules[ruledigitDual]() {
											goto l91
										}
										depth--
										add(ruledigitQuad, position96)
									}
									depth--
									add(ruledateFullYear, position95)
								}
								if buffer[position] != rune('-') {
									goto l91
								}
								position++
								{
									position97 := position
									depth++
									if !_rules[ruledigitDual]() {
										goto l91
									}
									depth--
									add(ruledateMonth, position97)
								}
								if buffer[position] != rune('-') {
									goto l91
								}
								position++
								{
									position98 := position
									depth++
									if !_rules[ruledigitDual]() {
										goto l91
									}
									depth--
									add(ruledateMDay, position98)
								}
								depth--
								add(rulefullDate, position94)
							}
							if buffer[position] != rune('T') {
								goto l91
							}
							position++
							{
								position99 := position
								depth++
								{
									position100 := position
									depth++
									if !_rules[ruletimeHour]() {
										goto l91
									}
									if buffer[position] != rune(':') {
										goto l91
									}
									position++
									if !_rules[ruletimeMinute]() {
										goto l91
									}
									if buffer[position] != rune(':') {
										goto l91
									}
									position++
									{
										position101 := position
										depth++
										if !_rules[ruledigitDual]() {
											goto l91
										}
										depth--
										add(ruletimeSecond, position101)
									}
									{
										position102, tokenIndex102, depth102 := position, tokenIndex, depth
										{
											position104 := position
											depth++
											if buffer[position] != rune('.') {
												goto l102
											}
											position++
											if !_rules[ruledigit]() {
												goto l102
											}
										l105:
											{
												position106, tokenIndex106, depth106 := position, tokenIndex, depth
												if !_rules[ruledigit]() {
													goto l106
												}
												goto l105
											l106:
												position, tokenIndex, depth = position106, tokenIndex106, depth106
											}
											depth--
											add(ruletimeSecfrac, position104)
										}
										goto l103
									l102:
										position, tokenIndex, depth = position102, tokenIndex102, depth102
									}
								l103:
									depth--
									add(rulepartialTime, position100)
								}
								{
									position107 := position
									depth++
									{
										position108, tokenIndex108, depth108 := position, tokenIndex, depth
										if buffer[position] != rune('Z') {
											goto l109
										}
										position++
										goto l108
									l109:
										position, tokenIndex, depth = position108, tokenIndex108, depth108
										{
											position110 := position
											depth++
											{
												position111, tokenIndex111, depth111 := position, tokenIndex, depth
												if buffer[position] != rune('-') {
													goto l112
												}
												position++
												goto l111
											l112:
												position, tokenIndex, depth = position111, tokenIndex111, depth111
												if buffer[position] != rune('+') {
													goto l91
												}
												position++
											}
										l111:
											if !_rules[ruletimeHour]() {
												goto l91
											}
											if buffer[position] != rune(':') {
												goto l91
											}
											position++
											if !_rules[ruletimeMinute]() {
												goto l91
											}
											depth--
											add(ruletimeNumoffset, position110)
										}
									}
								l108:
									depth--
									add(ruletimeOffset, position107)
								}
								depth--
								add(rulefullTime, position99)
							}
							depth--
							add(ruledatetime, position93)
						}
						depth--
						add(rulePegText, position92)
					}
					{
						add(ruleAction7, position)
					}
					goto l90
				l91:
					position, tokenIndex, depth = position90, tokenIndex90, depth90
					{
						position115 := position
						depth++
						{
							position116 := position
							depth++
							if !_rules[ruleinteger]() {
								goto l114
							}
							{
								position117, tokenIndex117, depth117 := position, tokenIndex, depth
								if !_rules[rulefrac]() {
									goto l118
								}
								{
									position119, tokenIndex119, depth119 := position, tokenIndex, depth
									if !_rules[ruleexp]() {
										goto l119
									}
									goto l120
								l119:
									position, tokenIndex, depth = position119, tokenIndex119, depth119
								}
							l120:
								goto l117
							l118:
								position, tokenIndex, depth = position117, tokenIndex117, depth117
								{
									position121, tokenIndex121, depth121 := position, tokenIndex, depth
									if !_rules[rulefrac]() {
										goto l121
									}
									goto l122
								l121:
									position, tokenIndex, depth = position121, tokenIndex121, depth121
								}
							l122:
								if !_rules[ruleexp]() {
									goto l114
								}
							}
						l117:
							depth--
							add(rulefloat, position116)
						}
						depth--
						add(rulePegText, position115)
					}
					{
						add(ruleAction8, position)
					}
					goto l90
				l114:
					position, tokenIndex, depth = position90, tokenIndex90, depth90
					{
						switch buffer[position] {
						case '{':
							{
								position125 := position
								depth++
								if buffer[position] != rune('{') {
									goto l88
								}
								position++
								{
									add(ruleAction15, position)
								}
								if !_rules[rulews]() {
									goto l88
								}
								{
									position127 := position
									depth++
								l128:
									{
										position129, tokenIndex129, depth129 := position, tokenIndex, depth
										if !_rules[rulekeyval]() {
											goto l129
										}
										{
											position130, tokenIndex130, depth130 := position, tokenIndex, depth
											{
												position132 := position
												depth++
												if !_rules[rulews]() {
													goto l130
												}
												if buffer[position] != rune(',') {
													goto l130
												}
												position++
												if !_rules[rulews]() {
													goto l130
												}
												depth--
												add(ruleinlineTableValSep, position132)
											}
											goto l131
										l130:
											position, tokenIndex, depth = position130, tokenIndex130, depth130
										}
									l131:
										goto l128
									l129:
										position, tokenIndex, depth = position129, tokenIndex129, depth129
									}
									depth--
									add(ruleinlineTableKeyValues, position127)
								}
								if !_rules[rulews]() {
									goto l88
								}
								if buffer[position] != rune('}') {
									goto l88
								}
								position++
								{
									add(ruleAction16, position)
								}
								depth--
								add(ruleinlineTable, position125)
							}
							break
						case '[':
							{
								position134 := position
								depth++
								{
									position135 := position
									depth++
									if buffer[position] != rune('[') {
										goto l88
									}
									position++
									{
										add(ruleAction22, position)
									}
									if !_rules[rulewsnl]() {
										goto l88
									}
									{
										position137 := position
										depth++
									l138:
										{
											position139, tokenIndex139, depth139 := position, tokenIndex, depth
											if !_rules[ruleval]() {
												goto l139
											}
											{
												add(ruleAction23, position)
											}
											{
												position141, tokenIndex141, depth141 := position, tokenIndex, depth
												{
													position143 := position
													depth++
													if !_rules[rulews]() {
														goto l141
													}
													if buffer[position] != rune(',') {
														goto l141
													}
													position++
													if !_rules[rulewsnl]() {
														goto l141
													}
													depth--
													add(rulearraySep, position143)
												}
												goto l142
											l141:
												position, tokenIndex, depth = position141, tokenIndex141, depth141
											}
										l142:
											{
												position144, tokenIndex144, depth144 := position, tokenIndex, depth
												{
													position146, tokenIndex146, depth146 := position, tokenIndex, depth
													if !_rules[rulecomment]() {
														goto l146
													}
													goto l147
												l146:
													position, tokenIndex, depth = position146, tokenIndex146, depth146
												}
											l147:
												if !_rules[rulenewline]() {
													goto l144
												}
												goto l145
											l144:
												position, tokenIndex, depth = position144, tokenIndex144, depth144
											}
										l145:
											goto l138
										l139:
											position, tokenIndex, depth = position139, tokenIndex139, depth139
										}
										depth--
										add(rulearrayValues, position137)
									}
									if !_rules[rulewsnl]() {
										goto l88
									}
									if buffer[position] != rune(']') {
										goto l88
									}
									position++
									depth--
									add(rulearray, position135)
								}
								depth--
								add(rulePegText, position134)
							}
							{
								add(ruleAction12, position)
							}
							break
						case 'f', 't':
							{
								position149 := position
								depth++
								{
									position150 := position
									depth++
									{
										position151, tokenIndex151, depth151 := position, tokenIndex, depth
										if buffer[position] != rune('t') {
											goto l152
										}
										position++
										if buffer[position] != rune('r') {
											goto l152
										}
										position++
										if buffer[position] != rune('u') {
											goto l152
										}
										position++
										if buffer[position] != rune('e') {
											goto l152
										}
										position++
										goto l151
									l152:
										position, tokenIndex, depth = position151, tokenIndex151, depth151
										if buffer[position] != rune('f') {
											goto l88
										}
										position++
										if buffer[position] != rune('a') {
											goto l88
										}
										position++
										if buffer[position] != rune('l') {
											goto l88
										}
										position++
										if buffer[position] != rune('s') {
											goto l88
										}
										position++
										if buffer[position] != rune('e') {
											goto l88
										}
										position++
									}
								l151:
									depth--
									add(ruleboolean, position150)
								}
								depth--
								add(rulePegText, position149)
							}
							{
								add(ruleAction11, position)
							}
							break
						case '"', '\'':
							{
								position154 := position
								depth++
								{
									position155 := position
									depth++
									{
										position156, tokenIndex156, depth156 := position, tokenIndex, depth
										{
											position158 := position
											depth++
											if buffer[position] != rune('\'') {
												goto l157
											}
											position++
											if buffer[position] != rune('\'') {
												goto l157
											}
											position++
											if buffer[position] != rune('\'') {
												goto l157
											}
											position++
											{
												position159 := position
												depth++
												{
													position160 := position
													depth++
												l161:
													{
														position162, tokenIndex162, depth162 := position, tokenIndex, depth
														{
															position163, tokenIndex163, depth163 := position, tokenIndex, depth
															if buffer[position] != rune('\'') {
																goto l163
															}
															position++
															if buffer[position] != rune('\'') {
																goto l163
															}
															position++
															if buffer[position] != rune('\'') {
																goto l163
															}
															position++
															goto l162
														l163:
															position, tokenIndex, depth = position163, tokenIndex163, depth163
														}
														{
															position164, tokenIndex164, depth164 := position, tokenIndex, depth
															{
																position166 := position
																depth++
																{
																	position167, tokenIndex167, depth167 := position, tokenIndex, depth
																	if buffer[position] != rune('\t') {
																		goto l168
																	}
																	position++
																	goto l167
																l168:
																	position, tokenIndex, depth = position167, tokenIndex167, depth167
																	if c := buffer[position]; c < rune(' ') || c > rune('\U0010ffff') {
																		goto l165
																	}
																	position++
																}
															l167:
																depth--
																add(rulemlLiteralChar, position166)
															}
															goto l164
														l165:
															position, tokenIndex, depth = position164, tokenIndex164, depth164
															if !_rules[rulenewline]() {
																goto l162
															}
														}
													l164:
														goto l161
													l162:
														position, tokenIndex, depth = position162, tokenIndex162, depth162
													}
													depth--
													add(rulemlLiteralBody, position160)
												}
												depth--
												add(rulePegText, position159)
											}
											if buffer[position] != rune('\'') {
												goto l157
											}
											position++
											if buffer[position] != rune('\'') {
												goto l157
											}
											position++
											if buffer[position] != rune('\'') {
												goto l157
											}
											position++
											{
												add(ruleAction21, position)
											}
											depth--
											add(rulemlLiteralString, position158)
										}
										goto l156
									l157:
										position, tokenIndex, depth = position156, tokenIndex156, depth156
										{
											position171 := position
											depth++
											if buffer[position] != rune('\'') {
												goto l170
											}
											position++
											{
												position172 := position
												depth++
											l173:
												{
													position174, tokenIndex174, depth174 := position, tokenIndex, depth
													{
														position175 := position
														depth++
														{
															switch buffer[position] {
															case '\t':
																if buffer[position] != rune('\t') {
																	goto l174
																}
																position++
																break
															case ' ', '!', '"', '#', '$', '%', '&':
																if c := buffer[position]; c < rune(' ') || c > rune('&') {
																	goto l174
																}
																position++
																break
															default:
																if c := buffer[position]; c < rune('(') || c > rune('\U0010ffff') {
																	goto l174
																}
																position++
																break
															}
														}

														depth--
														add(ruleliteralChar, position175)
													}
													goto l173
												l174:
													position, tokenIndex, depth = position174, tokenIndex174, depth174
												}
												depth--
												add(rulePegText, position172)
											}
											if buffer[position] != rune('\'') {
												goto l170
											}
											position++
											{
												add(ruleAction20, position)
											}
											depth--
											add(ruleliteralString, position171)
										}
										goto l156
									l170:
										position, tokenIndex, depth = position156, tokenIndex156, depth156
										{
											position179 := position
											depth++
											if buffer[position] != rune('"') {
												goto l178
											}
											position++
											if buffer[position] != rune('"') {
												goto l178
											}
											position++
											if buffer[position] != rune('"') {
												goto l178
											}
											position++
											{
												position180 := position
												depth++
											l181:
												{
													position182, tokenIndex182, depth182 := position, tokenIndex, depth
													{
														position183, tokenIndex183, depth183 := position, tokenIndex, depth
														{
															position185 := position
															depth++
															{
																position186, tokenIndex186, depth186 := position, tokenIndex, depth
																if !_rules[rulebasicChar]() {
																	goto l187
																}
																goto l186
															l187:
																position, tokenIndex, depth = position186, tokenIndex186, depth186
																if !_rules[rulenewline]() {
																	goto l184
																}
															}
														l186:
															depth--
															add(rulePegText, position185)
														}
														{
															add(ruleAction19, position)
														}
														goto l183
													l184:
														position, tokenIndex, depth = position183, tokenIndex183, depth183
														if !_rules[ruleescape]() {
															goto l182
														}
														if !_rules[rulenewline]() {
															goto l182
														}
														if !_rules[rulewsnl]() {
															goto l182
														}
													}
												l183:
													goto l181
												l182:
													position, tokenIndex, depth = position182, tokenIndex182, depth182
												}
												depth--
												add(rulemlBasicBody, position180)
											}
											if buffer[position] != rune('"') {
												goto l178
											}
											position++
											if buffer[position] != rune('"') {
												goto l178
											}
											position++
											if buffer[position] != rune('"') {
												goto l178
											}
											position++
											{
												add(ruleAction18, position)
											}
											depth--
											add(rulemlBasicString, position179)
										}
										goto l156
									l178:
										position, tokenIndex, depth = position156, tokenIndex156, depth156
										{
											position190 := position
											depth++
											{
												position191 := position
												depth++
												if buffer[position] != rune('"') {
													goto l88
												}
												position++
											l192:
												{
													position193, tokenIndex193, depth193 := position, tokenIndex, depth
													if !_rules[rulebasicChar]() {
														goto l193
													}
													goto l192
												l193:
													position, tokenIndex, depth = position193, tokenIndex193, depth193
												}
												if buffer[position] != rune('"') {
													goto l88
												}
												position++
												depth--
												add(rulePegText, position191)
											}
											{
												add(ruleAction17, position)
											}
											depth--
											add(rulebasicString, position190)
										}
									}
								l156:
									depth--
									add(rulestring, position155)
								}
								depth--
								add(rulePegText, position154)
							}
							{
								add(ruleAction10, position)
							}
							break
						default:
							{
								position196 := position
								depth++
								if !_rules[ruleinteger]() {
									goto l88
								}
								depth--
								add(rulePegText, position196)
							}
							{
								add(ruleAction9, position)
							}
							break
						}
					}

				}
			l90:
				depth--
				add(ruleval, position89)
			}
			return true
		l88:
			position, tokenIndex, depth = position88, tokenIndex88, depth88
			return false
		},
		/* 11 table <- <(stdTable / arrayTable)> */
		nil,
		/* 12 stdTable <- <('[' ws <tableKey> ws ']' Action13)> */
		nil,
		/* 13 arrayTable <- <('[' '[' ws <tableKey> ws (']' ']') Action14)> */
		nil,
		/* 14 inlineTable <- <('{' Action15 ws inlineTableKeyValues ws '}' Action16)> */
		nil,
		/* 15 inlineTableKeyValues <- <(keyval inlineTableValSep?)*> */
		nil,
		/* 16 tableKey <- <(key (tableKeySep key)*)> */
		func() bool {
			position203, tokenIndex203, depth203 := position, tokenIndex, depth
			{
				position204 := position
				depth++
				if !_rules[rulekey]() {
					goto l203
				}
			l205:
				{
					position206, tokenIndex206, depth206 := position, tokenIndex, depth
					{
						position207 := position
						depth++
						if !_rules[rulews]() {
							goto l206
						}
						if buffer[position] != rune('.') {
							goto l206
						}
						position++
						if !_rules[rulews]() {
							goto l206
						}
						depth--
						add(ruletableKeySep, position207)
					}
					if !_rules[rulekey]() {
						goto l206
					}
					goto l205
				l206:
					position, tokenIndex, depth = position206, tokenIndex206, depth206
				}
				depth--
				add(ruletableKey, position204)
			}
			return true
		l203:
			position, tokenIndex, depth = position203, tokenIndex203, depth203
			return false
		},
		/* 17 tableKeySep <- <(ws '.' ws)> */
		nil,
		/* 18 inlineTableValSep <- <(ws ',' ws)> */
		nil,
		/* 19 integer <- <(('-' / '+')? int)> */
		func() bool {
			position210, tokenIndex210, depth210 := position, tokenIndex, depth
			{
				position211 := position
				depth++
				{
					position212, tokenIndex212, depth212 := position, tokenIndex, depth
					{
						position214, tokenIndex214, depth214 := position, tokenIndex, depth
						if buffer[position] != rune('-') {
							goto l215
						}
						position++
						goto l214
					l215:
						position, tokenIndex, depth = position214, tokenIndex214, depth214
						if buffer[position] != rune('+') {
							goto l212
						}
						position++
					}
				l214:
					goto l213
				l212:
					position, tokenIndex, depth = position212, tokenIndex212, depth212
				}
			l213:
				{
					position216 := position
					depth++
					{
						position217, tokenIndex217, depth217 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('1') || c > rune('9') {
							goto l218
						}
						position++
						{
							position221, tokenIndex221, depth221 := position, tokenIndex, depth
							if !_rules[ruledigit]() {
								goto l222
							}
							goto l221
						l222:
							position, tokenIndex, depth = position221, tokenIndex221, depth221
							if buffer[position] != rune('_') {
								goto l218
							}
							position++
							if !_rules[ruledigit]() {
								goto l218
							}
						}
					l221:
					l219:
						{
							position220, tokenIndex220, depth220 := position, tokenIndex, depth
							{
								position223, tokenIndex223, depth223 := position, tokenIndex, depth
								if !_rules[ruledigit]() {
									goto l224
								}
								goto l223
							l224:
								position, tokenIndex, depth = position223, tokenIndex223, depth223
								if buffer[position] != rune('_') {
									goto l220
								}
								position++
								if !_rules[ruledigit]() {
									goto l220
								}
							}
						l223:
							goto l219
						l220:
							position, tokenIndex, depth = position220, tokenIndex220, depth220
						}
						goto l217
					l218:
						position, tokenIndex, depth = position217, tokenIndex217, depth217
						if !_rules[ruledigit]() {
							goto l210
						}
					}
				l217:
					depth--
					add(ruleint, position216)
				}
				depth--
				add(ruleinteger, position211)
			}
			return true
		l210:
			position, tokenIndex, depth = position210, tokenIndex210, depth210
			return false
		},
		/* 20 int <- <(([1-9] (digit / ('_' digit))+) / digit)> */
		nil,
		/* 21 float <- <(integer ((frac exp?) / (frac? exp)))> */
		nil,
		/* 22 frac <- <('.' digit (digit / ('_' digit))*)> */
		func() bool {
			position227, tokenIndex227, depth227 := position, tokenIndex, depth
			{
				position228 := position
				depth++
				if buffer[position] != rune('.') {
					goto l227
				}
				position++
				if !_rules[ruledigit]() {
					goto l227
				}
			l229:
				{
					position230, tokenIndex230, depth230 := position, tokenIndex, depth
					{
						position231, tokenIndex231, depth231 := position, tokenIndex, depth
						if !_rules[ruledigit]() {
							goto l232
						}
						goto l231
					l232:
						position, tokenIndex, depth = position231, tokenIndex231, depth231
						if buffer[position] != rune('_') {
							goto l230
						}
						position++
						if !_rules[ruledigit]() {
							goto l230
						}
					}
				l231:
					goto l229
				l230:
					position, tokenIndex, depth = position230, tokenIndex230, depth230
				}
				depth--
				add(rulefrac, position228)
			}
			return true
		l227:
			position, tokenIndex, depth = position227, tokenIndex227, depth227
			return false
		},
		/* 23 exp <- <(('e' / 'E') ('-' / '+')? digit (digit / ('_' digit))*)> */
		func() bool {
			position233, tokenIndex233, depth233 := position, tokenIndex, depth
			{
				position234 := position
				depth++
				{
					position235, tokenIndex235, depth235 := position, tokenIndex, depth
					if buffer[position] != rune('e') {
						goto l236
					}
					position++
					goto l235
				l236:
					position, tokenIndex, depth = position235, tokenIndex235, depth235
					if buffer[position] != rune('E') {
						goto l233
					}
					position++
				}
			l235:
				{
					position237, tokenIndex237, depth237 := position, tokenIndex, depth
					{
						position239, tokenIndex239, depth239 := position, tokenIndex, depth
						if buffer[position] != rune('-') {
							goto l240
						}
						position++
						goto l239
					l240:
						position, tokenIndex, depth = position239, tokenIndex239, depth239
						if buffer[position] != rune('+') {
							goto l237
						}
						position++
					}
				l239:
					goto l238
				l237:
					position, tokenIndex, depth = position237, tokenIndex237, depth237
				}
			l238:
				if !_rules[ruledigit]() {
					goto l233
				}
			l241:
				{
					position242, tokenIndex242, depth242 := position, tokenIndex, depth
					{
						position243, tokenIndex243, depth243 := position, tokenIndex, depth
						if !_rules[ruledigit]() {
							goto l244
						}
						goto l243
					l244:
						position, tokenIndex, depth = position243, tokenIndex243, depth243
						if buffer[position] != rune('_') {
							goto l242
						}
						position++
						if !_rules[ruledigit]() {
							goto l242
						}
					}
				l243:
					goto l241
				l242:
					position, tokenIndex, depth = position242, tokenIndex242, depth242
				}
				depth--
				add(ruleexp, position234)
			}
			return true
		l233:
			position, tokenIndex, depth = position233, tokenIndex233, depth233
			return false
		},
		/* 24 string <- <(mlLiteralString / literalString / mlBasicString / basicString)> */
		nil,
		/* 25 basicString <- <(<('"' basicChar* '"')> Action17)> */
		nil,
		/* 26 basicChar <- <(basicUnescaped / escaped)> */
		func() bool {
			position247, tokenIndex247, depth247 := position, tokenIndex, depth
			{
				position248 := position
				depth++
				{
					position249, tokenIndex249, depth249 := position, tokenIndex, depth
					{
						position251 := position
						depth++
						{
							switch buffer[position] {
							case ' ', '!':
								if c := buffer[position]; c < rune(' ') || c > rune('!') {
									goto l250
								}
								position++
								break
							case '#', '$', '%', '&', '\'', '(', ')', '*', '+', ',', '-', '.', '/', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', ':', ';', '<', '=', '>', '?', '@', 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z', '[':
								if c := buffer[position]; c < rune('#') || c > rune('[') {
									goto l250
								}
								position++
								break
							default:
								if c := buffer[position]; c < rune(']') || c > rune('\U0010ffff') {
									goto l250
								}
								position++
								break
							}
						}

						depth--
						add(rulebasicUnescaped, position251)
					}
					goto l249
				l250:
					position, tokenIndex, depth = position249, tokenIndex249, depth249
					{
						position253 := position
						depth++
						if !_rules[ruleescape]() {
							goto l247
						}
						{
							switch buffer[position] {
							case 'U':
								if buffer[position] != rune('U') {
									goto l247
								}
								position++
								if !_rules[rulehexQuad]() {
									goto l247
								}
								if !_rules[rulehexQuad]() {
									goto l247
								}
								break
							case 'u':
								if buffer[position] != rune('u') {
									goto l247
								}
								position++
								if !_rules[rulehexQuad]() {
									goto l247
								}
								break
							case '\\':
								if buffer[position] != rune('\\') {
									goto l247
								}
								position++
								break
							case '/':
								if buffer[position] != rune('/') {
									goto l247
								}
								position++
								break
							case '"':
								if buffer[position] != rune('"') {
									goto l247
								}
								position++
								break
							case 'r':
								if buffer[position] != rune('r') {
									goto l247
								}
								position++
								break
							case 'f':
								if buffer[position] != rune('f') {
									goto l247
								}
								position++
								break
							case 'n':
								if buffer[position] != rune('n') {
									goto l247
								}
								position++
								break
							case 't':
								if buffer[position] != rune('t') {
									goto l247
								}
								position++
								break
							default:
								if buffer[position] != rune('b') {
									goto l247
								}
								position++
								break
							}
						}

						depth--
						add(ruleescaped, position253)
					}
				}
			l249:
				depth--
				add(rulebasicChar, position248)
			}
			return true
		l247:
			position, tokenIndex, depth = position247, tokenIndex247, depth247
			return false
		},
		/* 27 escaped <- <(escape ((&('U') ('U' hexQuad hexQuad)) | (&('u') ('u' hexQuad)) | (&('\\') '\\') | (&('/') '/') | (&('"') '"') | (&('r') 'r') | (&('f') 'f') | (&('n') 'n') | (&('t') 't') | (&('b') 'b')))> */
		nil,
		/* 28 basicUnescaped <- <((&(' ' | '!') [ -!]) | (&('#' | '$' | '%' | '&' | '\'' | '(' | ')' | '*' | '+' | ',' | '-' | '.' | '/' | '0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9' | ':' | ';' | '<' | '=' | '>' | '?' | '@' | 'A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z' | '[') [#-[]) | (&(']' | '^' | '_' | '`' | 'a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z' | '{' | '|' | '}' | '~' | '\u007f' | '\u0080' | '\u0081' | '\u0082' | '\u0083' | '\u0084' | '\u0085' | '\u0086' | '\u0087' | '\u0088' | '\u0089' | '\u008a' | '\u008b' | '\u008c' | '\u008d' | '\u008e' | '\u008f' | '\u0090' | '\u0091' | '\u0092' | '\u0093' | '\u0094' | '\u0095' | '\u0096' | '\u0097' | '\u0098' | '\u0099' | '\u009a' | '\u009b' | '\u009c' | '\u009d' | '\u009e' | '\u009f' | '\u00a0' | '¡' | '¢' | '£' | '¤' | '¥' | '¦' | '§' | '¨' | '©' | 'ª' | '«' | '¬' | '\u00ad' | '®' | '¯' | '°' | '±' | '²' | '³' | '´' | 'µ' | '¶' | '·' | '¸' | '¹' | 'º' | '»' | '¼' | '½' | '¾' | '¿' | 'À' | 'Á' | 'Â' | 'Ã' | 'Ä' | 'Å' | 'Æ' | 'Ç' | 'È' | 'É' | 'Ê' | 'Ë' | 'Ì' | 'Í' | 'Î' | 'Ï' | 'Ð' | 'Ñ' | 'Ò' | 'Ó' | 'Ô' | 'Õ' | 'Ö' | '×' | 'Ø' | 'Ù' | 'Ú' | 'Û' | 'Ü' | 'Ý' | 'Þ' | 'ß' | 'à' | 'á' | 'â' | 'ã' | 'ä' | 'å' | 'æ' | 'ç' | 'è' | 'é' | 'ê' | 'ë' | 'ì' | 'í' | 'î' | 'ï' | 'ð' | 'ñ' | 'ò' | 'ó' | 'ô') []-􏿿]))> */
		nil,
		/* 29 escape <- <'\\'> */
		func() bool {
			position257, tokenIndex257, depth257 := position, tokenIndex, depth
			{
				position258 := position
				depth++
				if buffer[position] != rune('\\') {
					goto l257
				}
				position++
				depth--
				add(ruleescape, position258)
			}
			return true
		l257:
			position, tokenIndex, depth = position257, tokenIndex257, depth257
			return false
		},
		/* 30 mlBasicString <- <('"' '"' '"' mlBasicBody ('"' '"' '"') Action18)> */
		nil,
		/* 31 mlBasicBody <- <((<(basicChar / newline)> Action19) / (escape newline wsnl))*> */
		nil,
		/* 32 literalString <- <('\'' <literalChar*> '\'' Action20)> */
		nil,
		/* 33 literalChar <- <((&('\t') '\t') | (&(' ' | '!' | '"' | '#' | '$' | '%' | '&') [ -&]) | (&('(' | ')' | '*' | '+' | ',' | '-' | '.' | '/' | '0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9' | ':' | ';' | '<' | '=' | '>' | '?' | '@' | 'A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z' | '[' | '\\' | ']' | '^' | '_' | '`' | 'a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z' | '{' | '|' | '}' | '~' | '\u007f' | '\u0080' | '\u0081' | '\u0082' | '\u0083' | '\u0084' | '\u0085' | '\u0086' | '\u0087' | '\u0088' | '\u0089' | '\u008a' | '\u008b' | '\u008c' | '\u008d' | '\u008e' | '\u008f' | '\u0090' | '\u0091' | '\u0092' | '\u0093' | '\u0094' | '\u0095' | '\u0096' | '\u0097' | '\u0098' | '\u0099' | '\u009a' | '\u009b' | '\u009c' | '\u009d' | '\u009e' | '\u009f' | '\u00a0' | '¡' | '¢' | '£' | '¤' | '¥' | '¦' | '§' | '¨' | '©' | 'ª' | '«' | '¬' | '\u00ad' | '®' | '¯' | '°' | '±' | '²' | '³' | '´' | 'µ' | '¶' | '·' | '¸' | '¹' | 'º' | '»' | '¼' | '½' | '¾' | '¿' | 'À' | 'Á' | 'Â' | 'Ã' | 'Ä' | 'Å' | 'Æ' | 'Ç' | 'È' | 'É' | 'Ê' | 'Ë' | 'Ì' | 'Í' | 'Î' | 'Ï' | 'Ð' | 'Ñ' | 'Ò' | 'Ó' | 'Ô' | 'Õ' | 'Ö' | '×' | 'Ø' | 'Ù' | 'Ú' | 'Û' | 'Ü' | 'Ý' | 'Þ' | 'ß' | 'à' | 'á' | 'â' | 'ã' | 'ä' | 'å' | 'æ' | 'ç' | 'è' | 'é' | 'ê' | 'ë' | 'ì' | 'í' | 'î' | 'ï' | 'ð' | 'ñ' | 'ò' | 'ó' | 'ô') [(-􏿿]))> */
		nil,
		/* 34 mlLiteralString <- <('\'' '\'' '\'' <mlLiteralBody> ('\'' '\'' '\'') Action21)> */
		nil,
		/* 35 mlLiteralBody <- <(!('\'' '\'' '\'') (mlLiteralChar / newline))*> */
		nil,
		/* 36 mlLiteralChar <- <('\t' / [ -􏿿])> */
		nil,
		/* 37 hexdigit <- <((&('a' | 'b' | 'c' | 'd' | 'e' | 'f') [a-f]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F') [A-F]) | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]))> */
		func() bool {
			position266, tokenIndex266, depth266 := position, tokenIndex, depth
			{
				position267 := position
				depth++
				{
					switch buffer[position] {
					case 'a', 'b', 'c', 'd', 'e', 'f':
						if c := buffer[position]; c < rune('a') || c > rune('f') {
							goto l266
						}
						position++
						break
					case 'A', 'B', 'C', 'D', 'E', 'F':
						if c := buffer[position]; c < rune('A') || c > rune('F') {
							goto l266
						}
						position++
						break
					default:
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l266
						}
						position++
						break
					}
				}

				depth--
				add(rulehexdigit, position267)
			}
			return true
		l266:
			position, tokenIndex, depth = position266, tokenIndex266, depth266
			return false
		},
		/* 38 hexQuad <- <(hexdigit hexdigit hexdigit hexdigit)> */
		func() bool {
			position269, tokenIndex269, depth269 := position, tokenIndex, depth
			{
				position270 := position
				depth++
				if !_rules[rulehexdigit]() {
					goto l269
				}
				if !_rules[rulehexdigit]() {
					goto l269
				}
				if !_rules[rulehexdigit]() {
					goto l269
				}
				if !_rules[rulehexdigit]() {
					goto l269
				}
				depth--
				add(rulehexQuad, position270)
			}
			return true
		l269:
			position, tokenIndex, depth = position269, tokenIndex269, depth269
			return false
		},
		/* 39 boolean <- <(('t' 'r' 'u' 'e') / ('f' 'a' 'l' 's' 'e'))> */
		nil,
		/* 40 dateFullYear <- <digitQuad> */
		nil,
		/* 41 dateMonth <- <digitDual> */
		nil,
		/* 42 dateMDay <- <digitDual> */
		nil,
		/* 43 timeHour <- <digitDual> */
		func() bool {
			position275, tokenIndex275, depth275 := position, tokenIndex, depth
			{
				position276 := position
				depth++
				if !_rules[ruledigitDual]() {
					goto l275
				}
				depth--
				add(ruletimeHour, position276)
			}
			return true
		l275:
			position, tokenIndex, depth = position275, tokenIndex275, depth275
			return false
		},
		/* 44 timeMinute <- <digitDual> */
		func() bool {
			position277, tokenIndex277, depth277 := position, tokenIndex, depth
			{
				position278 := position
				depth++
				if !_rules[ruledigitDual]() {
					goto l277
				}
				depth--
				add(ruletimeMinute, position278)
			}
			return true
		l277:
			position, tokenIndex, depth = position277, tokenIndex277, depth277
			return false
		},
		/* 45 timeSecond <- <digitDual> */
		nil,
		/* 46 timeSecfrac <- <('.' digit+)> */
		nil,
		/* 47 timeNumoffset <- <(('-' / '+') timeHour ':' timeMinute)> */
		nil,
		/* 48 timeOffset <- <('Z' / timeNumoffset)> */
		nil,
		/* 49 partialTime <- <(timeHour ':' timeMinute ':' timeSecond timeSecfrac?)> */
		nil,
		/* 50 fullDate <- <(dateFullYear '-' dateMonth '-' dateMDay)> */
		nil,
		/* 51 fullTime <- <(partialTime timeOffset)> */
		nil,
		/* 52 datetime <- <(fullDate 'T' fullTime)> */
		nil,
		/* 53 digit <- <[0-9]> */
		func() bool {
			position287, tokenIndex287, depth287 := position, tokenIndex, depth
			{
				position288 := position
				depth++
				if c := buffer[position]; c < rune('0') || c > rune('9') {
					goto l287
				}
				position++
				depth--
				add(ruledigit, position288)
			}
			return true
		l287:
			position, tokenIndex, depth = position287, tokenIndex287, depth287
			return false
		},
		/* 54 digitDual <- <(digit digit)> */
		func() bool {
			position289, tokenIndex289, depth289 := position, tokenIndex, depth
			{
				position290 := position
				depth++
				if !_rules[ruledigit]() {
					goto l289
				}
				if !_rules[ruledigit]() {
					goto l289
				}
				depth--
				add(ruledigitDual, position290)
			}
			return true
		l289:
			position, tokenIndex, depth = position289, tokenIndex289, depth289
			return false
		},
		/* 55 digitQuad <- <(digitDual digitDual)> */
		nil,
		/* 56 array <- <('[' Action22 wsnl arrayValues wsnl ']')> */
		nil,
		/* 57 arrayValues <- <(val Action23 arraySep? (comment? newline)?)*> */
		nil,
		/* 58 arraySep <- <(ws ',' wsnl)> */
		nil,
		/* 60 Action0 <- <{ _ = buffer }> */
		nil,
		nil,
		/* 62 Action1 <- <{ p.SetTableString(begin, end) }> */
		nil,
		/* 63 Action2 <- <{ p.AddLineCount(end - begin) }> */
		nil,
		/* 64 Action3 <- <{ p.AddLineCount(end - begin) }> */
		nil,
		/* 65 Action4 <- <{ p.AddKeyValue() }> */
		nil,
		/* 66 Action5 <- <{ p.SetKey(p.buffer, begin, end) }> */
		nil,
		/* 67 Action6 <- <{ p.SetKey(p.buffer, begin, end) }> */
		nil,
		/* 68 Action7 <- <{ p.SetTime(begin, end) }> */
		nil,
		/* 69 Action8 <- <{ p.SetFloat64(begin, end) }> */
		nil,
		/* 70 Action9 <- <{ p.SetInt64(begin, end) }> */
		nil,
		/* 71 Action10 <- <{ p.SetString(begin, end) }> */
		nil,
		/* 72 Action11 <- <{ p.SetBool(begin, end) }> */
		nil,
		/* 73 Action12 <- <{ p.SetArray(begin, end) }> */
		nil,
		/* 74 Action13 <- <{ p.SetTable(p.buffer, begin, end) }> */
		nil,
		/* 75 Action14 <- <{ p.SetArrayTable(p.buffer, begin, end) }> */
		nil,
		/* 76 Action15 <- <{ p.StartInlineTable() }> */
		nil,
		/* 77 Action16 <- <{ p.EndInlineTable() }> */
		nil,
		/* 78 Action17 <- <{ p.SetBasicString(p.buffer, begin, end) }> */
		nil,
		/* 79 Action18 <- <{ p.SetMultilineString() }> */
		nil,
		/* 80 Action19 <- <{ p.AddMultilineBasicBody(p.buffer, begin, end) }> */
		nil,
		/* 81 Action20 <- <{ p.SetLiteralString(p.buffer, begin, end) }> */
		nil,
		/* 82 Action21 <- <{ p.SetMultilineLiteralString(p.buffer, begin, end) }> */
		nil,
		/* 83 Action22 <- <{ p.StartArray() }> */
		nil,
		/* 84 Action23 <- <{ p.AddArrayVal() }> */
		nil,
	}
	p.rules = _rules
}
