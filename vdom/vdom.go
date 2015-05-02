package vdom

type Attributes map[string]interface{}

type EvtHandler func(Event)
type Event interface{}

type Node interface {
	IsElement() bool
	NodeData() string
}

type TextNode struct {
	Data string
}

func (t *TextNode) IsElement() bool {
	return false
}

func (t *TextNode) NodeData() string {
	return t.Data
}

func NewTextNode(data string) *TextNode {
	return &TextNode{Data: data}
}

type Element struct {
	Tag         string
	Attrs       Attributes
	Children    []Node
	EvtHandlers map[string]EvtHandler
	rendered    interface{}
	Component   Component
	rendCache   *Element
	oldElem     *Element
	Key         string
}

type Component interface {
	Render(interface{}) *Element
	InternalState() interface{}
}

func (t *Element) IsElement() bool {
	return true
}

func (t *Element) NodeData() string {
	return t.Tag
}

func (t *Element) Render() *Element {
	if t.Component != nil {
		if t.rendCache != nil {
			return t.rendCache
		}

		var state interface{}
		if t.oldElem != nil && t.oldElem.Component != nil {
			state = t.oldElem.Component.InternalState()
		}

		t.rendCache = t.Component.Render(state)
		return t.rendCache
	}

	return t
}

func NewComElement(comName string, com Component) *Element {
	return &Element{
		Tag:       comName,
		Component: com,
	}
}

func NewElement(tag string, attrs Attributes, children []Node) *Element {
	return &Element{
		Tag:      tag,
		Attrs:    attrs,
		Children: children,
	}
}
