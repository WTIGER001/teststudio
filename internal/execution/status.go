package execution

func NormalizeTimeout(result Result) Result {
	if result.State == "failed" && result.FailureKind == "timeout" {
		result.State = "timed-out"
		if result.ObservedState == "failed" {
			result.ObservedState = "timed-out"
		}
	}
	return result
}
