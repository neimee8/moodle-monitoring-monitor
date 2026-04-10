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

type SnapshotType string

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

const CourseSnapshot = SnapshotType("course")
const SectionSnapshot = SnapshotType("section")

const buildTimingEvent = "build"
const buildTimingStage = ""
const requestTimingEvent = "request"

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

			Course:           course,
			Type:             snapshotType,
			Url:              url,
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
