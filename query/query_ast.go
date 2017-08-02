package query

import (
	"fmt"
	"strconv"

	"github.com/johnny-morrice/godless/crdt"
	"github.com/johnny-morrice/godless/crypto"
	"github.com/pkg/errors"
)

type QueryAST struct {
	Command    string
	TableKey   astVariable
	Select     QuerySelectAST `json:",omitempty"`
	Join       QueryJoinAST   `json:",omitempty"`
	PublicKeys []astVariable

	WhereStack     []*QueryWhereAST
	lastRowJoinKey astVariable
	lastRowJoin    *QueryRowJoinAST
}

type placeholderValidator interface {
	isValid(value interface{}) bool
}

type astVariable struct {
	validator     placeholderValidator
	position      int
	isPlaceholder bool
	isKey         bool
	text          string
}

func astLiteral(text string) astVariable {
	return astVariable{
		text: text,
	}
}

func astKey(text string) astVariable {
	return astVariable{
		text:  text,
		isKey: true,
	}
}

func astKeyPlaceholder(position int) astVariable {
	return astVariable{
		validator:     isAstString{},
		isKey:         true,
		isPlaceholder: true,
		position:      position,
	}
}

func astLiteralPlaceholder(position int) astVariable {
	return astVariable{
		validator:     isAstString{},
		isPlaceholder: true,
		position:      position,
	}
}

func astIntegerPlaceholder(position int) astVariable {
	return astVariable{
		validator:     isAstInt{},
		isPlaceholder: true,
		position:      position,
	}
}

type isAstString struct {
}

func (isAstString) isValid(val interface{}) bool {
	_, ok := val.(string)
	return ok
}

type isAstInt struct {
}

func (isAstInt) isValid(val interface{}) bool {
	_, ok := val.(int)
	return ok
}

func (ast *QueryAST) SetTableNamePlaceholder(begin int) {
	panic("not implemented")
}

func (ast *QueryAST) SetJoinRowKeyPlaceholder(begin int) {
	panic("not implemented")
}

func (ast *QueryAST) SetJoinValuePlaceholder(begin int) {
	panic("not implemented")
}

func (ast *QueryAST) SetJoinKeyPlaceholder(begin int) {
	panic("not implemented")
}

func (ast *QueryAST) SetLimitPlaceholder(begin int) {
	panic("not implemented")
}

func (ast *QueryAST) AddPredicateKeyPlaceholder(begin int) {
	panic("not implemented")
}

func (ast *QueryAST) AddPredicateLiteralPlaceholder(begin int) {
	panic("not implemented")
}

func (ast *QueryAST) AddCryptoKey(publicKey string) {
	ast.PublicKeys = append(ast.PublicKeys, astLiteral(publicKey))
}

func (ast *QueryAST) AddJoin() {
	ast.Command = "join"
}

func (ast *QueryAST) AddJoinRow() {
	row := &QueryRowJoinAST{}
	ast.Join.Rows = append(ast.Join.Rows, row)
	ast.lastRowJoin = row
}

func (ast *QueryAST) SetJoinRowKey(key string) {
	ast.lastRowJoin.RowKey = astKey(key)
}

func (ast *QueryAST) SetJoinKey(key string) {
	ast.lastRowJoinKey = astKey(key)
}

func (ast *QueryAST) SetJoinValue(value string) {
	joinValue := QueryRowJoinValueAST{
		Key:   ast.lastRowJoinKey,
		Value: astLiteral(value),
	}
	ast.lastRowJoin.Values = append(ast.lastRowJoin.Values, joinValue)
}

func (ast *QueryAST) PushWhere() {
	where := &QueryWhereAST{}

	if len(ast.WhereStack) == 0 {
		if ast.Select.Where != nil {
			panic("BUG invalid stack")
		}

		ast.Select.Where = where
	} else {
		lastWhere := ast.peekWhere()
		lastWhere.Clauses = append(lastWhere.Clauses, where)
	}

	ast.WhereStack = append(ast.WhereStack, where)
}

func (ast *QueryAST) PopWhere() {
	ast.WhereStack = ast.WhereStack[:len(ast.WhereStack)-1]
}

func (ast *QueryAST) SetWhereCommand(command string) {
	lastWhere := ast.peekWhere()
	lastWhere.Command = command
}

func (ast *QueryAST) peekWhere() *QueryWhereAST {
	if len(ast.WhereStack) == 0 {
		panic("BUG where stack empty!")
	}

	return ast.WhereStack[len(ast.WhereStack)-1]
}

func (ast *QueryAST) InitPredicate() {
	where := ast.peekWhere()
	where.Command = "predicate"
	where.Predicate = &QueryPredicateAST{}
}

func (ast *QueryAST) UsePredicateRowKey() {
	where := ast.peekWhere()
	where.Predicate.IncludeRowKey = true
}

func (ast *QueryAST) AddPredicateKey(key string) {
	where := ast.peekWhere()
	variable := astKey(key)
	where.Predicate.Values = append(where.Predicate.Values, variable)
}

func (ast *QueryAST) AddPredicateLiteral(literal string) {
	where := ast.peekWhere()
	variable := astLiteral(literal)
	where.Predicate.Values = append(where.Predicate.Values, variable)
}

func (ast *QueryAST) SetPredicateCommand(command string) {
	where := ast.peekWhere()
	where.Predicate.Command = command
}

func (ast *QueryAST) AddSelect() {
	ast.Command = "select"
}

func (ast *QueryAST) SetTableName(key string) {
	ast.TableKey = astKey(key)
}

func (ast *QueryAST) SetLimit(limit string) {
	ast.Select.Limit = limit
}

func (ast *QueryAST) Compile() (*Query, error) {
	query := &Query{}

	switch ast.Command {
	case "select":
		qselect, err := ast.Select.Compile()

		if err != nil {
			return nil, errors.Wrap(err, "BUG select compile failed")
		}

		query.OpCode = SELECT
		query.Select = qselect
	case "join":
		qjoin, err := ast.Join.Compile()

		if err != nil {
			return nil, errors.Wrap(err, "BUG join compile failed")
		}

		query.OpCode = JOIN
		query.Join = qjoin
	default:
		return nil, fmt.Errorf("BUG no command matching '%v'", ast.Command)
	}

	query.AST = ast
	query.TableKey = crdt.TableName(ast.TableKey.text)
	query.PublicKeys = make([]crypto.PublicKeyHash, len(ast.PublicKeys))

	for i, k := range ast.PublicKeys {
		query.PublicKeys[i] = crypto.PublicKeyHash(k.text)
	}

	return query, nil
}

type QueryJoinAST struct {
	Rows []*QueryRowJoinAST `json:",omitempty"`
}

func (ast *QueryJoinAST) Compile() (QueryJoin, error) {
	rows := make([]QueryRowJoin, len(ast.Rows))

	for i, r := range ast.Rows {
		rowJoin := QueryRowJoin{
			RowKey:  crdt.RowName(r.RowKey.text),
			Entries: map[crdt.EntryName]crdt.PointText{},
		}

		for _, val := range r.Values {
			plain, err := unquote(val.Value.text)

			if err != nil {
				return QueryJoin{}, errors.Wrap(err, "Error compiling join")
			}

			entry := crdt.EntryName(val.Key.text)
			point := crdt.PointText(plain)
			rowJoin.Entries[entry] = point
		}

		rows[i] = rowJoin
	}

	qjoin := QueryJoin{
		Rows: rows,
	}

	return qjoin, nil
}

type QueryRowJoinAST struct {
	RowKey astVariable
	Values []QueryRowJoinValueAST `json:",omitempty"`
}

type QueryRowJoinValueAST struct {
	Key   astVariable
	Value astVariable
}

type QuerySelectAST struct {
	Where *QueryWhereAST `json:",omitempty"`
	Limit string
}

func (ast *QuerySelectAST) Compile() (QuerySelect, error) {
	qselect := QuerySelect{}

	if ast.Limit != "" {
		limit, converr := strconv.ParseUint(ast.Limit, __BASE_10, __BITS_32)

		if converr != nil {
			return QuerySelect{}, errors.Wrap(converr, "BUG convert limit failed")
		}

		qselect.Limit = uint32(limit)
	}

	if ast.Where != nil {
		where, err := ast.Where.Compile()

		if err != nil {
			return QuerySelect{}, errors.Wrap(err, "BUG where clause compile failed")
		}

		qselect.Where = where
	}

	return qselect, nil
}

type QueryWhereAST struct {
	Command   string
	Clauses   []*QueryWhereAST   `json:",omitempty"`
	Predicate *QueryPredicateAST `json:",omitempty"`
}

func (ast *QueryWhereAST) Compile() (QueryWhere, error) {
	where := QueryWhere{}

	err := ast.Configure(&where)

	if err == nil {
		err = ast.CompileClauses(&where)
	}

	if err != nil {
		return QueryWhere{}, errors.Wrap(err, "QueryWhereAST.Compile failed")
	}

	return where, nil
}

func (ast *QueryWhereAST) Configure(where *QueryWhere) error {
	if ast.Command == "and" {
		where.OpCode = AND
	} else if ast.Command == "or" {
		where.OpCode = OR
	} else if ast.Command == "predicate" && ast.Predicate != nil {
		predicate, err := ast.Predicate.Compile()

		if err != nil {
			return errors.Wrap(err, "QueryWhereAST.Configure failed")
		}

		where.OpCode = PREDICATE
		where.Predicate = predicate
	} else {
		return fmt.Errorf("BUG unsupported where OpCode: '%v'", ast.Command)
	}

	return nil
}

func (ast *QueryWhereAST) CompileClauses(out *QueryWhere) error {
	astStack := []*QueryWhereAST{ast}
	WhereStack := []*QueryWhere{out}

	for len(astStack) > 0 {
		size := len(astStack)
		last := size - 1
		astTip := astStack[last]
		whereTip := WhereStack[last]

		astStack = astStack[:last]
		WhereStack = WhereStack[:last]

		childSize := len(astTip.Clauses)
		whereTip.Clauses = make([]QueryWhere, childSize)
		for i := 0; i < childSize; i++ {
			astChild := astTip.Clauses[i]
			whereChild := &whereTip.Clauses[i]
			err := astChild.Configure(whereChild)

			if err != nil {
				return errors.Wrap(err, "QueryWhereAST.CompileClauses failed")
			}

			astStack = append(astStack, astChild)
			WhereStack = append(WhereStack, whereChild)
		}
	}

	return nil
}

type QueryPredicateAST struct {
	Command       string
	Values        []astVariable
	IncludeRowKey bool
}

func (ast *QueryPredicateAST) Compile() (QueryPredicate, error) {
	predicate := QueryPredicate{}

	predicate.FunctionName = ast.Command

	// FIXME implement.
	// literals, err := unquoteAll(ast.Literals)
	//
	// if err != nil {
	// 	return QueryPredicate{}, errors.Wrap(err, "Error compiling predicate")
	// }
	//
	// pointLiterals := make([]crdt.PointText, len(literals))
	//
	// for i, lit := range ast.Literals {
	// 	pointLiterals[i] = crdt.PointText(lit)
	// }

	// predicate.Keys = makeEntryNames(ast.Keys)
	// predicate.Literals = pointLiterals
	predicate.IncludeRowKey = ast.IncludeRowKey

	return predicate, nil
}

func unquoteAllVars(vars []astVariable) ([]string, error) {
	text := make([]string, len(vars))

	for i, x := range vars {
		text[i] = x.text
	}

	return unquoteAll(text)
}

func unquoteAll(values []string) ([]string, error) {
	quoted := make([]string, len(values))

	for i, v := range values {
		q, err := unquote(v)

		if err != nil {
			return nil, err
		}

		quoted[i] = q
	}

	return quoted, nil
}

func unquoteMap(values map[string]string) (map[string]string, error) {
	quoted := map[string]string{}

	for k, v := range values {
		q, err := unquote(v)

		if err != nil {
			return nil, err
		}

		quoted[k] = q
	}

	return quoted, nil
}

func unquote(token string) (string, error) {
	token = fmt.Sprintf("\"%s\"", token)
	unquoted, err := strconv.Unquote(token)

	if err != nil {
		return "", errors.Wrap(err, fmt.Sprintf("Invalid string escape: '%s'", token))
	}

	return unquoted, nil
}

func quote(token string) string {
	token = strconv.Quote(token)
	return token[1 : len(token)-1]
}

func makeEntryNames(mess []string) []crdt.EntryName {
	es := make([]crdt.EntryName, len(mess))

	for i, s := range mess {
		es[i] = crdt.EntryName(s)
	}

	return es
}

const __BASE_10 = 10
const __BITS_32 = 64
