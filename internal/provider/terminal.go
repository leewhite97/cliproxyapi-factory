package provider

// A semantic end may be truncated but valid; EOF alone is not completion.
func isTerminalEvent(t string) bool {
	return t == "message_stop" || t == "response.completed" || t == "response.incomplete"
}
