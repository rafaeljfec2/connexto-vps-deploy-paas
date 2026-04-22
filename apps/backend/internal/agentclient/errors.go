package agentclient

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UnimplementedError struct {
	Method string
	Cause  error
}

func (e *UnimplementedError) Error() string {
	return "agent does not implement " + e.Method + ": please update the agent"
}

func (e *UnimplementedError) Unwrap() error {
	return e.Cause
}

func wrapUnimplemented(method string, err error) error {
	if err == nil {
		return nil
	}
	if st, ok := status.FromError(err); ok && st.Code() == codes.Unimplemented {
		return &UnimplementedError{Method: method, Cause: err}
	}
	return err
}

func IsUnimplemented(err error) bool {
	if err == nil {
		return false
	}
	var unimpl *UnimplementedError
	if errors.As(err, &unimpl) {
		return true
	}
	if st, ok := status.FromError(err); ok && st.Code() == codes.Unimplemented {
		return true
	}
	return false
}
