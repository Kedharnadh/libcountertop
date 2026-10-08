package oww

import _ "embed"

// Every openWakeWord classifier is trained against these two models.
var (
	//go:embed assets/melspectrogram.tflite
	melModel []byte

	//go:embed assets/embedding_model.tflite
	embedModel []byte
)
