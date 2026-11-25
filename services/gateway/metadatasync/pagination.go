package metadatasync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

type remotePage[T any] struct {
	Pager struct {
		CurrentPage *int `json:"current_page"`
		TotalPages  *int `json:"total_pages"`
		TotalItems  *int `json:"total_items,string"`
	} `json:"pager"`
	Rows []T `json:"rows"`
}

func fetchSnapshot[T any](ctx context.Context, w *Worker, path string) ([]T, int, error) {
	base, err := url.Parse(w.config.BaseURL)
	if err != nil {
		return nil, 0, fmt.Errorf("build %s URL: %w", path, err)
	}
	endpoint := base.JoinPath(path)

	var rows []T
	totalPages, totalItems := 1, 0
	page := 0
	for ; page < totalPages; page++ {
		result, err := fetchPage[T](ctx, w, *endpoint, page)
		if err != nil {
			return nil, page, fmt.Errorf("fetch %s page %d: %w", path, page, err)
		}

		pager := result.Pager
		if *pager.CurrentPage != page {
			return nil, page + 1, fmt.Errorf("%s page %d reports current page %d", path, page, *pager.CurrentPage)
		}

		if page == 0 {
			totalPages, totalItems = *pager.TotalPages, *pager.TotalItems
		} else if *pager.TotalPages != totalPages || *pager.TotalItems != totalItems {
			return nil, page + 1, fmt.Errorf("%s pagination totals changed on page %d", path, page)
		}

		if (totalItems > 0 && (totalPages == 0 || len(result.Rows) == 0)) ||
			(totalItems == 0 && totalPages > 1) || len(result.Rows) > totalItems-len(rows) {
			return nil, page + 1, fmt.Errorf("%s page %d has inconsistent row count or pagination totals", path, page)
		}

		rows = append(rows, result.Rows...)
	}

	if len(rows) != totalItems {
		return nil, page, fmt.Errorf("%s snapshot has %d rows, expected %d", path, len(rows), totalItems)
	}

	return rows, page, nil
}

func fetchPage[T any](ctx context.Context, w *Worker, endpoint url.URL, page int) (remotePage[T], error) {
	query := endpoint.Query()
	query.Set("_format", "json")
	query.Set("page", strconv.Itoa(page))
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return remotePage[T]{}, fmt.Errorf("create request: %w", err)
	}

	req.SetBasicAuth(w.config.Username, w.password)

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return remotePage[T]{}, fmt.Errorf("request page: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return remotePage[T]{}, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	result, decodeErr := decodePage[T](resp.Body)
	closeErr := resp.Body.Close()
	if decodeErr != nil {
		return remotePage[T]{}, errors.New("decode page: invalid response")
	}
	if closeErr != nil {
		return remotePage[T]{}, fmt.Errorf("close page: %w", closeErr)
	}

	return result, nil
}

func decodePage[T any](r io.Reader) (remotePage[T], error) {
	var result remotePage[T]
	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&result); err != nil {
		return remotePage[T]{}, err
	}

	pager := result.Pager
	if result.Rows == nil || pager.CurrentPage == nil || pager.TotalPages == nil || pager.TotalItems == nil {
		return remotePage[T]{}, errors.New("missing pager fields or rows")
	}

	if *pager.TotalPages < 0 || *pager.TotalItems < 0 {
		return remotePage[T]{}, errors.New("negative pagination totals")
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return remotePage[T]{}, errors.New("trailing data")
	}

	return result, nil
}
