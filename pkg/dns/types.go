package dns

import "fmt"

const (
	RRA            uint16 = 1
	RRTXT          uint16 = 16
	RROPT          uint16 = 41
	ClassIN        uint16 = 1
	EDNSUDPPayload uint16 = 1232
)

type RCode uint8

const (
	RCodeOK RCode = iota
	RCodeFormatError
	RCodeServerFailure
	RCodeNameError
)

func (r RCode) String() string {
	switch r {
	case RCodeOK:
		return "OK"
	case RCodeFormatError:
		return "FORMAT_ERROR"
	case RCodeServerFailure:
		return "SERVER_FAILURE"
	case RCodeNameError:
		return "NAME_ERROR"
	default:
		return fmt.Sprintf("RCODE_%d", uint8(r))
	}
}

func rcodeFromByte(v byte) (RCode, bool) {
	switch v {
	case 0:
		return RCodeOK, true
	case 1:
		return RCodeFormatError, true
	case 2:
		return RCodeServerFailure, true
	case 3:
		return RCodeNameError, true
	default:
		return 0, false
	}
}

type Question struct {
	Name  string
	Type  uint16
	Class uint16
}

type DecodedQuery struct {
	ID       uint16
	RD       bool
	CD       bool
	Question Question
	Payload  []byte
}

type QueryParams struct {
	ID      uint16
	QName   string
	QType   uint16
	QClass  uint16
	RD      bool
	CD      bool
	QDCount uint16
	IsQuery bool
}

type ResponseParams struct {
	ID       uint16
	RD       bool
	CD       bool
	Question Question
	Payload  []byte
	RCode    *RCode
}

type QueryReplyError struct {
	ID       uint16
	RD       bool
	CD       bool
	Question *Question
	RCode    RCode
}

func (e *QueryReplyError) Error() string {
	return "dns query requires reply: " + e.RCode.String()
}

type DropError struct{}

func (DropError) Error() string {
	return "dns packet should be dropped"
}

type Error struct {
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

func errf(format string, args ...any) error {
	return &Error{Message: fmt.Sprintf(format, args...)}
}
