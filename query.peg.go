package godless

import (
	"fmt"
	"math"
	"sort"
	"strconv"
)

const endSymbol rune = 1114112

/* The rule types inferred from the grammar are below. */
type pegRule uint8

const (
	ruleUnknown pegRule = iota
	ruleQuery
	ruleJoin
	ruleJoinKey
	ruleJoinRow
	ruleKeyJoin
	ruleValueJoin
	ruleSelect
	ruleSelectKey
	ruleLimit
	ruleWhere
	ruleWhereClause
	ruleAndClause
	ruleOrClause
	rulePredicateClause
	rulePredicate
	rulePredicateValue
	rulePredicateRowKey
	rulePredicateKey
	rulePredicateLiteralValue
	ruleLiteral
	rulePositiveInteger
	ruleKey
	ruleEscape
	ruleMustSpacing
	ruleSpacing
	ruleAction0
	ruleAction1
	rulePegText
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
)

var rul3s = [...]string{
	"Unknown",
	"Query",
	"Join",
	"JoinKey",
	"JoinRow",
	"KeyJoin",
	"ValueJoin",
	"Select",
	"SelectKey",
	"Limit",
	"Where",
	"WhereClause",
	"AndClause",
	"OrClause",
	"PredicateClause",
	"Predicate",
	"PredicateValue",
	"PredicateRowKey",
	"PredicateKey",
	"PredicateLiteralValue",
	"Literal",
	"PositiveInteger",
	"Key",
	"Escape",
	"MustSpacing",
	"Spacing",
	"Action0",
	"Action1",
	"PegText",
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
}

type token32 struct {
	pegRule
	begin, end uint32
}

func (t *token32) String() string {
	return fmt.Sprintf("\x1B[34m%v\x1B[m %v %v", rul3s[t.pegRule], t.begin, t.end)
}

type node32 struct {
	token32
	up, next *node32
}

func (node *node32) print(pretty bool, buffer string) {
	var print func(node *node32, depth int)
	print = func(node *node32, depth int) {
		for node != nil {
			for c := 0; c < depth; c++ {
				fmt.Printf(" ")
			}
			rule := rul3s[node.pegRule]
			quote := strconv.Quote(string(([]rune(buffer)[node.begin:node.end])))
			if !pretty {
				fmt.Printf("%v %v\n", rule, quote)
			} else {
				fmt.Printf("\x1B[34m%v\x1B[m %v\n", rule, quote)
			}
			if node.up != nil {
				print(node.up, depth+1)
			}
			node = node.next
		}
	}
	print(node, 0)
}

func (node *node32) Print(buffer string) {
	node.print(false, buffer)
}

func (node *node32) PrettyPrint(buffer string) {
	node.print(true, buffer)
}

type tokens32 struct {
	tree []token32
}

func (t *tokens32) Trim(length uint32) {
	t.tree = t.tree[:length]
}

func (t *tokens32) Print() {
	for _, token := range t.tree {
		fmt.Println(token.String())
	}
}

func (t *tokens32) AST() *node32 {
	type element struct {
		node *node32
		down *element
	}
	tokens := t.Tokens()
	var stack *element
	for _, token := range tokens {
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
	if stack != nil {
		return stack.node
	}
	return nil
}

func (t *tokens32) PrintSyntaxTree(buffer string) {
	t.AST().Print(buffer)
}

func (t *tokens32) PrettyPrintSyntaxTree(buffer string) {
	t.AST().PrettyPrint(buffer)
}

func (t *tokens32) Add(rule pegRule, begin, end, index uint32) {
	if tree := t.tree; int(index) >= len(tree) {
		expanded := make([]token32, 2*len(tree))
		copy(expanded, tree)
		t.tree = expanded
	}
	t.tree[index] = token32{
		pegRule: rule,
		begin:   begin,
		end:     end,
	}
}

func (t *tokens32) Tokens() []token32 {
	return t.tree
}

type QueryParser struct {
	QueryAST

	Buffer string
	buffer []rune
	rules  [45]func() bool
	parse  func(rule ...int) error
	reset  func()
	Pretty bool
	tokens32
}

func (p *QueryParser) Parse(rule ...int) error {
	return p.parse(rule...)
}

func (p *QueryParser) Reset() {
	p.reset()
}

type textPosition struct {
	line, symbol int
}

type textPositionMap map[int]textPosition

func translatePositions(buffer []rune, positions []int) textPositionMap {
	length, translations, j, line, symbol := len(positions), make(textPositionMap, len(positions)), 0, 1, 0
	sort.Ints(positions)

search:
	for i, c := range buffer {
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
	p   *QueryParser
	max token32
}

func (e *parseError) Error() string {
	tokens, error := []token32{e.max}, "\n"
	positions, p := make([]int, 2*len(tokens)), 0
	for _, token := range tokens {
		positions[p], p = int(token.begin), p+1
		positions[p], p = int(token.end), p+1
	}
	translations := translatePositions(e.p.buffer, positions)
	format := "parse error near %v (line %v symbol %v - line %v symbol %v):\n%v\n"
	if e.p.Pretty {
		format = "parse error near \x1B[34m%v\x1B[m (line %v symbol %v - line %v symbol %v):\n%v\n"
	}
	for _, token := range tokens {
		begin, end := int(token.begin), int(token.end)
		error += fmt.Sprintf(format,
			rul3s[token.pegRule],
			translations[begin].line, translations[begin].symbol,
			translations[end].line, translations[end].symbol,
			strconv.Quote(string(e.p.buffer[begin:end])))
	}

	return error
}

func (p *QueryParser) PrintSyntaxTree() {
	if p.Pretty {
		p.tokens32.PrettyPrintSyntaxTree(p.Buffer)
	} else {
		p.tokens32.PrintSyntaxTree(p.Buffer)
	}
}

func (p *QueryParser) Execute() {
	buffer, _buffer, text, begin, end := p.Buffer, p.buffer, "", 0, 0
	for _, token := range p.Tokens() {
		switch token.pegRule {

		case rulePegText:
			begin, end = int(token.begin), int(token.end)
			text = string(_buffer[begin:end])

		case ruleAction0:
			p.AddSelect()
		case ruleAction1:
			p.AddJoin()
		case ruleAction2:
			p.SetTableName(buffer[begin:end])
		case ruleAction3:
			p.AddJoinRow()
		case ruleAction4:
			p.SetJoinRowKey(buffer[begin:end])
		case ruleAction5:
			p.SetJoinKey(buffer[begin:end])
		case ruleAction6:
			p.SetJoinValue(buffer[begin:end])
		case ruleAction7:
			p.SetTableName(buffer[begin:end])
		case ruleAction8:
			p.SetLimit(buffer[begin:end])
		case ruleAction9:
			p.PushWhere()
		case ruleAction10:
			p.PopWhere()
		case ruleAction11:
			p.SetWhereCommand("and")
		case ruleAction12:
			p.SetWhereCommand("or")
		case ruleAction13:
			p.InitPredicate()
		case ruleAction14:
			p.SetPredicateCommand(buffer[begin:end])
		case ruleAction15:
			p.UsePredicateRowKey()
		case ruleAction16:
			p.AddPredicateKey(buffer[begin:end])
		case ruleAction17:
			p.AddPredicateLiteral(buffer[begin:end])

		}
	}
	_, _, _, _, _ = buffer, _buffer, text, begin, end
}

func (p *QueryParser) Init() {
	var (
		max                  token32
		position, tokenIndex uint32
		buffer               []rune
	)
	p.reset = func() {
		max = token32{}
		position, tokenIndex = 0, 0

		p.buffer = []rune(p.Buffer)
		if len(p.buffer) == 0 || p.buffer[len(p.buffer)-1] != endSymbol {
			p.buffer = append(p.buffer, endSymbol)
		}
		buffer = p.buffer
	}
	p.reset()

	_rules := p.rules
	tree := tokens32{tree: make([]token32, math.MaxInt16)}
	p.parse = func(rule ...int) error {
		r := 1
		if len(rule) > 0 {
			r = rule[0]
		}
		matches := p.rules[r]()
		p.tokens32 = tree
		if matches {
			p.Trim(tokenIndex)
			return nil
		}
		return &parseError{p, max}
	}

	add := func(rule pegRule, begin uint32) {
		tree.Add(rule, begin, position, tokenIndex)
		tokenIndex++
		if begin != position && position > max.end {
			max = token32{rule, begin, position}
		}
	}

	matchDot := func() bool {
		if buffer[position] != endSymbol {
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
		/* 0 Query <- <(Spacing ((Select Action0) / (Join Action1)) !.)> */
		func() bool {
			position0, tokenIndex0 := position, tokenIndex
			{
				position1 := position
				if !_rules[ruleSpacing]() {
					goto l0
				}
				{
					position2, tokenIndex2 := position, tokenIndex
					{
						position4 := position
						if buffer[position] != rune('s') {
							goto l3
						}
						position++
						if buffer[position] != rune('e') {
							goto l3
						}
						position++
						if buffer[position] != rune('l') {
							goto l3
						}
						position++
						if buffer[position] != rune('e') {
							goto l3
						}
						position++
						if buffer[position] != rune('c') {
							goto l3
						}
						position++
						if buffer[position] != rune('t') {
							goto l3
						}
						position++
						if !_rules[ruleMustSpacing]() {
							goto l3
						}
						{
							position5 := position
							{
								position6 := position
								if !_rules[ruleKey]() {
									goto l3
								}
								add(rulePegText, position6)
							}
							{
								add(ruleAction7, position)
							}
							add(ruleSelectKey, position5)
						}
						{
							position8, tokenIndex8 := position, tokenIndex
							if !_rules[ruleMustSpacing]() {
								goto l8
							}
							{
								position10 := position
								if buffer[position] != rune('w') {
									goto l8
								}
								position++
								if buffer[position] != rune('h') {
									goto l8
								}
								position++
								if buffer[position] != rune('e') {
									goto l8
								}
								position++
								if buffer[position] != rune('r') {
									goto l8
								}
								position++
								if buffer[position] != rune('e') {
									goto l8
								}
								position++
								if !_rules[ruleMustSpacing]() {
									goto l8
								}
								if !_rules[ruleWhereClause]() {
									goto l8
								}
								add(ruleWhere, position10)
							}
							goto l9
						l8:
							position, tokenIndex = position8, tokenIndex8
						}
					l9:
						{
							position11, tokenIndex11 := position, tokenIndex
							if !_rules[ruleMustSpacing]() {
								goto l11
							}
							{
								position13 := position
								if buffer[position] != rune('l') {
									goto l11
								}
								position++
								if buffer[position] != rune('i') {
									goto l11
								}
								position++
								if buffer[position] != rune('m') {
									goto l11
								}
								position++
								if buffer[position] != rune('i') {
									goto l11
								}
								position++
								if buffer[position] != rune('t') {
									goto l11
								}
								position++
								if !_rules[ruleMustSpacing]() {
									goto l11
								}
								{
									position14 := position
									{
										position15 := position
										if c := buffer[position]; c < rune('1') || c > rune('9') {
											goto l11
										}
										position++
									l16:
										{
											position17, tokenIndex17 := position, tokenIndex
											if c := buffer[position]; c < rune('0') || c > rune('9') {
												goto l17
											}
											position++
											goto l16
										l17:
											position, tokenIndex = position17, tokenIndex17
										}
										add(rulePositiveInteger, position15)
									}
									add(rulePegText, position14)
								}
								{
									add(ruleAction8, position)
								}
								add(ruleLimit, position13)
							}
							goto l12
						l11:
							position, tokenIndex = position11, tokenIndex11
						}
					l12:
						add(ruleSelect, position4)
					}
					{
						add(ruleAction0, position)
					}
					goto l2
				l3:
					position, tokenIndex = position2, tokenIndex2
					{
						position20 := position
						if buffer[position] != rune('j') {
							goto l0
						}
						position++
						if buffer[position] != rune('o') {
							goto l0
						}
						position++
						if buffer[position] != rune('i') {
							goto l0
						}
						position++
						if buffer[position] != rune('n') {
							goto l0
						}
						position++
						if !_rules[ruleMustSpacing]() {
							goto l0
						}
						{
							position21 := position
							{
								position22 := position
								if !_rules[ruleKey]() {
									goto l0
								}
								add(rulePegText, position22)
							}
							{
								add(ruleAction2, position)
							}
							add(ruleJoinKey, position21)
						}
						if !_rules[ruleMustSpacing]() {
							goto l0
						}
						if buffer[position] != rune('r') {
							goto l0
						}
						position++
						if buffer[position] != rune('o') {
							goto l0
						}
						position++
						if buffer[position] != rune('w') {
							goto l0
						}
						position++
						if buffer[position] != rune('s') {
							goto l0
						}
						position++
						if !_rules[ruleMustSpacing]() {
							goto l0
						}
						if !_rules[ruleJoinRow]() {
							goto l0
						}
					l24:
						{
							position25, tokenIndex25 := position, tokenIndex
							if !_rules[ruleSpacing]() {
								goto l25
							}
							if !_rules[ruleJoinRow]() {
								goto l25
							}
							goto l24
						l25:
							position, tokenIndex = position25, tokenIndex25
						}
						if !_rules[ruleSpacing]() {
							goto l0
						}
						add(ruleJoin, position20)
					}
					{
						add(ruleAction1, position)
					}
				}
			l2:
				{
					position27, tokenIndex27 := position, tokenIndex
					if !matchDot() {
						goto l27
					}
					goto l0
				l27:
					position, tokenIndex = position27, tokenIndex27
				}
				add(ruleQuery, position1)
			}
			return true
		l0:
			position, tokenIndex = position0, tokenIndex0
			return false
		},
		/* 1 Join <- <('j' 'o' 'i' 'n' MustSpacing JoinKey MustSpacing ('r' 'o' 'w' 's') MustSpacing JoinRow (Spacing JoinRow)* Spacing)> */
		nil,
		/* 2 JoinKey <- <(<Key> Action2)> */
		nil,
		/* 3 JoinRow <- <(Action3 '(' Spacing KeyJoin Spacing (',' Spacing ValueJoin Spacing)* ')')> */
		func() bool {
			position30, tokenIndex30 := position, tokenIndex
			{
				position31 := position
				{
					add(ruleAction3, position)
				}
				if buffer[position] != rune('(') {
					goto l30
				}
				position++
				if !_rules[ruleSpacing]() {
					goto l30
				}
				{
					position33 := position
					if buffer[position] != rune('@') {
						goto l30
					}
					position++
					if buffer[position] != rune('k') {
						goto l30
					}
					position++
					if buffer[position] != rune('e') {
						goto l30
					}
					position++
					if buffer[position] != rune('y') {
						goto l30
					}
					position++
					if !_rules[ruleSpacing]() {
						goto l30
					}
					if buffer[position] != rune('=') {
						goto l30
					}
					position++
					if !_rules[ruleSpacing]() {
						goto l30
					}
					{
						position34 := position
						if !_rules[ruleKey]() {
							goto l30
						}
						add(rulePegText, position34)
					}
					{
						add(ruleAction4, position)
					}
					add(ruleKeyJoin, position33)
				}
				if !_rules[ruleSpacing]() {
					goto l30
				}
			l36:
				{
					position37, tokenIndex37 := position, tokenIndex
					if buffer[position] != rune(',') {
						goto l37
					}
					position++
					if !_rules[ruleSpacing]() {
						goto l37
					}
					{
						position38 := position
						{
							position39 := position
							if !_rules[ruleKey]() {
								goto l37
							}
							add(rulePegText, position39)
						}
						{
							add(ruleAction5, position)
						}
						if !_rules[ruleSpacing]() {
							goto l37
						}
						if buffer[position] != rune('=') {
							goto l37
						}
						position++
						if !_rules[ruleSpacing]() {
							goto l37
						}
						if buffer[position] != rune('\'') {
							goto l37
						}
						position++
						{
							position41 := position
							if !_rules[ruleLiteral]() {
								goto l37
							}
							add(rulePegText, position41)
						}
						if buffer[position] != rune('\'') {
							goto l37
						}
						position++
						{
							add(ruleAction6, position)
						}
						add(ruleValueJoin, position38)
					}
					if !_rules[ruleSpacing]() {
						goto l37
					}
					goto l36
				l37:
					position, tokenIndex = position37, tokenIndex37
				}
				if buffer[position] != rune(')') {
					goto l30
				}
				position++
				add(ruleJoinRow, position31)
			}
			return true
		l30:
			position, tokenIndex = position30, tokenIndex30
			return false
		},
		/* 4 KeyJoin <- <('@' 'k' 'e' 'y' Spacing '=' Spacing <Key> Action4)> */
		nil,
		/* 5 ValueJoin <- <(<Key> Action5 Spacing '=' Spacing '\'' <Literal> '\'' Action6)> */
		nil,
		/* 6 Select <- <('s' 'e' 'l' 'e' 'c' 't' MustSpacing SelectKey (MustSpacing Where)? (MustSpacing Limit)?)> */
		nil,
		/* 7 SelectKey <- <(<Key> Action7)> */
		nil,
		/* 8 Limit <- <('l' 'i' 'm' 'i' 't' MustSpacing <PositiveInteger> Action8)> */
		nil,
		/* 9 Where <- <('w' 'h' 'e' 'r' 'e' MustSpacing WhereClause)> */
		nil,
		/* 10 WhereClause <- <(Action9 ((&('s') PredicateClause) | (&('o') OrClause) | (&('a') AndClause)) Action10)> */
		func() bool {
			position49, tokenIndex49 := position, tokenIndex
			{
				position50 := position
				{
					add(ruleAction9, position)
				}
				{
					switch buffer[position] {
					case 's':
						{
							position53 := position
							{
								add(ruleAction13, position)
							}
							{
								position55 := position
								{
									position56 := position
									{
										position57, tokenIndex57 := position, tokenIndex
										if buffer[position] != rune('s') {
											goto l58
										}
										position++
										if buffer[position] != rune('t') {
											goto l58
										}
										position++
										if buffer[position] != rune('r') {
											goto l58
										}
										position++
										if buffer[position] != rune('_') {
											goto l58
										}
										position++
										if buffer[position] != rune('e') {
											goto l58
										}
										position++
										if buffer[position] != rune('q') {
											goto l58
										}
										position++
										goto l57
									l58:
										position, tokenIndex = position57, tokenIndex57
										if buffer[position] != rune('s') {
											goto l49
										}
										position++
										if buffer[position] != rune('t') {
											goto l49
										}
										position++
										if buffer[position] != rune('r') {
											goto l49
										}
										position++
										if buffer[position] != rune('_') {
											goto l49
										}
										position++
										if buffer[position] != rune('n') {
											goto l49
										}
										position++
										if buffer[position] != rune('e') {
											goto l49
										}
										position++
										if buffer[position] != rune('q') {
											goto l49
										}
										position++
									}
								l57:
									add(rulePegText, position56)
								}
								{
									add(ruleAction14, position)
								}
								add(rulePredicate, position55)
							}
							if !_rules[ruleSpacing]() {
								goto l49
							}
							if buffer[position] != rune('(') {
								goto l49
							}
							position++
							if !_rules[ruleSpacing]() {
								goto l49
							}
							if !_rules[rulePredicateValue]() {
								goto l49
							}
						l60:
							{
								position61, tokenIndex61 := position, tokenIndex
								if buffer[position] != rune(',') {
									goto l61
								}
								position++
								if !_rules[ruleSpacing]() {
									goto l61
								}
								if !_rules[rulePredicateValue]() {
									goto l61
								}
								if !_rules[ruleSpacing]() {
									goto l61
								}
								goto l60
							l61:
								position, tokenIndex = position61, tokenIndex61
							}
							if buffer[position] != rune(')') {
								goto l49
							}
							position++
							add(rulePredicateClause, position53)
						}
						break
					case 'o':
						{
							position62 := position
							if buffer[position] != rune('o') {
								goto l49
							}
							position++
							if buffer[position] != rune('r') {
								goto l49
							}
							position++
							{
								add(ruleAction12, position)
							}
							if !_rules[ruleSpacing]() {
								goto l49
							}
							if buffer[position] != rune('(') {
								goto l49
							}
							position++
							if !_rules[ruleSpacing]() {
								goto l49
							}
							if !_rules[ruleWhereClause]() {
								goto l49
							}
							if !_rules[ruleSpacing]() {
								goto l49
							}
						l64:
							{
								position65, tokenIndex65 := position, tokenIndex
								if buffer[position] != rune(',') {
									goto l65
								}
								position++
								if !_rules[ruleSpacing]() {
									goto l65
								}
								if !_rules[ruleWhereClause]() {
									goto l65
								}
								if !_rules[ruleSpacing]() {
									goto l65
								}
								goto l64
							l65:
								position, tokenIndex = position65, tokenIndex65
							}
							if buffer[position] != rune(')') {
								goto l49
							}
							position++
							add(ruleOrClause, position62)
						}
						break
					default:
						{
							position66 := position
							if buffer[position] != rune('a') {
								goto l49
							}
							position++
							if buffer[position] != rune('n') {
								goto l49
							}
							position++
							if buffer[position] != rune('d') {
								goto l49
							}
							position++
							{
								add(ruleAction11, position)
							}
							if !_rules[ruleSpacing]() {
								goto l49
							}
							if buffer[position] != rune('(') {
								goto l49
							}
							position++
							if !_rules[ruleSpacing]() {
								goto l49
							}
							if !_rules[ruleWhereClause]() {
								goto l49
							}
							if !_rules[ruleSpacing]() {
								goto l49
							}
						l68:
							{
								position69, tokenIndex69 := position, tokenIndex
								if buffer[position] != rune(',') {
									goto l69
								}
								position++
								if !_rules[ruleSpacing]() {
									goto l69
								}
								if !_rules[ruleWhereClause]() {
									goto l69
								}
								if !_rules[ruleSpacing]() {
									goto l69
								}
								goto l68
							l69:
								position, tokenIndex = position69, tokenIndex69
							}
							if buffer[position] != rune(')') {
								goto l49
							}
							position++
							add(ruleAndClause, position66)
						}
						break
					}
				}

				{
					add(ruleAction10, position)
				}
				add(ruleWhereClause, position50)
			}
			return true
		l49:
			position, tokenIndex = position49, tokenIndex49
			return false
		},
		/* 11 AndClause <- <('a' 'n' 'd' Action11 Spacing '(' Spacing WhereClause Spacing (',' Spacing WhereClause Spacing)* ')')> */
		nil,
		/* 12 OrClause <- <('o' 'r' Action12 Spacing '(' Spacing WhereClause Spacing (',' Spacing WhereClause Spacing)* ')')> */
		nil,
		/* 13 PredicateClause <- <(Action13 Predicate Spacing '(' Spacing PredicateValue (',' Spacing PredicateValue Spacing)* ')')> */
		nil,
		/* 14 Predicate <- <(<(('s' 't' 'r' '_' 'e' 'q') / ('s' 't' 'r' '_' 'n' 'e' 'q'))> Action14)> */
		nil,
		/* 15 PredicateValue <- <((&('\'') PredicateLiteralValue) | (&('@') PredicateRowKey) | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9' | 'A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z' | '\\' | 'a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') PredicateKey))> */
		func() bool {
			position75, tokenIndex75 := position, tokenIndex
			{
				position76 := position
				{
					switch buffer[position] {
					case '\'':
						{
							position78 := position
							if buffer[position] != rune('\'') {
								goto l75
							}
							position++
							{
								position79 := position
								if !_rules[ruleLiteral]() {
									goto l75
								}
								add(rulePegText, position79)
							}
							if buffer[position] != rune('\'') {
								goto l75
							}
							position++
							{
								add(ruleAction17, position)
							}
							add(rulePredicateLiteralValue, position78)
						}
						break
					case '@':
						{
							position81 := position
							if buffer[position] != rune('@') {
								goto l75
							}
							position++
							if buffer[position] != rune('k') {
								goto l75
							}
							position++
							if buffer[position] != rune('e') {
								goto l75
							}
							position++
							if buffer[position] != rune('y') {
								goto l75
							}
							position++
							{
								add(ruleAction15, position)
							}
							add(rulePredicateRowKey, position81)
						}
						break
					default:
						{
							position83 := position
							{
								position84 := position
								if !_rules[ruleKey]() {
									goto l75
								}
								add(rulePegText, position84)
							}
							{
								add(ruleAction16, position)
							}
							add(rulePredicateKey, position83)
						}
						break
					}
				}

				add(rulePredicateValue, position76)
			}
			return true
		l75:
			position, tokenIndex = position75, tokenIndex75
			return false
		},
		/* 16 PredicateRowKey <- <('@' 'k' 'e' 'y' Action15)> */
		nil,
		/* 17 PredicateKey <- <(<Key> Action16)> */
		nil,
		/* 18 PredicateLiteralValue <- <('\'' <Literal> '\'' Action17)> */
		nil,
		/* 19 Literal <- <(Escape / (!'\'' .))*> */
		func() bool {
			{
				position90 := position
			l91:
				{
					position92, tokenIndex92 := position, tokenIndex
					{
						position93, tokenIndex93 := position, tokenIndex
						if !_rules[ruleEscape]() {
							goto l94
						}
						goto l93
					l94:
						position, tokenIndex = position93, tokenIndex93
						{
							position95, tokenIndex95 := position, tokenIndex
							if buffer[position] != rune('\'') {
								goto l95
							}
							position++
							goto l92
						l95:
							position, tokenIndex = position95, tokenIndex95
						}
						if !matchDot() {
							goto l92
						}
					}
				l93:
					goto l91
				l92:
					position, tokenIndex = position92, tokenIndex92
				}
				add(ruleLiteral, position90)
			}
			return true
		},
		/* 20 PositiveInteger <- <([1-9] [0-9]*)> */
		nil,
		/* 21 Key <- <(Escape / ((&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]) | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z])))+> */
		func() bool {
			position97, tokenIndex97 := position, tokenIndex
			{
				position98 := position
				{
					position101, tokenIndex101 := position, tokenIndex
					if !_rules[ruleEscape]() {
						goto l102
					}
					goto l101
				l102:
					position, tokenIndex = position101, tokenIndex101
					{
						switch buffer[position] {
						case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l97
							}
							position++
							break
						case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l97
							}
							position++
							break
						default:
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l97
							}
							position++
							break
						}
					}

				}
			l101:
			l99:
				{
					position100, tokenIndex100 := position, tokenIndex
					{
						position104, tokenIndex104 := position, tokenIndex
						if !_rules[ruleEscape]() {
							goto l105
						}
						goto l104
					l105:
						position, tokenIndex = position104, tokenIndex104
						{
							switch buffer[position] {
							case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
								if c := buffer[position]; c < rune('0') || c > rune('9') {
									goto l100
								}
								position++
								break
							case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
								if c := buffer[position]; c < rune('A') || c > rune('Z') {
									goto l100
								}
								position++
								break
							default:
								if c := buffer[position]; c < rune('a') || c > rune('z') {
									goto l100
								}
								position++
								break
							}
						}

					}
				l104:
					goto l99
				l100:
					position, tokenIndex = position100, tokenIndex100
				}
				add(ruleKey, position98)
			}
			return true
		l97:
			position, tokenIndex = position97, tokenIndex97
			return false
		},
		/* 22 Escape <- <('\\' ((&('v') 'v') | (&('t') 't') | (&('r') 'r') | (&('n') 'n') | (&('f') 'f') | (&('b') 'b') | (&('a') 'a') | (&('\\') '\\') | (&('?') '?') | (&('"') '"') | (&('\'') '\'')))> */
		func() bool {
			position107, tokenIndex107 := position, tokenIndex
			{
				position108 := position
				if buffer[position] != rune('\\') {
					goto l107
				}
				position++
				{
					switch buffer[position] {
					case 'v':
						if buffer[position] != rune('v') {
							goto l107
						}
						position++
						break
					case 't':
						if buffer[position] != rune('t') {
							goto l107
						}
						position++
						break
					case 'r':
						if buffer[position] != rune('r') {
							goto l107
						}
						position++
						break
					case 'n':
						if buffer[position] != rune('n') {
							goto l107
						}
						position++
						break
					case 'f':
						if buffer[position] != rune('f') {
							goto l107
						}
						position++
						break
					case 'b':
						if buffer[position] != rune('b') {
							goto l107
						}
						position++
						break
					case 'a':
						if buffer[position] != rune('a') {
							goto l107
						}
						position++
						break
					case '\\':
						if buffer[position] != rune('\\') {
							goto l107
						}
						position++
						break
					case '?':
						if buffer[position] != rune('?') {
							goto l107
						}
						position++
						break
					case '"':
						if buffer[position] != rune('"') {
							goto l107
						}
						position++
						break
					default:
						if buffer[position] != rune('\'') {
							goto l107
						}
						position++
						break
					}
				}

				add(ruleEscape, position108)
			}
			return true
		l107:
			position, tokenIndex = position107, tokenIndex107
			return false
		},
		/* 23 MustSpacing <- <((&('\n') '\n') | (&('\t') '\t') | (&(' ') ' '))+> */
		func() bool {
			position110, tokenIndex110 := position, tokenIndex
			{
				position111 := position
				{
					switch buffer[position] {
					case '\n':
						if buffer[position] != rune('\n') {
							goto l110
						}
						position++
						break
					case '\t':
						if buffer[position] != rune('\t') {
							goto l110
						}
						position++
						break
					default:
						if buffer[position] != rune(' ') {
							goto l110
						}
						position++
						break
					}
				}

			l112:
				{
					position113, tokenIndex113 := position, tokenIndex
					{
						switch buffer[position] {
						case '\n':
							if buffer[position] != rune('\n') {
								goto l113
							}
							position++
							break
						case '\t':
							if buffer[position] != rune('\t') {
								goto l113
							}
							position++
							break
						default:
							if buffer[position] != rune(' ') {
								goto l113
							}
							position++
							break
						}
					}

					goto l112
				l113:
					position, tokenIndex = position113, tokenIndex113
				}
				add(ruleMustSpacing, position111)
			}
			return true
		l110:
			position, tokenIndex = position110, tokenIndex110
			return false
		},
		/* 24 Spacing <- <((&('\n') '\n') | (&('\t') '\t') | (&(' ') ' '))*> */
		func() bool {
			{
				position117 := position
			l118:
				{
					position119, tokenIndex119 := position, tokenIndex
					{
						switch buffer[position] {
						case '\n':
							if buffer[position] != rune('\n') {
								goto l119
							}
							position++
							break
						case '\t':
							if buffer[position] != rune('\t') {
								goto l119
							}
							position++
							break
						default:
							if buffer[position] != rune(' ') {
								goto l119
							}
							position++
							break
						}
					}

					goto l118
				l119:
					position, tokenIndex = position119, tokenIndex119
				}
				add(ruleSpacing, position117)
			}
			return true
		},
		/* 26 Action0 <- <{ p.AddSelect() }> */
		nil,
		/* 27 Action1 <- <{ p.AddJoin() }> */
		nil,
		nil,
		/* 29 Action2 <- <{ p.SetTableName(buffer[begin:end]) }> */
		nil,
		/* 30 Action3 <- <{ p.AddJoinRow() }> */
		nil,
		/* 31 Action4 <- <{ p.SetJoinRowKey(buffer[begin:end]) }> */
		nil,
		/* 32 Action5 <- <{ p.SetJoinKey(buffer[begin:end]) }> */
		nil,
		/* 33 Action6 <- <{ p.SetJoinValue(buffer[begin:end]) }> */
		nil,
		/* 34 Action7 <- <{ p.SetTableName(buffer[begin:end]) }> */
		nil,
		/* 35 Action8 <- <{ p.SetLimit(buffer[begin:end])}> */
		nil,
		/* 36 Action9 <- <{ p.PushWhere() }> */
		nil,
		/* 37 Action10 <- <{ p.PopWhere() }> */
		nil,
		/* 38 Action11 <- <{ p.SetWhereCommand("and") }> */
		nil,
		/* 39 Action12 <- <{ p.SetWhereCommand("or") }> */
		nil,
		/* 40 Action13 <- <{ p.InitPredicate() }> */
		nil,
		/* 41 Action14 <- <{ p.SetPredicateCommand(buffer[begin:end]) }> */
		nil,
		/* 42 Action15 <- <{ p.UsePredicateRowKey() }> */
		nil,
		/* 43 Action16 <- <{ p.AddPredicateKey(buffer[begin:end]) }> */
		nil,
		/* 44 Action17 <- <{ p.AddPredicateLiteral(buffer[begin:end])}> */
		nil,
	}
	p.rules = _rules
}