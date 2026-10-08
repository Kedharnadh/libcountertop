package tflite

import (
	"bufio"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func referenceSignal(n int) []float32 {
	out := make([]float32, n)
	for i := range out {
		t := float64(i) / 16000
		env := 0.4 + 0.6*math.Sin(2*math.Pi*3*t)
		v := env * (6000*math.Sin(2*math.Pi*440*t) + 2500*math.Sin(2*math.Pi*1750*t))
		out[i] = float32(int16(v))
	}
	return out
}

func readGolden(t *testing.T, path string) []float32 {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Skip(err)
	}
	defer f.Close()

	var out []float32
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		v, err := strconv.ParseFloat(line, 32)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		out = append(out, float32(v))
	}
	if err := s.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestMelMatchesReferenceRuntime(t *testing.T) {
	want := readGolden(t, "testdata/mel_reference.txt")

	raw, err := os.ReadFile("../oww/assets/melspectrogram.tflite")
	if err != nil {
		t.Skip(err)
	}
	m, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	in, err := New(m)
	if err != nil {
		t.Fatal(err)
	}

	const samples = 1760
	in.ResizeInput(0, []int{1, samples})
	copy(in.Input(0).F32, referenceSignal(samples))
	if err := in.Invoke(); err != nil {
		t.Fatal(err)
	}

	got := in.Output(0).F32
	if len(got) != len(want) {
		t.Fatalf("got %d values, reference has %d", len(got), len(want))
	}

	var worst float64
	var at int
	for i := range want {
		if d := math.Abs(float64(got[i] - want[i])); d > worst {
			worst, at = d, i
		}
	}
	t.Logf(
		"largest difference %.6g at index %d (ours %.6f, reference %.6f)",
		worst,
		at,
		got[at],
		want[at],
	)

	if worst > 0.01 {
		t.Errorf("our mel output differs from the reference by %.6g", worst)
	}
}

func referenceMelFrames(n int) []float32 {
	out := make([]float32, n)
	for i := range out {
		f := float64(i)
		out[i] = float32(6 + 4*math.Sin(f*0.037) + 3*math.Cos(f*0.011) - 2*math.Sin(f*0.9))
	}
	return out
}

func TestEmbeddingMatchesReferenceRuntime(t *testing.T) {
	want := readGolden(t, "testdata/embedding_reference.txt")

	raw, err := os.ReadFile("../oww/assets/embedding_model.tflite")
	if err != nil {
		t.Skip(err)
	}
	m, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	in, err := New(m)
	if err != nil {
		t.Fatal(err)
	}

	in.ResizeInput(0, []int{1, 76, 32, 1})
	copy(in.Input(0).F32, referenceMelFrames(76*32))
	if err := in.Invoke(); err != nil {
		t.Fatal(err)
	}

	got := in.Output(0).F32
	if len(got) != len(want) {
		t.Fatalf("got %d values, reference has %d", len(got), len(want))
	}

	var worst float64
	var at int
	for i := range want {
		if d := math.Abs(float64(got[i] - want[i])); d > worst {
			worst, at = d, i
		}
	}
	t.Logf(
		"largest difference %.6g at index %d (ours %.6f, reference %.6f)",
		worst,
		at,
		got[at],
		want[at],
	)
	if worst > 0.01 {
		t.Errorf("our embedding output differs from the reference by %.6g", worst)
	}
}

func referenceEmbeddings() []float32 {
	out := make([]float32, 16*96)
	for i := range out {
		f := float64(i)
		out[i] = float32(0.6*math.Sin(f*0.037) + 0.4*math.Cos(f*0.011) - 0.3*math.Sin(f*0.9))
	}
	return out
}

func TestClassifierMatchesReferenceRuntime(t *testing.T) {
	model := os.Getenv("CLASSIFIER")
	if model == "" {
		t.Skip("set CLASSIFIER to a model testdata has a fixture for")
	}

	stem := strings.TrimSuffix(filepath.Base(model), ".tflite")
	want := readGolden(t, filepath.Join("testdata", "classifier_"+stem+"_reference.txt"))

	raw, err := os.ReadFile(model)
	if err != nil {
		t.Skip(err)
	}
	m, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	in, err := New(m)
	if err != nil {
		t.Fatal(err)
	}

	copy(in.Input(0).F32, referenceEmbeddings())
	if err := in.Invoke(); err != nil {
		t.Fatal(err)
	}

	got := in.Output(0).F32
	if len(got) != len(want) {
		t.Fatalf("got %d values, reference has %d", len(got), len(want))
	}

	var worst float64
	for i := range want {
		if d := math.Abs(float64(got[i] - want[i])); d > worst {
			worst = d
		}
	}
	t.Logf("ours %v, reference %v, largest difference %.6g", got, want, worst)

	if worst > 1e-4 {
		t.Errorf("our classifier output differs from the reference by %.6g", worst)
	}
}
