// Package errors contains app error struct and helpers
package errors

import (
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"strings"
)

type appErrorDetail struct {
	Value    any
	IsHidden bool
}

// AppError - структура для оборачивания сторонних ошибок в цепочку вызовов и хранения метаданных
type AppError struct {
	parent   error
	is       error
	chainMsg []string
	errMsg   string
	hints    []string
	meta     ErrorMeta
	details  map[string]appErrorDetail
}

// ErrorMeta - структура для хранения метаданных
type ErrorMeta struct {
	Code     errorCode
	TextCode string
}

// NewError - создает новую ошибку
func NewError(msg string, code errorCode, textCode string, hints []string) *AppError {
	return &AppError{
		chainMsg: []string{msg},
		errMsg:   msg,
		hints:    hints,
		meta: ErrorMeta{
			Code:     code,
			TextCode: textCode,
		},
	}
}

// Copy - создает копию ошибки
func (e *AppError) Copy() *AppError {
	cp := &AppError{
		parent: e,
		is:     e.is,
		errMsg: e.errMsg,
		hints:  e.hints,
		meta: ErrorMeta{
			Code:     e.meta.Code,
			TextCode: e.meta.TextCode,
		},
		details: nil,
	}

	if e.chainMsg != nil {
		cp.chainMsg = make([]string, len(e.chainMsg))
		copy(cp.chainMsg, e.chainMsg)
	}

	if e.details != nil {
		cp.details = make(map[string]appErrorDetail, len(e.details))
		maps.Copy(cp.details, e.details)
	}

	return cp
}

func (e *AppError) chain(chainMsg string) *AppError {
	cp := e.Copy()
	if chainMsg != "" {
		cp.chainMsg = append(cp.chainMsg, fmt.Sprintf("[%s]", chainMsg))
	}

	return cp
}

// WithParent - создает копию ошибки с измененной ссылкой на родительскую ошибку и цепочку вызовов
func (e *AppError) WithParent(err error) *AppError {
	cp := e.Copy()

	cp.parent = err
	cp.is = e
	curChainMsh := e.chainMsg
	cp.chainMsg = make([]string, 0, len(curChainMsh)+1)
	cp.chainMsg = append(cp.chainMsg, err.Error())
	cp.chainMsg = append(cp.chainMsg, curChainMsh...)

	return cp
}

// WithWrap - создает обертку для текущей ошибки на основе переданной ошибки
func (e *AppError) WithWrap(err error) *AppError {
	cp := e.Copy()

	cp.is = err
	cp.chainMsg = make([]string, 0, len(e.chainMsg)+1)
	cp.chainMsg = append(cp.chainMsg, e.chainMsg...)

	newChainMsg := getChainMsg(err)
	if newChainMsg != nil {
		cp.chainMsg = append(cp.chainMsg, newChainMsg...)
	}

	if newErrMsg, ok := NearestErrMsg(err); ok {
		cp.errMsg = newErrMsg
	} else {
		cp.errMsg = err.Error()
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		cp.meta = appErr.meta
	}

	return cp
}

// Is - реализация интерфейса error, проверка идентификации
func (e *AppError) Is(err error) bool {
	if err == nil {
		return false
	}

	if e == err || e.is == err {
		return true
	}

	return false
}

// Unwrap - реализация интерфейса error, возвращает родительскую ошибку
func (e *AppError) Unwrap() error {
	return e.parent
}

// Error - реализация интерфейса error, возвращает строку с цепочкой вызовов
func (e *AppError) Error() string {
	var bldr strings.Builder
	for i := len(e.chainMsg) - 1; i >= 0; i-- {
		bldr.WriteString(e.chainMsg[i])
		if i != 0 {
			bldr.WriteString(" : ")
		}
	}

	return bldr.String()
}

// Extend - наследовать ошибку без цепочки вызовов
func (e *AppError) Extend(errMsg string) *AppError {
	newErr := e.Copy()

	newErr.errMsg = errMsg
	newErr.chainMsg = []string{errMsg}

	return newErr
}

// ExtendWithChain - наследовать ошибку вместе с цепочкой
func (e *AppError) ExtendWithChain(errMsg string) *AppError {
	newErr := e.Copy()

	newErr.errMsg = errMsg
	newErr.chainMsg = append(newErr.chainMsg, errMsg)

	return newErr
}

// WithMeta - создает копию ошибки с измененным метаданными
func (e *AppError) WithMeta(meta ErrorMeta) *AppError {
	newErr := e.Copy()

	newErr.meta = meta

	return newErr
}

// WithCode - создает копию ошибки с измененным кодом
func (e *AppError) WithCode(code errorCode) *AppError {
	newErr := e.Copy()

	newErr.meta.Code = code

	return newErr
}

// WithTextCode - создает копию ошибки с измененным текстовым кодом
func (e *AppError) WithTextCode(textCode string) *AppError {
	newErr := e.Copy()

	newErr.meta.TextCode = textCode

	return newErr
}

// WithDetail - создает копию ошибки с измененным детализацией
func (e *AppError) WithDetail(key string, isHidden bool, value any) *AppError {
	newErr := e.Copy()

	if newErr.details == nil {
		newErr.details = make(map[string]appErrorDetail, 1)
	}

	newErr.details[key] = appErrorDetail{
		Value:    value,
		IsHidden: isHidden,
	}

	return newErr
}

// WithHints - создает копию ошибки с измененными подсказками для пользователей
func (e *AppError) WithHints(hints ...string) *AppError {
	newErr := e.Copy()

	newErr.hints = hints

	return newErr
}

// Hints - получить подсказки для пользователей
func (e *AppError) Hints() []string {
	return e.hints
}

// HintsStr - получить подсказки для пользователей в виде одной строки
func (e *AppError) HintsStr(delimiter string) string {
	return strings.Join(e.hints, delimiter)
}

// Meta - получить метаданные ошибки
func (e *AppError) Meta() ErrorMeta {
	return e.meta
}

// Detail - получить деталь ошибки
func (e *AppError) Detail(key string) (appErrorDetail, bool) {
	item, ok := e.details[key]

	return item, ok
}

// Details - получить детали ошибки
func (e *AppError) Details(withHidden bool) map[string]any {
	result := make(map[string]any, len(e.details))

	for k, v := range e.details {
		if withHidden || !v.IsHidden {
			result[k] = v.Value
		}
	}

	return result
}

// ErrMsg - получить текст ошибки без цепочки вызовов
func (e *AppError) ErrMsg() string {
	return e.errMsg
}

// LogValue - интерфейс для логеров
func (e *AppError) LogValue() slog.Value {
	jsonStruct := ToJSONStruct(e, true, false)
	return jsonStruct.ToSlogValue()
}
