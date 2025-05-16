package cachingproxy

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
)

type Proxy struct {
	allowedPrefixes []string
	client          *http.Client

	cache *expirable.LRU[string, []byte]
}

func NewCachingProxy(allowedPrefixes []string, cacheDuration time.Duration, client *http.Client) *Proxy {
	cache := expirable.NewLRU[string, []byte](1000, nil, cacheDuration)
	return &Proxy{allowedPrefixes: allowedPrefixes, client: client, cache: cache}
}

func (p *Proxy) isAllowed(url string) bool {
	for _, prefix := range p.allowedPrefixes {
		if len(url) < len(prefix) {
			continue // skip if the prefix is longer than url
		}

		urlPrefix := url[:len(prefix)]
		// Check if the prefix matches the url
		if urlPrefix == prefix {
			return true
		}
	}

	return false
}

func (p *Proxy) HandleRequest(url string, resp http.ResponseWriter) error {
	if !p.isAllowed(url) {
		return fmt.Errorf("url %s is not allowed", url)
	}

	// Check if the URL is already in the cache
	if cachedData, found := p.cache.Get(url); found {
		// Write the cached data to the writer
		_, err := resp.Write(cachedData)
		if err != nil {
			return fmt.Errorf("error writing cached data: %w", err)
		}
		return nil
	}

	upstream, err := p.client.Get(url)
	if err != nil {
		return fmt.Errorf("error making request: %w", err)
	}

	if upstream.StatusCode != http.StatusOK {
		resp.WriteHeader(upstream.StatusCode)
		_, err := io.Copy(resp, upstream.Body)
		if err != nil {
			return fmt.Errorf("error copying response body: %w", err)
		}
		return fmt.Errorf("error: upstream returned status %s", upstream.Status)
	}

	// Create a buffer to store the response body
	writer := &bytes.Buffer{}
	_, err = io.Copy(writer, upstream.Body)
	if err != nil {
		return fmt.Errorf("error storing response body: %w", err)
	}

	// Close the upstream response body
	err = upstream.Body.Close()
	if err != nil {
		return fmt.Errorf("error closing upstream response body: %w", err)
	}

	// Cache the response body
	_ = p.cache.Add(url, writer.Bytes())

	// Write the response body to the writer
	_, err = resp.Write(writer.Bytes())
	if err != nil {
		return fmt.Errorf("error writing response body: %w", err)
	}

	return nil
}
