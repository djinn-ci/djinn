package djinn

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type Logger interface {
	Log(args ...interface{})
}

type Error struct {
	StatusCode int
	Message    string
	Params     map[string][]string
}

func (e *Error) Error() string {
	buf := bytes.NewBufferString(strconv.FormatInt(int64(e.StatusCode), 10) + " " + e.Message + "\n\n")

	for field, errs := range e.Params {
		buf.WriteString(field + ":\n")

		for _, err := range errs {
			buf.WriteString("    " + err + "\n")
		}
	}
	return buf.String()
}

type Client struct {
	endpoint *url.URL
	log      Logger
	tok      string
}

func NewClient(tok, endpoint string) (*Client, error) {
	url, err := url.Parse(endpoint)

	if err != nil {
		return nil, err
	}

	return &Client{
		endpoint: url,
		tok:      tok,
	}, nil
}

func NewClientWithLogger(tok, endpoint string, log Logger) (*Client, error) {
	cli, err := NewClient(tok, endpoint)

	if err != nil {
		return nil, err
	}

	cli.log = log
	return cli, nil
}

// String returns the string representation of the client, this will look
// something like &djinn.Client{endpoint: endpoint, tok: tok}. This would
// typically be used during logging.
func (c *Client) String() string {
	return "&djinn.Client{endpoint: " + c.endpoint.String() + ", tok: " + c.tok + "}"
}

func (c *Client) err(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return nil
	}

	switch resp.StatusCode {
	case http.StatusBadRequest:
		var params map[string][]string

		if err := json.NewDecoder(resp.Body).Decode(&params); err != nil {
			return err
		}

		return &Error{
			StatusCode: resp.StatusCode,
			Message:    "Request contains missing or invalid parameters",
			Params:     params,
		}
	case http.StatusNotFound:
		fallthrough
	case http.StatusInternalServerError:
		fallthrough
	default:
		return &Error{
			StatusCode: resp.StatusCode,
			Message:    http.StatusText(resp.StatusCode),
		}
	}
}

var (
	apiPrefix = "/api"

	binaryContent = map[string]struct{}{
		"application/octet-stream": {},
		"application/x-qemu-disk":  {},
		"image/png":                {},
	}
)

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	var cli http.Client

	if strings.HasPrefix(req.URL.Path, apiPrefix+apiPrefix) {
		req.URL.Path = strings.TrimPrefix(req.URL.Path, apiPrefix)
	}

	req.Header.Set("Authorization", "Bearer "+c.tok)

	if c.log != nil {
		var buf bytes.Buffer

		if req.Body != nil {
			io.Copy(&buf, req.Body)
		}

		c.log.Log(req.Method, req.URL.String())

		for k, v := range req.Header {
			c.log.Log(k, v[0])
		}

		if l := buf.Len(); l > 0 {
			if l >= 1<<20 {
				c.log.Log("BLOB CONTENT - TOO BIG TO LOG")
			} else {
				s := buf.String()

				if _, ok := binaryContent[req.Header.Get("Content-Type")]; ok {
					s = base64.StdEncoding.EncodeToString(buf.Bytes())
				}
				c.log.Log(s)
			}
		}
		req.Body = io.NopCloser(&buf)
	}

	resp, err := cli.Do(req)

	if err != nil {
		return nil, err
	}

	if c.log != nil {
		var buf bytes.Buffer

		if resp.Body != nil {
			io.Copy(&buf, resp.Body)
		}

		c.log.Log(resp.Status)

		for k, v := range resp.Header {
			c.log.Log(k, v[0])
		}

		if l := buf.Len(); l > 0 {
			if l >= 1<<20 {
				c.log.Log("BLOB CONTENT - TOO BIG TO LOG")
			} else {
				s := buf.String()

				if _, ok := binaryContent[resp.Header.Get("Content-Type")]; ok {
					s = base64.StdEncoding.EncodeToString(buf.Bytes())
				}
				c.log.Log(s)
			}
		}
		resp.Body = io.NopCloser(&buf)
	}
	return resp, nil
}

func (c *Client) Get(url, accept string) (*http.Response, error) {
	req, err := http.NewRequest("GET", c.endpoint.String()+url, nil)

	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", accept)

	return c.Do(req)
}

func (c *Client) Post(url, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("POST", c.endpoint.String()+url, body)

	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json; charset=utf-8")

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return c.Do(req)
}

func (c *Client) Patch(url, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("PATCH", c.endpoint.String()+url, body)

	if err != nil {
		return nil, err
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return c.Do(req)
}

func (c *Client) Delete(url string) (*http.Response, error) {
	req, err := http.NewRequest("DELETE", c.endpoint.String()+url, nil)

	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json; charset=utf-8")

	return c.Do(req)
}
