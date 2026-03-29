package errors

import (
	"fmt"
	"strings"
)

// ChainError - структура для оборачивания сторонних ошибок в цепочку вызовов
type ChainError struct {
	parent   error
	chainMsg []string
}

func (e *ChainError) copy() *ChainError {
	cp := &ChainError{
		parent: e.parent,
	}

	if e.chainMsg != nil {
		cp.chainMsg = make([]string, len(e.chainMsg))
		copy(cp.chainMsg, e.chainMsg)
	}

	return cp
}

// Error - реализация интерфейса error, возвращает строку с цепочкой вызовов
func (e *ChainError) Error() string {
	var bldr strings.Builder
	for i := len(e.chainMsg) - 1; i >= 0; i-- {
		bldr.WriteString(e.chainMsg[i])
		if i != 0 {
			bldr.WriteString(" : ")
		}
	}

	return bldr.String()
}

// Is - реализация интерфейса error, проверка идентификации
func (e *ChainError) Is(err error) bool {
	if err == nil {
		return false
	}

	if e == err {
		return true
	}

	return false
}

// Unwrap - реализация интерфейса error, возвращает родительскую ошибку
func (e *ChainError) Unwrap() error {
	return e.parent
}

// Chain - оборачивает ошибку в цепочку
func Chain(err error, chainMsg string) error {
	if appErr, ok := err.(*AppError); ok {
		return appErr.chain(chainMsg)
	}

	if chainErr, ok := err.(*ChainError); ok {
		cp := chainErr.copy()
		if chainMsg != "" {
			cp.chainMsg = append(cp.chainMsg, fmt.Sprintf("[%s]", chainMsg))
		}
		return cp
	}

	return &ChainError{
		parent:   err,
		chainMsg: []string{fmt.Sprintf("[%s]", chainMsg)},
	}
}

// Chainf - оборачивает ошибку в цепочку с форматированием
func Chainf(err error, format string, args ...interface{}) error {
	chainMsg := fmt.Sprintf(format, args...)

	return Chain(err, chainMsg)
}

func getChainMsg(err error) []string {
	if err == nil {
		return nil
	}

	if chainErr, ok := err.(*ChainError); ok {
		return chainErr.chainMsg
	}

	if appErr, ok := err.(*AppError); ok {
		return appErr.chainMsg
	}

	return []string{err.Error()}
}
