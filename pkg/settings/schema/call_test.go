package schema

import "testing"

func TestCallAnswers(t *testing.T) {
	cases := []struct {
		name string
		call Call
		peer string
		want bool
	}{
		{"auto-answer off", Call{AnswerFrom: AnswerEveryone}, "a", false},
		{"everyone", Call{AutoAnswer: true, AnswerFrom: AnswerEveryone}, "a", true},
		{"unset means everyone", Call{AutoAnswer: true}, "a", true},
		{
			"selected and listed",
			Call{AutoAnswer: true, AnswerFrom: AnswerSelected, AnswerPeers: "b a"},
			"a",
			true,
		},
		{
			"selected and not listed",
			Call{AutoAnswer: true, AnswerFrom: AnswerSelected, AnswerPeers: "b"},
			"a",
			false,
		},
		{"selected with nobody", Call{AutoAnswer: true, AnswerFrom: AnswerSelected}, "a", false},
	}
	for _, c := range cases {
		if got := c.call.Answers(c.peer); got != c.want {
			t.Errorf("%s: Answers(%q) = %v, want %v", c.name, c.peer, got, c.want)
		}
	}
}
