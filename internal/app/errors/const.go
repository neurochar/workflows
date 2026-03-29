package errors

// Корневые ошибки
var (
	// ErrBadRequest - некорректный запрос
	ErrBadRequest = NewError("bad request", errorCodeBadRequest, "BAD_REQUEST", nil)

	// ErrUnauthorized - не авторизован
	ErrUnauthorized = NewError("unauthorized", errorCodeUnauthorized, "UNAUTHORIZED", nil)

	// ErrMethodNotAllowed - не разрешен
	ErrMethodNotAllowed = NewError("method not allowed", errorMethodNotAllowed, "METHOD_NOT_ALLOWED", nil)

	// ErrForbidden - нет доступа
	ErrForbidden = NewError("forbidden", errorCodeForbidden, "FORBIDDEN", nil)

	// ErrConflict - конфликт
	ErrConflict = NewError("conflict", errorCodeConflict, "CONFLICT", nil)

	// ErrRequestEntityTooLarge - слишком большой запрос
	ErrRequestEntityTooLarge = NewError("request entity too large", errorCodeRequestEntityTooLarge, "REQUEST_ENTITY_TOO_LARGE", nil)

	// ErrNotFound - не найден
	ErrNotFound = NewError("not found", errorCodeNotFound, "NOT_FOUND", nil)

	// ErrTooManyRequests - слишком много запросов
	ErrTooManyRequests = NewError("too many requests", errorTooManyRequests, "TOO_MANY_REQUESTS", nil)

	// ErrUnprocessable - невозможно обработать
	ErrUnprocessable = NewError("unprocessable", errorCodeUnprocessable, "UNPROCESSABLE", nil)

	// ErrInternal - внутренняя ошибка
	ErrInternal = NewError("internal error", errorCodeInternal, "INTERNAL_ERROR", nil)

	// ErrUnimplemented - не реализован
	ErrUnimplemented = NewError("unimplemented ", errorCodeUnimplemented, "UNIMPLEMENTED", nil)
)

// Ошибки хранилища
var (
	// ErrStoreOptimisticConflict - конфликт по оптимистичному блокированию
	ErrStoreOptimisticConflict = ErrConflict.Extend("store optimistic conflict")

	// ErrTxСoncurrentExec - ошибка транзакции
	ErrTxСoncurrentExec = ErrConflict.Extend("store transaction concurrent execution")

	// ErrStoreNoRows - no rows
	ErrStoreNoRows = ErrNotFound.Extend("store no rows")

	// ErrStoreUniqueViolation - unique violation
	ErrStoreUniqueViolation = ErrConflict.Extend("store unique violation").WithTextCode("UNIQUE_VIOLATION")

	// ErrStoreForeignKeyViolation - foreign key violation
	ErrStoreForeignKeyViolation = ErrBadRequest.Extend("store foreign key violation")

	// ErrStoreCheckViolation - check violation
	ErrStoreCheckViolation = ErrBadRequest.Extend("store check violation")

	// ErrStoreNotNullViolation - not null violation
	ErrStoreNotNullViolation = ErrBadRequest.Extend("store not null violation")

	// ErrStoreRestrictViolation - restrict violation
	ErrStoreRestrictViolation = ErrBadRequest.Extend("store restrict violation")

	// ErrStoreIntegrityViolation - integrity violation
	ErrStoreIntegrityViolation = ErrBadRequest.Extend("store integrity violation")
)

// Ошибки логики или валидации
var (
	// ErrVersionConflict - конфликт по версии
	ErrVersionConflict = ErrConflict.Extend("version conflict").WithTextCode("VERSION_CONFLICT")

	// ErrUniqueViolation - конфликт по уникальности
	ErrUniqueViolation = ErrConflict.Extend("unique violation").WithTextCode("UNIQUE_VIOLATION")

	// ErrBackoff - backoff
	ErrBackoff = ErrTooManyRequests.Extend("backoff").WithTextCode("BACKOFF")
)
