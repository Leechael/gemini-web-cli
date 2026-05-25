package protocol

// ArrayAt walks array-encoded protobuf JSON with numeric indexes and returns the array at that path.
// It lets RPC decoders describe field access as paths instead of repeating type assertions.
func ArrayAt(root any, path ...int) ([]any, bool) {
	value, ok := ValueAt(root, path...)
	if !ok {
		return nil, false
	}
	arr, ok := value.([]any)
	return arr, ok
}

// ValueAt walks array-encoded protobuf JSON with numeric indexes and returns the raw value at that path.
// It returns false when any path segment is not an array or is out of bounds.
func ValueAt(root any, path ...int) (any, bool) {
	cur := root
	for _, idx := range path {
		arr, ok := cur.([]any)
		if !ok || idx < 0 || idx >= len(arr) {
			return nil, false
		}
		cur = arr[idx]
	}
	return cur, true
}

// StringAt walks array-encoded protobuf JSON with numeric indexes and returns the string at that path.
// It returns an empty string when the path is missing or the value is not a string.
func StringAt(root any, path ...int) string {
	value, ok := ValueAt(root, path...)
	if !ok {
		return ""
	}
	s, _ := value.(string)
	return s
}

// IntAt walks array-encoded protobuf JSON with numeric indexes and returns the integer at that path.
// JSON numbers decode as float64; non-integral values return zero.
func IntAt(root any, path ...int) int {
	value, ok := ValueAt(root, path...)
	if !ok {
		return 0
	}
	switch v := value.(type) {
	case float64:
		if v == float64(int(v)) {
			return int(v)
		}
	case int:
		return v
	}
	return 0
}

// BoolAt walks array-encoded protobuf JSON with numeric indexes and returns the bool at that path.
// It returns false when the path is missing or the value is not a bool.
func BoolAt(root any, path ...int) bool {
	value, ok := ValueAt(root, path...)
	if !ok {
		return false
	}
	b, _ := value.(bool)
	return b
}

// FirstString returns the first non-empty string from values.
// RPC decoders use it when the same logical field appears in multiple observed slots.
func FirstString(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
