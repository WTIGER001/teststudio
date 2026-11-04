package execution

func CaptureAttachment(input CodedInput, result Result, file Attachment) bool {
	failed := input.FailedBefore || result.State == "failed" || result.State == "timed-out" || result.State == "unknown" || input.TimingMode == "fail" && input.SlowThresholdMs != nil && result.DurationMs >= *input.SlowThresholdMs
	mode := "always"
	switch file.Kind {
	case "log":
		return input.Capture.Logs
	case "screenshot":
		mode = input.Capture.Screenshots
	case "trace":
		mode = input.Capture.Trace
	case "video":
		mode = input.Capture.Video
	case "attachment":
	default:
		return false
	}
	return mode == "always" || mode == "failure" && failed
}
