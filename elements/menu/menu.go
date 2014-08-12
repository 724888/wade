package menu

import (
	"fmt"
	"strings"

	wd "github.com/phaikawl/wade"
)

type SwitchMenu struct {
	Current     string
	ActiveClass string
}

func (sm *SwitchMenu) Init(ce *wd.CustomElem) error {
	if sm.Current == "" {
		return fmt.Errorf(`"Current" attribute must be set.`)
	}

	cl := ce.Contents.Children("")
	if cl.Length != 1 || !cl.First().Is("ul") {
		return fmt.Errorf("switchmenu's contents must have exactly 1 child which is an <ul> element.")
	}

	for _, item := range wd.ToElemSlice(cl.First().Children("")) {
		if wd.IsWrapperElem(item) {
			item = item.Children("li").First()
		}

		if !item.Is("li") {
			return fmt.Errorf(`Direct children of the <ul> must be <li>.`)
		}

		if casestr := item.Attr("case"); casestr != "" {
			cases := strings.Split(casestr, " ")
			accepted := false
			for _, id := range cases {
				if strings.TrimSpace(id) == sm.Current {
					accepted = true
					item.AddClass(sm.ActiveClass)
					break
				}
			}

			if !accepted {
				item.RemoveClass(sm.ActiveClass)
			}
		} else {
			return fmt.Errorf(`"case" attribute must be set for each <li>.`)
		}
	}

	return nil
}

func Spec() map[string]interface{} {
	return map[string]interface{}{
		"switchmenu": SwitchMenu{},
	}
}
