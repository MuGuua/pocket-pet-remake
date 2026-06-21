package petprogression

import "errors"

var (
	ErrInvalidAllocateInput     = errors.New("invalid pet allocate attr input")
	ErrInsufficientAttrPoints   = errors.New("insufficient pet attr points")
	ErrLevelConfigNotFound      = errors.New("pet level config not found")
	ErrConvertConfigNotFound    = errors.New("pet attr convert config not found")
	ErrPetProgressionNotFound   = errors.New("pet progression state not found")
)
