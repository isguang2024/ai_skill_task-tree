package tasktree

import "fmt"

func ensureExpectedVersion(item jsonMap, expected *int, entity string) error {
	if expected == nil {
		return nil
	}
	current := int(asFloat(item["version"]))
	if current != *expected {
		return &appError{
			Code: 409,
			Msg:  fmt.Sprintf("%s version mismatch: expected %d, got %d", entity, *expected, current),
		}
	}
	return nil
}
