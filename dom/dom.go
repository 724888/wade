package dom

import (
	"errors"
	"fmt"
)

var (
	ErrorCantGetTagName    = errors.New("Not an element node, can't get tag name.")
	ErrorNoElementSelected = errors.New("No element selected.")
)

type (
	Dom interface {
		NewFragment(html string) Selection
		NewRootFragment() Selection
	}

	Event interface {
		Target() Selection
		CurrentTarget() Selection
		DelegateTarget() Selection
		RelatedTarget() Selection
		PreventDefault()
		StopPropagation()
		KeyCode() int
		Which() int
		MetaKey() bool
		PageXY() (int, int)
		Type() string
	}

	EventHandler func(Event)

	Attr struct {
		Name  string
		Value string
	}

	Selection interface {
		TagName() (string, error)
		Filter(selector string) Selection
		Children() Selection
		Contents() Selection
		First() Selection
		IsElement() bool
		Find(selector string) Selection
		Html() string
		SetHtml(html string)
		Length() int
		Elements() []Selection
		Append(Selection)
		Prepend(Selection)
		Remove()
		Clone() Selection
		ReplaceWith(Selection)
		OuterHtml() string
		Attr(attr string) (string, bool)
		SetAttr(attr string, value string)
		RemoveAttr(attr string)
		Val() string
		SetVal(val string)
		Parents() Selection
		Is(selector string) bool
		Unwrap()
		Parent() Selection
		Next() Selection
		Prev() Selection
		Before(sel Selection)
		After(sel Selection)
		Exists() bool
		Attrs() []Attr
		On(Event string, handler EventHandler)
		Listen(event string, selector string, handler EventHandler)
		Hide()
		Show()
		AddClass(class string)
		RemoveClass(class string)
		HasClass(class string) bool
		Text() string
		Dom
	}
)

// DebugInfo prints debug information for the element, including
// tag name, id and parent tree
func DebugInfo(sel Selection) string {
	sel = sel.First()
	tagname, _ := sel.TagName()
	str := tagname
	if id, ok := sel.Attr("id"); ok {
		str += "#" + id
	}
	str += " ("
	parents := sel.Parents().Elements()
	for j := len(parents) - 1; j >= 0; j-- {
		t, err := parents[j].TagName()
		if err == nil {
			str += t + ">"
		}
	}
	str += ")"

	return str
}

// ElementError returns an error with DebugInfo on the element
func ElementError(sel Selection, errstr string) error {
	return fmt.Errorf("Error on element {%v}: %v.", DebugInfo(sel), errstr)
}
