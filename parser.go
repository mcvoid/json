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
	depth = 1024
)

type charClass int8

// Character classes
const (
	charSpace charClass = iota // space
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

type state int8

// States
const (
	sr state = iota // start
	ok              // ok
	ob              // object
	ke              // key
	co              // colon
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
	eo                   // End object
	ee                   // End empty object
	ab                   // Accept bool
	an                   // Accept null
	ai                   // Accept number
	as                   // Accept string
)

// Modes
type mode int8

const (
	modeArray mode = iota
	modeDone
	modeKey
	modeObject
)

var ()

var asciiClasses = [129]charClass{
	_________, _________, _________, _________, _________, _________, _________, _________,
	_________, charWhite, charWhite, _________, _________, charWhite, _________, _________,
	_________, _________, _________, _________, _________, _________, _________, _________,
	_________, _________, _________, _________, _________, _________, _________, _________,

	charSpace, charEtc__, charQuote, charEtc__, charEtc__, charEtc__, charEtc__, charEtc__,
	charEtc__, charEtc__, charEtc__, charPlus_, charComma, charMinus, charPoint, charSlash,
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

var stateTransitionTable = [numStates][numClasses]state{
	/*  	            white                                                    1-9                                                ABCDF    etc
	.               sp  |   {   }   [   ]   :   ,   "   \   /   +   -   .   0   |   a   b   c   d   e   f   l   n   r   s   t   u   |   E   |  eof */
	/* start  sr*/ {sr, sr, so, __, sa, __, __, __, st, __, __, __, mi, __, ze, in, __, __, __, __, __, f1, __, n1, __, __, t1, __, __, __, __, __},
	/* ok     ok*/ {ok, ok, __, eo, __, ea, __, ep, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, ok},
	/* object ob*/ {ob, ob, __, ee, __, __, __, __, st, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __},
	/* key    ke*/ {ke, ke, __, __, __, __, __, __, st, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __},
	/* colon  co*/ {co, co, __, __, __, __, ek, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __},
	/* value  va*/ {va, va, so, __, sa, __, __, __, st, __, __, __, mi, __, ze, in, __, __, __, __, __, f1, __, n1, __, __, t1, __, __, __, __, __},
	/* array  ar*/ {ar, ar, so, __, sa, ea, __, __, st, __, __, __, mi, __, ze, in, __, __, __, __, __, f1, __, n1, __, __, t1, __, __, __, __, __},
	/* string st*/ {st, __, st, st, st, st, st, st, es, ec, st, st, st, st, st, st, st, st, st, st, st, st, st, st, st, st, st, st, st, st, st, __},
	/* escape ec*/ {__, __, __, __, __, __, __, __, st, st, st, __, __, __, __, __, __, st, __, __, __, st, __, st, st, __, st, u1, __, __, __, __},
	/* u1     u1*/ {__, __, __, __, __, __, __, __, __, __, __, __, __, __, u2, u2, u2, u2, u2, u2, u2, u2, __, __, __, __, __, __, u2, u2, __, __},
	/* u2     u2*/ {__, __, __, __, __, __, __, __, __, __, __, __, __, __, u3, u3, u3, u3, u3, u3, u3, u3, __, __, __, __, __, __, u3, u3, __, __},
	/* u3     u3*/ {__, __, __, __, __, __, __, __, __, __, __, __, __, __, u4, u4, u4, u4, u4, u4, u4, u4, __, __, __, __, __, __, u4, u4, __, __},
	/* u4     u4*/ {__, __, __, __, __, __, __, __, __, __, __, __, __, __, st, st, st, st, st, st, st, st, __, __, __, __, __, __, st, st, __, __},
	/* minus  mi*/ {__, __, __, __, __, __, __, __, __, __, __, __, __, __, ze, in, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __},
	/* zero   ze*/ {ok, ok, __, eo, __, ea, __, ep, __, __, __, __, __, fr, __, __, __, __, __, __, e1, __, __, __, __, __, __, __, __, e1, __, ok},
	/* int    in*/ {ok, ok, __, eo, __, ea, __, ep, __, __, __, __, __, fr, in, in, __, __, __, __, e1, __, __, __, __, __, __, __, __, e1, __, ok},
	/* frac   fr*/ {__, __, __, __, __, __, __, __, __, __, __, __, __, __, fs, fs, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __},
	/* fracs  fs*/ {ok, ok, __, eo, __, ea, __, ep, __, __, __, __, __, __, fs, fs, __, __, __, __, e1, __, __, __, __, __, __, __, __, e1, __, ok},
	/* e      e1*/ {__, __, __, __, __, __, __, __, __, __, __, e2, e2, __, e3, e3, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __},
	/* ex     e2*/ {__, __, __, __, __, __, __, __, __, __, __, __, __, __, e3, e3, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __},
	/* exp    e3*/ {ok, ok, __, eo, __, ea, __, ep, __, __, __, __, __, __, e3, e3, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, ok},
	/* tr     t1*/ {__, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, t2, __, __, __, __, __, __, __},
	/* tru    t2*/ {__, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, t3, __, __, __, __},
	/* true   t3*/ {__, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, ok, __, __, __, __, __, __, __, __, __, __, __},
	/* fa     f1*/ {__, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, f2, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __},
	/* fal    f2*/ {__, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, f3, __, __, __, __, __, __, __, __, __},
	/* fals   f3*/ {__, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, f4, __, __, __, __, __, __},
	/* false  f4*/ {__, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, ok, __, __, __, __, __, __, __, __, __, __, __},
	/* nu     n1*/ {__, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, n2, __, __, __, __},
	/* nul    n2*/ {__, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, n3, __, __, __, __, __, __, __, __, __},
	/* null   n3*/ {__, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __, ok, __, __, __, __, __, __, __, __, __},
}

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

func (p *parser) pushValue(v *Value) {
	p.valueTop++
	p.valueStack[p.valueTop] = v
}

func (p *parser) popValue() *Value {
	v := p.valueStack[p.valueTop]
	p.valueTop--
	return v
}

func (p *parser) peekValue() *Value {
	return p.valueStack[p.valueTop]
}

func (p *parser) pushMode(m mode) error {
	p.modeTop++
	if p.modeTop >= depth {
		return fmt.Errorf("%w: nested JSON max depth exceeded at byte %d", ErrParse, p.pos)
	}
	p.modeStack[p.modeTop] = m
	return nil
}

func (p *parser) popMode(m mode) error {
	if p.modeStack[p.modeTop] != m {
		return fmt.Errorf("%w: unmatched closing brace at %d", ErrParse, p.pos)
	}
	p.modeTop--
	return nil
}

func (p *parser) peekMode() mode {
	return p.modeStack[p.modeTop]
}

func (p *parser) reject() error {
	p.isRunning = false
	return fmt.Errorf("%w: invalid character reached at byte %d", ErrParse, p.pos)
}

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

func (p *parser) growArray() {
	if p.peekValue().jsonType == Array {
		return
	}
	val := p.popValue()
	arr := p.popValue()
	arr.arrayValue = append(arr.arrayValue, val)
	p.pushValue(arr)
}

func (p *parser) growObject() {
	v, k := p.popValue(), p.popValue().stringValue
	obj := p.popValue()
	obj.objectValue = append(obj.objectValue, pair{key: k, val: v})
	p.pushValue(obj)
}

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
			p.state = va
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
	default:
		return p.reject()
	}
	return nil
}

func Parse(r io.Reader) (*Value, error) {
	pda := &parser{
		isRunning: true,
		isEOF:     false,
		state:     sr,
		modeTop:   -1,
		valueTop:  -1,
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
				return nil, err
			}
		}
		if r == unicode.ReplacementChar {
			return nil, fmt.Errorf("%w: invalid UTF-8 character at %d", ErrParse, pda.pos)
		}
		if err := pda.consumeCharacter(r); err != nil {
			return nil, err
		}

		pda.pos += n
	}
	return pda.valueStack[0], nil
}

func ParseString(s string) (*Value, error) {
	return Parse(strings.NewReader(s))
}

func ParseBytes(b []byte) (*Value, error) {
	return ParseString(string(b))
}
