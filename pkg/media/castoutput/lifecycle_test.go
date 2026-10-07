package castoutput

import (
	"context"
	"testing"
	"testing/synctest"

	"github.com/ygelfand/libcountertop/pkg/display/videopage"
	"github.com/ygelfand/libcountertop/pkg/media/videostream"
)

type exiting struct {
	ending
	reading, cancelled, finish chan struct{}
}

func (s *exiting) Read(ctx context.Context, _ []int16) (int, error) {
	close(s.reading)
	<-ctx.Done()
	close(s.cancelled)
	<-s.finish
	return 0, ctx.Err()
}

func TestStopJoinsTheAudioWorker(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		o := newOutput()
		s := &exiting{reading: make(chan struct{}), cancelled: make(chan struct{}), finish: make(chan struct{})}
		o.Play(s)
		<-s.reading
		stopped := make(chan struct{})
		go func() { o.Stop(s); close(stopped) }()
		<-s.cancelled
		synctest.Wait()
		select {
		case <-stopped:
			t.Fatal("Stop returned while audio was still exiting")
		default:
		}
		close(s.finish)
		<-stopped
	})
}

func TestViewRetirementJoinsThePageWorker(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		o := newOutput()
		cancelled, done := make(chan struct{}), make(chan struct{})
		o.view, o.viewStop, o.viewDone, o.under = &videopage.Page{}, func() { close(cancelled) }, done, &videostream.Beneath{}
		retired := make(chan struct{})
		go func() { o.retireView(0); close(retired) }()
		<-cancelled
		synctest.Wait()
		select {
		case <-retired:
			t.Fatal("view retired before its page worker exited")
		default:
		}
		close(done)
		<-retired
	})
}
