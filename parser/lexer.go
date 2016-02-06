package parser

import "unicode/utf8"

type lexer struct {
	input string
	start int
	pos   int
	width int
	line  int
	items chan item
}

func (l *lexer) run() {
	for state := lexLine; state != nil; {
		state = state(l)
	}
	close(l.items)
}

func (l *lexer) emit(t itemType) {
	l.items <- item{t, l.input[l.start:l.pos], l.line}
	l.start = l.pos
}

const eof rune = 0

func (l *lexer) next() rune {
	if l.pos >= len(l.input) {
		l.width = 0
		return eof
	}

	var r rune
	r, l.width = utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += l.width
	return r
}

func (l *lexer) backup() {
	l.pos -= l.width
}

func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

func (l *lexer) ignore() {
	l.start = l.pos
}

func (l *lexer) accept(set string) {
	for {
		r := l.peek()
		found := false
		for _, s := range set {
			if r == s {
				l.next()
				found = true
				break
			}
		}
		if !found {
			return
		}
	}
}

func lex(input string) chan item {
	l := &lexer{input: input, items: make(chan item), line: 1}
	go l.run()
	return l.items
}

type lexStateFn func(*lexer) lexStateFn

func lexLine(l *lexer) lexStateFn {
	switch l.next() {
	case '#':
		l.accept("#")
		l.emit(itemHash)
		return lexText
	case '>':
		l.emit(itemArrow)
		return lexText
	case '\n':
		l.emit(itemEmptyLine)
		l.line++
		return lexLine
	case ' ', '\t':
		return lexIndent
	case eof:
		l.emit(itemEOF)
		return nil
	default:
		l.backup()
		return lexText
	}
}

func lexIndent(l *lexer) lexStateFn {
	l.accept(" \t")
	if l.start < l.pos {
		l.emit(itemIndent)
	}
	return lexText
}

func lexText(l *lexer) lexStateFn {
	for {
		switch l.next() {
		case '\n', eof:
			l.emit(itemText)
			l.line++
			return lexLine
		}
	}
}
