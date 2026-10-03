package sbi

import "errors"

// ErrMaintenance indicates that the page explicitly announces a service stop.
var ErrMaintenance = errors.New("sbi: maintenance page detected")

// ErrUnexpectedPage indicates that the expected asset data could not be identified.
var ErrUnexpectedPage = errors.New("sbi: expected page not available")
