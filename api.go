package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"

	"github.com/tomnomnom/linkheader"
)

const (
	API_URL = "https://api.github.com"
	GH_URL  = "https://github.com"
)

// materializeFile takes a physical file or stream (named pipe, user input,
// ...) and returns an io.Reader and the number of bytes that can be read
// from it.
func materializeFile(f *os.File) (io.Reader, int64, error) {
	fi, err := f.Stat()
	if err != nil {
		return nil, 0, err
	}

	// If the file is actually a char device (like user typed input)
	// or a named pipe (like a streamed in file), buffer it up.
	//
	// When uploading a file, you need to either explicitly set the
	// Content-Length header or send a chunked request. Since the
	// github upload server doesn't accept chunked encoding, we have
	// to set the size of the file manually. Since a stream doesn't have a
	// predefined length, it's read entirely into a byte buffer.
	if fi.Mode()&(os.ModeCharDevice|os.ModeNamedPipe) == 1 {
		vprintln("input was a stream, buffering up")

		var buf bytes.Buffer
		n, err := buf.ReadFrom(f)
		if err != nil {
			return nil, 0, errors.New("req: could not buffer up input stream: " + err.Error())
		}
		return &buf, n, err
	}

	// We know the os.File is most likely an actual file now.
	n, err := GetFileSize(f)
	return f, n, err
}

/* create a new request that sends the auth token */
func newAuthRequest(method, url, bodyType, token string, headers map[string]string, body io.Reader) (*http.Request, error) {
	vprintln("creating request:", method, url, bodyType, token)

	var n int64 // content length
	var err error
	if f, ok := body.(*os.File); ok {
		// Retrieve the content-length and buffer up if necessary.
		body, n, err = materializeFile(f)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	if n != 0 {
		vprintln("setting content-length to", n)
		req.ContentLength = n
	}

	if bodyType != "" {
		req.Header.Set("Content-Type", bodyType)
	}
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return req, nil
}

func DoAuthRequest(method, url, bodyType, token string, headers map[string]string, body io.Reader) (*http.Response, error) {
	req, err := newAuthRequest(method, url, bodyType, token, headers, body)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// nextLink returns the HTTP header Link annotated with 'next', "" otherwise.
func nextLink(links linkheader.Links) string {
	for _, link := range links {
		if link.Rel == "next" && link.URL != "" {
			return link.URL
		}
	}
	return ""
}

// githubGetPaginated returns a reader that yields the concatenation of the
// paginated responses to a query (URI).
//
// TODO: Rework the API so we can cleanly append per_page=100 as a URL
// parameter.
func githubGetPaginated(uri string) (io.ReadCloser, error) {
	resp, err := http.Get(ApiURL() + uri)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("expected '200 OK' but received '%v' (url: %s)", resp.Status, resp.Request.URL)
	}
	vprintln("GET (top-level)", ApiURL()+uri, "->", resp)

	// If the HTTP response is paginated, it will contain a Link header.
	links := linkheader.Parse(resp.Header.Get("Link"))
	if len(links) == 0 {
		return resp.Body, nil // No pagination.
	}

	// In this case, fetch all pages and concatenate them.
	r, w := io.Pipe()
	done := make(chan struct{})               // Backpressure from the pipe writer.
	responses := make(chan *http.Response, 5) // Allow 5 concurrent HTTP requests.
	responses <- resp

	// URL fetcher goroutine. Fetches paginated responses until no more
	// pages can be found. Closes the write end of the pipe if fetching a
	// page fails.
	go func() {
		defer close(responses) // Signal that no more requests are coming.
		for len(links) > 0 {
			URL := nextLink(links)
			if URL == "" {
				return // We're done.
			}

			resp, err := http.Get(URL)
			links = linkheader.Parse(resp.Header.Get("Link"))
			if err != nil {
				w.CloseWithError(err)
				return
			}
			select {
			case <-done:
				return // The body concatenator goroutine signals it has stopped.
			case responses <- resp: // Schedule the request body to be written to the pipe.
			}
		}
	}()

	// Body concatenator goroutine. Writes each response into the pipe
	// sequentially. Closes the write end of the pipe if the HTTP status is
	// not 200 or the body can't be read.
	go func() {
		defer func() {
			// Drain channel and close bodies, stop leaks.
			for resp := range responses {
				resp.Body.Close()
			}
		}()
		defer close(done) // Signal that we're done writing all requests, or an error occurred.
		for resp := range responses {
			if resp.StatusCode != http.StatusOK {
				resp.Body.Close()
				w.CloseWithError(fmt.Errorf("expected '200 OK' but received '%v' (url: %s)", resp.Status, resp.Request.URL))
				return
			}
			_, err := io.Copy(w, resp.Body)
			resp.Body.Close()
			if err != nil {
				vprintln("error: io.Copy: ", err)
				w.CloseWithError(err)
				return
			}
		}
		w.Close()
	}()

	return r, nil
}

// GithubGet fetches ApiURL()+uri from the GitHub API and unmarshals the
// respons into v. It takes care of pagination transparantly.
func GithubGet(uri string, v interface{}) error {
	rc, err := githubGetPaginated(uri)
	if err != nil {
		return err
	}
	defer rc.Close()
	var r io.Reader = rc
	if VERBOSITY > 0 {
		vprintln("BODY:")
		r = io.TeeReader(rc, os.Stderr)
	}

	// Github may return paginated responses. If so, githubGetPaginated will
	// return a reader which yields the concatenation of all pages. These
	// reponses are _separate_ JSON arrays. Standard json.Unmarshal() or
	// json.Decoder.Decode() will not have the expected result when
	// unmarshalling into v. For example, a 2-page response:
	//
	//   1. [{...}, {...}, {...}]
	//   2. [{...}]
	//
	// If v is a slice type, we'd like to decode the four objects from the
	// two pages into a single slice. However, if we just use
	// json.Decoder.Decode(), that won't work. v will be overridden each
	// time.
	//
	// For this reason, we use two very ugly things.
	//
	//   1. We analyze v with reflect to see if it's a slice.
	//   2. If so, we use the json.Decoder token API and reflection to
	//      dynamically add new elements into the slice, ignoring the
	//      boundaries between JSON arrays.
	//
	// This is a lot of work, and feels very stupid. An alternative would be
	// removing the outermost ][ in the intermediate responses, which would
	// be even more finnicky. Another alternative would be to explicitly
	// expose a pagination API, forcing clients of this code to deal with
	// it. That's how the go-github library does it. But why solve a problem
	// sensibly if one can power through it with reflection (half-joking)?

	sl := reflect.Indirect(reflect.ValueOf(v)) // Get the reflect.Value of the slice so we can append to it.
	t := sl.Type()
	if t.Kind() != reflect.Slice {
		// Not a slice, not going to handle special pagination JSON stream
		// semantics since it likely wouldn't work properly anyway. If this
		// is a non-paginated stream, it should work.
		return json.NewDecoder(r).Decode(v)
	}
	t = t.Elem() // Extract the type of the slice's elements.

	// Use streaming Token API to append all elements of the JSON stream
	// arrays (pagination) to the slice.
	for dec := json.NewDecoder(r); ; {
		tok, err := dec.Token()
		if err != nil {
			if err == io.EOF {
				return nil // Natural end of the JSON stream.
			}
			return err
		}
		vprintf("TOKEN %T: %v\n", tok, tok)
		// Check for tokens until we get an opening array brace. If we're
		// not in an array, we can't decode an array element later, which
		// would result in an error.
		if tok != json.Delim('[') {
			continue
		}

		// Read the array, appending all elements to the slice.
		for dec.More() {
			it := reflect.New(t) // Interface to a valid pointer to an object of the same type as the slice elements.
			vprintf("OBJECT %T: %v\n", it, it)
			if err := dec.Decode(it.Interface()); err != nil {
				return err
			}
			sl.Set(reflect.Append(sl, it.Elem()))
		}
	}
}

func ApiURL() string {
	if "" == EnvApiEndpoint {
		return API_URL
	} else {
		return EnvApiEndpoint
	}
}
