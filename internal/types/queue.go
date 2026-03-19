package types

type Queue[T any] struct {
	data       []T
	pointerIdx int
}

func NewQueue[T any](values ...T) *Queue[T] {
	return &Queue[T]{
		data: values,
	}
}

func (q *Queue[T]) Size() int {
	return len(q.data) - q.pointerIdx
}

func (q *Queue[T]) Add(values ...T) {
	q.data = append(q.data, values...)
}

func (q *Queue[T]) Pop() (T, bool) {
	if q.Size() == 0 {
		var zero T
		return zero, false
	}

	value := q.data[q.pointerIdx]

	// reset links
	var zero T
	q.data[q.pointerIdx] = zero

	q.pointerIdx++

	if q.pointerIdx > 0 && q.pointerIdx*2 >= len(q.data) {
		q.data = append([]T(nil), q.data[q.pointerIdx:]...)
		q.pointerIdx = 0
	}

	return value, true
}
