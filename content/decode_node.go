package content

import (
	"bytes"
	"encoding/json"
	"iter"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
)

// nodeKind is the JSON type of a node.
type nodeKind uint8

const (
	nodeInvalid nodeKind = iota
	nodeObject
	nodeArray
	nodeString
	nodeNumber
	nodeBool
	nodeNull
)

// node is the bytes of one JSON value of a document that json.Valid has
// accepted: a slice of the document, with no white space around it. The zero
// node is invalid. A node is not expanded until a field decoder asks for its
// members or elements, so a subtree nobody reads costs no memory beyond the
// document itself (a hostile 16 MiB array under an unknown key is scanned by
// json.Valid and never built).
type node []byte

// kind returns the JSON type of n from its first byte.
func (n node) kind() nodeKind {
	if len(n) == 0 { // equivalent mutant: a length is never negative, so == and <= agree
		return nodeInvalid
	}
	switch n[0] {
	case '{':
		return nodeObject
	case '[':
		return nodeArray
	case '"':
		return nodeString
	case 't', 'f':
		return nodeBool
	case 'n':
		return nodeNull
	}
	return nodeNumber
}

// number returns the literal text of a number node as json.Number, so a
// number is never rounded through a float64.
func (n node) number() json.Number { return json.Number(n) }

// parseDocument reports whether data is one valid JSON value, to the limits of
// encoding/json (syntax, trailing data and the nesting depth of 10000), and
// returns it as a node.
func parseDocument(data []byte) (node, bool) {
	if !json.Valid(data) {
		return nil, false
	}
	return node(bytes.Trim(data, " \t\r\n")), true
}

// nextValue reads the next value of dec, which reads from data, as a slice of
// data. A value the decoder cannot read ends the walk; json.Valid has
// accepted data, so that does not happen.
func nextValue(dec *json.Decoder, data []byte) (node, bool) {
	var raw json.RawMessage
	if err := dec.Decode(&raw); err != nil {
		return nil, false
	}
	end := int(dec.InputOffset())
	return node(bytes.TrimLeft(data[end-len(raw):end], " \t\r\n")), true
}

// elements yields the elements of an array node in order, stopping when yield
// returns false. It yields nothing for a node that is not an array.
func (n node) elements() iter.Seq2[int, node] {
	return func(yield func(int, node) bool) {
		if n.kind() != nodeArray {
			return
		}
		dec := json.NewDecoder(bytes.NewReader(n))
		if _, err := dec.Token(); err != nil { // the opening bracket
			return
		}
		for i := 0; dec.More(); i++ {
			child, ok := nextValue(dec, n)
			if !ok || !yield(i, child) {
				return
			}
		}
	}
}

// member is one key and value of an object, in document order. keyOK is false
// when the key held invalid UTF-8 or a lone surrogate (key then holds U+FFFD
// for it).
type member struct {
	key   string
	keyOK bool
	val   node
}

// members returns the members of an object node in document order, repeated
// keys included. It returns nil for a node that is not an object.
func (n node) members() []member {
	if n.kind() != nodeObject {
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(n))
	if _, err := dec.Token(); err != nil { // the opening brace
		return nil
	}
	var out []member
	for dec.More() {
		before := int(dec.InputOffset())
		if _, err := dec.Token(); err != nil { // the key; its text is read raw below
			break
		}
		rawKey := bytes.TrimLeft(n[before:int(dec.InputOffset())], " \t\r\n,")
		key, keyOK := unquote(rawKey)
		val, ok := nextValue(dec, n)
		if !ok {
			break
		}
		out = append(out, member{key: key, keyOK: keyOK, val: val})
	}
	return out
}

// text decodes a string node. ok is false when the literal held invalid UTF-8
// or an escaped surrogate with no partner, which encoding/json would silently
// replace with U+FFFD; the returned text then holds the replacement.
func (n node) text() (string, bool) {
	if n.kind() != nodeString || len(n) < 2 {
		return "", false
	}
	return unquote(n)
}

// unquote decodes the JSON string literal raw, quotes included. It trusts
// the escapes to be well formed (json.Valid has checked them) and reports
// ok=false for invalid UTF-8 and for an unpaired surrogate escape.
func unquote(raw []byte) (string, bool) {
	body := raw[1 : len(raw)-1]
	if bytes.IndexByte(body, '\\') < 0 && utf8.Valid(body) { // equivalent mutant when forced off: the slow path gives the same result, only slower
		return string(body), true
	}
	out := make([]byte, 0, len(body))
	ok := true
	for i := 0; i < len(body); { // equivalent mutant: i reaches len(body) exactly for well-formed escapes, so < and != agree
		if body[i] != '\\' {
			r, size := utf8.DecodeRune(body[i:])
			ok = ok && (r != utf8.RuneError || size != 1) // equivalent mutant: DecodeRune never returns a size above 1 for RuneError from an invalid byte, so != and > agree
			out = utf8.AppendRune(out, r)
			i += size
			continue
		}
		c := body[i+1]
		i += 2
		if c != 'u' {
			out = append(out, unescapeByte(c))
			continue
		}
		r := hex4(body[i : i+4])
		i += 4
		if !utf16.IsSurrogate(r) {
			out = utf8.AppendRune(out, r)
			continue
		}
		// equivalent mutants: a pair test on a low surrogate or on a non-escape still fails in DecodeRune below.
		if r < 0xDC00 && i+6 <= len(body) && body[i] == '\\' && body[i+1] == 'u' {
			if pair := utf16.DecodeRune(r, hex4(body[i+2:i+6])); pair != unicode.ReplacementChar { // equivalent mutant: DecodeRune returns U+FFFD or a rune above it, so != and > agree
				out = utf8.AppendRune(out, pair)
				i += 6
				continue
			}
		}
		ok = false
		out = utf8.AppendRune(out, utf8.RuneError)
	}
	return string(out), ok
}

// unescapeByte maps the character after a backslash (other than u) to the
// byte it stands for.
func unescapeByte(c byte) byte {
	switch c {
	case 'b':
		return '\b'
	case 'f':
		return '\f'
	case 'n':
		return '\n'
	case 'r':
		return '\r'
	case 't':
		return '\t'
	}
	return c // '"', '\\' and '/'
}

// hex4 parses four hexadecimal digits.
func hex4(b []byte) rune {
	var r rune
	for _, c := range b {
		r *= 16
		switch {
		case c >= '0' && c <= '9': // equivalent mutant: the digit 0 adds nothing, so >= and > agree
			r += rune(c - '0')
		case c >= 'a' && c <= 'f':
			r += rune(c-'a') + 10
		case c >= 'A' && c <= 'F':
			r += rune(c-'A') + 10
		}
	}
	return r
}
