package rpcs

import "encoding/json"

// LogCounter is one metric counter tuple for ozz5Z.
type LogCounter struct {
	ID    string
	Value int
}

// EncodeBulkLogCounter returns RPC ozz5Z — BulkLogCounter.
func EncodeBulkLogCounter(counters []LogCounter) (rpcID, payload string) {
	triples := make([]any, 0, len(counters))
	for _, c := range counters {
		triples = append(triples, []any{nil, c.ID, c.Value})
	}
	arr := []any{[]any{triples}}
	bytes, _ := json.Marshal(arr)
	return "ozz5Z", string(bytes)
}
