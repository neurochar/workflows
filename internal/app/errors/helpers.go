package errors

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ExtractError - ищет ближайшую ошибку AppError
func ExtractError(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}

	return nil, false
}

// NearestHints - ищет ближайший непустой слайс подсказок
func NearestHints(err error) ([]string, bool) {
	for {
		if err == nil {
			return nil, false
		}
		if appErr, ok := err.(*AppError); ok {
			if len(appErr.hints) > 0 {
				return appErr.hints, true
			}
			err = appErr.Unwrap()
			continue
		}
		if chainErr, ok := err.(*ChainError); ok {
			err = chainErr.Unwrap()
			continue
		}
		err = errors.Unwrap(err)
	}
}

// NearestErrMsg - ищет только сообщение об ошибке в AppError
func NearestErrMsg(err error) (string, bool) {
	for {
		if err == nil {
			return "", false
		}
		if appErr, ok := err.(*AppError); ok {
			if len(appErr.errMsg) > 0 {
				return appErr.errMsg, true
			}
			err = appErr.Unwrap()
			continue
		}
		if chainErr, ok := err.(*ChainError); ok {
			err = chainErr.Unwrap()
			continue
		}
		err = errors.Unwrap(err)
	}
}

// NearestError - ищет либо сообщение об ошибке в Error, либо текст сторонней ошибки
func NearestError(err error) (string, bool) {
	for {
		if err == nil {
			return "", false
		}
		if appErr, ok := err.(*AppError); ok {
			if len(appErr.errMsg) > 0 {
				return appErr.errMsg, true
			}
			err = appErr.Unwrap()
			continue
		}
		if chainErr, ok := err.(*ChainError); ok {
			err = chainErr.Unwrap()
			continue
		}
		if err.Error() != "" {
			return err.Error(), true
		}
		err = errors.Unwrap(err)
	}
}

// WithHints - добавляет подсказки к error.
// Ищет ближайший AppError и копируется. Если не найдено - создается копия ErrInternal
func WithHints(err error, hints ...string) *AppError {
	var foundAppErr *AppError

	lookingErr := err
	for lookingErr != nil {

		if appErr, ok := lookingErr.(*AppError); ok {
			foundAppErr = appErr
			break
		}
		if chainErr, ok := lookingErr.(*ChainError); ok {
			lookingErr = chainErr.Unwrap()
			continue
		}
		lookingErr = errors.Unwrap(lookingErr)
	}

	var cp *AppError
	if foundAppErr != nil {
		cp = foundAppErr.Copy()
	} else {
		cp = ErrInternal.Copy()
	}
	cp.parent = err
	cp.hints = hints

	return cp
}

// CheckIsTxСoncurrentExec - проверяет, является ли ошибка pgx о конкурентном выполнении транзакции
func CheckIsTxСoncurrentExec(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (pgErr.Code == "40001" || pgErr.Code == "25P02") {
		return true
	}
	return errors.Is(err, ErrTxСoncurrentExec)
}

// ConvertPgxToAppErr - конвертирует ошибку pgx в ошибку приложения
func ConvertPgxToAppErr(err error) (error, bool) {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrStoreNoRows.WithWrap(err), true
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "40001":
			return ErrTxСoncurrentExec.WithWrap(err), true
		case "25P02":
			return ErrTxСoncurrentExec.WithWrap(err), true
		case "23505":
			return ErrStoreUniqueViolation.WithWrap(err).WithDetail("column", true, pgErr.ColumnName), true
		case "23503":
			return ErrStoreForeignKeyViolation.WithWrap(err).WithDetail("column", true, pgErr.ColumnName), true
		case "23502":
			return ErrStoreNotNullViolation.WithWrap(err).WithDetail("column", true, pgErr.ColumnName), true
		case "23514":
			return ErrStoreCheckViolation.WithWrap(err).WithDetail("constraint", true, pgErr.ConstraintName), true
		case "23001":
			return ErrStoreRestrictViolation.WithWrap(err).WithDetail("constraint", true, pgErr.ConstraintName), true
		case "23000":
			return ErrStoreIntegrityViolation.WithWrap(err).WithDetail("constraint", true, pgErr.ConstraintName), true
		default:
			return ErrInternal.WithWrap(err), false
		}
	}
	return err, false
}
