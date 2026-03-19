package pages

import (
	"fmt"
	"monitor/internal/parsing"
	"monitor/internal/requests"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type SnapshotType string

type Snapshot struct {
	TimeBuildingStarted time.Time

	Course     string
	Type       SnapshotType
	Url        string
	Err        error
	StatusCode int
	LoggedIn   bool
	Retries    int
	Doc        *goquery.Document
}

const CourseSnapshot = SnapshotType("course")
const SectionSnapshot = SnapshotType("section")

func BuildSnapshots(
	snapshotCh chan Snapshot,
	courseWg *sync.WaitGroup,
	sectionWg *sync.WaitGroup,
	reqTemplate *requests.Request,
	parser *parsing.Parser,
	course string,
	snapshotType SnapshotType,
	url string,
) error {
	if snapshotType == CourseSnapshot {
		courseWg.Add(1)
	} else if snapshotType == SectionSnapshot {
		sectionWg.Add(1)
	} else {
		return fmt.Errorf("unsupported snapshot type: %s", snapshotType)
	}

	go func() {
		snapshot := Snapshot{
			TimeBuildingStarted: time.Now().Local(),

			Course: course,
			Type:   snapshotType,
			Url:    url,
		}

		defer func() {
			snapshotCh <- snapshot

			if snapshotType == CourseSnapshot {
				courseWg.Done()
			} else {
				sectionWg.Done()
			}
		}()

		req := reqTemplate.DeepCopy()
		req.Url = url
		resp := req.Do()

		snapshot.StatusCode = resp.StatusCode
		snapshot.Retries = resp.Retries

		if resp.Err != nil {
			snapshot.Err = fmt.Errorf("error on request: ", resp.Err)
			return
		}

		if resp.StatusCode != http.StatusOK {
			return
		}

		snapshot.LoggedIn = !strings.Contains(resp.FinalUrl, "login")

		if !snapshot.LoggedIn {
			return
		}

		doc, err := parser.MakeDoc(string(resp.Body))

		if err != nil {
			snapshot.Err = fmt.Errorf("error on parsing: ", err)
			return
		}

		snapshot.Doc = doc

		if snapshot.Type == CourseSnapshot {
			links := parser.ExtractSectionLinks(snapshot.Doc)

			for _, link := range links {
				BuildSnapshots(
					snapshotCh,
					courseWg,
					sectionWg,
					reqTemplate,
					parser,
					course,
					SectionSnapshot,
					link,
				)
			}
		}
	}()

	return nil
}
