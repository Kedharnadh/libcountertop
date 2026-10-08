package oww

import (
	"fmt"

	"github.com/ygelfand/libcountertop/pkg/inference/tflite"
)

const (
	// The only rate openWakeWord models are trained for.
	SampleRate = 16000

	Step = 1280

	melLookback = 3 * 160

	melBins = 32

	embedFrames = 76
	embedStride = 8

	embedDims = 96
)

const EmbeddingDims = embedDims

type Engine struct {
	mel *tflite.Interpreter

	embed *tflite.Stream

	raw []int16

	mels  []float32
	feats []float32

	classifiers []*Classifier
	scores      map[string]float32
	maxFrames   int

	melScale  float32
	melOffset float32
}

// openWakeWord's mel transform; samples reach the model at int16 magnitudes, undivided.
const (
	melScale  = 0.1
	melOffset = 2
)

func New() (*Engine, error) {
	parsed, err := tflite.Parse(melModel)
	if err != nil {
		return nil, fmt.Errorf("oww: mel model: %w", err)
	}
	mel, err := tflite.New(parsed)
	if err != nil {
		return nil, fmt.Errorf("oww: mel model: %w", err)
	}

	parsed, err = tflite.Parse(embedModel)
	if err != nil {
		return nil, fmt.Errorf("oww: embedding model: %w", err)
	}
	embed, err := tflite.NewStream(parsed, []int{1, embedFrames, melBins, 1})
	if err != nil {
		return nil, fmt.Errorf("oww: embedding model: %w", err)
	}

	e := &Engine{
		mel:   mel,
		embed: embed,
		// Models are trained on frames at multiples of the hop from the first sample.
		raw:       make([]int16, 0, melLookback+Step),
		mels:      make([]float32, 0, embedStride*melBins),
		scores:    map[string]float32{},
		melScale:  melScale,
		melOffset: melOffset,
	}

	return e, nil
}

type Classifier struct {
	ID string

	in     *tflite.Interpreter
	frames int
}

func (e *Engine) Load(id string, model []byte) (*Classifier, error) {
	m, err := tflite.Parse(model)
	if err != nil {
		return nil, fmt.Errorf("oww: %s: %w", id, err)
	}
	in, err := tflite.New(m)
	if err != nil {
		return nil, fmt.Errorf("oww: %s: %w", id, err)
	}

	shape := in.Input(0).Shape
	if len(shape) != 3 || shape[2] != embedDims {
		return nil, fmt.Errorf(
			"oww: %s: input shape %v is not a %d-wide embedding window",
			id,
			shape,
			embedDims,
		)
	}

	c := &Classifier{ID: id, in: in, frames: shape[1]}
	e.classifiers = append(e.classifiers, c)
	if c.frames > e.maxFrames {
		e.maxFrames = c.frames
	}
	return c, nil
}

func (e *Engine) Unload(id string) {
	kept := e.classifiers[:0]
	e.maxFrames = 0
	for _, c := range e.classifiers {
		if c.ID == id {
			continue
		}
		kept = append(kept, c)
		if c.frames > e.maxFrames {
			e.maxFrames = c.frames
		}
	}
	e.classifiers = kept
	delete(e.scores, id)
}

func (e *Engine) Process(pcm []int16) (map[string]float32, error) {
	e.raw = append(e.raw, pcm...)
	clear(e.scores)
	stepped := false

	for len(e.raw) >= melLookback+Step {
		if err := e.step(e.raw[:melLookback+Step]); err != nil {
			return nil, err
		}
		e.raw = append(e.raw[:0], e.raw[Step:]...)
		stepped = true

		if len(e.feats)/embedDims < e.maxFrames {
			continue
		}
		for _, c := range e.classifiers {
			score, err := c.score(e.feats)
			if err != nil {
				return nil, err
			}
			if score > e.scores[c.ID] {
				e.scores[c.ID] = score
			}
		}
	}

	if !stepped || len(e.scores) == 0 {
		return nil, nil
	}
	return e.scores, nil
}

func (e *Engine) step(window []int16) error {
	e.mel.ResizeInput(0, []int{1, len(window)})
	in := e.mel.Input(0)
	for i, s := range window {
		in.F32[i] = float32(s)
	}
	if err := e.mel.Invoke(); err != nil {
		return fmt.Errorf("oww: mel: %w", err)
	}

	out := e.mel.Output(0)
	frames := out.Count() / melBins
	if frames != embedStride {
		return fmt.Errorf("oww: mel produced %d frames for a step, want %d", frames, embedStride)
	}
	e.mels = e.mels[:0]
	for _, v := range out.F32 {
		e.mels = append(e.mels, v*e.melScale+e.melOffset)
	}

	emb, err := e.embed.Write(e.mels)
	if err != nil {
		return err
	}
	if len(emb) == 0 {
		return nil
	}
	if len(emb) != embedDims {
		return fmt.Errorf("oww: embedding produced %d values, want %d", len(emb), embedDims)
	}

	e.feats = append(e.feats, emb...)
	if keep := e.maxFrames * embedDims; keep > 0 && len(e.feats) > keep {
		e.feats = append(e.feats[:0], e.feats[len(e.feats)-keep:]...)
	}
	return nil
}

func (c *Classifier) score(feats []float32) (float32, error) {
	c.in.ResizeInput(0, []int{1, c.frames, embedDims})
	copy(c.in.Input(0).F32, feats[len(feats)-c.frames*embedDims:])

	if err := c.in.Invoke(); err != nil {
		return 0, fmt.Errorf("oww: %s: %w", c.ID, err)
	}
	out := c.in.Output(0)
	if out.Count() == 0 {
		return 0, fmt.Errorf("oww: %s produced no output", c.ID)
	}
	return out.F32[0], nil
}
