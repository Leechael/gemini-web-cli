package types

// ModelOutput holds the parsed response from Gemini.
type ModelOutput struct {
	Text             string
	TextDelta        string
	RCid             string
	Metadata         []string // [cid, rid, rcid, ...]
	Images           []Image
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
	AdvancedOnly bool
	Headers      map[string]string
}

// Known models matching the Python library constants.
var Models = []Model{
	{
		Name:        "unspecified",
		DisplayName: "Auto-select",
		Headers:     map[string]string{},
	},
	{
		Name:        "gemini-2.0-flash",
		DisplayName: "Gemini 2.0 Flash",
		Headers: map[string]string{
			"x-goog-ext-525001261-jspb": `[1,null,null,null,"fbb127bbb056c959",null,null,0,[4],null,null,1]`,
			"x-goog-ext-73010989-jspb":  "[0]",
			"x-goog-ext-73010990-jspb":  "[0]",
		},
	},
	{
		Name:         "gemini-2.5-pro",
		DisplayName:  "Gemini 2.5 Pro",
		AdvancedOnly: true,
		Headers: map[string]string{
			"x-goog-ext-525001261-jspb": `[1,null,null,null,"e6fa609c3fa255c0",null,null,0,[4],null,null,2]`,
			"x-goog-ext-73010989-jspb":  "[0]",
			"x-goog-ext-73010990-jspb":  "[0]",
		},
	},
	{
		Name:         "gemini-2.5-flash",
		DisplayName:  "Gemini 2.5 Flash",
		AdvancedOnly: true,
		Headers: map[string]string{
			"x-goog-ext-525001261-jspb": `[1,null,null,null,"3acb4e219170d42a",null,null,0,[4],null,null,1]`,
			"x-goog-ext-73010989-jspb":  "[0]",
			"x-goog-ext-73010990-jspb":  "[0]",
		},
	},
	{
		Name:         "gemini-3.1-pro",
		DisplayName:  "Gemini 3.1 Pro",
		AdvancedOnly: true,
		Headers: map[string]string{
			"x-goog-ext-525001261-jspb": `[1,null,null,null,"e6fa609c3fa255c0",null,null,0,[4],null,null,2]`,
			"x-goog-ext-73010989-jspb":  "[0]",
			"x-goog-ext-73010990-jspb":  "[0]",
		},
	},
	{
		Name:         "gemini-3.0-flash",
		DisplayName:  "Gemini 3.0 Flash",
		AdvancedOnly: false,
		Headers: map[string]string{
			"x-goog-ext-525001261-jspb": `[1,null,null,null,"fbb127bbb056c959",null,null,0,[4],null,null,1]`,
			"x-goog-ext-73010989-jspb":  "[0]",
			"x-goog-ext-73010990-jspb":  "[0]",
		},
	},
}

// FindModel looks up a model by name, returns nil if not found.
func FindModel(name string) *Model {
	for i := range Models {
		if Models[i].Name == name {
			return &Models[i]
		}
	}
	return nil
}
