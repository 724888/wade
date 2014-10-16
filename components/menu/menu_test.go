package menu

import (
	"testing"

	"github.com/phaikawl/wade/bind"
	"github.com/phaikawl/wade/dom/goquery"
	"github.com/stretchr/testify/require"
)

type (
	Scope struct {
		Choice string
	}
)

func TestSwitchMenu(t *testing.T) {
	b := bind.NewTestBindEngine()
	b.ComponentManager().RegisterComponents(Components())
	scope := &Scope{
		Choice: "a",
	}

	root := goquery.GetDom().NewFragment(`
	<wroot>
		<wSwitchMenu @Current="$Choice">
			<ul>
				<li case="a"></li>
				<li case="b"></li>
				<li case="c"></li>
			</ul>
		</wSwitchMenu>
	</wroot>
	`)

	b.Bind(root, scope, false)
	lis := root.Find("ul").Children().Elements()
	require.Equal(t, lis[0].HasClass("active"), true)
	require.Equal(t, lis[1].HasClass("active"), false)
	require.Equal(t, lis[2].HasClass("active"), false)

	scope.Choice = "b"
	b.Watcher().Digest(&scope.Choice)
	require.Equal(t, lis[0].HasClass("active"), false)
	require.Equal(t, lis[1].HasClass("active"), true)
	require.Equal(t, lis[2].HasClass("active"), false)

	scope.Choice = "kkf"
	b.Watcher().Digest(&scope.Choice)
	require.Equal(t, lis[0].HasClass("active"), false)
	require.Equal(t, lis[1].HasClass("active"), false)
	require.Equal(t, lis[2].HasClass("active"), false)
}
