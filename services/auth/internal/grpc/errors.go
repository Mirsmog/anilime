package grpcapi

import (
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func errInvalidArgument(code, msg string, fieldViolations map[string]string) error {
	st := status.New(codes.InvalidArgument, msg)
	info := &errdetails.ErrorInfo{Reason: code, Domain: "auth"}

	bad := &errdetails.BadRequest{}
	for field, desc := range fieldViolations {
		bad.FieldViolations = append(bad.FieldViolations, &errdetails.BadRequest_FieldViolation{Field: field, Description: desc})
	}

	st2, err := st.WithDetails(info, bad)
	if err != nil {
		return st.Err()
	}
	return st2.Err()
}

func errAlreadyExists(code, msg string) error {
	st := status.New(codes.AlreadyExists, msg)
	info := &errdetails.ErrorInfo{Reason: code, Domain: "auth"}
	st2, err := st.WithDetails(info)
	if err != nil {
		return st.Err()
	}
	return st2.Err()
}

func errUnauthenticated(code, msg string) error {
	st := status.New(codes.Unauthenticated, msg)
	info := &errdetails.ErrorInfo{Reason: code, Domain: "auth"}
	st2, err := st.WithDetails(info)
	if err != nil {
		return st.Err()
	}
	return st2.Err()
}

//nolint:unparam // code is kept for future internal error categorization
func errInternal(code, msg string) error {
	st := status.New(codes.Internal, msg)
	info := &errdetails.ErrorInfo{Reason: code, Domain: "auth"}
	st2, err := st.WithDetails(info)
	if err != nil {
		return st.Err()
	}
	return st2.Err()
}
