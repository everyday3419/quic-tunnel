package http

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/rs/zerolog"
)

type RequestHandler func(*http.Request) (*http.Response, error)

type HTTPProcessor struct {
	logger *zerolog.Logger
}

func NewHTTPProcessor(logger *zerolog.Logger) *HTTPProcessor {
	return &HTTPProcessor{logger: logger}
}

func (p *HTTPProcessor) ProcessStream(stream quic.Stream, handler RequestHandler) error {
	if err := stream.SetReadDeadline(time.Now().Add(15 * time.Second)); err != nil {
		return err
	}

	buf := make([]byte, 4096)
	n, err := stream.Read(buf)
	if err != nil && err != io.EOF {
		return err
	}

	req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(buf[:n])))
	if err != nil {
		return err
	}

	req.URL.Scheme = "https"
	req.URL.Host = req.Host
	req.RequestURI = ""

	resp, err := handler(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	p.logger.Info().
		Str("protocol", resp.Proto).
		Int("status", resp.StatusCode).
		Msg("received HTTP response")

	if err := stream.SetWriteDeadline(time.Now().Add(15 * time.Second)); err != nil {
		return err
	}
	return resp.Write(stream)
}
