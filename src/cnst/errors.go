package cnst

import "errors"

var (
	ErrEmptyKey      = errors.New("empty key")
	ErrExist         = errors.New("the resource already exists")
	ErrNotExist      = errors.New("the resource does not exist")
	ErrEmptyPassword = errors.New("password is empty")
	ErrEmptyUsername = errors.New("username is empty")
	ErrEmptyRequest  = errors.New("empty request")
	ErrIsDirectory   = errors.New("file is directory")
	ErrInvalidOption = errors.New("invalid option")
	ErrWrongDataType = errors.New("wrong data type")
	ErrShareAccess   = errors.New("share not allowed")
)
