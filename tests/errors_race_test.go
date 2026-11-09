package tests

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	gooseErr "github.com/awesome-goose/goose/errors"
)

// The package-level sentinels are shared by every request. Wrapping one per
// request (kernel.go: ErrFailedToHydrateInput.WithError(err)) must not change
// it, or concurrent requests overwrite each other's wrapped error. Run with
// -race: before the With* methods returned copies this was a data race, and
// a request could read another request's error back.
func TestSentinelWithIsSafeConcurrently(t *testing.T) {
	sentinel := gooseErr.ErrFailedToHydrateInput
	before := *sentinel

	var wg sync.WaitGroup
	failures := make(chan string, 64)
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			inner := fmt.Errorf("request %d", i)
			for j := 0; j < 200; j++ {
				got := sentinel.WithError(inner).WithMeta(i).WithDetail(inner.Error()).WithSuggestedAction("retry")
				if got.Err != inner || got.Meta != i || got.Detail != inner.Error() {
					failures <- fmt.Sprintf("request %d read %v / %v", i, got.Err, got.Meta)
					return
				}
				if !errors.Is(got, gooseErr.ErrFailedToHydrateInput) || !errors.Is(got, inner) {
					failures <- fmt.Sprintf("request %d: errors.Is lost the sentinel or the cause", i)
					return
				}
			}
		}(i)
	}
	wg.Wait()
	close(failures)
	for f := range failures {
		t.Error(f)
	}
	if sentinel.Err != before.Err || sentinel.Meta != before.Meta || sentinel.Detail != before.Detail || sentinel.SuggestedAction != before.SuggestedAction {
		t.Fatalf("the sentinel was changed: %+v", *sentinel)
	}
}
