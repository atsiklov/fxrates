package domain

import "errors"

var (
	ErrRateNotFound   = errors.New("rate not found")
	ErrRateNotApplied = errors.New("rate update not applied")
)
