package server

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol/rpcs"
	"github.com/Leechael/gemini-web-cli/internal/types"
	mcp "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

const maxMCPResearchListCount = 100

func (s *Server) resolveMCPModel(override string) (*types.Model, error) {
	name := override
	if name == "" {
		name = s.mcpDefaultModel
	}
	if name == "" || name == "auto" {
		model := types.FindModel("unspecified")
		if model == nil {
			return nil, fmt.Errorf("model %q not found", "unspecified")
		}
		return model, nil
	}
	if m := s.pool.ResolveModel(name); m != nil {
		return m, nil
	}
	if m := types.FindModel(name); m != nil {
		return m, nil
	}
	return nil, fmt.Errorf("model %q not found", name)
}

func (s *Server) registerMCPTools(srv *mcpserver.MCPServer) {
	researchCreateTool := mcp.NewTool("gemini_research_create",
		mcp.WithDescription("Submit a deep research task to Gemini and return a task id for status polling."),
		mcp.WithString("prompt",
			mcp.Required(),
			mcp.Description("Research topic or prompt."),
		),
		mcp.WithString("model",
			mcp.Description("Deep Research uses Gemini auto-selection. Omit this field or use auto/unspecified; explicit model names are rejected."),
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
		mcp.WithString("notebook",
			mcp.Description("Notebook id to scope the new chat to (with or without the notebooks/ prefix)."),
		),
	)
	srv.AddTool(askTool, s.handleMCPAsk)

	listModelsTool := mcp.NewTool("gemini_list_models",
		mcp.WithDescription("List available Gemini model names and display names."),
	)
	srv.AddTool(listModelsTool, s.handleMCPListModels)

	notebookCreateTool := mcp.NewTool("gemini_notebook_create",
		mcp.WithDescription("Create a Gemini notebook. Returns the notebooks/<uuid> resource name."),
		mcp.WithString("title",
			mcp.Required(),
			mcp.Description("Notebook title."),
		),
	)
	srv.AddTool(notebookCreateTool, s.handleMCPNotebookCreate)

	notebookGetTool := mcp.NewTool("gemini_notebook_get",
		mcp.WithDescription("Get a notebook's title, emoji, and source list."),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Notebook id, with or without the notebooks/ prefix."),
		),
	)
	srv.AddTool(notebookGetTool, s.handleMCPNotebookGet)

	notebookListChatsTool := mcp.NewTool("gemini_notebook_list_chats",
		mcp.WithDescription("List the chats belonging to a notebook, newest first."),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Notebook id, with or without the notebooks/ prefix."),
		),
	)
	srv.AddTool(notebookListChatsTool, s.handleMCPNotebookListChats)

	notebookAddFileSourceTool := mcp.NewTool("gemini_notebook_add_file_source",
		mcp.WithDescription("Upload a local file (on the machine running gemini-web-cli serve) and attach it to a notebook as a source."),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Notebook id, with or without the notebooks/ prefix."),
		),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Local file path to upload."),
		),
	)
	srv.AddTool(notebookAddFileSourceTool, s.handleMCPNotebookAddFileSource)

	notebookAddURLSourceTool := mcp.NewTool("gemini_notebook_add_url_source",
		mcp.WithDescription("Attach a web URL to a notebook as a source."),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Notebook id, with or without the notebooks/ prefix."),
		),
		mcp.WithString("url",
			mcp.Required(),
			mcp.Description("The http(s) URL to attach."),
		),
	)
	srv.AddTool(notebookAddURLSourceTool, s.handleMCPNotebookAddURLSource)

	notebookRemoveSourceTool := mcp.NewTool("gemini_notebook_remove_source",
		mcp.WithDescription("Remove a source from a notebook."),
		mcp.WithString("source",
			mcp.Required(),
			mcp.Description("Full source resource name: notebooks/<uuid>/sources/<sid>."),
		),
	)
	srv.AddTool(notebookRemoveSourceTool, s.handleMCPNotebookRemoveSource)
}

func (s *Server) handleMCPResearchCreate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	prompt, err := req.RequireString("prompt")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	modelName := req.GetString("model", "")
	if modelName != "" && modelName != "auto" && modelName != "unspecified" {
		return mcp.NewToolResultError("deep research only supports model auto/unspecified"), nil
	}
	var model *types.Model
	if modelName == "" {
		model = types.FindModel("unspecified")
		if model == nil {
			return mcp.NewToolResultError("model \"unspecified\" not found"), nil
		}
	} else {
		model, err = s.resolveMCPModel(modelName)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
	}
	plan, err := s.pool.CreateAndStartDeepResearch(ctx, prompt, model)
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

	status, err := s.pool.CheckDeepResearch(ctx, id)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	log.Printf("mcp research status state=%q text_len=%d", status.State, status.TextLen)
	result := map[string]any{
		"id":       id,
		"state":    status.State,
		"text_len": status.TextLen,
	}
	return mcp.NewToolResultJSON(result)
}

func (s *Server) handleMCPResearchResult(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	text, sources, err := s.pool.GetDeepResearchResult(ctx, id)
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

	log.Printf("mcp research result text_len=%d sources=%d", len(text), len(respSources))
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
	if count > maxMCPResearchListCount {
		return mcp.NewToolResultError(fmt.Sprintf("count must be <= %d", maxMCPResearchListCount)), nil
	}
	cursor := req.GetString("cursor", "")

	reports, nextCursor, err := s.pool.ListResearchReportsPage(ctx, count, cursor)
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
	model, err := s.resolveMCPModel(modelName)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	metadata := make([]string, 10)
	metadata[0] = id
	if latest, err := s.pool.FetchLatestChatResponse(ctx, id); err == nil && latest != nil {
		if latest.Rid != "" {
			metadata[1] = latest.Rid
		}
		if latest.RCid != "" {
			metadata[2] = latest.RCid
		}
	} else if err != nil {
		log.Printf("mcp research reply continuing without latest metadata chat_id=%q err=%q", id, err.Error())
	}

	output, err := s.pool.SendMessageDeepResearch(ctx, prompt, metadata, model)
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
	model, err := s.resolveMCPModel(modelName)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	notebook := req.GetString("notebook", "")
	output, err := s.pool.GenerateContent(ctx, prompt, model, notebook)
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
	models := s.availableModels()

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

func (s *Server) handleMCPNotebookCreate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	title, err := req.RequireString("title")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	resource, err := s.pool.CreateNotebook(ctx, title)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultJSON(map[string]any{"resource": resource, "title": title})
}

type mcpNotebookSourceItem struct {
	Resource   string `json:"resource"`
	FileName   string `json:"file_name"`
	MimeType   string `json:"mime_type"`
	UploadedAt int64  `json:"uploaded_at_unix,omitempty"`
}

func mcpNotebookJSON(nb *rpcs.Notebook) map[string]any {
	sources := make([]mcpNotebookSourceItem, 0, len(nb.Sources))
	for _, src := range nb.Sources {
		sources = append(sources, mcpNotebookSourceItem{
			Resource:   src.ResourceName,
			FileName:   src.FileName,
			MimeType:   src.MimeType,
			UploadedAt: src.UploadedUnix,
		})
	}
	return map[string]any{
		"resource":     nb.ResourceName,
		"title":        nb.Title,
		"emoji":        nb.Emoji,
		"sources":      sources,
		"created_unix": nb.CreatedUnix,
		"updated_unix": nb.UpdatedUnix,
		"source_count": len(sources),
	}
}

func (s *Server) handleMCPNotebookGet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	nb, err := s.pool.GetNotebook(ctx, id)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if nb == nil {
		return mcp.NewToolResultError("notebook not found"), nil
	}
	return mcp.NewToolResultJSON(mcpNotebookJSON(nb))
}

func (s *Server) handleMCPNotebookListChats(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	items, err := s.pool.ListNotebookChats(ctx, id)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	type chatItem struct {
		Cid     string `json:"cid"`
		Title   string `json:"title"`
		Updated string `json:"updated,omitempty"`
	}
	out := make([]chatItem, 0, len(items))
	for _, it := range items {
		out = append(out, chatItem{Cid: it.Cid, Title: it.Title, Updated: it.UpdatedAt})
	}
	return mcp.NewToolResultJSON(map[string]any{"chats": out})
}

func (s *Server) handleMCPNotebookAddFileSource(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	path, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	// Upload and attach must run on the account owning the notebook.
	nb, err := s.pool.AddNotebookFileSource(ctx, id, path)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultJSON(mcpNotebookJSON(nb))
}

func (s *Server) handleMCPNotebookAddURLSource(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	url, err := req.RequireString("url")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return mcp.NewToolResultError("url must start with http:// or https://"), nil
	}
	nb, err := s.pool.AddNotebookURLSource(ctx, id, url)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultJSON(mcpNotebookJSON(nb))
}

func (s *Server) handleMCPNotebookRemoveSource(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	source, err := req.RequireString("source")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if err := s.pool.RemoveNotebookSource(ctx, source); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultJSON(map[string]any{"removed": source})
}
