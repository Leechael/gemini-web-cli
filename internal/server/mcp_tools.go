package server

import (
	"context"
	"sort"

	"github.com/Leechael/gemini-web-cli/internal/types"
	mcp "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func (s *Server) resolveMCPModel(override string) *types.Model {
	name := override
	if name == "" {
		name = s.mcpDefaultModel
	}
	if name == "" || name == "auto" {
		return types.FindModel("unspecified")
	}
	if m := s.client.ResolveModel(name); m != nil {
		return m
	}
	if m := types.FindModel(name); m != nil {
		return m
	}
	return types.FindModel("unspecified")
}

func (s *Server) registerMCPTools(srv *mcpserver.MCPServer) {
	researchCreateTool := mcp.NewTool("gemini_research_create",
		mcp.WithDescription("Submit a deep research task to Gemini and return a task id for status polling."),
		mcp.WithString("prompt",
			mcp.Required(),
			mcp.Description("Research topic or prompt."),
		),
		mcp.WithString("model",
			mcp.Description("Model name override. Omit to use the server's --mcp-default-model."),
		),
	)
	srv.AddTool(researchCreateTool, s.handleMCPResearchCreate)

	researchStatusTool := mcp.NewTool("gemini_research_status",
		mcp.WithDescription("Check the state of a submitted deep research task."),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Research task id (chat id)."),
		),
	)
	srv.AddTool(researchStatusTool, s.handleMCPResearchStatus)

	researchResultTool := mcp.NewTool("gemini_research_result",
		mcp.WithDescription("Fetch the final result text and source citations of a completed deep research task."),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Research task id (chat id)."),
		),
	)
	srv.AddTool(researchResultTool, s.handleMCPResearchResult)

	researchListTool := mcp.NewTool("gemini_research_list",
		mcp.WithDescription("List completed deep research reports from the library, with pagination."),
		mcp.WithNumber("count",
			mcp.Description("Max reports to return per page (default 13)."),
		),
		mcp.WithString("cursor",
			mcp.Description("Pagination cursor from a previous call's next_cursor; omit for the first page."),
		),
	)
	srv.AddTool(researchListTool, s.handleMCPResearchList)

	researchReplyTool := mcp.NewTool("gemini_research_reply",
		mcp.WithDescription("Send a follow-up prompt to an existing deep research chat to refine or continue the research. Returns the immediate acknowledgement text; poll gemini_research_status afterwards."),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Research task id (chat id) to continue."),
		),
		mcp.WithString("prompt",
			mcp.Required(),
			mcp.Description("The follow-up/refinement prompt."),
		),
		mcp.WithString("model",
			mcp.Description("Model name override. Omit to use the server's --mcp-default-model."),
		),
	)
	srv.AddTool(researchReplyTool, s.handleMCPResearchReply)

	askTool := mcp.NewTool("gemini_ask",
		mcp.WithDescription("Send a single-turn prompt to Gemini (search-like, no conversation state)."),
		mcp.WithString("prompt",
			mcp.Required(),
			mcp.Description("The prompt to send."),
		),
		mcp.WithString("model",
			mcp.Description("Model name override. Omit to use the server's --mcp-default-model."),
		),
	)
	srv.AddTool(askTool, s.handleMCPAsk)

	listModelsTool := mcp.NewTool("gemini_list_models",
		mcp.WithDescription("List available Gemini model names and display names."),
	)
	srv.AddTool(listModelsTool, s.handleMCPListModels)
}

func (s *Server) handleMCPResearchCreate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	prompt, err := req.RequireString("prompt")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	modelName := req.GetString("model", "")
	model := s.resolveMCPModel(modelName)
	if model == nil {
		return mcp.NewToolResultError("model not found"), nil
	}

	plan, err := s.client.CreateAndStartDeepResearch(ctx, prompt, model)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result := map[string]any{
		"id":       plan.Cid,
		"title":    plan.Title,
		"eta_text": plan.ETAText,
		"steps":    plan.Steps,
	}
	return mcp.NewToolResultJSON(result)
}

func (s *Server) handleMCPResearchStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	status, err := s.client.CheckDeepResearch(ctx, id)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result := map[string]any{
		"id":    id,
		"state": status.State,
	}
	return mcp.NewToolResultJSON(result)
}

func (s *Server) handleMCPResearchResult(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	text, sources, err := s.client.GetDeepResearchResult(ctx, id)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	keys := make([]int, 0, len(sources))
	for key := range sources {
		keys = append(keys, key)
	}
	sort.Ints(keys)

	type sourceItem struct {
		URL   string `json:"url"`
		Title string `json:"title"`
	}
	respSources := make([]sourceItem, 0, len(sources))
	for _, key := range keys {
		s := sources[key]
		respSources = append(respSources, sourceItem{
			URL:   s.URL,
			Title: s.Title,
		})
	}

	result := map[string]any{
		"id":      id,
		"text":    text,
		"sources": respSources,
	}
	return mcp.NewToolResultJSON(result)
}

func (s *Server) handleMCPResearchList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	count := req.GetInt("count", 13)
	if count <= 0 {
		count = 13
	}
	cursor := req.GetString("cursor", "")

	reports, nextCursor, err := s.client.ListResearchReportsPage(ctx, count, cursor)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	type reportItem struct {
		Cid       string `json:"cid"`
		Title     string `json:"title,omitempty"`
		Snippet   string `json:"snippet,omitempty"`
		ReportID  string `json:"report_id,omitempty"`
		CreatedAt int64  `json:"created_at,omitempty"`
	}
	items := make([]reportItem, 0, len(reports))
	for _, r := range reports {
		items = append(items, reportItem{
			Cid:       r.Cid,
			Title:     r.Title,
			Snippet:   r.Snippet,
			ReportID:  r.ReportID,
			CreatedAt: r.CreatedAt,
		})
	}

	result := map[string]any{
		"reports":     items,
		"next_cursor": nextCursor,
	}
	return mcp.NewToolResultJSON(result)
}

func (s *Server) handleMCPResearchReply(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	prompt, err := req.RequireString("prompt")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	modelName := req.GetString("model", "")
	model := s.resolveMCPModel(modelName)
	if model == nil {
		return mcp.NewToolResultError("model not found"), nil
	}

	metadata := make([]string, 10)
	metadata[0] = id
	if latest, err := s.client.FetchLatestChatResponse(ctx, id); err == nil && latest != nil {
		if latest.Rid != "" {
			metadata[1] = latest.Rid
		}
		if latest.RCid != "" {
			metadata[2] = latest.RCid
		}
	}

	output, err := s.client.SendMessageDeepResearch(ctx, prompt, metadata, model)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	text := ""
	if output != nil {
		text = output.Text
	}
	result := map[string]any{
		"chat_id": id,
		"text":    text,
	}
	return mcp.NewToolResultJSON(result)
}

func (s *Server) handleMCPAsk(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	prompt, err := req.RequireString("prompt")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	modelName := req.GetString("model", "")
	model := s.resolveMCPModel(modelName)
	if model == nil {
		return mcp.NewToolResultError("model not found"), nil
	}

	output, err := s.client.GenerateContent(ctx, prompt, model)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if output == nil {
		return mcp.NewToolResultError("empty response from Gemini"), nil
	}

	type imageItem struct {
		URL   string `json:"url,omitempty"`
		Title string `json:"title,omitempty"`
	}
	type videoItem struct {
		URL       string `json:"url,omitempty"`
		Thumbnail string `json:"thumbnail,omitempty"`
	}
	type mediaItem struct {
		Title  string `json:"title,omitempty"`
		MP3URL string `json:"mp3_url,omitempty"`
		MP4URL string `json:"mp4_url,omitempty"`
		VTTURL string `json:"vtt_url,omitempty"`
	}

	result := map[string]any{
		"text": output.Text,
	}

	if len(output.Images) > 0 {
		images := make([]imageItem, 0, len(output.Images))
		for _, img := range output.Images {
			images = append(images, imageItem{URL: img.URL, Title: img.Title})
		}
		result["images"] = images
	}
	if len(output.Videos) > 0 {
		videos := make([]videoItem, 0, len(output.Videos))
		for _, vid := range output.Videos {
			videos = append(videos, videoItem{URL: vid.URL, Thumbnail: vid.Thumbnail})
		}
		result["videos"] = videos
	}
	if len(output.Media) > 0 {
		media := make([]mediaItem, 0, len(output.Media))
		for _, m := range output.Media {
			media = append(media, mediaItem{
				Title:  m.Title,
				MP3URL: m.MP3URL,
				MP4URL: m.MP4URL,
				VTTURL: m.VTTURL,
			})
		}
		result["media"] = media
	}
	return mcp.NewToolResultJSON(result)
}

func (s *Server) handleMCPListModels(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	models := s.client.AvailableModels()

	type modelItem struct {
		Name        string `json:"name"`
		DisplayName string `json:"display_name,omitempty"`
		Advanced    bool   `json:"advanced,omitempty"`
	}
	items := make([]modelItem, 0, len(models))
	for _, m := range models {
		items = append(items, modelItem{
			Name:        m.Name,
			DisplayName: m.DisplayName,
			Advanced:    m.AdvancedOnly,
		})
	}

	result := map[string]any{
		"models": items,
	}
	return mcp.NewToolResultJSON(result)
}
