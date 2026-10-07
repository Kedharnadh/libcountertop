package call

import (
	"github.com/ygelfand/libcountertop/pkg/say"
)

func (c *Calls) notice(peer string, reason Reason) {
	var body string
	switch reason {
	case ReasonDeclined:
		body = say.T("call.declined")
	case ReasonUnanswered:
		body = say.T("call.unanswered")
	case ReasonBusy:
		body = say.T("call.busy")
	case ReasonUnavailable:
		body = say.T("call.unavailable")
	case ReasonDropped:
		body = say.T("call.dropped")
	case ReasonFailed:
		body = say.T("call.failed")
	default:
		return
	}
	c.options.Messages.Info(peer, body, noticeFor)
}
