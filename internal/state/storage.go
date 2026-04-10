package state

import (
	"monitor/internal/parsing"
)

type Storage map[string]parsing.Activities

func NewStorage() *Storage {
	s := Storage(make(map[string]parsing.Activities))
	return &s
}

func (s Storage) Exists(course string) bool {
	_, ok := s[course]
	return ok
}

func (s Storage) Set(course string, activities parsing.Activities) {
	s[course] = activities
}

func (s Storage) Diff(course string, activities parsing.Activities) (parsing.Activities, parsing.Activities) {
	return s[course].Diff(activities)
}
