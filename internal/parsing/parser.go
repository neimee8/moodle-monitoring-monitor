package parsing

import (
	"monitor/internal/config"
	"monitor/internal/requests"
	"monitor/internal/types"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type Parser struct {
	cfg *config.Config
	req *requests.Request
}

func NewParser(cfg *config.Config, reqTemplate *requests.Request) *Parser {
	return &Parser{
		cfg: cfg,
		req: reqTemplate.DeepCopy(),
	}
}

func (p *Parser) MakeDoc(html string) (*goquery.Document, error) {
	return goquery.NewDocumentFromReader(strings.NewReader(html))
}

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

func (p *Parser) ExtractActivities(doc *goquery.Document) types.Set[Activity] {
	activities := types.NewSet[Activity]()

	doc.Find("li.activity-wrapper").Each(func(_ int, li *goquery.Selection) {
		classAttr, _ := li.Attr("class")
		classes := strings.Fields(classAttr)

		modType := ""

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
