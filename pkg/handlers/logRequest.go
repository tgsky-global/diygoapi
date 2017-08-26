package handlers

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/gilcrest/go-API-template/pkg/config/env"
	"go.uber.org/zap"
)

// LogRequest wraps several logging functions
//   printRequest - sends request output from httputil.DumpRequest to STDERR
//   loggerRequest - uses logger util to log requests
//   log2DB - logs request to relational database
func LogRequest(env *env.Env, req *http.Request) error {
	// Check Redis key:value pair to determine if printing is on
	// for the service
	// TODO - Implement Redis cache
	if 0 == 0 {
		err := printRequest(req)
		if err != nil {
			return err
		}
	}
	if 0 == 0 {
		err := logRequest(env, req)
		if err != nil {
			return err
		}
	}
	return nil
}

// PrintRequest wraps the call to httputil.DumpRequest
func printRequest(req *http.Request) error {

	// func DumpRequest(req *http.Request, body bool) ([]byte, error)
	requestDump, err := httputil.DumpRequest(req, true)
	if err != nil {
		return HTTPStatusError{http.StatusBadRequest, err}
	}
	fmt.Println(string(requestDump))
	return nil
}

// drainBody reads all of b to memory and then returns two equivalent
// ReadClosers yielding the same bytes.
//
// It returns an error if the initial slurp of all bytes fails. It does not attempt
// to make the returned ReadClosers have identical error-matching behavior.
// Function lifted straight from httputil package
func drainBody(b io.ReadCloser) (r1, r2 io.ReadCloser, err error) {
	if b == http.NoBody {
		// No copying needed. Preserve the magic sentinel meaning of NoBody.
		return http.NoBody, http.NoBody, nil
	}
	var buf bytes.Buffer
	if _, err = buf.ReadFrom(b); err != nil {
		return nil, b, err
	}
	if err = b.Close(); err != nil {
		return nil, b, err
	}
	return ioutil.NopCloser(&buf), ioutil.NopCloser(bytes.NewReader(buf.Bytes())), nil
}

func dumpBody(req *http.Request) ([]byte, error) {
	var err error
	save := req.Body
	save, req.Body, err = drainBody(req.Body)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer

	chunked := len(req.TransferEncoding) > 0 && req.TransferEncoding[0] == "chunked"

	if req.Body != nil {
		var dest io.Writer = &b
		if chunked {
			dest = httputil.NewChunkedWriter(dest)
		}
		_, err = io.Copy(dest, req.Body)
		if chunked {
			dest.(io.Closer).Close()
			io.WriteString(&b, "\r\n")
		}
	}

	req.Body = save
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func logBody(lgr *zap.Logger, req *http.Request) (*zap.Logger, error) {
	// func dumpBody(req *http.Request) ([]byte, error)
	requestDump, err := dumpBody(req)
	if err != nil {
		return nil, HTTPStatusError{http.StatusBadRequest, err}
	}
	lgr = lgr.With(zap.String("Body", string(requestDump)))
	return lgr, nil
}

func logHeader(lgr *zap.Logger, req *http.Request) (*zap.Logger, error) {
	var i int
	for key, valSlice := range req.Header {
		for _, val := range valSlice {
			i++
			header := fmt.Sprintf("%s: %s", key, val)
			lgr = lgr.With(zap.String(fmt.Sprintf("Header(%d)", i), header))
		}
	}
	return lgr, nil
}

func logRequest(env *env.Env, req *http.Request) error {

	// type Request struct {
	//            Method string
	//            URL *url.URL
	//            Proto      string // "HTTP/1.0"
	//            ProtoMajor int    // 1
	//            ProtoMinor int    // 0
	//            Header Header
	//            Body io.ReadCloser
	//            ContentLength int64
	//            TransferEncoding []string
	//            Close bool
	//            Host string
	//            Form url.Values
	//            PostForm url.Values
	//            MultipartForm *multipart.Form
	//            Trailer Header
	//            RemoteAddr string
	//            RequestURI string
	//            TLS *tls.ConnectionState
	//    }

	logger := env.Logger
	defer env.Logger.Sync()

	logger.Debug("logRequest started")
	defer logger.Debug("logRequest ended")

	logger, _ = logHeader(logger, req)
	logger, _ = logBody(logger, req)

	logger.Info("Request received",
		zap.String("HTTP method", req.Method),
		zap.String("URL Path", req.URL.Path[1:]),
		zap.String("URL", req.URL.String()),
		zap.String("Protocol", req.Proto),
		zap.Int("ProtoMajor", req.ProtoMajor),
		zap.Int("ProtoMinor", req.ProtoMinor),
		zap.Int64("Content Length", req.ContentLength),
		zap.String("Transfer-Encoding", strings.Join(req.TransferEncoding, ",")),
		zap.Bool("Close", req.Close),
		zap.String("Host", req.Host),
		//fmt.Fprintf(w, "Form Values = %s\n", url.Values)
		//fmt.Fprintf(w, "Post Form Values = %s\n", url.Values)
		//fmt.Fprintf(w, "MultpartForm Values = %s\n", *multipart.Form)
		//fmt.Fprintf(w, "Trailer", Header)
		zap.String("RemoteAddr", req.RemoteAddr),
		zap.String("RequestURI", req.RequestURI),
		//fmt.Fprintf(w, "TLS", *tls.ConnectionState)
	)

	return nil
}
