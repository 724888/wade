package main

const (
	CreateElementOpener    = "vdom.NewElement"
	CreateTextNodeOpener   = "vdom.NewTextNode"
	CreateComElementOpener = "vdom.NewComElement"
	AttributeMapOpener     = "vdom.Attributes"
	NodeTypeName           = "vdom.Node"
	NodeListOpener         = "[]vdom.Node"
	RenderFuncOpener       = "func %vRender(stateData interface{}) *vdom.Element "
	RenderEmbedString      = "(this %v) "
	ComponentDataOpener    = "wade.Com"
	ComponentSetStateCode  = "this.%v = stateData.(%v)"
)

func Prelude(pkgName string) string {
	return `package ` + pkgName + `

// THIS FILE IS AUTOGENERATED BY WADE.GO FUEL
// CHANGES WILL BE OVERWRITTEN, PLEASE DUN
import (
	"fmt"

	"github.com/gowade/wade/vdom"
	"github.com/gowade/wade"
)

func init() {
	_, _, _ = fmt.Printf, vdom.NewElement, wade.CreateComponent
}

`
}
