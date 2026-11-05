package azuresandbox

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/workerexchange"
	"github.com/wtiger001/teststudio/internal/workerprotocol"
)

func matchesWork(proof Proof, work workerexchange.Record) bool {
	sum := sha256.Sum256(work.Input)
	if hex.EncodeToString(sum[:]) != work.InputHash {
		return false
	}
	if proof.Operation == "discover" {
		var input workerprotocol.DiscoveryInput
		return json.Unmarshal(work.Input, &input) == nil && workerprotocol.ValidateDiscoveryInput(input) == nil && input.Image == proof.Image && input.Runner == proof.Runner && input.Project == proof.Reference.Project && input.SourceDigest == proof.SourceDigest
	}
	var input execution.CodedInput
	if json.Unmarshal(work.Input, &input) != nil || input.RunID != proof.Reference.Run || input.Test.ID != proof.Reference.Test || input.Test.Runner != proof.Runner || input.Artifact.Image != proof.Image || input.Artifact.SourceProject != "" && input.Artifact.SourceProject != proof.Reference.Project {
		return false
	}
	_, err := execution.MarshalWorkItem(input)
	return err == nil
}
