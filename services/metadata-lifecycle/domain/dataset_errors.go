package domain

import "errors"

var (
	ErrDatasetForbidden = errors.New("dataset operation forbidden")
	ErrDatasetConflict  = errors.New("dataset lifecycle conflict")
	ErrDatasetNotFound  = errors.New("dataset not found")
	ErrInvalidDataset   = errors.New("invalid dataset")
	ErrDatasetInternal  = errors.New("dataset operation failed")
)
