package tasktree

func itemResult(item jsonMap) string {
	result := asString(item["result"])
	if result != "" {
		return result
	}
	switch asString(item["status"]) {
	case "done", "canceled":
		return asString(item["status"])
	default:
		return ""
	}
}

func hasClosedResult(item jsonMap) bool {
	return itemResult(item) != ""
}

func isDoneResult(item jsonMap) bool {
	return itemResult(item) == "done"
}

func isCanceledResult(item jsonMap) bool {
	return itemResult(item) == "canceled"
}

func shouldResetProgressOnReopen(item jsonMap) bool {
	return isDoneResult(item) || asFloat(item["progress"]) >= 1
}

func rollupClosedResult(items []jsonMap) string {
	if len(items) == 0 {
		return ""
	}
	doneOnly := true
	canceledOnly := true
	for _, item := range items {
		switch itemResult(item) {
		case "done":
			canceledOnly = false
		case "canceled":
			doneOnly = false
		default:
			doneOnly = false
			canceledOnly = false
		}
	}
	switch {
	case doneOnly:
		return "done"
	case canceledOnly:
		return "canceled"
	default:
		return "mixed"
	}
}

func statusFromResult(result string) string {
	switch result {
	case "done":
		return "done"
	case "canceled":
		return "canceled"
	case "mixed":
		return "closed"
	default:
		return ""
	}
}

