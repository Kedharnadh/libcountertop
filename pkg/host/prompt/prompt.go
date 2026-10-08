package prompt

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"charm.land/huh/v2"
	"github.com/charmbracelet/x/term"
)

var ErrCanceled = errors.New("canceled")

func IsTerminal() bool { return term.IsTerminal(os.Stdout.Fd()) }

func Ask(ctx context.Context, out io.Writer, field huh.Field) error {
	err := huh.NewForm(huh.NewGroup(field)).
		WithOutput(out).
		RunWithContext(ctx)
	if errors.Is(err, huh.ErrUserAborted) {
		return ErrCanceled
	}
	return err
}

func Choose[T any](
	ctx context.Context,
	out io.Writer,
	title string,
	items []T,
	label func(T) string,
	other string,
) (T, error) {
	var chosen T

	options := make([]huh.Option[int], 0, len(items)+1)
	for i, item := range items {
		options = append(options, huh.NewOption(label(item), i))
	}
	if other != "" {
		options = append(options, huh.NewOption(other, len(items)))
	}

	var at int
	field := huh.NewSelect[int]().Title(title).Options(options...).Value(&at)
	if err := Ask(ctx, out, field); err != nil {
		return chosen, err
	}
	if at < len(items) {
		chosen = items[at]
	}
	return chosen, nil
}

func Line(ctx context.Context, out io.Writer, title, value string, secret bool) (string, error) {
	field := huh.NewInput().Title(title).Value(&value)
	if secret {
		field = field.EchoMode(huh.EchoModePassword)
	} else {
		field = field.Validate(notEmpty)
	}

	err := Ask(ctx, out, field)
	return value, err
}

func Confirm(ctx context.Context, out io.Writer, title string) (bool, error) {
	return confirm(ctx, out, title, false)
}

func ConfirmDefaultYes(ctx context.Context, out io.Writer, title string) (bool, error) {
	return confirm(ctx, out, title, true)
}

func confirm(ctx context.Context, out io.Writer, title string, yes bool) (bool, error) {
	err := Ask(ctx, out, huh.NewConfirm().Title(title).Value(&yes))
	return yes, err
}

func Typed(ctx context.Context, out io.Writer, title, want string) (bool, error) {
	var answer string
	field := huh.NewInput().Title(title).Value(&answer).Validate(func(s string) error {
		if s != want {
			return fmt.Errorf("type %s to continue", want)
		}
		return nil
	})

	if err := Ask(ctx, out, field); err != nil {
		return false, err
	}
	return answer == want, nil
}

func notEmpty(s string) error {
	if s == "" {
		return errors.New("cannot be empty")
	}
	return nil
}
