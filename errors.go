package slipstream

import "errors"

func errConfig(message string) error {
	return errors.New(message)
}
