package types

// Queue is a FIFO queue backed by a slice.
type Queue[T any] struct {
	data       []T
	pointerIdx int
}

// NewQueue returns a queue populated with the provided values.
func NewQueue[T any](values ...T) *Queue[T] {
	return &Queue[T]{
		data: values,
	}
}

// Size returns the number of values that can still be popped.
func (q *Queue[T]) Size() int {
	return len(q.data) - q.pointerIdx
}

// Add appends values to the back of the queue.
func (q *Queue[T]) Add(values ...T) {
	q.data = append(q.data, values...)
}

// Pop removes and returns the next value in FIFO order.
func (q *Queue[T]) Pop() (T, bool) {
	if q.Size() == 0 {
		var zero T
		return zero, false
	}

	value := q.data[q.pointerIdx]

	// Clear the slot so popped values can be garbage-collected.
	var zero T
	q.data[q.pointerIdx] = zero

	q.pointerIdx++

	if q.pointerIdx > 0 && q.pointerIdx*2 >= len(q.data) {
		// Periodically compact the slice to avoid retaining a long consumed prefix.
		q.data = append([]T(nil), q.data[q.pointerIdx:]...)
		q.pointerIdx = 0
	}

	return value, true
}
