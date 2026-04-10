package parsing

import (
	"fmt"
	"monitor/internal/utils"
	"strings"
)

type Activity struct {
	Id    string
	Type  string
	Title string
	Link  string
}

type Activities []Activity

func (a Activity) Repr() string {
	return fmt.Sprintf(
		"%s %s: %s\n%s",
		a.Id,
		strings.ToUpper(a.Type),
		a.Title,
		a.Link,
	)
}

func (a Activity) ReprHtml() string {
	return fmt.Sprintf(
		"<code>%s</code> %s: %s\n%s",
		a.Id,
		strings.ToUpper(a.Type),
		a.Title,
		a.Link,
	)
}

func (a Activities) Repr() string {
	repr := ""

	for _, activity := range a {
		repr += activity.Repr() + "\n"
	}

	return repr
}

func (a Activities) ReprHtml() string {
	repr := ""

	for _, activity := range a {
		repr += activity.ReprHtml() + "\n"
	}

	return repr
}

func (a Activities) Diff(b Activities) (added, removed Activities) {
	return utils.SliceDiffComparable(a, b)
}
