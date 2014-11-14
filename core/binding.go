package core

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/phaikawl/wade/com"

	. "github.com/phaikawl/wade/scope"
)

const (
	ReservedPrefix = "data-w-"
	//BoundAttr      = ReservedPrefix + "bound"
	//BindInfoAttr   = ReservedPrefix + "binds"

	AttrBindPrefix   = '@'
	BinderBindPrefix = '#'
	WrapperTag       = "w-ghost"
)

type (
	Application interface {
		ErrChanPut(error)
		ComponentInit(prototype com.Prototype)
	}

	Binding struct {
		app        Application
		tm         *com.Manager
		domBinders map[string]DomBinder
		helpers    HelpersSymbolTable

		watcher   *Watcher
		scope     *Scope
		pageModel interface{}
	}

	drmElem struct {
		dom.Selection
		rmList *[]dom.Selection
	}

	BindingError struct {
		Err error
	}
)

func (be BindingError) Error() string {
	return be.Err.Error()
}

type DummyApp struct {
}

func (da DummyApp) ErrChanPut(err error) {
}

func (da DummyApp) ComponentInit(proto com.Prototype) {
}

func NewTestBindEngine() *Binding {
	return NewBindEngine(DummyApp{}, com.NewManager(nil), BasicWatchBackend{})
}

func NewBindEngine(app Application, tm *com.Manager, jsWatcher WatchBackend) *Binding {
	b := &Binding{
		app:        app,
		tm:         tm,
		watcher:    NewWatcher(jsWatcher),
		domBinders: defaultBinders(),
		helpers:    NewHelpersSymbolTable(defaultHelpers()),
	}

	b.scope = NewScope([]SymbolTable{b.helpers})
	return b
}

func (b *Binding) Watcher() *Watcher {
	return b.watcher
}

//RegisterBinder registers a binder
func (b *Binding) RegisterBinder(name string, binder DomBinder) {
	if _, exists := b.domBinders[name]; exists {
		panic(fmt.Sprintf(`A binder with that name "%v" already exists.`, name))
	}

	b.domBinders[name] = binder
}

func (b *Binding) ComponentManager() *com.Manager {
	return b.tm
}

// RegisterHelper registers a function as a helper with the given name.
// Helpers are global.
func (b *Binding) RegisterHelper(name string, fn interface{}) {
	typ := reflect.TypeOf(fn)
	if typ.Kind() != reflect.Func {
		panic(fmt.Sprintf("Invalid helper %v, must be a function.", name))
	}

	if typ.NumOut() == 0 {
		panic(fmt.Sprintf("Invalid helper %v, a helper must return something.", name))
	}

	if _, exist, _ := b.helpers.Lookup(name); !exist {
		b.helpers.RegisterFunc(name, fn)
		return
	}

	panic(fmt.Sprintf("Helper with name %v already exists.", name))
	return
}

func (b *Binding) watchModel(value interface{}, binds *barray, root *expr, bs *bindScope, callback WatchCallback) error {
	for _, bi := range binds.slice {
		if !bi.BindObj().FieldRefl.CanAddr() {
			return fmt.Errorf(`Cannot watch field "%v" because it's an unaddressable value. Perhaps you don't really need to watch for its changes, if that's the case, you can use a pipe ("|") at the beginning`, bi.BindObj().Field)
		}

		//use watchjs to watch for changes to the model
		b.watcher.Watch(value, func(oldAddr uintptr, repl interface{}) (newVal interface{}) {
			newVal, _ = bs.evaluateRec(root, nil, oldAddr, repl)
			return
		}, bi.BindObj(), callback)
	}

	return nil
}

func bstrPanic(mess, bindstring string, elem dom.Selection) {
	if !elem.Exists() {
		return
	}

	panic(dom.ElementError(elem, fmt.Sprintf(mess+`. While processing bind string "%v"`, bindstring)))
}

func reportError(err error, bstr string, elem dom.Selection) {
	if err != nil {
		bstrPanic(err.Error(), bstr, elem)
	}
}

func (e drmElem) Remove() {
	*e.rmList = append(*e.rmList, e.Selection)
}

func (e drmElem) ReplaceWith(sel dom.Selection) {
	e.Before(sel)
	e.Remove()
}

func (b *Binding) processAttrBind(attr string, bstr string, elem dom.Selection, bs *bindScope, once bool) (err error) {
	roote, binds, v, er := bs.evaluate(bstr)
	if er != nil {
		bstrPanic(er.Error(), bstr, elem)
	}

	var setFn func(value interface{})
	if _, ok := v.(string); ok {
		setFn = func(value interface{}) {
			elem.SetAttr(attr, value.(string))
		}
	} else {
		setFn = func(value interface{}) {
			elem.SetProp(attr, value)
		}
	}

	setFn(v)

	if !once {
		err = b.watchModel(v, binds, roote, bs, func(newResult interface{}) {
			setFn(newResult)
		})
	}

	if err != nil {
		bstrPanic(err.Error(), bstr, elem)
	}

	return
}

func (b *Binding) processFieldBind(field string, bstr string, elem dom.Selection, bs *bindScope, once bool, ce ) {
	roote, binds, v, er := bs.evaluate(bstr)
	if er != nil {
		bstrPanic(er.Error(), bstr, elem)
	}

	oe, ok, err := EvaluateObjField(field, reflect.ValueOf(ce.Model()))
	if err != nil {
		bstrPanic(err.Error(), bstr, elem)
	}

	if !ok {
		bstrPanic(fmt.Sprintf(`No such field "%v" to bind to`, field), bstr, elem)
	}

	checkCompat := func(src, dst reflect.Type) {
		if !src.AssignableTo(dst) {
			bstrPanic(fmt.Sprintf(`Unassignable, incompatible types "%v" and "%v" of the model field and the value`,
				src.String(), dst.String()), bstr, elem)
		}
	}

	checkCompat(reflect.TypeOf(v), oe.FieldRefl.Type())
	oe.FieldRefl.Set(reflect.ValueOf(v))

	if !once {
		err = b.watchModel(v, binds, roote, bs, func(newResult interface{}) {
			nr := reflect.ValueOf(newResult)
			checkCompat(nr.Type(), oe.FieldRefl.Type())
			oe.FieldRefl.Set(nr)

			err := ce.Update()
			if err != nil {
				b.app.ErrChanPut(BindingError{dom.ElementError(ce.Element(), err.Error())})
			}
		})

		if err != nil {
			bstrPanic(err.Error(), bstr, elem)
		}
	}
}

func (b *Binding) bindCustomElem(elem dom.Selection, tag *com.Component, bs *bindScope, once bool, scopeElem dom.Selection) {
	if !elem.Exists() {
		return
	}

	if bound, ok := elem.Prop(BoundAttr); ok && bound.(bool) == true {
		return
	}

	bindinfo := ""
	if scopeElem != nil {
		if setag, _ := scopeElem.TagName(); tag.Name == setag {
			panic(dom.ElementError(elem,
				fmt.Sprintf(`Infinite loop detected. Usage of custom tag "%v" inside its own definition.`, tag.Name),
			))
		}
	}

	customElem, err := tag.NewElem(elem, b.app, bs.scope)
	if err != nil {
		panic(dom.ElementError(elem, fmt.Sprintf(`Cannot initialize the custom element, error in its Init(). Error: %v`, err.Error())))
	}

	for _, hattr := range elem.Attrs() {
		if hattr.Name[0] == AttrBindPrefix {
			attr := hattr.Name[1:]
			elem.RemoveAttr(hattr.Name)
			bindinfo += fmt.Sprintf("{%v: [%v]} ", hattr.Name, hattr.Value)

			field := strings.Split(attr, ".")[0]
			if ok, fieldName := tag.HasAttr(field); ok {
				b.processFieldBind(fieldName, hattr.Value, elem, bs, once, customElem)
			} else {
				b.processAttrBind(attr, hattr.Value, elem, bs, once)
			}
		}
	}

	err = customElem.PrepareContents(func(contentElems dom.Selection, once bool) {
		b.bindWithScope(contentElems, bs.scope, once, true, scopeElem)
	})

	if err != nil {
		panic(dom.ElementError(elem, err.Error()))
	}

	b.bindWithScope(elem, b.newModelScope(customElem.Model()), once, false, elem)

	if bindinfo != "" {
		old, _ := elem.Attr(BindInfoAttr)
		elem.SetAttr(BindInfoAttr, old+bindinfo)
	}
}

var (
	NameRegexp = regexp.MustCompile(`\w+`)
)

func checkName(strs []string) error {
	for _, str := range strs {
		if !NameRegexp.MatchString(str) {
			return fmt.Errorf("Invalid name %v", str)
		}
	}

	return nil
}

func parseBinderLHS(astr string) (binder string, args []string, err error) {
	lp := strings.IndexRune(astr, '(')
	if lp != -1 {
		if astr[len(astr)-1] != ')' {
			err = fmt.Errorf("Invalid syntax for left hand side of binding")
			return
		}

		binder = astr[:lp]
		args = strings.Split(astr[lp+1:len(astr)-1], ",")
	} else {
		binder = astr
		args = []string{}
	}

	err = checkName(append(args, binder))

	return
}

func (b *Binding) processBinderBind(astr, bstr string, elem dom.Selection, bs *bindScope, once bool) (removedElems []dom.Selection, err error) {
	binderName, args, err := parseBinderLHS(astr)
	if err != nil {
		bstrPanic(err.Error(), astr, elem)
	}

	removedElems = make([]dom.Selection, 0)

	if binder, ok := b.domBinders[binderName]; ok {
		binder = binder.BindInstance()

		roote, binds, v, err2 := bs.evaluate(bstr)
		if err2 != nil {
			err = err2
			return
		}

		domBind := DomBind{
			Elem:    drmElem{elem, &removedElems},
			Value:   v,
			Args:    args,
			binding: b,
			scope:   bs.scope,
		}

		if binds.size == 1 {
			fmodel := binds.slice[0].BindObj().FieldRefl
			err = binder.Watch(domBind, func(newVal string) {
				if !fmodel.CanSet() {
					bstrPanic("2-way data binding on unchangable field", bstr, elem)
				}
				fmodel.Set(reflect.ValueOf(newVal))
			})

			if err != nil {
				return
			}
		}

		//gopherjs:blocking
		err = binder.Bind(domBind)
		if err != nil {
			return
		}

		//gopherjs:blocking
		err = binder.Update(domBind)
		if err != nil {
			return
		}

		if !once {
			udb := domBind
			udb.Elem = elem

			err = b.watchModel(v, binds, roote, bs, func(newResult interface{}) {
				udb.OldValue = udb.Value
				udb.Value = newResult
				//gopherjs:blocking
				er := binder.Update(udb)
				if er != nil {
					b.app.ErrChanPut(BindingError{er})
				}
			})

			if err != nil {
				bstrPanic(err.Error(), bstr, elem)
				return
			}
		}

	} else {
		err = fmt.Errorf(`Dom binder "%v" does not exist`, binderName)
	}

	return
}

var (
	MustacheRegex = regexp.MustCompile("{{([^{}]+)}}")
)

func (b *Binding) processMustaches(elem dom.Selection, once bool, bs *bindScope) error {
	text := elem.Text()
	if strings.Index(text, "{{") == -1 {
		return nil
	}

	matches := MustacheRegex.FindAllStringSubmatch(text, -1)
	if matches != nil {
		splitted := MustacheRegex.Split(text, -1)

		textNodes := elem.NewEmptySelection()
		for i, m := range matches {
			cr, blist, v, err := bs.evaluate(m[1])
			if err != nil {
				return err
			}

			node := elem.NewTextNode(toString(v))

			if !once {
				err = b.watchModel(v, blist, cr, bs, func(val interface{}) {
					node.SetText(toString(val))
				})

				if err != nil {
					return err
				}
			}

			if splitted[i] != "" {
				bf := elem.NewTextNode(splitted[i])
				textNodes = textNodes.Add(bf)
			}

			textNodes = textNodes.Add(node)
		}

		if splitted[len(splitted)-1] != "" {
			bf := elem.NewTextNode(splitted[len(splitted)-1])
			textNodes = textNodes.Add(bf)
		}

		elem.ReplaceWith(textNodes)
	}

	return nil
}

func (b *Binding) bindDomRec(elem dom.Selection,
	bs *bindScope,
	once bool,
	additionalBinds []dom.Attr,
	scopeElem dom.Selection) (replaced dom.Selection) {

	replaced = elem

	/*if !elem.Exists() {
		return
	}*/

	isWrapper := icommon.IsWrapperElem(elem)
	var abinds []dom.Attr
	if isWrapper {
		abinds = make([]dom.Attr, 0)
	}

	isElement := elem.IsElement()

	var tag *com.Component
	isCustom := false
	if isElement {
		tag, isCustom = b.tm.GetComponent(elem)
	}

	attrs := make([]dom.Attr, 0)
	if additionalBinds != nil {
		attrs = append(attrs, additionalBinds...)
	}

	if isElement {
		attrs = append(attrs, elem.Attrs()...)
	}

	bindinfo := ""

	removedElems := make([][]dom.Selection, 0)
	// perform binding
	for _, attr := range attrs {
		astr, bstr := attr.Name[1:], attr.Value

		switch attr.Name[0] {
		case AttrBindPrefix:
			if isCustom || !isElement {
				continue
			}

			if isWrapper {
				elem.Children().SetAttr(attr.Name, attr.Value)
				continue
			}

			elem.RemoveAttr(attr.Name)
			b.processAttrBind(astr, bstr, elem, bs, once)

		case BinderBindPrefix:
			if isWrapper {
				abinds = append(abinds, attr)
				continue
			}

			if isElement {
				elem.RemoveAttr(attr.Name)
			}
			rmdElems, err := b.processBinderBind(astr, bstr, elem, bs, once)
			if err != nil {
				bstrPanic(err.Error(), bstr, elem)
			}

			removedElems = append(removedElems, rmdElems)

			/*if !elem.Exists() {
				return
			}*/

		default:
			continue
		}

		bindinfo += fmt.Sprintf("{%v: [%v]} ", attr.Name, attr.Value)
	}

	if elem.IsTextNode() {
		err := b.processMustaches(elem, once, bs)
		if err != nil {
			bstrPanic(err.Error(), elem.Text(), elem.Parent())
		}

		return
	}

	if isWrapper {
		conts := elem.Contents()
		elem.ReplaceWith(conts)

		//gopherjs:blocking
		conts.BEach(func(_ int, child dom.Selection) {
			if child.IsElement() {
				b.bindDomRec(child, bs, once, abinds, scopeElem)
			}
		})

		replaced = conts
		return
	} else {
		if bindinfo != "" {
			elem.SetAttr(BindInfoAttr, bindinfo)
		}

		if !isCustom {
			//gopherjs:blocking
			elem.Contents().BEach(func(_ int, child dom.Selection) {
				b.bindDomRec(child, bs, once, nil, scopeElem)
			})
		} else {
			b.bindCustomElem(elem, tag, bs, once, scopeElem)
		}
	}

	for _, l := range removedElems {
		for _, e := range l {
			e.Remove()
		}
	}

	return
}

func (b *Binding) newModelScope(model interface{}) *Scope {
	s := NewModelScope(model)
	s.Merge(b.scope)
	return s
}

func ScopeFromModels(models []interface{}) (s *Scope) {
	s = NewScope([]SymbolTable{})
	for _, model := range models {
		if model != nil {
			s.AddSymTables(NewModelSymbolTable(model))
		}
	}

	return
}

func (b *Binding) BindModels(rootElem dom.Selection, models []interface{}, once bool) {
	if !rootElem.Children().First().Exists() {
		panic("Invalid root element for bind. It must be a node in a real html document, a <wroot> or a child of <wroot>.")
	}

	s := ScopeFromModels(models)
	s.Merge(b.scope)

	b.bindWithScope(rootElem, s, once, false, rootElem)
}

func (b *Binding) Bind(rootElem dom.Selection, model interface{}, once bool) {
	b.BindModels(rootElem, []interface{}{model}, once)
}

func (b *Binding) rootList(rootElems dom.Selection, bindRoot bool) []dom.Selection {
	if bindRoot {
		return rootElems.Elements()
	}

	return rootElems.Contents().Elements()
}

func (b *Binding) bindWithScope(rootElems dom.Selection, s *Scope, once bool, bindRoot bool, scopeElem dom.Selection) {
	bs := &bindScope{s}
	elems := b.rootList(rootElems, bindRoot)

	for _, e := range elems {
		b.bindDomRec(e, bs, once, nil, scopeElem)
	}

	for _, re := range rootElems.Elements() {
		if re.IsElement() {
			re.SetProp(BoundAttr, true)
		}
	}
}
