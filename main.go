// Package main runs the Moodle monitoring loop.
package main

import (
	"errors"
	"fmt"
	"monitor/internal/config"
	"monitor/internal/messages"
	"monitor/internal/pages"
	"monitor/internal/parsing"
	"monitor/internal/requests"
	"monitor/internal/sessions"
	"monitor/internal/settings"
	statepkg "monitor/internal/state"
	"monitor/internal/timepkg"
	"monitor/internal/types"
	"monitor/internal/utils"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

const (
	// buildSnapshotsTimingEvent groups timing entries for snapshot building.
	buildSnapshotsTimingEvent = "snapshot building"
	// buildSnapshotsTimingStage is the default stage label for snapshot building.
	buildSnapshotsTimingStage = ""
	// snapshotProcessingTimingEvent groups timing entries for snapshot processing.
	snapshotProcessingTimingEvent = "snapshot processing"
)

var cfg = config.Load()
var stg *settings.Settings

var reqTemplate *requests.Request
var msgSender *messages.Sender
var state *statepkg.State
var parser *parsing.Parser
var sessionManager *sessions.SessionManager

var initialized = false
var stopping atomic.Bool
var lastLoggedOutMsg = time.Date(1970, time.January, 0, 0, 0, 0, 0, time.Local)
var lastTimedOutSessions = sessions.Sessions{}

// initialize wires the shared services used by the monitoring loop.
func initialize() {
	interruptRequestCallback := func() bool {
		return stopping.Load()
	}

	if stg == nil {
		stg = settings.Load(cfg)
	}

	if reqTemplate == nil {
		reqTemplate = requests.NewRequest(cfg)
		reqTemplate.Headers = map[string][]string{
			"User-Agent": []string{cfg.MoodleUserAgentHeader},
		}
		reqTemplate.Retries = cfg.MoodleRequestRetries
		reqTemplate.InterruptRequestCallback = interruptRequestCallback
		reqTemplate.Semaphore = requests.NewSemaphore(cfg)
	}

	if msgSender == nil {
		msgSender = messages.NewSender(
			cfg,
			interruptRequestCallback,
		)
	}

	if state == nil {
		state = statepkg.Load(cfg)
	}

	if parser == nil {
		parser = parsing.NewParser(cfg, reqTemplate)
	}

	if sessionManager == nil {
		sessionManager = sessions.NewSessionManager(stg)
	}

	initialized = true
}

// main initializes the monitor and runs the polling cycle until shutdown.
func main() {
	defer func() {
		fmt.Print(cfg.Sep)

		if r := recover(); r != nil {
			fmt.Printf("🕰️ [%s]\n❌ Panic: %s\n\n", time.Now().Local().Format(cfg.TimeFormat), r)
			fmt.Println("🪜 Stack trace:")
			fmt.Print(string(debug.Stack()))
		} else {
			fmt.Printf("🕰️ [%s]\n🛑 Stopped\n\n", time.Now().Local().Format(cfg.TimeFormat))
		}

		if initialized {
			err := state.Save(cfg)

			if err != nil {
				fmt.Printf(
					"🕰️ [%s]\n❌ %s\n\n",
					time.Now().Local().Format(cfg.TimeFormat),
					utils.Capitalize(err.Error()),
				)
			}
		}

		fmt.Print(cfg.Sep)
	}()

	fmt.Print(cfg.Sep)
	fmt.Printf("🕰️ [%s]\n⚙️ Initializing...\n\n", time.Now().Local().Format(cfg.TimeFormat))

	initialize()

	exitOnTimerCh := make(chan types.Void)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		stopping.Store(true)
		exitOnTimerCh <- types.Void{}
	}()

	fmt.Printf("🕰️ [%s]\n✅ Initialized successfully!\n\n", time.Now().Local().Format(cfg.TimeFormat))
	fmt.Print(cfg.Sep)

	for true {
		if stopping.Load() {
			return
		}

		fmt.Print(cfg.Sep)
		fmt.Print("⚙️ Starting cycle\n\n")

		timing := timepkg.NewTiming()
		timing.Start(buildSnapshotsTimingEvent, buildSnapshotsTimingStage)

		var snapshotCourseWg, snapshotSectionWg sync.WaitGroup
		snapshotCh := make(chan pages.Snapshot, cfg.SnapshotChannelBufferSize)

		for course, path := range stg.Courses {
			err := pages.BuildSnapshots(
				cfg,
				sessionManager,
				snapshotCh,
				&snapshotCourseWg,
				&snapshotSectionWg,
				reqTemplate,
				parser,
				course,
				pages.CourseSnapshot,
				cfg.MoodleBaseUrl+path,
			)

			if err != nil {
				panic("build snapshot error: " + err.Error())
			}
		}

		go func() {
			snapshotCourseWg.Wait()
			snapshotSectionWg.Wait()
			close(snapshotCh)
			timing.End(buildSnapshotsTimingEvent, buildSnapshotsTimingStage)
		}()

		fmt.Print(cfg.Sep)

		courseActivitySetMap := make(map[string]types.Set[parsing.Activity])
		sectionCounters := make(map[string]int)
		errorCounters := make(map[string]int)
		badStatusCounters := make(map[string]int)
		snapshotProcessingStage := -1

		for snapshot := range snapshotCh {
			if stopping.Load() {
				return
			}

			func() {
				// Keep stage identifiers unique so per-snapshot timing entries stay distinct.
				snapshotProcessingStage++
				timing.Start(snapshotProcessingTimingEvent, strconv.Itoa(snapshotProcessingStage))

				defer func() {
					timing.End(snapshotProcessingTimingEvent, strconv.Itoa(snapshotProcessingStage))
					fmt.Print(cfg.Sep)
				}()

				activitySet, ok := courseActivitySetMap[snapshot.Course]

				if !ok {
					activitySet = types.NewSet[parsing.Activity]()
					courseActivitySetMap[snapshot.Course] = activitySet
				}

				sectionCounters[snapshot.Course]++

				fmt.Print(cfg.Sep)
				fmt.Printf("🕰️ [%s]\n", time.Now().Local().Format(cfg.TimeFormat))
				fmt.Println("⚙️ Snapshot processing started")
				fmt.Printf("🕰️ Snapshot building started at: %s\n", snapshot.TimeBuildingStarted.Format(cfg.TimeFormat))
				fmt.Printf("⏳ Time spent on request: %v\n", snapshot.RequestDuration)
				fmt.Printf("⏳ Time spent on building: %v\n", snapshot.BuildingDuration)
				fmt.Printf("🏫 Course: %s\n", snapshot.Course)
				fmt.Printf("📦 Snapshot type: %s\n", snapshot.Type)
				fmt.Printf("📍 Url: %s\n", snapshot.Url)
				fmt.Printf("🔄 Retries: %d\n", snapshot.Retries)
				fmt.Printf("⚠️ Timed out sessions: %d\n\n", snapshot.TimedOutSessions)

				if snapshot.Retries > 0 {
					msgSender.Do(fmt.Sprintf("🔄 Request to %s retried %d times!", snapshot.Url, snapshot.Retries), true)
				}

				if snapshot.Err == nil {
					fmt.Println("✅ No error")
				} else if errors.Is(snapshot.Err, sessions.NoValidSessionsError) {
					msgSender.Do("‼️ No valid sessions.", true)
					panic(snapshot.Err)
				} else {
					errorCounters[snapshot.Course]++
					text := fmt.Sprintf("❌ Error: %v", snapshot.Err)

					fmt.Print(text + "\n\n")

					if snapshot.Type == pages.CourseSnapshot {
						msgSender.Do(fmt.Sprintf("%s [%s]", text, snapshot.Course), true)
					}

					return
				}

				if snapshot.StatusCode == http.StatusOK {
					fmt.Println("✅ Status ok")
				} else {
					badStatusCounters[snapshot.Course]++
					text := fmt.Sprintf("❌ Bad response status: %d", snapshot.StatusCode)

					fmt.Print(text + "\n\n")

					if snapshot.Type == pages.CourseSnapshot {
						msgSender.Do(fmt.Sprintf("%s [%s]", text, snapshot.Course), true)
					}

					return
				}

				if snapshot.LoggedIn {
					fmt.Print("✅ Logged in\n\n")
				} else {
					text := "❌ Logged out"

					fmt.Print(text + "\n\n")

					if snapshot.Type == pages.CourseSnapshot &&
						time.Since(lastLoggedOutMsg) > time.Duration(cfg.LoggedOutMsgCooldownSeconds)*time.Second {

						msgSender.Do(fmt.Sprintf("%s [%s]", text, snapshot.Course), true)
						lastLoggedOutMsg = time.Now().Local()
					}

					return
				}

				activitySet.Merge(parser.ExtractActivities(snapshot.Doc))
			}()
		}

		fmt.Print(cfg.Sep)
		fmt.Printf("🕰️ [%s]\n", time.Now().Local().Format(cfg.TimeFormat))
		fmt.Println("🏁 Cycle finished")

		pagesVisited := 0

		for _, sections := range sectionCounters {
			pagesVisited += sections
		}

		fmt.Printf("📍 Pages visited: %d\n\n", pagesVisited)

		for course, _ := range stg.Courses {
			func() {
				fmt.Print(cfg.Sep)

				activities := parsing.Activities{}

				// Continue the same timing sequence for per-course summary processing.
				snapshotProcessingStage++
				timing.Start(snapshotProcessingTimingEvent, strconv.Itoa(snapshotProcessingStage))

				defer func() {
					timing.End(snapshotProcessingTimingEvent, strconv.Itoa(snapshotProcessingStage))
					fmt.Print(cfg.Sep)

					if len(activities) > 0 {
						state.Storage.Set(course, activities)
					}
				}()

				sectionCounter := sectionCounters[course]
				errorCounter := errorCounters[course]
				badStatusCounter := badStatusCounters[course]

				fmt.Printf("🏫 Course: %s\n", course)
				fmt.Printf("⤵️ Sections: %d\n", sectionCounter)

				if errorCounter == 0 {
					fmt.Println("✅ No errors on course")
				} else {
					fmt.Printf("❌ Errors: %d\n", errorCounter)
				}

				if badStatusCounter == 0 {
					fmt.Print("✅ No bad response statuses on course\n\n")
				} else {
					fmt.Printf("❌ Bad response statuses: %d\n\n", badStatusCounter)
				}

				activitySet := courseActivitySetMap[course]

				if activitySet.Size() == 0 {
					fmt.Print("⚠️ Activities have not been extracted\n\n")
					return
				}

				activities = parsing.Activities(activitySet.ToSlice())

				if !state.Storage.Exists(course) {
					fmt.Print("🧠 New course content remembered\n\n")
					return
				}

				addedActivities, removedActivities := state.Storage.Diff(course, activities)

				if len(addedActivities) == 0 && len(removedActivities) == 0 {
					fmt.Print("✅ Nothing new\n\n")
					return
				}

				fmt.Print("🚨 Changes\n\n")
				text := fmt.Sprintf("🚨 Changes in %s\n\n", course)

				if len(addedActivities) > 0 {
					fmt.Println(fmt.Sprintf("🆕 Added:\n\n%s", addedActivities.Repr()))
					text += fmt.Sprintf("🆕 Added:\n\n%s\n", addedActivities.ReprHtml())
				}

				if len(removedActivities) > 0 {
					fmt.Println(fmt.Sprintf("🗑️ Removed:\n\n%s", removedActivities.Repr()))
					text += fmt.Sprintf("🗑️ Removed:\n\n%s\n", removedActivities.ReprHtml())
				}

				msgSender.Do(text, false)
			}()
		}

		fmt.Print(cfg.Sep)

		allTimedOutSessions := sessionManager.GetTimedOutSessions()
		newTimedOutSessions, _ := lastTimedOutSessions.Diff(allTimedOutSessions)

		if len(newTimedOutSessions) > 0 {
			msgSender.Do(fmt.Sprintf("⚠️ New timed out sessions:\n%s", newTimedOutSessions.Repr()), true)
		}

		if len(allTimedOutSessions) > 0 {
			fmt.Printf("⚠️ Timed out sessions:\n\n%s", allTimedOutSessions.Repr())
			lastTimedOutSessions = allTimedOutSessions
		}

		durationRepr, err := timing.ReprAvailableDurationsOfEvents()

		if err != nil {
			panic("error on timing duration representation: " + err.Error())
		}

		fmt.Printf("%s\n", durationRepr)
		fmt.Print(cfg.Sep)
		fmt.Print(cfg.Sep)
		fmt.Print("\n\n\n\n\n\n")
		fmt.Print(cfg.Sep)

		timer := time.NewTimer(time.Duration(stg.MonitorRequestCycleCooldownSeconds) * time.Second)

		select {
		case <-exitOnTimerCh:
			return

		case <-timer.C:
		}
	}
}
