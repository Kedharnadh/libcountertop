package schema

import "github.com/ygelfand/libcountertop/pkg/say"

type AlertStyle string

const (
	AlertBanner AlertStyle = "banner"
	AlertCard   AlertStyle = "card"
	AlertFull   AlertStyle = "full"
	AlertToast  AlertStyle = "toast"
)

const DefaultAlertStyle = AlertBanner

func (s AlertStyle) Label() string {
	switch s {
	case AlertBanner:
		return say.T("alertstyle.banner")
	case AlertCard:
		return say.T("alertstyle.card")
	case AlertFull:
		return say.T("alertstyle.full")
	case AlertToast:
		return say.T("alertstyle.toast")
	}
	return string(s)
}

func AlertStyles() []AlertStyle {
	return []AlertStyle{AlertBanner, AlertCard, AlertFull, AlertToast}
}
