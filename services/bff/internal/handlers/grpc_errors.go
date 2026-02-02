package handlers

import (
	"net/http"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/example/anime-platform/internal/platform/api"
)

func writeGRPCError(w http.ResponseWriter, requestID string, err error) {
	st, ok := status.FromError(err)
	if !ok {
		api.Internal(w, requestID)
		return
	}

	code := "INTERNAL"
	details := map[string]any{}
	for _, d := range st.Details() {
		switch v := d.(type) {
		case *errdetails.ErrorInfo:
			if v.GetReason() != "" {
				code = v.GetReason()
			}
		case *errdetails.BadRequest:
			for _, fv := range v.GetFieldViolations() {
				if fv.GetField() != "" {
					details[fv.GetField()] = fv.GetDescription()
				}
			}
		}
	}
	if len(details) == 0 {
		details = nil
	}

	switch st.Code() {
	case codes.InvalidArgument:
		api.BadRequest(w, code, st.Message(), requestID, details)
	case codes.Unauthenticated:
		api.Unauthorized(w, code, st.Message(), requestID)
	case codes.PermissionDenied:
		api.Forbidden(w, code, st.Message(), requestID)
	case codes.NotFound:
		api.NotFound(w, code, st.Message(), requestID)
	case codes.AlreadyExists:
		api.Conflict(w, code, st.Message(), requestID, details)
	case codes.ResourceExhausted:
		api.RateLimited(w, code, st.Message(), requestID, details)
	default:
		api.Internal(w, requestID)
	}
}
