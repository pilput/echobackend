package service

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"echobackend/config"
	apperrors "echobackend/internal/apperror"
	"echobackend/internal/dto"
	"echobackend/internal/model"
	"echobackend/internal/platform/openrouter"
	"echobackend/internal/repository"
)

type ChatConversationService interface {
	CreateConversation(ctx context.Context, userID string, conversation *dto.CreateChatConversationRequest) (*dto.ChatConversationResponse, error)
	CreateConversationStream(ctx context.Context, userID string, req *dto.CreateChatConversationStreamRequest) (*dto.ChatStreamResult, <-chan string, <-chan dto.ChatMessageResponse, <-chan error, error)
	GetConversationByID(ctx context.Context, id string, userID string) (*dto.ChatConversationResponse, error)
	GetUserConversations(ctx context.Context, userID string, offset int, limit int) ([]*dto.ChatConversationResponse, int64, error)
	UpdateConversation(ctx context.Context, id string, userID string, conversation *dto.UpdateChatConversationRequest) (*dto.ChatConversationResponse, error)
	DeleteConversation(ctx context.Context, id string, userID string) error
	CreateMessage(ctx context.Context, userID, conversationID string, req *dto.CreateChatMessageRequest) ([]*dto.ChatMessageResponse, error)
	CreateStreamingMessage(ctx context.Context, userID, conversationID string, req *dto.CreateChatMessageRequest) (*dto.ChatStreamResult, <-chan string, <-chan dto.ChatMessageResponse, <-chan error, error)
	SaveStreamingMessage(ctx context.Context, conversationID, userID, content string, model *string, usage openrouter.Usage) (*dto.ChatMessageResponse, error)
	GetMessages(ctx context.Context, conversationID, userID string) ([]*dto.ChatMessageResponse, error)
	GetMessage(ctx context.Context, id, userID string) (*dto.ChatMessageResponse, error)
	DeleteMessage(ctx context.Context, id, userID string) (*dto.ChatMessageResponse, error)
}

type chatConversationService struct {
	conversationRepo repository.ChatConversationRepository
	openRouter       openrouter.Client
	config           *config.Config
}

func NewChatConversationService(conversationRepo repository.ChatConversationRepository, openRouter openrouter.Client, cfg *config.Config) ChatConversationService {
	return &chatConversationService{
		conversationRepo: conversationRepo,
		openRouter:       openRouter,
		config:           cfg,
	}
}

func (s *chatConversationService) CreateConversation(ctx context.Context, userID string, conversation *dto.CreateChatConversationRequest) (*dto.ChatConversationResponse, error) {
	chatConversation := &model.ChatConversation{
		Title:  conversation.Title,
		UserID: userID,
	}

	createdConversation, err := s.conversationRepo.CreateConversation(ctx, chatConversation)
	if err != nil {
		return nil, err
	}

	return dto.ChatConversationToResponse(createdConversation), nil
}

func (s *chatConversationService) CreateConversationStream(ctx context.Context, userID string, req *dto.CreateChatConversationStreamRequest) (*dto.ChatStreamResult, <-chan string, <-chan dto.ChatMessageResponse, <-chan error, error) {
	chatConversation := &model.ChatConversation{
		Title:  buildConversationTitle(req.Title, req.Content),
		UserID: userID,
	}
	createdConversation, err := s.conversationRepo.CreateConversation(ctx, chatConversation)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	msgReq := &dto.CreateChatMessageRequest{
		Content:     req.Content,
		Role:        "user",
		Model:       req.Model,
		Temperature: req.Temperature,
	}
	result, chunks, complete, errCh, err := s.createStreamingMessageInternal(ctx, userID, createdConversation.ID, msgReq)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	result.ConversationID = createdConversation.ID
	return result, chunks, complete, errCh, nil
}

func (s *chatConversationService) GetConversationByID(ctx context.Context, id string, userID string) (*dto.ChatConversationResponse, error) {
	conversation, err := s.conversationRepo.GetConversationByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if conversation.UserID != userID {
		return nil, apperrors.ErrConversationNotOwned
	}

	return dto.ChatConversationToDetailResponse(conversation), nil
}

func (s *chatConversationService) GetUserConversations(ctx context.Context, userID string, offset int, limit int) ([]*dto.ChatConversationResponse, int64, error) {
	conversations, total, err := s.conversationRepo.GetUserConversations(ctx, userID, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	var responses []*dto.ChatConversationResponse
	for _, conversation := range conversations {
		responses = append(responses, dto.ChatConversationToResponse(conversation))
	}

	return responses, total, nil
}

func (s *chatConversationService) UpdateConversation(ctx context.Context, id string, userID string, conversation *dto.UpdateChatConversationRequest) (*dto.ChatConversationResponse, error) {
	existingConversation, err := s.conversationRepo.GetConversationByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if existingConversation.UserID != userID {
		return nil, apperrors.ErrConversationNotOwned
	}

	updates := make(map[string]any)
	if conversation.Title != "" {
		updates["title"] = conversation.Title
	}
	if conversation.IsPinned != nil {
		updates["is_pinned"] = *conversation.IsPinned
		if *conversation.IsPinned {
			now := time.Now()
			updates["pinned_at"] = &now
		} else {
			updates["pinned_at"] = nil
		}
	}

	updatedConversation, err := s.conversationRepo.UpdateConversation(ctx, id, updates)
	if err != nil {
		return nil, err
	}

	return dto.ChatConversationToResponse(updatedConversation), nil
}

func (s *chatConversationService) DeleteConversation(ctx context.Context, id string, userID string) error {
	existingConversation, err := s.conversationRepo.GetConversationByID(ctx, id)
	if err != nil {
		return err
	}

	if existingConversation.UserID != userID {
		return apperrors.ErrConversationNotOwned
	}

	return s.conversationRepo.DeleteConversation(ctx, id)
}

func (s *chatConversationService) CreateMessage(ctx context.Context, userID, conversationID string, req *dto.CreateChatMessageRequest) ([]*dto.ChatMessageResponse, error) {
	conversation, err := s.getOwnedConversation(ctx, conversationID, userID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	message := &model.ChatMessage{
		ConversationID: conversationID,
		UserID:         userID,
		Role:           normalizedRole(req.Role),
		Content:        req.Content,
		Model:          req.Model,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	userMessage, err := s.conversationRepo.CreateMessage(ctx, message)
	if err != nil {
		return nil, err
	}
	if _, err := s.conversationRepo.UpdateConversation(ctx, conversationID, map[string]any{"updated_at": now}); err != nil {
		return nil, err
	}

	responses := []*dto.ChatMessageResponse{dto.ChatMessageToResponse(userMessage)}
	if userMessage.Role != "user" || s.openRouter == nil {
		return responses, nil
	}

	contextMessages, err := s.conversationRepo.GetMessagesByConversationIDAsc(ctx, conversationID, userID)
	if err != nil {
		return nil, err
	}
	if conversation.Title == "New conversation" && len(contextMessages) == 1 {
		if _, err := s.conversationRepo.UpdateConversation(ctx, conversationID, map[string]any{
			"title":      buildConversationTitle(nil, req.Content),
			"updated_at": now,
		}); err != nil {
			return nil, err
		}
	}

	reply, err := s.openRouter.GenerateResponse(ctx, toOpenRouterMessages(contextMessages), req.Model, normalizedTemperature(req.Temperature))
	if err != nil {
		// Fail-open: AI provider errors are non-fatal; the stored user message is still returned.
		openRouterLog.Warn("generate response failed; returning user message only", "error", err)
		return responses, nil //nolint:nilerr // intentional fail-open when the AI provider errors
	}
	if len(reply.Choices) == 0 {
		return responses, nil
	}

	assistant := &model.ChatMessage{
		ConversationID:   conversationID,
		UserID:           userID,
		Role:             reply.Choices[0].Message.Role,
		Content:          reply.Choices[0].Message.Content,
		Model:            s.effectiveModel(req.Model),
		PromptTokens:     reply.Usage.PromptTokens,
		CompletionTokens: reply.Usage.CompletionTokens,
		TotalTokens:      reply.Usage.TotalTokens,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	createdAssistant, err := s.conversationRepo.CreateMessage(ctx, assistant)
	if err != nil {
		return nil, err
	}
	if _, err := s.conversationRepo.UpdateConversation(ctx, conversationID, map[string]any{"updated_at": createdAssistant.UpdatedAt}); err != nil {
		return nil, err
	}
	responses = append(responses, dto.ChatMessageToResponse(createdAssistant))
	return responses, nil
}

func (s *chatConversationService) CreateStreamingMessage(ctx context.Context, userID, conversationID string, req *dto.CreateChatMessageRequest) (*dto.ChatStreamResult, <-chan string, <-chan dto.ChatMessageResponse, <-chan error, error) {
	return s.createStreamingMessageInternal(ctx, userID, conversationID, req)
}

func (s *chatConversationService) createStreamingMessageInternal(ctx context.Context, userID, conversationID string, req *dto.CreateChatMessageRequest) (*dto.ChatStreamResult, <-chan string, <-chan dto.ChatMessageResponse, <-chan error, error) {
	conversation, err := s.getOwnedConversation(ctx, conversationID, userID)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	now := time.Now()
	message := &model.ChatMessage{
		ConversationID: conversationID,
		UserID:         userID,
		Role:           normalizedRole(req.Role),
		Content:        req.Content,
		Model:          req.Model,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	userMessage, err := s.conversationRepo.CreateMessage(ctx, message)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if _, err := s.conversationRepo.UpdateConversation(ctx, conversationID, map[string]any{"updated_at": now}); err != nil {
		return nil, nil, nil, nil, err
	}

	result := &dto.ChatStreamResult{
		UserMessage: dto.ChatMessageToResponse(userMessage),
	}
	if userMessage.Role != "user" || s.openRouter == nil {
		complete := make(chan dto.ChatMessageResponse)
		close(complete)
		errCh := make(chan error)
		close(errCh)
		return result, nil, complete, errCh, nil
	}

	contextMessages, err := s.conversationRepo.GetMessagesByConversationIDAsc(ctx, conversationID, userID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if conversation.Title == "New conversation" && len(contextMessages) == 1 {
		if _, err := s.conversationRepo.UpdateConversation(ctx, conversationID, map[string]any{
			"title":      buildConversationTitle(nil, req.Content),
			"updated_at": now,
		}); err != nil {
			return nil, nil, nil, nil, err
		}
	}

	upstreamChunks, usageCh, upstreamErrCh := s.openRouter.GenerateStream(ctx, toOpenRouterMessages(contextMessages), req.Model, normalizedTemperature(req.Temperature))
	outChunks := make(chan string)
	complete := make(chan dto.ChatMessageResponse, 1)
	errCh := make(chan error, 1)

	go func() {
		defer close(outChunks)
		defer close(complete)
		defer close(errCh)

		var content strings.Builder
		for chunk := range upstreamChunks {
			content.WriteString(chunk)
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			case outChunks <- chunk:
			}
		}

		if err, ok := <-upstreamErrCh; ok && err != nil {
			errCh <- err
			return
		}

		usage := openrouter.Usage{}
		if u, ok := <-usageCh; ok {
			usage = u
		}
		if content.Len() == 0 {
			errCh <- errors.New("OpenRouter returned an empty response")
			return
		}
		saved, err := s.SaveStreamingMessage(ctx, conversationID, userID, content.String(), req.Model, usage)
		if err != nil {
			errCh <- err
			return
		}
		complete <- *saved
	}()

	return result, outChunks, complete, errCh, nil
}

func (s *chatConversationService) SaveStreamingMessage(ctx context.Context, conversationID, userID, content string, modelID *string, usage openrouter.Usage) (*dto.ChatMessageResponse, error) {
	now := time.Now()
	message := &model.ChatMessage{
		ConversationID:   conversationID,
		UserID:           userID,
		Role:             "assistant",
		Content:          content,
		Model:            s.effectiveModel(modelID),
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		TotalTokens:      usage.TotalTokens,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	created, err := s.conversationRepo.CreateMessage(ctx, message)
	if err != nil {
		return nil, err
	}
	if _, err := s.conversationRepo.UpdateConversation(ctx, conversationID, map[string]any{"updated_at": now}); err != nil {
		return nil, err
	}
	return dto.ChatMessageToResponse(created), nil
}

func (s *chatConversationService) GetMessages(ctx context.Context, conversationID, userID string) ([]*dto.ChatMessageResponse, error) {
	if _, err := s.getOwnedConversation(ctx, conversationID, userID); err != nil {
		return nil, err
	}
	messages, err := s.conversationRepo.GetMessagesByConversationID(ctx, conversationID, userID)
	if err != nil {
		return nil, err
	}
	responses := make([]*dto.ChatMessageResponse, 0, len(messages))
	for _, message := range messages {
		responses = append(responses, dto.ChatMessageToResponse(message))
	}
	return responses, nil
}

func (s *chatConversationService) GetMessage(ctx context.Context, id, userID string) (*dto.ChatMessageResponse, error) {
	message, err := s.conversationRepo.GetMessageByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	return dto.ChatMessageToResponse(message), nil
}

func (s *chatConversationService) DeleteMessage(ctx context.Context, id, userID string) (*dto.ChatMessageResponse, error) {
	message, err := s.conversationRepo.GetMessageByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	if err := s.conversationRepo.DeleteMessage(ctx, id, userID); err != nil {
		return nil, err
	}
	return dto.ChatMessageToResponse(message), nil
}

func (s *chatConversationService) getOwnedConversation(ctx context.Context, id, userID string) (*model.ChatConversation, error) {
	conversation, err := s.conversationRepo.GetConversationByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if conversation.UserID != userID {
		return nil, apperrors.ErrConversationNotOwned
	}
	return conversation, nil
}

func buildConversationTitle(title *string, fallbackContent string) string {
	if title != nil && strings.TrimSpace(*title) != "" {
		// Judul eksplisit dari client: hormati isinya, cukup rapikan
		// whitespace dan batasi ke kapasitas kolom DB (varchar 255).
		return truncateRunes(normalizeSpace(*title), maxExplicitTitleRunes)
	}
	trimmed := normalizeSpace(fallbackContent)
	if trimmed == "" {
		return defaultConversationTitle
	}
	// Ambil baris/kalimat pertama agar tidak kepotong tengah kalimat.
	candidate := firstSentence(trimmed)
	words := strings.Fields(candidate)
	words = stripLeadingFillers(words)
	words = stripTrailingFillers(words)
	if len(words) == 0 {
		words = strings.Fields(candidate)
	}
	titleStr := truncateWords(words, maxTitleWords, maxTitleRunes)
	if titleStr == "" {
		return defaultConversationTitle
	}
	return titleStr
}

const (
	defaultConversationTitle = "New conversation"
	// Judul auto dari pesan pertama: pendek ala daftar chat.
	maxTitleWords = 8
	maxTitleRunes = 60
	// Judul eksplisit: ikuti kapasitas kolom DB.
	maxExplicitTitleRunes = 255
)

// normalizeSpace memadatkan semua whitespace menjadi satu spasi.
func normalizeSpace(s string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}

// truncateRunes memotong string per rune (aman untuk UTF-8).
func truncateRunes(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes])
}

// firstSentence mengambil baris pertama, lalu potong di akhir kalimat
// pertama (. ! ? …) bila prefix-nya sudah cukup bermakna (>=3 kata).
// Juga membersihkan prefix markdown/list seperti "# ", "> ", "- ", "1. ".
func firstSentence(s string) string {
	if i := strings.IndexRune(s, '\n'); i >= 0 {
		if first := strings.TrimSpace(s[:i]); first != "" {
			s = first
		}
	}
	s = strings.TrimLeft(s, "#>*-•–— \t")
	if dot := strings.Index(s, ". "); dot > 0 {
		if head := strings.TrimSpace(s[:dot]); len(strings.Fields(head)) >= 3 {
			s = head
		}
	}
	end := -1
	for i, r := range s {
		if r == '!' || r == '?' || r == '…' {
			end = i + len(string(r))
			break
		}
		if r == '.' {
			end = i + 1
			break
		}
	}
	if end > 0 {
		if head := strings.TrimSpace(s[:end]); len(strings.Fields(head)) >= 3 {
			s = head
		}
	}
	// Strip "1. ", "12) " ala numbered list di awal.
	rest := strings.TrimLeft(s, "0123456789")
	if len(rest) < len(s) {
		if r, _ := utf8.DecodeRuneInString(rest); r == '.' || r == ')' {
			if t := strings.TrimSpace(rest[utf8.RuneLen(r):]); t != "" {
				s = t
			}
		}
	}
	return strings.TrimSpace(strings.Trim(s, " \t\"'“”‘’.,:;!?…-–—"))
}

// Kata pengisi di awal pesan (ID/EN) yang buruk untuk judul.
var leadingFillerWords = map[string]struct{}{
	"tolong": {}, "mohon": {}, "please": {}, "pls": {}, "plis": {},
	"coba": {}, "cobalah": {}, "bisakah": {}, "bisa": {}, "bolehkah": {},
	"gimana": {}, "bagaimana": {}, "cara": {},
	"buatkan": {}, "buatin": {}, "bikinkan": {}, "bikinin": {},
	"tuliskan": {}, "tulis": {}, "tuliskanlah": {},
	"jelaskan": {}, "jelasin": {}, "jelasken": {},
	"kasih": {}, "kasi": {}, "berikan": {}, "beritahu": {},
	"beritahukan": {}, "tunjukkan": {}, "tunjukin": {},
	"apa": {}, "apakah": {}, "itu": {}, "ini": {},
	"halo": {}, "haloo": {}, "hallo": {}, "hai": {}, "hi": {}, "hello": {},
}

// Kata pengisi di akhir pesan yang buruk untuk judul.
var trailingFillerWords = map[string]struct{}{
	"ya": {}, "yah": {}, "dong": {}, "donk": {}, "sih": {},
	"deh": {}, "loh": {}, "lho": {}, "kah": {}, "tuh": {},
}

func stripLeadingFillers(words []string) []string {
	stripped := 0
	for len(words) > 0 && stripped < 2 {
		if _, ok := leadingFillerWords[strings.ToLower(words[0])]; !ok {
			break
		}
		words = words[1:]
		stripped++
	}
	return words
}

func stripTrailingFillers(words []string) []string {
	for len(words) > 3 {
		if _, ok := trailingFillerWords[strings.ToLower(words[len(words)-1])]; !ok {
			break
		}
		words = words[:len(words)-1]
	}
	return words
}

// truncateWords menggabung maxWords kata pertama dan memastikan panjang
// tidak melebihi maxRunes, selalu potong di batas kata (tidak pernah
// memotong tengah kata maupun tengah rune UTF-8).
func truncateWords(words []string, maxWords, maxRunes int) string {
	if len(words) > maxWords {
		words = words[:maxWords]
	}
	out := strings.Join(words, " ")
	out = strings.Trim(out, " \t\"'“”‘’.,:;!?…-–—")
	if out == "" {
		return ""
	}
	runes := []rune(out)
	if len(runes) <= maxRunes {
		return out
	}
	cut := maxRunes
	for cut > 0 && runes[cut] != ' ' {
		cut--
	}
	if cut == 0 {
		// Kata pertama sendiri lebih panjang dari batas: potong keras.
		return string(runes[:maxRunes])
	}
	return strings.TrimRight(string(runes[:cut]), " \t.,:;!?…-–—")
}

func normalizedRole(role string) string {
	role = strings.TrimSpace(role)
	if role == "" {
		return "user"
	}
	return role
}

func normalizedTemperature(v *float64) float64 {
	if v == nil || *v < 0 || *v > 2 {
		return 0.7
	}
	return *v
}

func toOpenRouterMessages(messages []*model.ChatMessage) []openrouter.Message {
	result := make([]openrouter.Message, 0, len(messages))
	for _, message := range messages {
		result = append(result, openrouter.Message{
			Role:    message.Role,
			Content: message.Content,
		})
	}
	return result
}

func (s *chatConversationService) effectiveModel(model *string) *string {
	if model != nil && strings.TrimSpace(*model) != "" {
		trimmed := strings.TrimSpace(*model)
		return &trimmed
	}
	if s.config == nil {
		return nil
	}
	defaultModel := strings.TrimSpace(s.config.OpenRouter.DefaultModel)
	if defaultModel == "" {
		return nil
	}
	return &defaultModel
}
