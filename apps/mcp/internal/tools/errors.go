package tools

import "errors"

func errInvalidArg(msg string) error {
	return errors.New("invalid argument: " + msg)
}
