package rpcs

import (
	"encoding/json"
	"fmt"
)

// LogClientEventOpts carries caller-supplied fields for dI8W6e.
type LogClientEventOpts struct {
	DeviceID string
	ModelID  int
	EventID  int
}

// EncodeLogClientEvent is intentionally stubbed until a MyActivity iframe HAR
// sample confirms the device-id payload shape.
func EncodeLogClientEvent(opts LogClientEventOpts) (rpcID, payload string, err error) {
	return "dI8W6e", "", fmt.Errorf("dI8W6e requires device-id; see protocol/rpcs/housekeeping_log_event.go TODO")
}

// LogModelSelectionOpts carries TFNzk model-selection log fields.
type LogModelSelectionOpts struct {
	Selector1 int
	Selector2 int
	ModelID   int
	ExpIDs    []int
}

// EncodeLogModelSelection returns RPC TFNzk — LogModelSelection.
func EncodeLogModelSelection(opts LogModelSelectionOpts) (rpcID, payload string) {
	expArr := make([]any, len(opts.ExpIDs))
	for i, e := range opts.ExpIDs {
		expArr[i] = e
	}
	arr := []any{
		[]any{opts.Selector1, nil, opts.Selector2},
		nil,
		[]any{[]any{opts.ModelID}, expArr},
	}
	bytes, _ := json.Marshal(arr)
	return "TFNzk", string(bytes)
}
