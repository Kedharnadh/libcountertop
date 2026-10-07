package homeassistant

import (
	"context"
)

func (h *Controller) Labels(ctx context.Context) ([]Label, error) {
	raw, err := h.Template(ctx, LabelsTemplate)
	if err != nil {
		return nil, err
	}
	return ParseLabels(raw)
}
