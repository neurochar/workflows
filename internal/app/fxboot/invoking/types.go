package invoking

import "context"

// InvokeInit - invoke after init
type InvokeInit struct {
	StartBeforeOpen func(context.Context) error
	StartAfterOpen  func(context.Context) error
	Stop            func(context.Context) error
}
