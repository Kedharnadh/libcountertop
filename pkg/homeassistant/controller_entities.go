package homeassistant

import (
	"context"
)

func (h *Controller) Entities(ctx context.Context, f Filter) ([]Entity, error) {
	raw, err := h.Template(ctx, f.Template())
	if err != nil {
		return nil, err
	}
	return ParseEntities(raw)
}
