package state

import (
	"monitor/internal/parsing"
)

// Storage maps a course name to its known activities.
type Storage map[string]parsing.Activities

// NewStorage returns an empty activity storage.
func NewStorage() *Storage {
	s := Storage(make(map[string]parsing.Activities))
	return &s
}

// Exists reports whether a course already has persisted activities.
func (s Storage) Exists(course string) bool {
	_, ok := s[course]
	return ok
}

// Set stores the current activities for a course.
func (s Storage) Set(course string, activities parsing.Activities) {
	s[course] = activities
}

// Diff returns activities added in the new slice and removed from the stored one.
func (s Storage) Diff(course string, activities parsing.Activities) (parsing.Activities, parsing.Activities) {
	return s[course].Diff(activities)
}
