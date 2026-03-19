package state

import "monitor/internal/types"

type Storage map[string]types.Activities

func NewStorage() *Storage {
	s := Storage(make(map[string]types.Activities))
	return &s
}

func (s Storage) Exists(course string) bool {
	_, ok := s[course]
	return ok
}

func (s Storage) Set(course string, activities types.Activities) {
	s[course] = activities
}

func (s Storage) Diff(course string, activities types.Activities) (types.Activities, types.Activities) {
	return s[course].Diff(activities)
}
