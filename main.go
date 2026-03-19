package main

import (
	"fmt"
	"monitor/internal/config"
	"monitor/internal/messages"
	"monitor/internal/pages"
	"monitor/internal/parsing"
	"monitor/internal/requests"
	"monitor/internal/settings"
	statepkg "monitor/internal/state"
	"monitor/internal/timepkg"
	"monitor/internal/types"
	"monitor/internal/utils"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

const buildSnapshotsTimingKey = "snapshot building"
const snapshotProcessingTimingKey = "snapshot processing"

var cfg = config.Load()
var stg *settings.Settings

var reqTemplate *requests.Request
var msgSender *messages.Sender
var state *statepkg.State
var parser *parsing.Parser
var timing *timepkg.Timing

var initialized = false
var stopping atomic.Bool
var lastLoggedOutMsg = time.Date(1970, time.January, 0, 0, 0, 0, 0, time.Local)

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
			"User-Agent": []string{"Mozilla/5.0"},
		}
		reqTemplate.Cookies = map[string]string{
			"MoodleSession": stg.MoodleSession,
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

	if timing == nil {
		timing = timepkg.NewTiming()
	}

	initialized = true
}

func main() {
	defer func() {
		fmt.Print(cfg.Sep)

		if r := recover(); r != nil {
			fmt.Printf("🕰️  [%s]\n❌ Panic: %s\n\n", time.Now().Local().Format(cfg.TimeFormat), r)
			fmt.Println("🪜 Stack trace:")
			fmt.Print(string(debug.Stack()))
		} else {
			fmt.Printf("🕰️  [%s]\n🛑 Stopped\n\n", time.Now().Local().Format(cfg.TimeFormat))
		}

		if initialized {
			err := state.Save(cfg)

			if err != nil {
				fmt.Printf(
					"🕰️  [%s]\n❌ %s\n\n",
					time.Now().Local().Format(cfg.TimeFormat),
					utils.Capitalize(err.Error()),
				)
			}
		}

		fmt.Print(cfg.Sep)
	}()

	fmt.Print(cfg.Sep)
	fmt.Printf("🕰️  [%s]\n⚙️ Initializing...\n\n", time.Now().Local().Format(cfg.TimeFormat))

	initialize()

	exitOnTimerCh := make(chan types.Void)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		stopping.Store(true)
		exitOnTimerCh <- types.Void{}
	}()

	fmt.Printf("🕰️  [%s]\n✅ Initialized successfully!\n\n", time.Now().Local().Format(cfg.TimeFormat))
	fmt.Print(cfg.Sep)

	for true {
		if stopping.Load() {
			return
		}

		fmt.Print(cfg.Sep)
		fmt.Print("⚙️ Starting cycle\n\n")

		timing.Start(buildSnapshotsTimingKey, "")
		var snapshotCourseWg, snapshotSectionWg sync.WaitGroup
		snapshotCh := make(chan pages.Snapshot, cfg.SnapshotChannelBufferSize)

		for course, path := range stg.Courses {
			err := pages.BuildSnapshots(
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
			timing.End(buildSnapshotsTimingKey, "")
		}()

		fmt.Print(cfg.Sep)

		courseActivitySetMap := make(map[string]types.Set[types.Activity])
		sectionCounters := make(map[string]int)
		errorCounters := make(map[string]int)
		badStatusCounters := make(map[string]int)
		i := 0

		for snapshot := range snapshotCh {
			if stopping.Load() {
				return
			}

			timing.Start(snapshotProcessingTimingKey, string(i))

			activitySet, ok := courseActivitySetMap[snapshot.Course]

			if !ok {
				activitySet = types.NewSet[types.Activity]()
				courseActivitySetMap[snapshot.Course] = activitySet
			}

			sectionCounters[snapshot.Course]++

			fmt.Printf(cfg.Sep)
			fmt.Printf("🕰️  [%s]\n", time.Now().Local().Format(cfg.TimeFormat))
			fmt.Println("⚙️  Snapshot processing started")
			fmt.Printf("⏱️  Snapshot building started at: %s\n", snapshot.TimeBuildingStarted.Format(cfg.TimeFormat))
			fmt.Printf("🏫 Course: %s\n", snapshot.Course)
			fmt.Printf("📦 Snapshot type: %s\n", snapshot.Type)
			fmt.Printf("📍 Url: %s\n", snapshot.Url)
			fmt.Printf("🔄 Retries: %d\n\n", snapshot.Retries)

			if snapshot.Err == nil {
				fmt.Println("✅ No error")
			} else {
				errorCounters[snapshot.Course]++
				text := fmt.Sprintf("❌ Error: %v", snapshot.Err)

				fmt.Print(text + "\n\n")
				fmt.Print(cfg.Sep)

				if snapshot.Type == pages.CourseSnapshot {
					msgSender.Do(fmt.Sprintf("%s [%s]", text, snapshot.Course), true)
				}

				timing.End(snapshotProcessingTimingKey, string(i))
				continue
			}

			if snapshot.StatusCode == http.StatusOK {
				fmt.Println("✅ Status ok")
			} else {
				badStatusCounters[snapshot.Course]++
				text := fmt.Sprintf("❌ Bad response status: %d", snapshot.StatusCode)

				fmt.Print(text + "\n\n")
				fmt.Print(cfg.Sep)

				if snapshot.Type == pages.CourseSnapshot {
					msgSender.Do(fmt.Sprintf("%s [%s]", text, snapshot.Course), true)
				}

				timing.End(snapshotProcessingTimingKey, string(i))
				continue
			}

			if snapshot.LoggedIn {
				fmt.Print("✅ Logged in\n\n")
			} else {
				text := "❌ Logged out"

				fmt.Print(text + "\n\n")
				fmt.Print(cfg.Sep)

				if snapshot.Type == pages.CourseSnapshot &&
					time.Since(lastLoggedOutMsg) > time.Duration(cfg.LoggedOutMsgCooldownSeconds)*time.Second {

					msgSender.Do(fmt.Sprintf("%s [%s]", text, snapshot.Course), true)
					lastLoggedOutMsg = time.Now().Local()
				}

				timing.End(snapshotProcessingTimingKey, string(i))
				continue
			}

			activitySet.Merge(parser.ExtractActivities(snapshot.Doc))

			fmt.Print(cfg.Sep)
			timing.End(snapshotProcessingTimingKey, string(i))

			i++
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
			fmt.Print(cfg.Sep)

			timing.End(snapshotProcessingTimingKey, string(i))
			sectionCounter := sectionCounters[course]
			errorCounter := errorCounters[course]
			badStatusCounter := badStatusCounters[course]

			fmt.Printf("🏫 Course: %s\n", course)
			fmt.Printf("⤵️  Sections: %d\n", sectionCounter)

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
				fmt.Print(cfg.Sep)
				continue
			}

			if !state.Storage.Exists(course) {
				fmt.Print("🧠 New course content remembered\n\n")
				fmt.Print(cfg.Sep)
				continue
			}

			activities := types.Activities(activitySet.ToSlice())
			added, removed := state.Storage.Diff(course, activities)

			if len(added) == 0 && len(removed) == 0 {
				fmt.Print("✅ Nothing new\n\n")
				fmt.Print(cfg.Sep)
				continue
			}

			fmt.Print("🚨 Changes\n\n")
			text := fmt.Sprintf("🚨 Changes in %s\n\n", course)

			if len(added) > 0 {
				fmt.Println(fmt.Sprintf("🆕 Added:\n\n%s", added.Repr()))
				text += fmt.Sprintf("🆕 Added:\n\n%s\n", added.ReprHtml())
			}

			if len(removed) > 0 {
				fmt.Println(fmt.Sprintf("🗑️ Removed:\n\n%s", removed.Repr()))
				text += fmt.Sprintf("🗑️ Removed:\n\n%s\n", removed.ReprHtml())
			}

			msgSender.Do(text, false)
			state.Storage.Set(course, activities)

			fmt.Print(cfg.Sep)
			timing.End(snapshotProcessingTimingKey, string(i))

			i++
		}

		durationRepr, err := timing.ReprAvailableDurationsOfEvents()

		if err != nil {
			panic("error on timing duration represent: " + err.Error())
		}

		fmt.Print(cfg.Sep)
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
