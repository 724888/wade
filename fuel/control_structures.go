package main

import (
	"fmt"

	"github.com/gowade/html"
	"github.com/gowade/wade/utils/htmlutils"
)

func lnode(code string) *codeNode {
	return &codeNode{
		typ:  SliceVarCodeNode,
		code: code,
	}
}

func (c *HTMLCompiler) forLoopCode(node *html.Node, vda *varDeclArea) (*codeNode, error) {
	keyName, valName := "_", "_"
	var rangeAttr html.Attribute
	for _, attr := range node.Attr {
		switch attr.Key {
		case "k":
			keyName = attr.Val
		case "v":
			valName = attr.Val
		case "range":
			rangeAttr = attr

		default:
			return nil, fmt.Errorf(`Invalid attribute "%v" for for loop.`, attr.Key)
		}
	}

	if rangeAttr.Val == "" {
		return nil, fmt.Errorf(`for loop's "range" attribute cannot be empty.`)
	}

	if !rangeAttr.IsMustache {
		return nil, fmt.Errorf(
			`for loop's "range" attribute must be assigned to a `+
				`mustache representing a Go slice value. Got string value "%v" instead.`,
			rangeAttr.Val)

	}

	varName := vda.newVar("for")
	forVda := newVarDeclArea()
	apList := []*codeNode{{
		typ:  SliceVarCodeNode,
		code: varName,
	}}
	apList = append(apList, c.genChildren(node, forVda, nil)...)

	forVda.saveToCN()

	vda.setVarDecl(
		varName,
		ncn(fmt.Sprintf(`%v := %v{}`, varName, NodeListOpener)),
		&codeNode{
			typ:  BlockCodeNode,
			code: fmt.Sprintf(`for __k, __v := range %v`, rangeAttr.Val),
			children: []*codeNode{
				ncn(fmt.Sprintf(`%v, %v := __k, __v`, keyName, valName)),
				forVda.codeNode,
				&codeNode{
					typ:      AppendListCodeNode,
					code:     fmt.Sprintf("%v = ", varName),
					children: apList,
				},
			},
		})

	return lnode(fmt.Sprintf(varName)), nil
}

func (c *HTMLCompiler) ifControlCode(node *html.Node, vda *varDeclArea) (*codeNode, error) {
	var rcond html.Attribute
	for _, attr := range node.Attr {
		switch attr.Key {
		case "cond":
			rcond = attr

		default:
			return nil, fmt.Errorf(`Invalid attribute "%v" for if.`, attr.Key)
		}
	}

	cond := rcond.Val
	if cond == "" {
		return nil, fmt.Errorf(`if structure's "cond" attribute cannot be empty`)
	}

	if !rcond.IsMustache {
		return nil, fmt.Errorf(
			`if tag's "cond" attribute must be assigned to a `+
				`mustache respresenting a Go boolean expression. Got string value "%v" instead.`, cond)
	}

	varName := vda.newVar("if")
	ifVda := newVarDeclArea()

	child := c.generateRec(node.FirstChild, ifVda, nil)[0]
	ifVda.saveToCN()

	child.code = fmt.Sprintf("%v = ", varName) + child.code
	vda.setVarDecl(
		varName,
		ncn(fmt.Sprintf(`var %v vdom.Node`, varName)),
		&codeNode{
			typ:  BlockCodeNode,
			code: fmt.Sprintf(`if %v `, cond),
			children: []*codeNode{
				ifVda.codeNode,
				child,
			},
		})

	return ncn(varName), nil
}

func (c *HTMLCompiler) caseControlCode(node *html.Node, varName string, expr html.Attribute) (*codeNode, error) {
	caseVda := newVarDeclArea()

	child := c.generateRec(node.FirstChild, caseVda, nil)[0]
	caseVda.saveToCN()

	var caseCode string
	if expr.Val != "" {
		caseCode = fmt.Sprintf(`case %v:`, attributeValueCode(expr))
	} else {
		caseCode = "default:"
	}

	child.code = fmt.Sprintf("%v = ", varName) + child.code
	return &codeNode{
		typ:  WrapperCodeNode,
		code: caseCode,
		children: []*codeNode{
			caseVda.codeNode,
			child,
		},
	}, nil
}

func (compiler *HTMLCompiler) switchControlCode(node *html.Node, vda *varDeclArea) (*codeNode, error) {
	var exprAttr html.Attribute
	var hasExpr bool

	for _, attr := range node.Attr {
		switch attr.Key {
		case "expr":
			exprAttr = attr
			hasExpr = true

		default:
			return nil, fmt.Errorf(`Invalid attribute "%v" for switch.`, attr.Key)
		}
	}

	varName := vda.newVar("switch")
	var cases []*codeNode
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode {
			continue
		}

		htmlutils.RemoveGarbageTextChildren(node)
		var caseExprAttr html.Attribute
		switch c.Data {
		case "case":
			for _, attr := range c.Attr {
				if attr.Key == "expr" {
					caseExprAttr = attr
				} else {
					return nil, fmt.Errorf(`Invalid attribute "%v" for case tag.`, attr.Key)
				}
			}

			if caseExprAttr.Val == "" {
				return nil, fmt.Errorf(`case tag's "expr" attribute cannot be empty.`,
					caseExprAttr.Key)
			}

		case "default":
			for _, attr := range c.Attr {
				return nil, fmt.Errorf(`switch's default tag`+
					` shouldn't have any attributes, "%v" given.`, attr.Key)
			}

		default:
			return nil, fmt.Errorf(`switch tag's child elements` +
				` can only be "case" or "default" tag.`)
		}

		cn, err := compiler.caseControlCode(c, varName, caseExprAttr)
		if err != nil {
			return nil, err
		}
		cases = append(cases, cn)
	}

	var exprCode string
	if hasExpr {
		exprCode = attributeValueCode(exprAttr)
	}

	vda.setVarDecl(
		varName,
		ncn(fmt.Sprintf(`var %v vdom.Node`, varName)),
		&codeNode{
			typ:      BlockCodeNode,
			code:     fmt.Sprintf("switch %v ", exprCode),
			children: cases,
		})

	return ncn(varName), nil
}
