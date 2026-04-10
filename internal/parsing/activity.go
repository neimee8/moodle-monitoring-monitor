package parsing

import (
	"fmt"
	"monitor/internal/utils"
	"strings"
)

// Activity describes a single Moodle course activity.
type Activity struct {
	Id    string
	Type  string
	Title string
	Link  string
}

// Activities is a list of Moodle course activities.
type Activities []Activity

// Repr returns a plain-text representation of the activity.
func (a Activity) Repr() string {
	return fmt.Sprintf(
		"%s %s: %s\n%s",
		a.Id,
		strings.ToUpper(a.Type),
		a.Title,
		a.Link,
	)
}

// ReprHtml returns an HTML-formatted representation of the activity.
func (a Activity) ReprHtml() string {
	return fmt.Sprintf(
		"<code>%s</code> %s: %s\n%s",
		a.Id,
		strings.ToUpper(a.Type),
		a.Title,
		a.Link,
	)
}

// Repr returns a plain-text representation of all activities.
func (a Activities) Repr() string {
	repr := ""

	for _, activity := range a {
		repr += activity.Repr() + "\n"
	}

	return repr
}

// ReprHtml returns an HTML-formatted representation of all activities.
func (a Activities) ReprHtml() string {
	repr := ""

	for _, activity := range a {
		repr += activity.ReprHtml() + "\n"
	}

	return repr
}

// Diff returns activities added in b and removed from a.
func (a Activities) Diff(b Activities) (added, removed Activities) {
	return utils.SliceDiffComparable(a, b)
}
