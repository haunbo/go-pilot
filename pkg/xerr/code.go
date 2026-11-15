package xerr

// Generic API error codes. Applications can define their own codes and pass
// them to New or Wrap.
const (
	// CodeOK is the success code.
	CodeOK = 0

	// CodeInvalidParams indicates invalid request parameters.
	CodeInvalidParams = 10001
	// CodeUnauthorized indicates missing or invalid authentication.
	CodeUnauthorized = 10002
	// CodeForbidden indicates insufficient permissions.
	CodeForbidden = 10003
	// CodeNotFound indicates a missing resource.
	CodeNotFound = 10004
	// CodeConflict indicates a conflicting resource state.
	CodeConflict = 10005
	// CodeTooManyRequests indicates rate limiting.
	CodeTooManyRequests = 10006
	// CodeInternal indicates an internal server error.
	CodeInternal = 10007
)
