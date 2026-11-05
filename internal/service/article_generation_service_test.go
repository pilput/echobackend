package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"echobackend/config"
	apperrors "echobackend/internal/apperror"
	"echobackend/internal/dto"
	"echobackend/internal/platform/openrouter"
)

// fakeArticleStream returns upstream channels that deliver chunks, then err
// (if any), then usage — like openrouter.Client.GenerateStream.
func fakeArticleStream(chunks []string, streamErr error) (<-chan string, <-chan openrouter.Usage, <-chan error) {
	chunkCh := make(chan string, len(chunks))
	for _, c := range chunks {
		chunkCh <- c
	}
	close(chunkCh)
	usageCh := make(chan openrouter.Usage, 1)
	errCh := make(chan error, 1)
	if streamErr != nil {
		errCh <- streamErr
	} else {
		usageCh <- openrouter.Usage{TotalTokens: 42}
	}
	close(usageCh)
	close(errCh)
	return chunkCh, usageCh, errCh
}

func drainArticle(t *testing.T, chunks <-chan string, complete <-chan dto.GeneratedArticle, errCh <-chan error) ([]string, *dto.GeneratedArticle, error) {
	t.Helper()
	var got []string
	for c := range chunks {
		got = append(got, c)
	}
	if err, ok := <-errCh; ok && err != nil {
		if _, ok := <-complete; ok {
			t.Fatal("complete must not carry a value after an error")
		}
		return got, nil, err
	}
	article, ok := <-complete
	if !ok {
		t.Fatal("neither complete nor error was delivered")
	}
	return got, &article, nil
}

func TestBuildArticleUserPrompt(t *testing.T) {
	cases := []struct {
		name string
		req  dto.GenerateArticleRequest
		want string
	}{
		{
			name: "defaults, no title",
			req:  dto.GenerateArticleRequest{Prompt: "Go generics"},
			want: "Write a blog article about: Go generics\nTone: informative and clear\nLength: about 800 words",
		},
		{
			name: "all fields with title",
			req:  dto.GenerateArticleRequest{Prompt: "Hiking in Bali", Tone: "storytelling", Length: "long", Title: "My Trip"},
			want: "Write a blog article about: Hiking in Bali\nTone: narrative, storytelling style\nLength: about 1500 words\nWorking title: My Trip",
		},
		{
			name: "professional short, blank title ignored",
			req:  dto.GenerateArticleRequest{Prompt: "Q3 results", Tone: "professional", Length: "short", Title: "   "},
			want: "Write a blog article about: Q3 results\nTone: professional and authoritative\nLength: about 400 words",
		},
		{
			name: "casual medium",
			req:  dto.GenerateArticleRequest{Prompt: "coffee", Tone: "casual", Length: "medium"},
			want: "Write a blog article about: coffee\nTone: casual and friendly\nLength: about 800 words",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := buildArticleUserPrompt(&tc.req); got != tc.want {
				t.Fatalf("got:\n%q\nwant:\n%q", got, tc.want)
			}
		})
	}
}

func TestArticleGeneration_UnavailableNeverCallsOpenRouter(t *testing.T) {
	openRouter := &mockOpenRouterService{
		generateStreamFn: func(ctx context.Context, messages []openrouter.Message, model *string, temperature float64) (<-chan string, <-chan openrouter.Usage, <-chan error) {
			t.Fatal("OpenRouter must not be called when AI is unavailable")
			return nil, nil, nil
		},
	}
	cases := []struct {
		name string
		svc  ArticleGenerationService
	}{
		{"no client", NewArticleGenerationService(nil, aiTestConfig())},
		{"no config", NewArticleGenerationService(openRouter, nil)},
		{"blank key", NewArticleGenerationService(openRouter, &config.Config{OpenRouter: config.OpenRouterConfig{APIKey: "  ", DefaultModel: "m"}})},
		{"no default model", NewArticleGenerationService(openRouter, &config.Config{OpenRouter: config.OpenRouterConfig{APIKey: "k", DefaultModel: " "}})},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.svc.Available() {
				t.Fatal("Available() = true, want false")
			}
			chunks, complete, errCh, err := tc.svc.GenerateArticleStream(context.Background(), &dto.GenerateArticleRequest{Prompt: "abc"})
			if !errors.Is(err, apperrors.ErrAIArticleGenerationUnavailable) || chunks != nil || complete != nil || errCh != nil {
				t.Fatalf("got err %v, want ErrAIArticleGenerationUnavailable and nil channels", err)
			}
		})
	}
}

func TestArticleGeneration_StreamsChunksThenComplete(t *testing.T) {
	var gotMessages []openrouter.Message
	var gotModel string
	var gotTemp float64
	openRouter := &mockOpenRouterService{
		generateStreamFn: func(ctx context.Context, messages []openrouter.Message, model *string, temperature float64) (<-chan string, <-chan openrouter.Usage, <-chan error) {
			gotMessages, gotTemp = messages, temperature
			if model != nil {
				gotModel = *model
			}
			return fakeArticleStream([]string{"<h1>Hi</h1>", "<p>there</p>"}, nil)
		},
	}
	svc := NewArticleGenerationService(openRouter, aiTestConfig())
	if !svc.Available() {
		t.Fatal("Available() = false, want true")
	}
	req := &dto.GenerateArticleRequest{Prompt: "greetings", Title: "Hello"}
	chunks, complete, errCh, err := svc.GenerateArticleStream(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	got, article, streamErr := drainArticle(t, chunks, complete, errCh)
	if streamErr != nil {
		t.Fatalf("unexpected stream error: %v", streamErr)
	}
	if strings.Join(got, "|") != "<h1>Hi</h1>|<p>there</p>" {
		t.Fatalf("chunks = %v", got)
	}
	if article.Content != "<h1>Hi</h1><p>there</p>" || article.Model != "test-model" {
		t.Fatalf("article = %+v", article)
	}
	if gotModel != "test-model" || gotTemp != 0.7 {
		t.Fatalf("model %q temp %v, want test-model 0.7", gotModel, gotTemp)
	}
	if len(gotMessages) != 2 || gotMessages[0].Role != "system" || gotMessages[0].Content != articleSystemPrompt || gotMessages[1].Role != "user" {
		t.Fatalf("messages = %+v", gotMessages)
	}
	if !strings.HasSuffix(gotMessages[1].Content, "\nWorking title: Hello") {
		t.Fatalf("user prompt missing title: %q", gotMessages[1].Content)
	}
	if !strings.HasPrefix(articleSystemPrompt, "You are an expert blog writer for pilput") || !strings.Contains(articleSystemPrompt, "\n\nOutput rules:\n- Respond with HTML only.") {
		t.Fatalf("system prompt shape changed:\n%s", articleSystemPrompt)
	}
}

func TestArticleGeneration_EmptyResponseIsError(t *testing.T) {
	openRouter := &mockOpenRouterService{
		generateStreamFn: func(ctx context.Context, messages []openrouter.Message, model *string, temperature float64) (<-chan string, <-chan openrouter.Usage, <-chan error) {
			return fakeArticleStream(nil, nil)
		},
	}
	svc := NewArticleGenerationService(openRouter, aiTestConfig())
	chunks, complete, errCh, err := svc.GenerateArticleStream(context.Background(), &dto.GenerateArticleRequest{Prompt: "abc"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	_, _, streamErr := drainArticle(t, chunks, complete, errCh)
	if !errors.Is(streamErr, apperrors.ErrAIEmptyResponse) || streamErr.Error() != "OpenRouter returned an empty response" {
		t.Fatalf("stream error = %v", streamErr)
	}
}

func TestArticleGeneration_ForwardsUpstreamError(t *testing.T) {
	upstream := errors.New(`OpenRouter API error: 429 {"error":{"message":"Rate limited"}}`)
	openRouter := &mockOpenRouterService{
		generateStreamFn: func(ctx context.Context, messages []openrouter.Message, model *string, temperature float64) (<-chan string, <-chan openrouter.Usage, <-chan error) {
			return fakeArticleStream([]string{"partial"}, upstream)
		},
	}
	svc := NewArticleGenerationService(openRouter, aiTestConfig())
	chunks, complete, errCh, err := svc.GenerateArticleStream(context.Background(), &dto.GenerateArticleRequest{Prompt: "abc"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	got, _, streamErr := drainArticle(t, chunks, complete, errCh)
	if !errors.Is(streamErr, upstream) || len(got) != 1 {
		t.Fatalf("got chunks %v err %v, want 1 chunk and upstream error", got, streamErr)
	}
}

func TestArticleGeneration_StopsOnContextCancel(t *testing.T) {
	upstreamChunks := make(chan string)
	openRouter := &mockOpenRouterService{
		generateStreamFn: func(ctx context.Context, messages []openrouter.Message, model *string, temperature float64) (<-chan string, <-chan openrouter.Usage, <-chan error) {
			usageCh := make(chan openrouter.Usage)
			errCh := make(chan error)
			go func() {
				defer close(upstreamChunks)
				defer close(usageCh)
				defer close(errCh)
				upstreamChunks <- "a"
			}()
			return upstreamChunks, usageCh, errCh
		},
	}
	svc := NewArticleGenerationService(openRouter, aiTestConfig())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	chunks, complete, errCh, err := svc.GenerateArticleStream(ctx, &dto.GenerateArticleRequest{Prompt: "abc"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	// Nobody reads chunks; the producer must still finish via ctx.Done().
	streamErr := <-errCh
	if !errors.Is(streamErr, context.Canceled) {
		t.Fatalf("stream error = %v, want context.Canceled", streamErr)
	}
	for range chunks {
	}
	if _, ok := <-complete; ok {
		t.Fatal("complete must be empty after cancellation")
	}
}
