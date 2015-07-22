package dom

import (
	"github.com/gopherjs/gopherjs/js"
)

var (
	document        Document
	NewEventHandler func(EventHandler) interface{}
)

type EventHandler func(Event)

type Event interface {
	PreventDefault()
	StopPropagation()
	JS() *js.Object
}

func GetDocument() Document {
	if document == nil {
		panic(" document has not been set.")
	}
	return document
}

type Document interface {
	Title() string
	SetTitle(title string)

	Node
}

func SetDocument(doc Document) {
	document = doc
}

type NodeType int

const (
	NopNode NodeType = iota
	ElementNode
	TextNode
)

type Node interface {
	Type() NodeType
	Find(query string) []Node
	Data() string
	Children() []Node

	SetAttr(string, interface{})
	SetProp(string, interface{})
	RemoveAttr(string)

	Clear()
	JS() *js.Object
	SetClass(string, bool)
}

type Driver interface {
	ToInputEl(Node) InputEl
	ToFormEl(Node) FormEl
}

type FormEl interface {
	Node
	IsValid() bool
}

type InputEl interface {
	Node
	Value() string
	SetValue(string)
	Checked() bool
	SetChecked(bool)
}
