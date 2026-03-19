package types

type Set[T comparable] map[T]Void

func NewSet[T comparable](values ...T) Set[T] {
	s := Set[T](make(map[T]Void))
	s.Add(values...)

	return s
}

func (s Set[T]) Size() int {
	return len(s)
}

func (s Set[T]) Add(values ...T) {
	if s == nil {
		s = NewSet(values...)
		return
	}

	for _, value := range values {
		s[value] = Void{}
	}
}

func (s Set[T]) Remove(value T) {
	delete(s, value)
}

func (s Set[T]) Exists(value T) bool {
	_, ok := s[value]
	return ok
}

func (s Set[T]) Merge(set Set[T]) {
	if s == nil {
		s = NewSet[T]()
		return
	}

	for value := range set {
		s[value] = Void{}
	}
}

func (s Set[T]) ToSlice() []T {
	slice := make([]T, 0, len(s))

	for value := range s {
		slice = append(slice, value)
	}

	return slice
}
