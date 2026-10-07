package control

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type TouchPhase uint8

const (
	TouchDown TouchPhase = iota
	TouchMove
	TouchUp
)

type Contact struct {
	ID, X, Y int
	Phase    TouchPhase
	At       time.Time
}

type InputOptions struct {
	Deliver            func(Contact)
	Buttons            []string
	Press              func(string) error
	Tap, Step, Between time.Duration
	Steps              int
}

// Input serializes synthetic gestures and releases a held contact even when canceled.
type Input struct {
	options InputOptions
	gate    chan struct{}
	id      int
}

func NewInput(options InputOptions) *Input {
	if options.Step <= 0 {
		options.Step = 8 * time.Millisecond
	}
	if options.Tap <= 0 {
		options.Tap = options.Step
	}
	if options.Between <= 0 {
		options.Between = 120 * time.Millisecond
	}
	if options.Steps <= 0 {
		options.Steps = 20
	}
	options.Buttons = slices.Clone(options.Buttons)
	s := &Input{options: options, gate: make(chan struct{}, 1)}
	s.gate <- struct{}{}
	return s
}

func (s *Input) locked(ctx context.Context, run func() error) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.gate:
	}
	defer func() { s.gate <- struct{}{} }()
	if err := ctx.Err(); err != nil {
		return err
	}
	return run()
}

func delay(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func coordinates(args []string, count int) ([]int, error) {
	out := make([]int, count)
	for i := range count {
		v, err := strconv.Atoi(args[i])
		if err != nil {
			return nil, fmt.Errorf("coordinate %q: %w", args[i], err)
		}
		out[i] = v
	}
	return out, nil
}

func (s *Input) gesture(ctx context.Context, x, y int, move func(func(int, int)) error) error {
	return s.locked(ctx, func() error {
		s.id-- // Kernel tracking IDs are nonnegative; synthetic IDs occupy a separate range.
		id := s.id
		s.options.Deliver(Contact{ID: id, X: x, Y: y, Phase: TouchDown, At: time.Now()})
		defer func() { s.options.Deliver(Contact{ID: id, X: x, Y: y, Phase: TouchUp, At: time.Now()}) }()
		return move(func(nx, ny int) {
			x, y = nx, ny
			s.options.Deliver(Contact{ID: id, X: x, Y: y, Phase: TouchMove, At: time.Now()})
		})
	})
}

func (s *Input) Commands() []*cobra.Command {
	var commands []*cobra.Command
	if s.options.Deliver != nil {
		commands = append(commands,
			&cobra.Command{Use: "tap X Y", Short: "Touch once and release", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
				at, err := coordinates(args, 2)
				if err != nil {
					return err
				}
				return s.gesture(cmd.Context(), at[0], at[1], func(func(int, int)) error { return delay(cmd.Context(), s.options.Tap) })
			}},
			&cobra.Command{Use: "hold X Y MILLISECONDS", Short: "Hold a contact and release", Args: cobra.ExactArgs(3), RunE: func(cmd *cobra.Command, args []string) error {
				at, err := coordinates(args, 2)
				if err != nil {
					return err
				}
				ms, err := strconv.Atoi(args[2])
				if err != nil || ms < 0 || ms > 60000 {
					return fmt.Errorf("hold must be 0 to 60000 milliseconds")
				}
				return s.gesture(cmd.Context(), at[0], at[1], func(move func(int, int)) error {
					for left := time.Duration(ms) * time.Millisecond; left > 0; left -= s.options.Step {
						if err := delay(cmd.Context(), min(left, s.options.Step)); err != nil {
							return err
						}
						move(at[0], at[1])
					}
					return nil
				})
			}},
			&cobra.Command{Use: "swipe X1 Y1 X2 Y2 [STEPS]", Short: "Drag between viewed coordinates", Args: cobra.RangeArgs(4, 5), RunE: func(cmd *cobra.Command, args []string) error {
				at, err := coordinates(args, 4)
				if err != nil {
					return err
				}
				steps := s.options.Steps
				if len(args) == 5 {
					steps, err = strconv.Atoi(args[4])
				}
				if err != nil || steps < 1 || steps > int(time.Minute/s.options.Step) {
					return fmt.Errorf("steps must produce a swipe of at most one minute")
				}
				return s.gesture(cmd.Context(), at[0], at[1], func(move func(int, int)) error {
					for i := 1; i <= steps; i++ {
						if err := delay(cmd.Context(), s.options.Step); err != nil {
							return err
						}
						move(at[0]+(at[2]-at[0])*i/steps, at[1]+(at[3]-at[1])*i/steps)
					}
					return nil
				})
			}},
		)
	}
	if s.options.Press != nil {
		commands = append(commands, &cobra.Command{Use: "button " + strings.Join(s.options.Buttons, "|") + " [TIMES]", Short: "Press a hardware button", Args: cobra.RangeArgs(1, 2), RunE: func(cmd *cobra.Command, args []string) error {
			if !slices.Contains(s.options.Buttons, args[0]) {
				return fmt.Errorf("no button %q, want %s", args[0], strings.Join(s.options.Buttons, ", "))
			}
			times := 1
			var err error
			if len(args) == 2 {
				times, err = strconv.Atoi(args[1])
			}
			if err != nil || times < 1 || times > 500 {
				return fmt.Errorf("times must be 1 to 500")
			}
			return s.locked(cmd.Context(), func() error {
				for i := range times {
					if i > 0 {
						if err := delay(cmd.Context(), s.options.Between); err != nil {
							return err
						}
					}
					if err := s.options.Press(args[0]); err != nil {
						return err
					}
				}
				return nil
			})
		}})
	}
	return commands
}
