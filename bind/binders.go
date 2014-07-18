package bind

import (
	"fmt"
	"reflect"
	"strings"

	jq "github.com/gopherjs/jquery"
)

const (
	WadePageAttr = "data-wade-page"
)

func defaultBinders() map[string]DomBinder {
	return map[string]DomBinder{
		"value": &ValueBinder{},
		"html":  &HtmlBinder{},
		"attr":  &AttrBinder{},
		"on":    &EventBinder{},
		"each":  new(EachBinder),
		"page":  &PageBinder{},
		"if":    new(IfBinder),
		"ifn":   &UnlessBinder{&IfBinder{}},
	}
}

// ValueBinder is a 2-way binder that binds an element's value attribute.
// It takes no extra dash args.
// Meant to be used for <input>.
//
// Usage:
//	bind-value="Expression"
type ValueBinder struct{ *BaseBinder }

// Update sets the element's value attribute to a new value
func (b *ValueBinder) Update(d DomBind) {
	d.Elem.SetVal(toString(d.Value))
}

// Watch watches for javascript change event on the element
func (b *ValueBinder) Watch(elem jq.JQuery, ufn ModelUpdateFn) {
	tagname := strings.ToUpper(elem.Prop("tagName").(string))
	if tagname != "INPUT" {
		panic("Can only watch for changes on html input, textarea and select.")
	}

	elem.On(jq.CHANGE, func(evt jq.Event) {
		ufn(elem.Val())
	})
}
func (b *ValueBinder) BindInstance() DomBinder { return b }

// ValueBinder is a 1-way binder that binds an element's html content to
// the value of a model field.
// It takes no extra dash args.
//
// Usage:
//	bind-html="Expression"
type HtmlBinder struct{ BaseBinder }

// Update sets the element's html content to a new value
func (b *HtmlBinder) Update(d DomBind) {
	d.Elem.SetHtml(toString(d.Value))
}
func (b *HtmlBinder) BindInstance() DomBinder { return b }

// AttrBinder is a 1-way binder that binds a specified element's attribute
// to a model field value.
// It takes 1 extra dash arg that is the name of the html attribute to be bound.
//
// Usage:
//	bind-attr-thatAttribute="Expression"
type AttrBinder struct{ BaseBinder }

func (b *AttrBinder) Update(d DomBind) {
	if len(d.Args) != 1 {
		panic(fmt.Sprintf(`Incorrect number of args %v for html attribute binder.
Usage: bind-attr-thatAttribute="Field".`, len(d.Args)))
	}
	d.Elem.SetAttr(d.Args[0], toString(d.Value))
}
func (b *AttrBinder) BindInstance() DomBinder { return b }

// EventBinder is a 1-way binder that binds a method of the model to an event
// that occurs on the element.
// It takes 1 extra dash arg that is the event name, for example "click",
// "change",...
//
// Usage:
//	bind-on-thatEventName="HandlerMethod"
type EventBinder struct{ BaseBinder }

func (b *EventBinder) Bind(d DomBind) {
	fni := d.Value
	if fni == nil {
		d.Panic("Event must be bound to a function, not a nil. If you're trying to call a function on this event, please use a method that returns a func().")
	}
	fn, ok := fni.(func())
	if !ok {
		panic(fmt.Sprintf("Wrong type %v for EventBinder's handler, must be of type func().",
			reflect.TypeOf(fni).String()))
	}
	if len(d.Args) > 1 {
		panic("Too many dash arguments to event bind.")
	}
	d.Elem.On(d.Args[0], func(evt jq.Event) {
		evt.PreventDefault()
		fn()
	})
}
func (b *EventBinder) BindInstance() DomBinder { return b }

type indexFunc func(i int, v reflect.Value) (interface{}, reflect.Value)

// EachBinder is a 1-way binder that repeats an element according to a map
// or slice. It outputs a key and a value bound to each item.
// It takes no extra dash arg. The extra output after "->" are the names that
// receives the key and value, those names can be used inside the elment's
// content. Each key and value pair is bound separately to each element.
//
// Usage:
//	bind-each="Expression"
// Or
//	bind-each="Expression -> outputKey, outputValue"
// Example:
//	<div bind-each="Errors -> type, msg">
//		<p>Error type: <% type %></p>
//		<p>Message: <% msg %></p>
//	</div>
type EachBinder struct {
	*BaseBinder
	marker    jq.JQuery
	prototype jq.JQuery
	indexFn   indexFunc
	size      int
}

func (b *EachBinder) BindInstance() DomBinder {
	return new(EachBinder)
}

func getIndexFunc(value interface{}) indexFunc {
	kind := reflect.TypeOf(value).Kind()
	switch kind {
	case reflect.Slice:
		return func(i int, val reflect.Value) (interface{}, reflect.Value) {
			return i, val.Index(i)
		}
	case reflect.Map:
		return func(i int, val reflect.Value) (interface{}, reflect.Value) {
			key := val.MapKeys()[i]
			return key.String(), val.MapIndex(key)
		}
	default:
		panic(fmt.Sprintf("Wrong kind %v of target for the each binder, must be a slice or map.", kind.String()))
	}
}

func (b *EachBinder) Bind(d DomBind) {
	d.Elem.RemoveAttr(BindPrefix + "each")
	b.indexFn = getIndexFunc(d.Value)
	b.marker = gJQ("<!-- wade each -->").InsertBefore(d.Elem).First()
	b.prototype = d.Elem.Clone()

	d.Elem.Remove()
	d.RemoveBinding(d.Elem)
}

func (b *EachBinder) Update(d DomBind) {
	val := reflect.ValueOf(d.Value)

	for i := val.Len(); i < b.size; i++ {
		b.marker.Next().Remove()
	}

	for i := b.size; i < val.Len(); i++ {
		b.marker.After(b.prototype.Clone())
	}

	b.size = val.Len()

	prev := b.marker

	for i := 0; i < b.size; i++ {
		k, v := b.indexFn(i, val)
		nx := b.prototype.Clone()
		prev.Next().ReplaceWith(nx)
		d.ProduceOutputs(nx, true, true, k, v.Interface())
		prev = nx
	}
}

// PageBinder is used for <a> elements to set its href to the real page url
// and save necessary information for the proper page switching when the user
// clicks on the link. It should be used with the url() helper.
//
// Typical usage:
//	bind-page="url(`page-id`, arg1, arg2...)"
type PageBinder struct{ BaseBinder }

func (b *PageBinder) Update(d DomBind) {
	if strings.ToLower(d.Elem.Prop("tagName").(string)) != "a" {
		panic("bind-page can only be used for links (<a> elements).")
	}
	uinf := d.Value.(UrlInfo)
	d.Elem.SetAttr("href", uinf.fullUrl)
	d.Elem.SetAttr(WadePageAttr, uinf.path)
}
func (b *PageBinder) BindInstance() DomBinder { return b }

// IfBinder shows or remove an element according to a boolean field value.
//
// Usage:
//	bind-if="BooleanExpression"
type IfBinder struct {
	*BaseBinder
	placeholder jq.JQuery
}

func (b *IfBinder) Bind(d DomBind) {
	b.placeholder = gJQ("<!-- hidden elem -->")
}

func (b *IfBinder) Update(d DomBind) {
	shown := d.Value.(bool)
	if shown && !jqExists(d.Elem) {
		b.placeholder.ReplaceWith(d.Elem)
		return
	}

	if !shown && jqExists(d.Elem) {
		d.Elem.ReplaceWith(b.placeholder)
	}
}
func (b *IfBinder) BindInstance() DomBinder { return new(IfBinder) }

// UnlessBinder is the reverse of IfBinder.
//
// Usage:
//	bind-ifn="BooleanExpression"
type UnlessBinder struct {
	*IfBinder
}

func (b *UnlessBinder) Update(d DomBind) {
	d.Value = !(d.Value.(bool))
	b.IfBinder.Update(d)
}
func (b *UnlessBinder) BindInstance() DomBinder { return &UnlessBinder{new(IfBinder)} }
