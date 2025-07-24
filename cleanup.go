package main

import "errors"

type CleanupFuncs []func() error

func (cf *CleanupFuncs) Defer(f func() error) {
	*cf = append(*cf, f)
}

func (cf *CleanupFuncs) Cleanup() error {
	errs := make([]error, 0)
	for i := len(*cf) - 1; i >= 0; i-- {
		if ferr := (*cf)[i](); ferr != nil {
			errs = append(errs, ferr)
		}
	}
	return errors.Join(errs...)
}
