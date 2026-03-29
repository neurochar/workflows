package errors

type markError struct {
	source error
	mark   error
}

// Error - реализация интерфейса error
func (e *markError) Error() string {
	return e.source.Error()
}

// Unwrap - реализация интерфейса error, возвращает родительскую ошибку
func (e *markError) Unwrap() error {
	return e.source
}

// Is - реализация интерфейса error, проверка идентификации
func (e *markError) Is(err error) bool {
	if err == nil {
		return false
	}
	return e.source == err || e.mark == err
}

// Mark - поменить source ошибку mark ошибкой (для errors.Is)
func Mark(source error, mark error) error {
	return &markError{source: source, mark: mark}
}
