package tools

func ReadOnlyAnnotations() map[string]bool {
	return map[string]bool{
		"readOnlyHint":    true,
		"destructiveHint": false,
		"idempotentHint":  true,
		"openWorldHint":   false,
	}
}

func DestructiveAnnotations() map[string]bool {
	return map[string]bool{
		"readOnlyHint":    false,
		"destructiveHint": true,
		"idempotentHint":  false,
		"openWorldHint":   false,
	}
}

func SafeWriteAnnotations() map[string]bool {
	return map[string]bool{
		"readOnlyHint":    false,
		"destructiveHint": false,
		"idempotentHint":  true,
		"openWorldHint":   false,
	}
}

func NonIdempotentWriteAnnotations() map[string]bool {
	return map[string]bool{
		"readOnlyHint":    false,
		"destructiveHint": false,
		"idempotentHint":  false,
		"openWorldHint":   false,
	}
}
