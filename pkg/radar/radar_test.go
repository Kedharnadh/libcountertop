package radar

import (
	"context"
	"errors"
	"image"
	"sync/atomic"
	"testing"
	"time"
)

func TestAFailedBuildIsNotRetriedOnEveryFrame(t *testing.T) {
	var tries atomic.Int32
	r := New(Options{Home: func(context.Context) (float64, float64, error) {
		tries.Add(1)
		return 0, 0, errors.New("no home")
	}})
	req := Request{Size: image.Pt(800, 480), Miles: 200}

	deadline := time.Now().Add(5 * time.Second)
	for {
		_, err, _ := r.Get(req)
		if err != nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the build never failed")
		}
		time.Sleep(5 * time.Millisecond)
	}
	for range 200 {
		if _, err, _ := r.Get(req); err == nil {
			t.Fatal("the failure was forgotten")
		}
	}
	time.Sleep(20 * time.Millisecond)

	if n := tries.Load(); n != 1 {
		t.Errorf("home looked up %d times, want 1 until the retry wait passes", n)
	}
}

func TestANewRequestCancelsTheBuildInProgress(t *testing.T) {
	started := make(chan struct{}, 2)
	canceled := make(chan struct{}, 2)
	r := New(Options{Home: func(ctx context.Context) (float64, float64, error) {
		started <- struct{}{}
		<-ctx.Done()
		canceled <- struct{}{}
		return 0, 0, ctx.Err()
	}})
	near := Request{Size: image.Pt(800, 480), Miles: 50}
	far := Request{Size: image.Pt(800, 480), Miles: 400}

	_, _, _ = r.Get(near)
	<-started
	_, _, _ = r.Get(far)

	select {
	case <-canceled:
	case <-time.After(5 * time.Second):
		t.Fatal("the first build kept running after the range changed")
	}
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("the new range never started building")
	}
	r.mu.Lock()
	cancel := r.cancel
	r.mu.Unlock()
	cancel()
}
