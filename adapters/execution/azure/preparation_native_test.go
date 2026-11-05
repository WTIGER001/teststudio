package azureworker

import (
	"testing"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzurePreparationNativeRequiresSeparateExplicitReader(t *testing.T) {
	s, _, _ := preparationHostTest(t)
	network := preparationNetworkDefaults(s)
	s.Network = &network
	c, writer := s.Worker, s.Writer
	if err := s.ValidateNative(); err != nil {
		t.Fatal("separate explicit reader rejected", err)
	}
	for _, id := range []string{"", "default", allDenyPrincipals, writer.ClientID} {
		changed := s
		changed.ReaderClientID = id
		requireDeadlineError(t, changed.ValidateNative(), hostedexecution.ErrConfiguration)
	}
	changed := s
	changed.Writer.Identity = c.PullIdentity
	requireDeadlineError(t, changed.ValidateNative(), hostedexecution.ErrConfiguration)
	changed = s
	changed.Host = nil
	requireDeadlineError(t, changed.ValidateNative(), hostedexecution.ErrConfiguration)
	changed = s
	changed.Network = nil
	requireDeadlineError(t, changed.ValidateNative(), hostedexecution.ErrConfiguration)
	for _, mutate := range []func(*PreparationHostSettings){
		func(h *PreparationHostSettings) { h.Environment = c.WorkerEnvironment },
		func(h *PreparationHostSettings) { h.ReaderIdentity = writer.Identity },
		func(h *PreparationHostSettings) { h.JournalStorageName = "../foreign" },
		func(h *PreparationHostSettings) { h.GoRuntime = "foreign.azurecr.io/runtime:latest" },
	} {
		host := *s.Host
		mutate(&host)
		changed = s
		changed.Host = &host
		requireDeadlineError(t, changed.ValidateNative(), hostedexecution.ErrConfiguration)
	}
}
