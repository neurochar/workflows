package errors

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ToGrpcStatus - преобразует error в GrpcStatus
func ToGrpcStatus(err error) error {
	if err == nil {
		return nil
	}

	appErr, ok := ExtractError(err)
	if !ok {
		st, ok := status.FromError(err)
		if !ok {
			return status.Error(codes.Unknown, err.Error())
		}

		return st.Err()
	}

	code := mapToGrpcCode(appErr.Meta().Code)
	textCode := appErr.Meta().TextCode

	meta := make(map[string]string)

	errMsg := textCode

	hints, ok := NearestHints(err)
	if ok {
		hitText := strings.Join(hints, "; ")

		if errMsg != "" {
			errMsg = fmt.Sprintf("%s: %s", errMsg, hitText)
		} else {
			errMsg = hitText
		}

		for i, hint := range hints {
			meta[fmt.Sprintf("hint_%d", i)] = hint
		}
	}

	details := appErr.Details(false)
	for k, v := range details {
		var value string

		switch v := v.(type) {
		case string:
			value = v
		case []byte:
			value = string(v)
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			value = fmt.Sprintf("%d", v)
		case float32, float64:
			value = fmt.Sprintf("%d", v)
		default:
			value = fmt.Sprintf("%v", v)
		}

		meta[fmt.Sprintf("detail_%s", k)] = value
	}

	detail := &errdetails.ErrorInfo{
		Reason:   textCode,
		Metadata: meta,
	}

	st, stErr := status.New(code, errMsg).WithDetails(detail)
	if stErr != nil {
		return status.Error(codes.Unknown, stErr.Error())
	}

	return st.Err()
}

var grpcCodeByAppCode = map[errorCode]codes.Code{
	errorCodeBadRequest:    codes.InvalidArgument,
	errorCodeUnauthorized:  codes.Unauthenticated,
	errorCodeForbidden:     codes.PermissionDenied,
	errorCodeNotFound:      codes.NotFound,
	errorCodeConflict:      codes.Aborted,
	errorCodeUnprocessable: codes.FailedPrecondition,
	errorTooManyRequests:   codes.ResourceExhausted,
	errorCodeInternal:      codes.Internal,
}

func mapToGrpcCode(app errorCode) codes.Code {
	if c, ok := grpcCodeByAppCode[app]; ok {
		return c
	}
	return codes.Unknown
}

func FromGRPCError(err error) error {
	if err == nil {
		return nil
	}

	st, ok := status.FromError(err)
	if !ok {
		return err
	}

	var info *errdetails.ErrorInfo
	for _, d := range st.Details() {
		if v, ok := d.(*errdetails.ErrorInfo); ok {
			info = v
			break
		}
	}

	textCode := ""
	if info != nil && info.Reason != "" {
		textCode = info.Reason
	}

	var base *AppError
	if textCode != "" {
		if ae := appErrorByTextCode(textCode); ae != nil {
			base = ae
		}
	}
	if base == nil {
		base = appErrorByGrpcCode(st.Code())
	}
	if base == nil {
		return err
	}

	var hints []string
	details := map[string]any{}

	if info != nil && info.Metadata != nil {
		type pair struct {
			i int
			v string
		}

		hp := make([]pair, 0, len(info.Metadata))

		for k, v := range info.Metadata {
			switch {
			case strings.HasPrefix(k, "hint_"):
				if idx, err := strconv.Atoi(strings.TrimPrefix(k, "hint_")); err == nil {
					hp = append(hp, pair{i: idx, v: v})
				}

			case strings.HasPrefix(k, "detail_"):
				key := strings.TrimPrefix(k, "detail_")
				if key != "" {
					details[key] = v
				}
			}
		}

		slices.SortFunc(hp, func(a, b pair) int {
			return cmp.Compare(a.i, b.i)
		})

		hints = make([]string, 0, len(hp))
		for _, p := range hp {
			if p.v != "" {
				hints = append(hints, p.v)
			}
		}
	}

	cp := base.Extend(st.Message())

	if textCode != "" {
		cp = cp.WithTextCode(textCode)
	}

	if len(hints) > 0 {
		cp = cp.WithHints(hints...)
	}

	for k, v := range details {
		cp = cp.WithDetail(k, false, v)
	}

	return cp
}

func appErrorByTextCode(textCode string) *AppError {
	switch textCode {
	case "BAD_REQUEST":
		return ErrBadRequest
	case "UNAUTHORIZED":
		return ErrUnauthorized
	case "FORBIDDEN":
		return ErrForbidden
	case "NOT_FOUND":
		return ErrNotFound
	case "CONFLICT":
		return ErrConflict
	case "TOO_MANY_REQUESTS":
		return ErrTooManyRequests
	case "UNPROCESSABLE":
		return ErrUnprocessable
	case "INTERNAL_ERROR":
		return ErrInternal
	case "UNIQUE_VIOLATION":
		return ErrUniqueViolation
	case "VERSION_CONFLICT":
		return ErrVersionConflict
	case "BACKOFF":
		return ErrBackoff
	default:
		return nil
	}
}

func appErrorByGrpcCode(code codes.Code) *AppError {
	switch code {
	case codes.InvalidArgument:
		return ErrBadRequest
	case codes.Unauthenticated:
		return ErrUnauthorized
	case codes.PermissionDenied:
		return ErrForbidden
	case codes.NotFound:
		return ErrNotFound
	case codes.Aborted:
		return ErrConflict
	case codes.ResourceExhausted:
		return ErrTooManyRequests
	case codes.FailedPrecondition:
		return ErrUnprocessable
	case codes.Internal:
		return ErrInternal
	case codes.Unimplemented:
		return ErrUnimplemented
	default:
		return ErrInternal
	}
}
