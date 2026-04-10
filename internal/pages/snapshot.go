package pages

import (
	"fmt"
	"monitor/internal/config"
	"monitor/internal/parsing"
	"monitor/internal/requests"
	"monitor/internal/sessions"
	"monitor/internal/timepkg"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// SnapshotType identifies the kind of page snapshot that was built.
type SnapshotType string

// Snapshot contains the result of a single page snapshot build.
type Snapshot struct {
	TimeBuildingStarted time.Time
	RequestDuration     time.Duration
	BuildingDuration    time.Duration

	Course           string
	Type             SnapshotType
	Url              string
	Err              error
	StatusCode       int
	LoggedIn         bool
	Retries          int
	TimedOutSessions int
	Doc              *goquery.Document
}

const (
	// CourseSnapshot marks a snapshot built from a course page.
	CourseSnapshot = SnapshotType("course")
	// SectionSnapshot marks a snapshot built from a section page.
	SectionSnapshot = SnapshotType("section")
)

const (
	// buildTimingEvent groups timing entries for the full snapshot build.
	buildTimingEvent = "build"
	// buildTimingStage is the default stage label for snapshot building.
	buildTimingStage = ""
	// requestTimingEvent groups timing entries for HTTP requests within a build.
	requestTimingEvent = "request"
)

// BuildSnapshots builds a snapshot for the given page and recursively schedules section snapshots for course pages.
func BuildSnapshots(
	cfg *config.Config,
	sessionManager *sessions.SessionManager,
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

			// The counter is incremented before each attempt, so start at -1 to report zero timeouts on the first try.
			TimedOutSessions: -1,
		}

		timing := timepkg.NewTiming()
		timing.Start(buildTimingEvent, buildTimingStage)

		defer func() {
			timing.End(buildTimingEvent, buildTimingStage)
			snapshot.BuildingDuration, _ = timing.DurationOfEvent(buildTimingEvent)
			snapshot.RequestDuration, _ = timing.DurationOfEvent(requestTimingEvent)

			snapshotCh <- snapshot

			if snapshotType == CourseSnapshot {
				courseWg.Done()
			} else {
				sectionWg.Done()
			}
		}()

		req := reqTemplate.DeepCopy()
		req.Url = url
		var resp *requests.Response

		for true {
			snapshot.TimedOutSessions++
			session, err := sessionManager.GetSession()

			if err != nil {
				snapshot.LoggedIn = false
				snapshot.Err = fmt.Errorf("error on session selection: %w", err)
				break
			}

			req.Cookies = map[string]string{
				cfg.MoodleSessionCookieName: session.Value,
			}

			timing.Start(requestTimingEvent, strconv.Itoa(snapshot.TimedOutSessions))
			resp = req.Do()
			timing.End(requestTimingEvent, strconv.Itoa(snapshot.TimedOutSessions))

			snapshot.StatusCode = resp.StatusCode
			snapshot.Retries = resp.Retries

			if resp.Err != nil {
				snapshot.Err = fmt.Errorf("error on request: %w", resp.Err)
				return
			}

			if resp.StatusCode != http.StatusOK {
				return
			}

			snapshot.LoggedIn = !strings.Contains(resp.FinalUrl, "login")

			if snapshot.LoggedIn {
				break
			}

			sessionManager.TimedOut(session)
		}

		if !snapshot.LoggedIn {
			return
		}

		doc, err := parser.MakeDoc(string(resp.Body))

		if err != nil {
			snapshot.Err = fmt.Errorf("error on parsing: %w", err)
			return
		}

		snapshot.Doc = doc

		if snapshot.Type == CourseSnapshot {
			// Course pages fan out into section snapshots after the main page is parsed.
			links := parser.ExtractSectionLinks(snapshot.Doc)

			for _, link := range links {
				BuildSnapshots(
					cfg,
					sessionManager,
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
