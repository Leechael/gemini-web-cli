package types

import (
	"encoding/json"
	"fmt"
)

// ModelOutput holds the parsed response from Gemini.
type ModelOutput struct {
	Text             string
	TextDelta        string
	RCid             string
	Metadata         []string // [cid, rid, rcid, ...]
	Images           []Image
	Videos           []Video
	Media            []GeneratedMedia
	Done             bool
	DeepResearchPlan *DeepResearchPlanData // non-nil when a plan is detected
}

// DeepResearchPlanData holds the extracted plan from candidate structured data.
type DeepResearchPlanData struct {
	Title         string
	Steps         []string
	ETAText       string
	ConfirmPrompt string
}

// Image represents a web or generated image in the response.
type Image struct {
	URL       string
	Title     string
	Alt       string
	Generated bool
}

// Video represents a generated video in the response.
type Video struct {
	URL       string
	Thumbnail string
}

// GeneratedMedia represents generated music/audio media in the response.
type GeneratedMedia struct {
	MP3URL       string
	MP3Thumbnail string
	MP4URL       string
	MP4Thumbnail string
}

// DeepResearchPlan holds the plan returned by create_deep_research_plan.
type DeepResearchPlan struct {
	Cid     string
	Title   string
	ETAText string
	Steps   []string
}

// ChatItem represents a single chat in the list.
type ChatItem struct {
	Cid       string
	Title     string
	UpdatedAt string
}

// ChatTurn represents a single user/assistant turn in a conversation.
type ChatTurn struct {
	UserPrompt        string
	AssistantResponse string
	RCid              string
	Rid               string
	Images            []Image
	Videos            []Video
	Media             []GeneratedMedia
}

// GroundingSource represents a search citation.
type GroundingSource struct {
	URL   string
	Title string
}

// Model represents a Gemini model configuration.
type Model struct {
	Name         string
	DisplayName  string
	Description  string
	AdvancedOnly bool
	Headers      map[string]string
}

// ModelHeaderKey is the primary header key used for model selection.
const ModelHeaderKey = "x-goog-ext-525001261-jspb"

// BuildModelHeader constructs the HTTP headers required for model selection.
func BuildModelHeader(modelID string, capacityTail string) map[string]string {
	return map[string]string{
		ModelHeaderKey:             fmt.Sprintf(`[1,null,null,null,"%s",null,null,0,[4],null,null,%s]`, modelID, capacityTail),
		"x-goog-ext-73010989-jspb": "[0]",
		"x-goog-ext-73010990-jspb": "[0]",
	}
}

// ModelID extracts the internal model ID from the model headers.
func (m *Model) ModelID() string {
	hdr := m.Headers[ModelHeaderKey]
	if hdr == "" {
		return ""
	}
	// Parse: [1,null,null,null,"<id>",...]
	var arr []any
	if err := json.Unmarshal([]byte(hdr), &arr); err != nil {
		return ""
	}
	if len(arr) > 4 {
		if s, ok := arr[4].(string); ok {
			return s
		}
	}
	return ""
}

// Known models matching the Python library constants.
var Models = []Model{
	{Name: "unspecified", DisplayName: "Auto-select", Headers: map[string]string{}},
	{Name: "gemini-3-pro", DisplayName: "Gemini 3 Pro", Headers: BuildModelHeader("9d8ca3786ebdfbea", "1")},
	{Name: "gemini-3-flash", DisplayName: "Gemini 3 Flash", Headers: BuildModelHeader("fbb127bbb056c959", "1")},
	{Name: "gemini-3-flash-thinking", DisplayName: "Gemini 3 Flash Thinking", Headers: BuildModelHeader("5bf011840784117a", "1")},
	{Name: "gemini-3-pro-plus", DisplayName: "Gemini 3 Pro Plus", AdvancedOnly: true, Headers: BuildModelHeader("e6fa609c3fa255c0", "4")},
	{Name: "gemini-3-flash-plus", DisplayName: "Gemini 3 Flash Plus", AdvancedOnly: true, Headers: BuildModelHeader("56fdd199312815e2", "4")},
	{Name: "gemini-3-flash-thinking-plus", DisplayName: "Gemini 3 Flash Thinking Plus", AdvancedOnly: true, Headers: BuildModelHeader("e051ce1aa80aa576", "4")},
	{Name: "gemini-3-pro-advanced", DisplayName: "Gemini 3 Pro Advanced", AdvancedOnly: true, Headers: BuildModelHeader("e6fa609c3fa255c0", "2")},
	{Name: "gemini-3-flash-advanced", DisplayName: "Gemini 3 Flash Advanced", AdvancedOnly: true, Headers: BuildModelHeader("56fdd199312815e2", "2")},
	{Name: "gemini-3-flash-thinking-advanced", DisplayName: "Gemini 3 Flash Thinking Advanced", AdvancedOnly: true, Headers: BuildModelHeader("e051ce1aa80aa576", "2")},
}

// FallbackModelName is the model to use when error 1052 (model unavailable) is encountered.
const FallbackModelName = "gemini-3-flash"

// FindModel looks up a model by name, returns nil if not found.
func FindModel(name string) *Model {
	for i := range Models {
		if Models[i].Name == name {
			return &Models[i]
		}
	}
	return nil
}
