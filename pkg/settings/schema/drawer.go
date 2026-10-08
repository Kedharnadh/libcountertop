package schema

import "github.com/ygelfand/libcountertop/pkg/say"

type Edge string

const (
	EdgeLeft   Edge = "left"
	EdgeRight  Edge = "right"
	EdgeTop    Edge = "top"
	EdgeBottom Edge = "bottom"
)

const DefaultEdge = EdgeRight

func (e Edge) Label() string {
	switch e {
	case EdgeLeft:
		return say.T("edge.left")
	case EdgeRight:
		return say.T("edge.right")
	case EdgeTop:
		return say.T("edge.top")
	case EdgeBottom:
		return say.T("edge.bottom")
	}
	return string(e)
}

func Edges() []Edge { return []Edge{EdgeLeft, EdgeRight, EdgeTop, EdgeBottom} }

func Sides() []Edge { return []Edge{EdgeLeft, EdgeRight} }
