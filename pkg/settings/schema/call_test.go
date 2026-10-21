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
		{
			"everyone includes the API",
			Call{AutoAnswer: true, AnswerFrom: AnswerEveryone},
			APICaller,
			true,
		},
		{
			"selected, API allowed",
			Call{AutoAnswer: true, AnswerFrom: AnswerSelected, AnswerAPI: true},
			APICaller,
			true,
		},
		{
			"selected, API not allowed",
			Call{AutoAnswer: true, AnswerFrom: AnswerSelected, AnswerPeers: APICaller},
			APICaller,
			false,
		},
		{
			"API allowed but auto-answer off",
			Call{AnswerFrom: AnswerSelected, AnswerAPI: true},
			APICaller,
			false,
		},
	}
	for _, c := range cases {
		if got := c.call.Answers(c.peer); got != c.want {
			t.Errorf("%s: Answers(%q) = %v, want %v", c.name, c.peer, got, c.want)
		}
	}
}

func TestCallAnswersUser(t *testing.T) {
	cases := []struct {
		name string
		call Call
		user string
		want bool
	}{
		{"auto-answer off", Call{AnswerFrom: AnswerEveryone, AnswerUsers: "u1"}, "u1", false},
		{"everyone", Call{AutoAnswer: true, AnswerFrom: AnswerEveryone}, "u1", true},
		{
			"selected and listed",
			Call{AutoAnswer: true, AnswerFrom: AnswerSelected, AnswerUsers: "u2 u1"},
			"u1",
			true,
		},
		{
			"selected, listed only as a panel",
			Call{AutoAnswer: true, AnswerFrom: AnswerSelected, AnswerPeers: "u1"},
			"u1",
			false,
		},
		{
			"selected and not listed",
			Call{AutoAnswer: true, AnswerFrom: AnswerSelected, AnswerUsers: "u2"},
			"u1",
			false,
		},
	}
	for _, c := range cases {
		if got := c.call.AnswersUser(c.user); got != c.want {
			t.Errorf("%s: AnswersUser(%q) = %v, want %v", c.name, c.user, got, c.want)
		}
	}
}
