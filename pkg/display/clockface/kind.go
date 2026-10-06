package face

// Kind is a stable clock-face identifier, independent of a product settings schema.
type Kind string

const (
 Plain Kind = "plain"
 Stack Kind = "stack"
 Cards Kind = "cards"
 Analog Kind = "analog"
 AnalogSeconds Kind = "analog-seconds"
 Segments Kind = "segments"
 Words Kind = "words"
 Overlap Kind = "overlap"
 Default = Plain
)

// Kinds returns the available clock-face identifiers.
func Kinds() []Kind {
 return []Kind{Plain,Stack,Cards,Analog,AnalogSeconds,Segments,Words,Overlap}
}

