package types

// Set is a generic set implemented as a map.
type Set[T comparable] map[T]Void

// NewSet returns a set populated with the provided values.
func NewSet[T comparable](values ...T) Set[T] {
	s := Set[T](make(map[T]Void))
	s.Add(values...)

	return s
}

// Size returns the number of values in the set.
func (s Set[T]) Size() int {
	return len(s)
}

// Add inserts values into the set.
func (s Set[T]) Add(values ...T) {
	if s == nil {
		s = NewSet(values...)
		return
	}

	for _, value := range values {
		s[value] = Void{}
	}
}

// Remove deletes a value from the set.
func (s Set[T]) Remove(value T) {
	delete(s, value)
}

// Exists reports whether the value is present in the set.
func (s Set[T]) Exists(value T) bool {
	_, ok := s[value]
	return ok
}

// Merge inserts all values from another set.
func (s Set[T]) Merge(set Set[T]) {
	if s == nil {
		s = NewSet[T]()
		return
	}

	for value := range set {
		s[value] = Void{}
	}
}

// ToSlice returns the set contents as a slice.
func (s Set[T]) ToSlice() []T {
	slice := make([]T, 0, len(s))

	for value := range s {
		slice = append(slice, value)
	}

	return slice
}
