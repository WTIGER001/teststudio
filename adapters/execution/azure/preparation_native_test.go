package azureworker

import (
	"testing"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzurePreparationNativeRequiresSeparateExplicitReader(t *testing.T) {
	c, writer, _, _, _ := preparationWriterTest(t)
	s := PreparationSettings{Worker: c, Writer: writer, ReaderClientID: "aaaaaaaa-1111-2222-3333-444444444444"}
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
}
