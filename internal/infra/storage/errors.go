package storage

import (
	appErrors "github.com/neurochar/workflows/internal/app/errors"
)

var ErrBucketAlreadyExists = appErrors.ErrConflict.Extend("bucket already exists")
