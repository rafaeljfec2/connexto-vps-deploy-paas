package transport

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/paasdeploy/mcp/internal/toolkit"
)

// maxBucketPeekBytes caps how much of the request body the bucket classifier
// is allowed to read for parsing. JSON-RPC requests issued by MCP clients are
// tiny in practice; the cap exists to neutralize abuse of the read window.
//
// When the body exceeds this cap we DO NOT truncate the request — the peeked
// chunk is concatenated with the untouched tail via io.MultiReader so the
// downstream StreamableHTTPHandler still sees the full payload. We only
// classify conservatively (BucketMutate) because we cannot inspect a payload
// we did not fully read.
const maxBucketPeekBytes = 256 * 1024

// jsonRPCEnvelope is the minimal shape we need to classify a single request
// as a mutating tool call. Every field is optional so non-tools/call payloads
// (initialize, ping, list operations, ...) deserialize without error.
type jsonRPCEnvelope struct {
	Method string `json:"method"`
	Params struct {
		Name string `json:"name"`
	} `json:"params"`
}

// multiReadCloser composes a Reader (typically io.MultiReader) with a Closer
// (the original request body) so we can rebind r.Body without leaking the
// underlying connection.
type multiReadCloser struct {
	io.Reader
	io.Closer
}

// ClassifyOutcome is returned alongside the bucket so the caller can update
// observability counters describing why the classifier decided what it did.
// "" means classification succeeded cleanly; the other values represent
// conservative escalations to BucketMutate.
type ClassifyOutcome string

const (
	ClassifyOK         ClassifyOutcome = ""
	ClassifyReadError  ClassifyOutcome = "read_error"
	ClassifyOverflow   ClassifyOutcome = "overflow"
	ClassifyParseError ClassifyOutcome = "parse_error"
)

// classifyRequest decides which rate-limit bucket should bill the request.
//
// Order of resolution:
//
//  1. GET/HEAD/OPTIONS use the read bucket (SSE channel; no tool calls).
//  2. The request body is peeked (up to maxBucketPeekBytes) and parsed as
//     JSON-RPC. Both single-envelope ({...}) and batch ([{...}, {...}])
//     payloads are inspected. If ANY tools/call entry targets a tool
//     registered as write or destructive, the mutate bucket is billed.
//  3. The optional X-MCP-Bucket header may UPGRADE the bucket to mutate
//     (defensive override for clients that batch destructive RPC), but it can
//     never DOWNGRADE a mutating call to read.
//  4. Everything else is read.
//
// On overflow OR parse failure we degrade conservatively to mutate so a
// crafted payload cannot bypass the stricter quota by becoming
// "unclassifiable". This is intentional: defense-in-depth wins over
// false-positive throttling.
//
// classifyRequest also rebinds r.Body so downstream handlers see the original
// payload unchanged (peeked + tail when overflow, peeked-only when fully
// consumed).
func classifyRequest(r *http.Request) (Bucket, ClassifyOutcome, error) {
	if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
		return BucketRead, ClassifyOK, nil
	}

	bucket := BucketRead
	outcome := ClassifyOK
	if r.Body != nil && r.Body != http.NoBody {
		peeked, overflow, err := peekBody(r)
		if err != nil {
			return BucketMutate, ClassifyReadError, err
		}
		if overflow {
			bucket = BucketMutate
			outcome = ClassifyOverflow
		} else if len(peeked) > 0 {
			classified, classifyErr := classifyJSONRPC(peeked)
			if classifyErr != nil {
				bucket = BucketMutate
				outcome = ClassifyParseError
			} else if classified == BucketMutate {
				bucket = BucketMutate
			}
		}
	}

	if header := strings.ToLower(strings.TrimSpace(r.Header.Get("X-MCP-Bucket"))); header == "mutate" || header == "destructive" {
		bucket = BucketMutate
	}

	return bucket, outcome, nil
}

// peekBody reads up to maxBucketPeekBytes from r.Body and rebinds r.Body so
// downstream handlers see the full original payload. The boolean return
// indicates whether the body exceeded the peek cap (in which case the tail
// was NOT consumed, only logically chained).
func peekBody(r *http.Request) ([]byte, bool, error) {
	original := r.Body
	peeked, err := io.ReadAll(io.LimitReader(original, maxBucketPeekBytes+1))
	if err != nil {
		r.Body = io.NopCloser(bytes.NewReader(peeked))
		return nil, false, err
	}
	if len(peeked) > maxBucketPeekBytes {
		head := peeked[:maxBucketPeekBytes]
		overflowByte := peeked[maxBucketPeekBytes:]
		r.Body = &multiReadCloser{
			Reader: io.MultiReader(bytes.NewReader(head), bytes.NewReader(overflowByte), original),
			Closer: original,
		}
		return head, true, nil
	}
	r.Body = io.NopCloser(bytes.NewReader(peeked))
	return peeked, false, nil
}

// classifyJSONRPC inspects the peeked payload to decide whether it targets a
// mutating tool. It supports both single-envelope and batch JSON-RPC requests
// (the MCP SDK still accepts batches for protocolVersion < 2025-06-18).
//
// Returns an error when neither shape parses; the caller is expected to
// translate that into a conservative BucketMutate classification so a
// malformed payload cannot bypass mutate quotas by being "unclassifiable".
func classifyJSONRPC(body []byte) (Bucket, error) {
	trimmed := bytes.TrimLeft(body, " \t\r\n")
	if len(trimmed) == 0 {
		return BucketRead, nil
	}
	if trimmed[0] == '[' {
		var batch []jsonRPCEnvelope
		if err := json.Unmarshal(body, &batch); err != nil {
			return BucketMutate, err
		}
		for _, env := range batch {
			if env.Method == "tools/call" && toolkit.IsMutatingTool(env.Params.Name) {
				return BucketMutate, nil
			}
		}
		return BucketRead, nil
	}
	var env jsonRPCEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return BucketMutate, err
	}
	if env.Method == "tools/call" && toolkit.IsMutatingTool(env.Params.Name) {
		return BucketMutate, nil
	}
	return BucketRead, nil
}
