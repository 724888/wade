package bind

import (
	"fmt"
	"strings"
)

type PageManager interface {
	PageUrl(string, []interface{}) (string, error)
	Url(string) string
}

type UrlInfo struct {
	path    string
	fullUrl string
}

func RegisterUrlHelper(pm PageManager, b *Binding) {
	b.RegisterHelper("url", func(pageid string, params ...interface{}) UrlInfo {
		url, err := pm.PageUrl(pageid, params)
		if err != nil {
			panic(fmt.Errorf(`url helper error: "%v", when getting url for page "%v"`, err.Error(), pageid))
		}
		return UrlInfo{url, pm.Url(url)}
	})
}

func defaultHelpers() map[string]interface{} {
	return map[string]interface{}{
		"toUpper": strings.ToUpper,
		"toLower": strings.ToLower,
		"concat": func(s1, s2 string) string {
			return s1 + s2
		},
	}
}
