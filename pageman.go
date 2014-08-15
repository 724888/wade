package wade

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/gopherjs/gopherjs/js"
	jq "github.com/gopherjs/jquery"
	"github.com/phaikawl/wade/bind"
)

const (
	WadeReservedPrefix = "wade-rsvd-"
	WadeExcludeAttr    = WadeReservedPrefix + "exclude"

	GlobalDisplayScope = "__global__"
)

var (
	gRouteParamRegexp = regexp.MustCompile(`\:\w+`)
)

// PageManager is Page Manager
type pageManager struct {
	appEnv       AppEnv
	router       js.Object
	currentPage  *page
	startPageId  string
	basePath     string
	notFoundPage *page
	container    jq.JQuery
	tcontainer   jq.JQuery

	binding        *bind.Binding
	tm             *custagMan
	pc             *PageCtrl
	displayScopes  map[string]displayScope
	globalDs       *globalDisplayScope
	formattedTitle string
}

func newPageManager(appEnv AppEnv, startPage, basePath string,
	tcontainer jq.JQuery, binding *bind.Binding, tm *custagMan) *pageManager {

	container := gJQ("<div class='wade-wrapper'></div>")
	container.AppendTo(gJQ("body"))
	pm := &pageManager{
		appEnv:        appEnv,
		router:        js.Global.Get("RouteRecognizer").New(),
		currentPage:   nil,
		basePath:      basePath,
		startPageId:   startPage,
		notFoundPage:  nil,
		container:     container,
		tcontainer:    tcontainer,
		binding:       binding,
		tm:            tm,
		displayScopes: make(map[string]displayScope),
		globalDs:      &globalDisplayScope{},
	}

	pm.displayScopes[GlobalDisplayScope] = pm.globalDs
	return pm
}

// Set the target element that receives Wade's real HTML output,
// by default the container is <body>
func (pm *pageManager) SetOutputContainer(elementId string) {
	parent := gJQ("#" + elementId)
	if parent.Length == 0 {
		panic(fmt.Sprintf("No such element #%v.", elementId))
	}

	parent.Append(pm.container)
}

func (pm *pageManager) CurrentPageId() string {
	return pm.currentPage.id
}

func (pm *pageManager) cutPath(path string) string {
	if strings.HasPrefix(path, pm.basePath) {
		path = path[len(pm.basePath):]
	}
	return path
}

func (pm *pageManager) page(id string) *page {
	if ds, hasDs := pm.displayScopes[id]; hasDs {
		if page, ok := ds.(*page); ok {
			return page
		}
	}

	panic(fmt.Sprintf(`No such page "%v" found.`, id))
}

func (pm *pageManager) displayScope(id string) displayScope {
	if ds, ok := pm.displayScopes[id]; ok {
		return ds
	}
	panic(fmt.Sprintf(`No such page or page group "%v" found.`, id))
}

func (pm *pageManager) SetNotFoundPage(pageId string) {
	pm.notFoundPage = pm.page(pageId)
}

// Url returns the full url for a path
func (pm *pageManager) Url(path string) string {
	return pm.basePath + path
}

func documentUrl() string {
	location := gHistory.Get("location")
	if location.IsNull() || location.IsUndefined() {
		location = js.Global.Get("document").Get("location")
	}
	return location.Get("pathname").Str()
}

func (pm *pageManager) setupPageOnLoad() {
	path := pm.cutPath(documentUrl())
	if path == "/" {
		startPage := pm.page(pm.startPageId)
		path = startPage.path
		gHistory.Call("replaceState", nil, startPage.title, pm.Url(path))
	}
	pm.updatePage(path, false)
}

func (pm *pageManager) RedirectToPage(page string, params ...interface{}) {
	url, err := pm.PageUrl(page, params...)
	if err != nil {
		panic(err.Error())
	}
	pm.updatePage(url, true)
}

func (pm *pageManager) RedirectToUrl(url string) {
	js.Global.Get("window").Set("location", url)
}

func (pm *pageManager) prepare() {
	// preprocess wsection elements
	pm.tcontainer.Find("wsection").Each(func(_ int, e jq.JQuery) {
		name := strings.TrimSpace(e.Attr("name"))
		if name == "" {
			panic(`Error: a <wsection> doesn't have or have empty name`)
		}
		for _, c := range name {
			if !unicode.IsDigit(c) && !unicode.IsLetter(c) && c != '-' {
				panic(fmt.Sprintf("Invalid character '%q' in wsection name.", c))
			}
		}
		e.SetAttr("id", WadeReservedPrefix+name)
	})

	if pm.container.Length == 0 {
		panic(fmt.Sprintf("Cannot find the page container #%v.", pm.container))
	}

	gJQ(js.Global.Get("window")).On("popstate", func() {
		pm.updatePage(documentUrl(), false)
	})

	//Handle link events
	pm.container.On(jq.CLICK, "a", func(e jq.Event) {
		a := gJQ(e.Target)

		pagepath := a.Attr(bind.WadePageAttr)
		if pagepath == "" { //not a wade page link, let the browser do its job
			return
		}

		e.PreventDefault()

		pm.updatePage(pagepath, true)
	})

	pm.setupPageOnLoad()
}

func walk(elem jq.JQuery, pm *pageManager) {
	elem.Children("").Each(func(_ int, e jq.JQuery) {
		belong := e.Attr("w-belong")
		if belong == "" {
			walk(e, pm)
		} else {
			if ds, ok := pm.displayScopes[belong]; ok {
				if ds.hasPage(pm.currentPage.id) {
					walk(e, pm)
				} else {
					e.Remove()
				}
			} else {
				panic(fmt.Sprintf(`Invalid value "%v" for w-belong, no such page or page group is registered.`, belong))
			}
		}
	})
}

func (pm *pageManager) updatePage(url string, pushState bool) {
	url = pm.cutPath(url)
	matches := pm.router.Call("recognize", url)
	println("path: " + url)
	if matches.IsUndefined() || matches.Length() == 0 {
		if pm.notFoundPage != nil {
			pm.updatePage(pm.notFoundPage.path, false)
		} else {
			panic("Page not found. No 404 handler declared.")
		}
	}

	match := matches.Index(0)
	pageId := match.Get("handler").Invoke().Str()
	page := pm.page(pageId)
	if pushState {
		gHistory.Call("pushState", nil, page.title, pm.Url(url))
	}
	params := make(map[string]interface{})
	prs := match.Get("params")
	if !prs.IsUndefined() {
		params = prs.Interface().(map[string]interface{})
	}

	if pm.currentPage != page {
		pm.formattedTitle = page.title
		pm.container.Hide()
		pm.currentPage = page
		pcontents := pm.tcontainer.Clone()
		walk(pcontents, pm)
		pm.container.SetHtml(pcontents.Html())

		pm.container.Find("wrep").Each(func(_ int, e jq.JQuery) {
			e.Remove()
			pm.container.Find("#" + WadeReservedPrefix + e.Attr("target")).
				SetHtml(e.Html())
		})

		pm.container.Find("wsection").Each(func(_ int, e jq.JQuery) {
			e.Children("").First().Unwrap()
		})

		go func() {
			pm.bind(params)
			wrapperElemsUnwrap(pm.container)
			pm.container.Show()
		}()

		tElem := gJQ("<title>").SetHtml(pm.formattedTitle)
		oElem := gJQ("head").Find("title")
		if oElem.Length == 0 {
			gJQ("head").Append(tElem)
		} else {
			oElem.ReplaceWith(tElem)
		}
	}
}

// PageUrl returns the url and route parameters for the specified pageId
func (pm *pageManager) PageUrl(pageId string, params ...interface{}) (u string, err error) {
	err = nil
	page := pm.page(pageId)

	n := len(params)
	if n == 0 {
		u = page.path
		return
	}

	i := 0
	repl := func(src string) (out string) {
		out = src
		if i >= n {
			err = fmt.Errorf("Not enough parameters supplied for the route.")
			return
		}
		out = fmt.Sprintf("%v", params[i])
		i += 1
		return
	}

	u = gRouteParamRegexp.ReplaceAllStringFunc(page.path, repl)
	if i != n {
		err = fmt.Errorf("Too many parameters supplied for the route")
		return
	}
	return
}

func (pm *pageManager) BasePath() string {
	return pm.basePath
}

func (pm *pageManager) newPageCtrl(page *page, params map[string]interface{}) *PageCtrl {
	return &PageCtrl{
		app:     pm.appEnv,
		pm:      pm,
		p:       page,
		params:  params,
		helpers: make(map[string]interface{}),
	}
}

func (pm *pageManager) CurrentPage() ThisPage {
	return pm.pc
}

func (pm *pageManager) bind(params map[string]interface{}) {
	models := make([]interface{}, 0)
	controllers := make([]PageControllerFunc, 0)

	pc := pm.newPageCtrl(pm.currentPage, params)

	add := func(ds displayScope) {
		if ctrls := ds.Controllers(); ctrls != nil {
			for _, controller := range ctrls {
				controllers = append(controllers, controller)
			}
		}
	}

	add(pm.globalDs)
	for _, grp := range pm.currentPage.groups {
		add(grp)
	}
	add(pm.currentPage)

	if len(controllers) > 0 {
		completeChan := make(chan bool, 1)
		queueChan := make(chan bool, len(controllers))
		for _, controller := range controllers {
			go func(controller PageControllerFunc) {
				//gopherjs:blocking
				models = append(models, controller(pc))
				queueChan <- true
				if len(queueChan) == len(controllers) {
					completeChan <- true
				}
			}(controller)
		}
		<-completeChan
	}

	if len(models) == 0 {
		pm.binding.Bind(pm.container, nil, true, false)
	} else {
		pm.binding.BindModels(pm.container, models, false, false)
	}

	pm.pc = pc
}

// RegisterController adds a new controller function for the specified
// page / page group.
func (pm *pageManager) registerController(displayScope string, fn PageControllerFunc) {
	ds := pm.displayScope(displayScope)
	ds.addController(fn)
}

// RegisterDisplayScopes registers the given map of pages and page groups
func (pm *pageManager) registerDisplayScopes(m map[string]DisplayScope) {
	for id, item := range m {
		if id == "" {
			panic("id of page/page group cannot be empty.")
		}

		pm.displayScopes[id] = item.Register(id, pm)
	}
}
