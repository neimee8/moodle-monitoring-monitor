package timepkg

import (
	"errors"
	"fmt"
	"time"
)

type timeSpan struct {
	start *time.Time
	end   *time.Time
}

type Timing struct {
	eventOrder []string
	events     map[string]map[string]*timeSpan
}

func NewTiming() *Timing {
	return &Timing{
		events: make(map[string]map[string]*timeSpan),
	}
}

func (timing *Timing) Start(event string, stage string) error {
	_, eventExists := timing.events[event]
	stageStarted := false

	if eventExists {
		_, stageStarted = timing.events[event][stage]
	}

	if stageStarted {
		return fmt.Errorf("event %s on stage %s is already started", event, stage)
	}

	if !eventExists {
		timing.eventOrder = append(timing.eventOrder, event)
	}

	t := time.Now().Local()

	if timing.events[event] == nil {
		timing.events[event] = make(map[string]*timeSpan)
	}

	timing.events[event][stage] = &timeSpan{
		start: &t,
	}

	return nil
}

func (timing *Timing) End(event string, stage string) error {
	_, eventExists := timing.events[event]
	stageStarted := false
	stageEnded := false

	if eventExists {
		_, stageStarted = timing.events[event][stage]
	}

	if stageStarted {
		stageEnded = timing.events[event][stage].end != nil
	}

	if !stageStarted {
		return fmt.Errorf("event %s on stage %s have not started yet", event, stage)
	}

	if stageEnded {
		return fmt.Errorf("event %s on stage %s is already ended", event, stage)
	}

	t := time.Now().Local()
	timing.events[event][stage].end = &t

	return nil
}

func (timing Timing) DurationOfStage(event string, stage string) (time.Duration, error) {
	var zeroDuration time.Duration
	_, eventExists := timing.events[event]
	stageStarted := false
	stageEnded := false

	if eventExists {
		_, stageStarted = timing.events[event][stage]
	}

	if stageStarted {
		stageEnded = timing.events[event][stage].end != nil
	}

	if !stageStarted {
		return zeroDuration, fmt.Errorf("event %s on stage %s have not started yet", event, stage)
	}

	if !stageEnded {
		return zeroDuration, fmt.Errorf("event %s on stage %s have not ended yet", event, stage)
	}

	span := timing.events[event][stage]
	start := *span.start
	end := *span.end

	return end.Sub(start), nil
}

func (timing Timing) DurationOfEvent(event string) (time.Duration, error) {
	var zeroDuration time.Duration
	_, eventExists := timing.events[event]

	if !eventExists {
		return zeroDuration, fmt.Errorf("event %s have not started yet", event)
	}

	stages := timing.events[event]
	duration := zeroDuration

	for _, span := range stages {
		if span.start == nil || span.end == nil {
			continue
		}

		duration += span.end.Sub(*span.start)
	}

	if duration == zeroDuration {
		return zeroDuration, fmt.Errorf("no stages of event %s have been started yet", event)
	}

	return duration, nil
}

func (timing Timing) ReprDurationOfStage(event string, stage string) (string, error) {
	var zeroDuration time.Duration
	duration, err := timing.DurationOfStage(event, stage)

	if err != nil {
		return zeroDuration.String(), err
	}

	return fmt.Sprintf("%s: %v", event, duration), nil
}

func (timing Timing) ReprDurationOfEvent(event string) (string, error) {
	var zeroDuration time.Duration
	duration, err := timing.DurationOfEvent(event)

	if err != nil {
		return zeroDuration.String(), err
	}

	return fmt.Sprintf("%s: %v", event, duration), nil
}

func (timing Timing) ReprAvailableDurationsOfEvents() (string, error) {
	prefix := "⏳ Time spent:\n"
	text := prefix

	for _, event := range timing.eventOrder {
		durationRepr, err := timing.ReprDurationOfEvent(event)

		if err != nil {
			continue
		}

		text += fmt.Sprintf("%s\n", durationRepr)
	}

	if text == prefix {
		return "", errors.New("no finished events found")
	}

	return text, nil
}
