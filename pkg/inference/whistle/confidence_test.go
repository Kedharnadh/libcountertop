package whistle

import (
	"encoding/binary"
	"math"
	"testing"
)

func TestTranscriptCarriesWordProbabilities(t *testing.T) {
	head := make([]byte, 16)
	binary.LittleEndian.PutUint32(head[8:], 120)
	body := `{"text":"show radar","words":[` +
		`{"word":" show","start":0.1,"end":0.4,"probability":0.92},` +
		`{"word":" radar","start":0.4,"end":0.9,"probability":0.64}]}`
	tr, err := transcript(message{kind: msgText, payload: append(head, body...)})
	if err != nil {
		t.Fatal(err)
	}
	if len(tr.Words) != 2 || tr.Words[1].Text != " radar" {
		t.Fatalf("words = %+v", tr.Words)
	}
	lowest, mean := tr.Confidence()
	if lowest != 0.64 || math.Abs(mean-0.78) > 1e-9 {
		t.Errorf("Confidence() = %v, %v; want 0.64, 0.78", lowest, mean)
	}
	if got := wordProbabilities(tr.Words); got != "show:0.92 radar:0.64" {
		t.Errorf("wordProbabilities = %q", got)
	}
}

func TestNoWordsNoConfidence(t *testing.T) {
	if lowest, mean := (Transcript{}).Confidence(); lowest != 0 || mean != 0 {
		t.Errorf("Confidence() = %v, %v; want zeros", lowest, mean)
	}
}
