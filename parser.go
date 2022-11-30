package json

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
)

/*
This table-driven parser is a port of Doug Crockford's json-c
It has been modified and expanded in the following ways:
- whole thing is ported to Go
- grammar expanded to include primitives (null, number, string, boolean) as top-level values
- include a primitive int type
- parses a value rather than merely checking validity
- unit tested
*/

const (
	// Can only handle this many nested arrays and object.
	// If you data is deeper than this, you have bigger problems
	// than the parser failing.
	depth = 1024
)

// The different input categories that provide the "columns" of the state transition table.
type charClass int8

// Character classes
const (
	charSpace charClass = iota // space
	charLF___                  // Line feed
	charWhite                  // other whitespace
	charLCurB                  // {
	charRCurB                  // }
	charLSqrB                  // [
	charRSqrB                  // ]
	charColon                  // :
	charComma                  // ,
	charQuote                  // "
	charBacks                  // \
	charSlash                  // /
	charStar_                  // *
	charPlus_                  // +
	charMinus                  // -
	charPoint                  // .
	charZero_                  // 0
	charDigit                  // 123456789
	charLow_A                  // a
	charLow_B                  // b
	charLow_C                  // c
	charLow_D                  // d
	charLow_E                  // e
	charLow_F                  // f
	charLow_L                  // l
	charLow_N                  // n
	charLow_R                  // r
	charLow_S                  // s
	charLow_T                  // t
	charLow_U                  // u
	charABCDF                  // ABCDF
	charCap_E                  // E
	charEtc__                  // everything else
	charEof__                  // EOF
	numClasses
	_________ = -1 // error
)

// States tell the parser the grammar rule that is trying to be matched,
// and which kinds of characters are expected next.
type state int8

// States
const (
	sr state = iota // start
	ok              // ok
	ob              // object
	ke              // key
	co              // colon
	tc              // trailing comma
	va              // value
	ar              // array
	st              // string
	ec              // escape
	u1              // u1
	u2              // u2
	u3              // u3
	u4              // u4
	mi              // minus
	ze              // zero
	in              // integer
	fr              // fraction
	fs              // fraction
	e1              // e
	e2              // ex
	e3              // exp
	t1              // tr
	t2              // tru
	t3              // true
	f1              // fa
	f2              // fal
	f3              // fals
	f4              // false
	n1              // nu
	n2              // nul
	n3              // null
	c1              // comment
	c2              // line comment
	c3              // block comment
	c4              // block comment closeing star
	numStates
)

// actions
const (
	__ state = -1 - iota // Error
	ek                   // End key
	ep                   // End pair or array element
	es                   // End string
	sa                   // Start array
	so                   // Start object
	ea                   // End array
	aa                   // End empty array
	eo                   // End object
	ee                   // End empty object
	ab                   // Accept bool
	an                   // Accept null
	ai                   // Accept number
	as                   // Accept string
	sc                   // start comment
	ce                   // comment end
	cc                   // EOF on commented line
)

// Modes for the mode stack
// makes this state machine a pushdown automaton
// lets us parse recursive structures and do things
// like brace matching
type mode int8

const (
	// We're currently processing the contents of an array
	modeArray mode = iota
	// We're at the base value. If this is at the top of the stack,
	// we've either just finished or just ended parsing.
	modeDone
	// We're processing a key string, signals when the string is done,
	// we need to look for a ':' rather than just going back to OK
	modeKey
	// We're currently processing the contents of an object
	modeObject
)

// table for mapping an ascii byte to a character class
// EOF is special, and is enreachable via a single byte
var asciiClasses = [129]charClass{
	_________, _________, _________, _________, _________, _________, _________, _________,
	_________, charWhite, charLF___, _________, _________, charWhite, _________, _________,
	_________, _________, _________, _________, _________, _________, _________, _________,
	_________, _________, _________, _________, _________, _________, _________, _________,

	charSpace, charEtc__, charQuote, charEtc__, charEtc__, charEtc__, charEtc__, charEtc__,
	charEtc__, charEtc__, charStar_, charPlus_, charComma, charMinus, charPoint, charSlash,
	charZero_, charDigit, charDigit, charDigit, charDigit, charDigit, charDigit, charDigit,
	charDigit, charDigit, charColon, charEtc__, charEtc__, charEtc__, charEtc__, charEtc__,

	charEtc__, charABCDF, charABCDF, charABCDF, charABCDF, charCap_E, charABCDF, charEtc__,
	charEtc__, charEtc__, charEtc__, charEtc__, charEtc__, charEtc__, charEtc__, charEtc__,
	charEtc__, charEtc__, charEtc__, charEtc__, charEtc__, charEtc__, charEtc__, charEtc__,
	charEtc__, charEtc__, charEtc__, charLSqrB, charBacks, charRSqrB, charEtc__, charEtc__,

	charEtc__, charLow_A, charLow_B, charLow_C, charLow_D, charLow_E, charLow_F, charEtc__,
	charEtc__, charEtc__, charEtc__, charEtc__, charLow_L, charEtc__, charLow_N, charEtc__,
	charEtc__, charEtc__, charLow_R, charLow_S, charLow_T, charLow_U, charEtc__, charEtc__,
	charEtc__, charEtc__, charEtc__, charLCurB, charEtc__, charRCurB, charEtc__, charEtc__,
	charEof__,
}

// Maps a state + input to a new state. Some states (-1 and lower) are actions with special property rules
var stateTransitionTable = [numStates][numClasses]state{
	/*  	                white                                                        1-9                                                ABCDF    etc
	.               sp  \n  |   {   }   [   ]   :   ,   "   \   /   *   +   -   .   0   |   a   b   c   d   e   f   l   n   r   s   t   u   |   E   |  eof */
	/* start  sr*/ {sr, sr, sr, so, __, sa, __, __, __, st, __, sc, __, __, mi, __, ze, in, __, __, __, __, __, f1, __, n1, __, __, t1, __, __, __, __, __},
	/* ok     ok*/ {ok, ok, ok, __, eo, __, ea, __, ep, __, __, sc, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, ok},
	/* object ob*/ {ob, ob, ob, __, ee, __, __, __, __, st, __, sc, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __},
	/* key    ke*/ {ke, ke, ke, __, ee, __, __, __, __, st, __, sc, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __},
	/* colon  co*/ {co, co, co, __, __, __, __, ek, __, __, __, sc, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __},
	/* comma  tc*/ {tc, tc, tc, so, __, sa, aa, __, __, st, __, sc, __, __, mi, __, ze, in, __, __, __, __, __, f1, __, n1, __, __, t1, __, __, __, __, __},
	/* value  va*/ {va, va, va, so, __, sa, __, __, __, st, __, sc, __, __, mi, __, ze, in, __, __, __, __, __, f1, __, n1, __, __, t1, __, __, __, __, __},
	/* array  ar*/ {ar, ar, ar, so, __, sa, aa, __, __, st, __, sc, __, __, mi, __, ze, in, __, __, __, __, __, f1, __, n1, __, __, t1, __, __, __, __, __},
	/* string st*/ {st, __, __, st, st, st, st, st, st, es, ec, st, st, st, st, st, st, st, st, st, st, st, st, st, st, st, st, st, st, st, st, st, st, __},
	/* escape ec*/ {__, __, __, __, __, __, __, __, __, st, st, st, __, __, __, __, __, __, __, st, __, __, __, st, __, st, st, __, st, u1, __, __, __, __},
	/* u1     u1*/ {__, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, u2, u2, u2, u2, u2, u2, u2, u2, __, __, __, __, __, __, u2, u2, __, __},
	/* u2     u2*/ {__, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, u3, u3, u3, u3, u3, u3, u3, u3, __, __, __, __, __, __, u3, u3, __, __},
	/* u3     u3*/ {__, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, u4, u4, u4, u4, u4, u4, u4, u4, __, __, __, __, __, __, u4, u4, __, __},
	/* u4     u4*/ {__, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, st, st, st, st, st, st, st, st, __, __, __, __, __, __, st, st, __, __},
	/* minus  mi*/ {__, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, ze, in, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __},
	/* zero   ze*/ {ok, ok, ok, __, eo, __, ea, __, ep, __, __, sc, __, __, __, fr, __, __, __, __, __, __, e1, __, __, __, __, __, __, __, __, e1, __, ok},
	/* int    in*/ {ok, ok, ok, __, eo, __, ea, __, ep, __, __, sc, __, __, __, fr, in, in, __, __, __, __, e1, __, __, __, __, __, __, __, __, e1, __, ok},
	/* frac   fr*/ {__, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, fs, fs, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __},
	/* fracs  fs*/ {ok, ok, ok, __, eo, __, ea, __, ep, __, __, sc, __, __, __, __, fs, fs, __, __, __, __, e1, __, __, __, __, __, __, __, __, e1, __, ok},
	/* e      e1*/ {__, __, __, __, __, __, __, __, __, __, __, __, __, e2, e2, __, e3, e3, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __},
	/* ex     e2*/ {__, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, e3, e3, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __},
	/* exp    e3*/ {ok, ok, ok, __, eo, __, ea, __, ep, __, __, sc, __, __, __, __, e3, e3, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, ok},
	/* tr     t1*/ {__, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, t2, __, __, __, __, __, __, __},
	/* tru    t2*/ {__, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, t3, __, __, __, __},
	/* true   t3*/ {__, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, ok, __, __, __, __, __, __, __, __, __, __, __},
	/* fa     f1*/ {__, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, f2, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __},
	/* fal    f2*/ {__, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, f3, __, __, __, __, __, __, __, __, __},
	/* fals   f3*/ {__, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, f4, __, __, __, __, __, __},
	/* false  f4*/ {__, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, ok, __, __, __, __, __, __, __, __, __, __, __},
	/* nu     n1*/ {__, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, n2, __, __, __, __},
	/* nul    n2*/ {__, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, n3, __, __, __, __, __, __, __, __, __},
	/* null   n3*/ {__, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, ok, __, __, __, __, __, __, __, __, __},
	/* /      c1*/ {__, __, __, __, __, __, __, __, __, __, __, c2, c3, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __},
	/* // \n  c2*/ {c2, ce, c2, c2, c2, c2, c2, c2, c2, c2, c2, c2, c2, c2, c2, c2, c2, c2, c2, c2, c2, c2, c2, c2, c2, c2, c2, c2, c2, c2, c2, c2, c2, cc},
	/* /* *   c3*/ {c3, c3, c3, c3, c3, c3, c3, c3, c3, c3, c3, c3, c4, c3, c3, c3, c3, c3, c3, c3, c3, c3, c3, c3, c3, c3, c3, c3, c3, c3, c3, c3, c3, __},
	/* /* * / c4*/ {c3, c3, c3, c3, c3, c3, c3, c3, c3, c3, c3, ce, c4, c3, c3, c3, c3, c3, c3, c3, c3, c3, c3, c3, c3, c3, c3, c3, c3, c3, c3, c3, c3, __},
}

// The pushdown automaton to handle the parsing.
type parser struct {
	isRunning  bool
	isEOF      bool
	state      state
	modeTop    int
	valueTop   int
	modeStack  [depth]mode
	valueStack [depth * 3]*Value
	buffer     string
	pos        int
}

// Puts a value onto the value stack. Correct parsing should end
// with a single value left on the stack.
func (p *parser) pushValue(v *Value) {
	p.valueTop++
	p.valueStack[p.valueTop] = v
}

// Pulls a value from the stack.
func (p *parser) popValue() *Value {
	v := p.valueStack[p.valueTop]
	p.valueTop--
	return v
}

// Push a mode to the mode stack. Correct parsing should end
// with modeDone being the only thing left on the stack.
func (p *parser) pushMode(m mode) error {
	p.modeTop++
	if p.modeTop >= depth {
		return fmt.Errorf("%w: nested JSON max depth exceeded at byte %d", ErrParse, p.pos)
	}
	p.modeStack[p.modeTop] = m
	return nil
}

// Pulls a mode from the stack.
func (p *parser) popMode(m mode) error {
	if p.modeStack[p.modeTop] != m {
		return fmt.Errorf("%w: unmatched closing brace at %d", ErrParse, p.pos)
	}
	p.modeTop--
	return nil
}

// Sees what the top of the stack is without removing it.
func (p *parser) peekMode() mode {
	return p.modeStack[p.modeTop]
}

// An impossible input under correct JSON grammar has been reached. Can happen for several reasons.
func (p *parser) reject() error {
	p.isRunning = false
	return fmt.Errorf("%w: invalid character reached at byte %d", ErrParse, p.pos)
}

// We're at a point where,due to a closing brace, we are done with a literal value,
// but it hasn't been added to the stack yet. So we clip it here and push the value.
// This only happens for numbers (and integers), as the other values have explicit
// terminating characters.
func (p *parser) terminateLiterals(r rune) {
	switch p.state {
	case ze, in:
		// Accept an integer value
		val, _ := strconv.ParseInt(p.buffer, 10, 64)
		p.pushValue(&Value{jsonType: Integer, integerValue: val})
		p.buffer = ""
	case fs, e3:
		// Accept an Number value
		val, _ := strconv.ParseFloat(p.buffer, 64)
		p.pushValue(&Value{jsonType: Number, numberValue: val})
		p.buffer = ""
	}
}

// We're in array mode, and found a child object, so add it to the array
// as we go on. This way at most one child object is on the stack for an
// array at any time, and the rest are held in the array itself.
func (p *parser) growArray() {
	val := p.popValue()
	arr := p.popValue()
	arr.arrayValue = append(arr.arrayValue, val)
	p.pushValue(arr)
}

// We're in object mode, and found a child k/v pair, so add it to the object
// as we go on. This way at most one child pair is on the stack for an
// object at any time, and the rest are held in the object itself.
func (p *parser) growObject() {
	v, k := p.popValue(), p.popValue().stringValue
	obj := p.popValue()
	obj.objectValue = append(obj.objectValue, pair{key: k, val: v})
	p.pushValue(obj)
}

// Run one step of the PDA. Also handles the logic of the action states.
func (p *parser) consumeCharacter(r rune) error {
	var nextClass charClass
	var nextState state

	if p.isEOF {
		nextClass = charEof__
	} else if r >= 128 {
		nextClass = charEtc__
	} else {
		nextClass = asciiClasses[r]
	}

	if nextClass == _________ {
		return p.reject()
	}

	nextState = stateTransitionTable[p.state][nextClass]
	// Handle regular state transitions
	if nextState >= 0 {
		switch nextState {
		case t1, t2, t3, f1, f2, f3, f4, mi, ze, in, fr, fs, e1, e2, e3, st, ec, u1, u2, u3, u4:
			p.buffer = p.buffer + string(r)
		case ok:
			switch p.state {
			case n3:
				// Accept a null value
				p.pushValue(&Value{jsonType: Null})
				p.buffer = ""
			case f4, t3:
				// Accept a bool value
				p.buffer = p.buffer + string(r)
				val, _ := strconv.ParseBool(p.buffer)
				p.pushValue(&Value{jsonType: Boolean, booleanValue: val})
				p.buffer = ""
			case ze, in:
				// Accept an integer value
				val, _ := strconv.ParseInt(p.buffer, 10, 64)
				p.pushValue(&Value{jsonType: Integer, integerValue: val})
				p.buffer = ""
			case fs, e3:
				// Accept an Number value
				val, _ := strconv.ParseFloat(p.buffer, 64)
				p.pushValue(&Value{jsonType: Number, numberValue: val})
				p.buffer = ""
			}
		}

		p.state = nextState
		return nil
	}

	// Handle actions
	switch nextState {
	case ee:
		// End Empty Object
		p.popMode(modeKey)
		p.state = ok
		//
	case eo:
		// End non-empty object

		if err := p.popMode(modeObject); err != nil {
			return p.reject()
		}
		p.terminateLiterals(r)
		p.growObject()
		p.state = ok
	case aa:
		// End empty array
		p.popMode(modeArray)
		p.state = ok
	case ea:
		// End array

		if err := p.popMode(modeArray); err != nil {
			return p.reject()
		}
		p.terminateLiterals(r)
		p.growArray()
		p.state = ok
	case so:
		// Start object
		if err := p.pushMode(modeKey); err != nil {
			return p.reject()
		}

		p.pushValue(&Value{jsonType: Object, objectValue: []pair{}})
		p.state = ob
	case sa:
		// Start array
		if err := p.pushMode(modeArray); err != nil {
			return p.reject()
		}
		p.pushValue(&Value{jsonType: Array, arrayValue: []*Value{}})
		p.state = ar
	case es:
		// End String
		// Accept the built string value
		p.buffer = p.buffer + string(r)
		val, _ := strconv.Unquote(strings.Replace(p.buffer, `\/`, `/`, -1))
		p.pushValue(&Value{jsonType: String, stringValue: val})
		p.buffer = ""
		switch p.peekMode() {
		case modeKey:
			p.state = co
		default:
			p.state = ok
		}
	case ep:
		// End an array element or object pair
		// See comma
		p.terminateLiterals(r)

		switch p.peekMode() {
		case modeArray:
			p.growArray()
			p.state = tc
		case modeObject:
			p.growObject()
			p.popMode(modeObject)
			p.pushMode(modeKey)
			p.state = ke
		default:
			return p.reject()
		}
	case ek:
		// See colon
		p.popMode(modeKey)
		p.pushMode(modeObject)
		p.state = va
	case sc:
		p.pushMode(mode(p.state))
		p.state = c1
	case ce:
		p.state = state(p.peekMode())
		p.popMode(mode(p.state))
	case cc:
		// We have an eof, so get back to the previous state
		// before the comment and rerun the logic before stopping
		p.state = state(p.peekMode())
		p.popMode(mode(p.state))
		p.consumeCharacter(r)
	default:
		return p.reject()
	}
	return nil
}

// Parses a JSON value from a Reader. If it cannot read a valid value,
// it returns a null value and a non-nil error.
// Returns the parsed value and nil error otherwise.
func Parse(r io.Reader) (*Value, error) {
	pda := &parser{
		isRunning:  true,
		isEOF:      false,
		state:      sr,
		modeTop:    -1,
		valueTop:   -1,
		valueStack: [depth * 3]*Value{{}},
	}
	pda.pushMode(modeDone)

	b := bufio.NewReader(r)

	// main loop
	for pda.isRunning {
		r, n, err := b.ReadRune()
		if err != nil {
			if errors.Is(err, io.EOF) {
				pda.isEOF = true
				pda.isRunning = false
			} else {
				return &Value{}, err
			}
		}
		if r == unicode.ReplacementChar {
			return &Value{}, fmt.Errorf("%w: invalid UTF-8 character at %d", ErrParse, pda.pos)
		}
		if err := pda.consumeCharacter(r); err != nil {
			return &Value{}, err
		}

		pda.pos += n
	}
	return pda.valueStack[0], nil
}

// Parses a JSON value from a string. If it cannot read a valid value,
// it returns a null value and a non-nil error.
// Returns the parsed value and nil error otherwise.
func ParseString(s string) (*Value, error) {
	return Parse(strings.NewReader(s))
}

// Parses a JSON value from a byte slice. If it cannot read a valid value,
// it returns a null value and a non-nil error.
// Returns the parsed value and nil error otherwise.
func ParseBytes(b []byte) (*Value, error) {
	return ParseString(string(b))
}
