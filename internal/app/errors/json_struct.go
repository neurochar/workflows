package errors

import "log/slog"

// JSONStruct - структура для json
type JSONStruct struct {
	Message  string         `json:"message"`
	ChainMsg string         `json:"chainMsg,omitempty"`
	TextCode string         `json:"textCode,omitempty"`
	Code     int            `json:"code,omitempty"`
	Hints    []string       `json:"hints,omitempty"`
	Details  map[string]any `json:"details,omitempty"`
}

// ToSlogValue - преобразует error в slog.Value
func (e *JSONStruct) ToSlogValue() slog.Value {
	attrs := []slog.Attr{
		slog.String("message", e.Message),
		slog.String("textCode", e.TextCode),
		slog.Int("code", e.Code),
	}

	if e.ChainMsg != e.Message {
		attrs = append(attrs, slog.String("chainMsg", e.ChainMsg))
	}

	if len(e.Hints) > 0 {
		attrs = append(attrs, slog.Any("hints", e.Hints))
	}

	if len(e.Details) > 0 {
		attrs = append(attrs, slog.Any("details", e.Details))
	}

	return slog.GroupValue(attrs...)
}

// ToJSONStruct - преобразует error в ErrorJsonStruct
func ToJSONStruct(err error, addHiddenDetails bool, addHints bool) JSONStruct {
	if err == nil {
		return JSONStruct{}
	}

	appErr, ok := ExtractError(err)
	if !ok {
		return JSONStruct{
			Message: err.Error(),
		}
	}

	errStruct := JSONStruct{
		Message:  appErr.ErrMsg(),
		ChainMsg: appErr.Error(),
		TextCode: appErr.meta.TextCode,
		Code:     int(appErr.meta.Code),
		Details:  appErr.Details(addHiddenDetails),
	}

	if addHints {
		errStruct.Hints = appErr.Hints()
	}

	return errStruct
}
