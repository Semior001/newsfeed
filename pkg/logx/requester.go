package logx

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-pkgz/requester/middleware"
	"github.com/samber/lo"
	"golang.org/x/exp/slog"
)

// RoundTripperOpts contains options for client logger.
type RoundTripperOpts struct {
	Level         slog.Level
	SecretHeaders []string
}

// LoggingRoundTripper logs every client request.
func LoggingRoundTripper(lg *slog.Logger, opts RoundTripperOpts) middleware.RoundTripperHandler {
	return func(next http.RoundTripper) http.RoundTripper {
		return middleware.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			le := logEntry{}

			le.Request.URL = req.URL.String()
			le.Request.Method = req.Method
			le.Request.Headers = map[string]string{}
			for k, vals := range req.Header {
				if lo.Contains(opts.SecretHeaders, k) {
					le.Request.Headers[k] = "***"
					continue
				}
				le.Request.Headers[k] = strings.Join(vals, ",")
			}
			req.Body, le.Request.RequestBody = copyAndTrim(req.Body)

			lg.LogAttrs(req.Context(), opts.Level, "request sent", slog.Any("request", le.Request))

			start := time.Now()
			resp, err := next.RoundTrip(req)
			le.Elapsed = time.Since(start)
			le.Error = err

			le.Response.Headers = map[string]string{}
			for k, vals := range resp.Header {
				if lo.Contains(opts.SecretHeaders, k) {
					le.Response.Headers[k] = "***"
					continue
				}
				le.Response.Headers[k] = strings.Join(vals, ",")
			}

			resp.Body, le.Response.ResponseBody = copyAndTrim(resp.Body)
			le.Response.StatusCode = resp.StatusCode

			lg.LogAttrs(req.Context(), opts.Level, "response received",
				slog.Any("response", le.Response),
				slog.Any("elapsed", le.Elapsed),
				slog.Any("err", le.Error),
			)

			return resp, err
		})
	}
}

type logEntry struct {
	Request struct {
		Method      string
		URL         string
		Headers     map[string]string
		RequestBody string
	}
	Response struct {
		StatusCode   int
		Headers      map[string]string
		ResponseBody string
	}
	Error   error
	Elapsed time.Duration
}

const trimBodyAt = 1024

func copyAndTrim(r io.ReadCloser) (rd io.ReadCloser, result string) {
	if r == nil {
		return nil, ""
	}

	rd, result, read := readPortion(r, trimBodyAt)
	if read == trimBodyAt {
		result = result[:trimBodyAt] + "..."
	}
	result = strings.ReplaceAll(result, "\n", "")
	result = strings.ReplaceAll(result, "\t", "")

	return rd, result
}

func readPortion(src io.ReadCloser, limit int64) (rd io.ReadCloser, portion string, read int64) {
	buf := &bytes.Buffer{}

	read, err := io.CopyN(buf, src, limit)
	if err != nil {
		return io.NopCloser(bytes.NewReader(buf.Bytes())), buf.String(), read
	}

	return &closer{rd: io.MultiReader(buf, src), closeFn: src.Close}, buf.String(), read
}

type closer struct {
	rd      io.Reader
	closeFn func() error
}

func (c *closer) Read(p []byte) (n int, err error) { return c.rd.Read(p) }
func (c *closer) Close() error                     { return c.closeFn() }
