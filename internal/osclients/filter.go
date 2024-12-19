package osclients

type result[T any] struct {
	ok  T
	err error
}

func (r result[T]) Ok() T      { return r.ok }
func (r result[T]) Err() error { return r.err }

// Result carries either a result or a non-nil error.
type Result[T any] interface {
	Ok() T
	Err() error
}

func NewResultOk[T any](ok T) result[T] {
	return result[T]{ok: ok}
}

func NewResultErr[T any](err error) result[T] {
	return result[T]{err: err}
}

func Filter[T any, R Result[T]](in <-chan R, filters ...func(T) bool) <-chan R {
	out := make(chan (R))
	go func() {
		defer close(out)
	next:
		for result := range in {
			if err := result.Err(); err != nil {
				out <- result
				continue
			}
			for _, filter := range filters {
				if !filter(result.Ok()) {
					continue next
				}
			}
			out <- result
		}
	}()
	return out
}

func JustOne[T any, R Result[*T]](in <-chan R, duplicateError error) (*T, error) {
	var found *T
	for result := range in {
		if err := result.Err(); err != nil {
			return nil, err
		}
		if found != nil {
			return nil, duplicateError
		}
		ok := result.Ok()
		found = ok
	}
	return found, nil
}
