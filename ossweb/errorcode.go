package ossweb

type ErrorCodeType int32

const (
	ERROR_SUCCESS      ErrorCodeType = 0
	ERROR_UNAUTHORIZED ErrorCodeType = 1 //StatusUnauthorized
	ERROR_BADARGS      ErrorCodeType = 2
	Policy_AVG         ErrorCodeType = 3
)
