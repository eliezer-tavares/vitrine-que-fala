package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

// HTTPAIClient talks to ai-vision-service: downloads the image, posts it as
// multipart to /describe, and returns the per-locale alt-text map; Embed
// posts texts to /embed for the semantic-search index (ADR-0009). The
// timeout is generous because CPU-only models load lazily (ADR-0005).
type HTTPAIClient struct {
	BaseURL string
	HTTP    *http.Client
}

func NewHTTPAIClient(baseURL string) *HTTPAIClient {
	return &HTTPAIClient{BaseURL: baseURL, HTTP: &http.Client{Timeout: 90 * time.Second}}
}

func (c *HTTPAIClient) Describe(ctx context.Context, imageURL string) (map[string]string, error) {
	imgReq, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, err
	}
	imgResp, err := c.HTTP.Do(imgReq)
	if err != nil {
		return nil, err
	}
	defer imgResp.Body.Close()
	if imgResp.StatusCode >= 400 {
		return nil, errors.New("image download failed with status " + imgResp.Status)
	}
	img, err := io.ReadAll(io.LimitReader(imgResp.Body, 8<<20))
	if err != nil {
		return nil, err
	}

	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	fw, err := mw.CreateFormFile("image", "product.jpg")
	if err != nil {
		return nil, err
	}
	if _, err := fw.Write(img); err != nil {
		return nil, err
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/describe", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		return nil, errors.New("ai service failed with status " + res.Status)
	}

	var out struct {
		AltTexts map[string]string `json:"alt_texts"`
		OCR      []string          `json:"ocr_text"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.AltTexts, nil
}

// Embed turns texts into vectors via POST /embed (multilingual-e5-small on
// the service side; the stub pipeline answers with deterministic vectors).
func (c *HTTPAIClient) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	payload, err := json.Marshal(map[string]any{"texts": texts, "input_type": "passage"})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/embed", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("embed failed with status %s", res.Status)
	}
	var out struct {
		Embeddings [][]float64 `json:"embeddings"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Embeddings, nil
}
