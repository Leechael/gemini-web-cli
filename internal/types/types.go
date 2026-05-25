package types

import (
	"encoding/json"
	"fmt"
	"strings"
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
	VTTURL       string
	Title        string
	Artist       string
	Genre        string
	Moods        []string
}

// ExtractImages extracts web and generated images from candidate media data.
func ExtractImages(imageData any) []Image {
	arr, ok := imageData.([]any)
	if !ok || len(arr) == 0 {
		return nil
	}

	var images []Image
	if len(arr) > 1 {
		if webImgs, ok := arr[1].([]any); ok {
			for _, wi := range webImgs {
				wiArr, ok := wi.([]any)
				if !ok {
					continue
				}
				img := Image{}
				if src := stringAt(wiArr, 0, 0, 0); src != "" {
					img.URL = src
				}
				if title := stringAt(wiArr, 7, 0); title != "" {
					img.Title = title
				}
				if img.URL != "" {
					images = append(images, img)
				}
			}
		}
	}

	if len(arr) > 7 && arr[7] != nil {
		for _, giArr := range findGeneratedImageItems(arr[7]) {
			img := Image{Generated: true}
			if u := stringAt(giArr, 3, 3); u != "" {
				img.URL = u
			}
			if isImageURL(img.URL) {
				images = append(images, img)
			}
		}
	}

	if arr0, ok := arr[0].(map[string]any); ok {
		if editResults := arr0["8"]; editResults != nil {
			for _, giArr := range findGeneratedImageItems(editResults) {
				img := Image{Generated: true}
				if u := stringAt(giArr, 3, 3); u != "" {
					img.URL = u
				}
				if isImageURL(img.URL) {
					images = append(images, img)
				}
			}
		}
	}

	return images
}

func isImageURL(url string) bool {
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}

func findGeneratedImageItems(root any) [][]any {
	var items [][]any
	var walk func(any)
	walk = func(v any) {
		arr, ok := v.([]any)
		if !ok {
			return
		}
		if stringAt(arr, 3, 3) != "" {
			items = append(items, arr)
			return
		}
		for _, child := range arr {
			walk(child)
		}
	}
	walk(root)
	return items
}

// ExtractVideos extracts generated videos from candidate media data.
func ExtractVideos(imageData any) []Video {
	arr, ok := imageData.([]any)
	if !ok || len(arr) == 0 {
		return nil
	}

	if len(arr) > 59 && arr[59] != nil {
		if videos := extractVideoURLs(arr[59]); len(videos) > 0 {
			return videos
		}
	}
	for _, elem := range arr {
		if m, ok := elem.(map[string]any); ok {
			if v, exists := m["60"]; exists {
				if videos := extractVideoURLs(v); len(videos) > 0 {
					return videos
				}
			}
		}
	}
	return nil
}

func extractVideoURLs(data any) []Video {
	current, ok := data.([]any)
	if !ok || len(current) == 0 {
		return nil
	}
	for i := 0; i < 4; i++ {
		next := arrayAt(current, 0)
		if next == nil {
			return nil
		}
		current = next
	}

	urls := arrayAt(current, 7)
	if urls == nil || len(urls) < 2 {
		return nil
	}
	thumbnail, _ := urls[0].(string)
	videoURL, _ := urls[1].(string)
	if videoURL == "" {
		return nil
	}
	return []Video{{URL: videoURL, Thumbnail: thumbnail}}
}

// ExtractMedia extracts generated music and audio media from candidate media data.
func ExtractMedia(imageData any) []GeneratedMedia {
	arr, ok := imageData.([]any)
	if !ok || len(arr) == 0 {
		return nil
	}

	var mediaData []any
	if len(arr) > 86 && arr[86] != nil {
		mediaData, _ = arr[86].([]any)
	}
	if mediaData == nil {
		for _, elem := range arr {
			if m, ok := elem.(map[string]any); ok {
				for _, key := range []string{"86", "87"} {
					if v, exists := m[key]; exists {
						mediaData, _ = v.([]any)
						if mediaData != nil {
							break
						}
					}
				}
				if mediaData != nil {
					break
				}
			}
		}
	}
	if mediaData == nil {
		return nil
	}

	var mp3URL, mp3Thumb string
	mp3Part := arrayAt(mediaData, 0)
	if mp3Part != nil {
		mp3Inner := arrayAt(mp3Part, 1)
		if mp3Inner != nil {
			mp3URLs := arrayAt(mp3Inner, 7)
			if mp3URLs != nil && len(mp3URLs) >= 2 {
				mp3Thumb, _ = mp3URLs[0].(string)
				mp3URL, _ = mp3URLs[1].(string)
			}
		}
	}

	var mp4URL, mp4Thumb, vttURL string
	mp4Part := arrayAt(mediaData, 1)
	if mp4Part != nil {
		mp4Inner := arrayAt(mp4Part, 1)
		if mp4Inner != nil {
			mp4URLs := arrayAt(mp4Inner, 7)
			if mp4URLs != nil && len(mp4URLs) >= 2 {
				mp4Thumb, _ = mp4URLs[0].(string)
				mp4URL, _ = mp4URLs[1].(string)
			}
		}
		vttInner := arrayAt(mp4Part, 3)
		if vttInner != nil {
			vttURLs := arrayAt(vttInner, 7)
			if vttURLs != nil && len(vttURLs) >= 2 {
				vttURL, _ = vttURLs[1].(string)
			}
		}
	}
	if mp3URL == "" && mp4URL == "" {
		return nil
	}

	media := GeneratedMedia{
		MP3URL:       mp3URL,
		MP3Thumbnail: mp3Thumb,
		MP4URL:       mp4URL,
		MP4Thumbnail: mp4Thumb,
		VTTURL:       vttURL,
	}
	if meta := arrayAt(mediaData, 2); meta != nil {
		if len(meta) > 0 {
			media.Title, _ = meta[0].(string)
		}
		if len(meta) > 2 {
			media.Artist, _ = meta[2].(string)
		}
		if len(meta) > 4 {
			media.Genre, _ = meta[4].(string)
		}
		if len(meta) > 5 {
			if moods, ok := meta[5].([]any); ok {
				for _, item := range moods {
					if s, ok := item.(string); ok && s != "" {
						media.Moods = append(media.Moods, s)
					}
				}
			}
		}
	}
	return []GeneratedMedia{media}
}

func arrayAt(arr []any, idx int) []any {
	if idx < 0 || idx >= len(arr) {
		return nil
	}
	a, _ := arr[idx].([]any)
	return a
}

func stringAt(root any, indices ...int) string {
	current := root
	for _, idx := range indices {
		arr, ok := current.([]any)
		if !ok || idx < 0 || idx >= len(arr) {
			return ""
		}
		current = arr[idx]
	}
	s, _ := current.(string)
	return s
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
	CreatedAtUnix     int64
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
func BuildModelHeader(modelID string, selector int) map[string]string {
	if selector == 0 {
		selector = 1
	}
	return map[string]string{
		ModelHeaderKey:             fmt.Sprintf(`[1,null,null,null,"%s",null,null,0,[4,5,6,8],null,null,2,null,null,%d,1,"FDC4D579-7A5D-4C69-A864-7188BDCFC8FF"]`, modelID, selector),
		"x-goog-ext-73010989-jspb": "[0]",
		"x-goog-ext-73010990-jspb": "[0,0,0]",
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
	{Name: "gemini-3.1-flash-lite", DisplayName: "Gemini 3.1 Flash-Lite", Headers: BuildModelHeader("8c46e95b1a07cecc", 6)},
	{Name: "gemini-3.5-flash", DisplayName: "Gemini 3.5 Flash", Headers: BuildModelHeader("56fdd199312815e2", 1)},
	{Name: "gemini-3.1-pro", DisplayName: "Gemini 3.1 Pro", AdvancedOnly: true, Headers: BuildModelHeader("e6fa609c3fa255c0", 3)},
	{Name: "gemini-3-pro", DisplayName: "Gemini 3 Pro", Headers: BuildModelHeader("9d8ca3786ebdfbea", 3)},
	{Name: "gemini-3-flash", DisplayName: "Gemini 3 Flash", Headers: BuildModelHeader("fbb127bbb056c959", 1)},
	{Name: "gemini-3-flash-thinking", DisplayName: "Gemini 3 Flash Thinking", Headers: BuildModelHeader("5bf011840784117a", 2)},
	{Name: "gemini-3-pro-plus", DisplayName: "Gemini 3 Pro Plus", AdvancedOnly: true, Headers: BuildModelHeader("e6fa609c3fa255c0", 3)},
	{Name: "gemini-3-flash-plus", DisplayName: "Gemini 3 Flash Plus", AdvancedOnly: true, Headers: BuildModelHeader("56fdd199312815e2", 1)},
	{Name: "gemini-3-flash-thinking-plus", DisplayName: "Gemini 3 Flash Thinking Plus", AdvancedOnly: true, Headers: BuildModelHeader("e051ce1aa80aa576", 2)},
	{Name: "gemini-3-pro-advanced", DisplayName: "Gemini 3 Pro Advanced", AdvancedOnly: true, Headers: BuildModelHeader("e6fa609c3fa255c0", 3)},
	{Name: "gemini-3-flash-advanced", DisplayName: "Gemini 3 Flash Advanced", AdvancedOnly: true, Headers: BuildModelHeader("56fdd199312815e2", 1)},
	{Name: "gemini-3-flash-thinking-advanced", DisplayName: "Gemini 3 Flash Thinking Advanced", AdvancedOnly: true, Headers: BuildModelHeader("e051ce1aa80aa576", 2)},
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
