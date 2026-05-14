package errcode

const (
	HTTPSuccess                   = 200
	HTTPInvalidRequest            = 40001
	HTTPUnauthorized              = 40101
	HTTPInternalServer            = 50000
	WSCodeSuccess          uint32 = 0
	WSCodeTokenInvalid            = 10001
	WSCodeSessionInvalid          = 10002
	WSCodeUnauthorized            = 10003
	WSCodeInvalidPacket           = 10004
	WSCodeUnsupportedCmd          = 10005
	WSCodeWorldEnterFailed        = 20001
	WSCodePlayerNotFound          = 20002
	WSCodeWorldMoveFailed         = 20003
)
