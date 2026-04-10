package parsing

import (
	"monitor/internal/config"
	"monitor/internal/requests"
	"monitor/internal/types"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Parser extracts links and activities from Moodle HTML pages.
type Parser struct {
	cfg *config.Config
	req *requests.Request
}

// NewParser returns a parser configured for the current Moodle instance.
func NewParser(cfg *config.Config, reqTemplate *requests.Request) *Parser {
	return &Parser{
		cfg: cfg,
		req: reqTemplate.DeepCopy(),
	}
}

// MakeDoc parses raw HTML into a goquery document.
func (p *Parser) MakeDoc(html string) (*goquery.Document, error) {
	return goquery.NewDocumentFromReader(strings.NewReader(html))
}

// ExtractSectionLinks returns unique canonical section links found on a course page.
func (p *Parser) ExtractSectionLinks(doc *goquery.Document) []string {
	seen := types.NewSet[string]()
	links := make([]string, 0)

	doc.Find(`h3.sectionname > a[href*="/course/view.php"][href*="section="]`).Each(func(_ int, s *goquery.Selection) {
		href, ok := s.Attr("href")

		if !ok || href == "" {
			return
		}

		u, err := url.Parse(href)

		if err != nil {
			return
		}

		if u.Host != p.cfg.MoodleHost || u.Path != "/course/view.php" {
			return
		}

		q := u.Query()
		id := q.Get("id")
		section := q.Get("section")

		if id == "" || section == "" {
			return
		}

		// Normalize the query so equivalent links collapse to the same canonical URL.
		canonicalQuery := url.Values{}
		canonicalQuery.Set("id", id)
		canonicalQuery.Set("section", section)

		u.RawQuery = canonicalQuery.Encode()
		u.Fragment = ""

		link := u.String()

		if seen.Exists(link) {
			return
		}

		seen.Add(link)
		links = append(links, link)
	})

	return links
}

// ExtractActivities returns the deduplicated activities found on a page.
func (p *Parser) ExtractActivities(doc *goquery.Document) types.Set[Activity] {
	activities := types.NewSet[Activity]()

	doc.Find("li.activity-wrapper").Each(func(_ int, li *goquery.Selection) {
		classAttr, _ := li.Attr("class")
		classes := strings.Fields(classAttr)

		modType := ""

		// Moodle encodes the activity type in a CSS class with the modtype_ prefix.
		for _, className := range classes {
			if strings.HasPrefix(className, "modtype_") {
				modType = strings.TrimPrefix(className, "modtype_")
				break
			}
		}

		a := li.Find("a[href]").First()

		if a.Length() == 0 {
			return
		}

		href, ok := a.Attr("href")

		if !ok || href == "" {
			return
		}

		title := strings.TrimSpace(a.Text())
		parsed, err := url.Parse(href)

		if err != nil {
			return
		}

		activities.Add(Activity{
			Id:    parsed.Query().Get("id"),
			Type:  modType,
			Title: title,
			Link:  href,
		})
	})

	return activities
}
