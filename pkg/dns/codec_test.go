package dns

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type vectorFile struct {
	Vectors []vectorCase `json:"vectors"`
}

type vectorCase struct {
	Name       string `json:"name"`
	Domain     string `json:"domain"`
	ID         uint16 `json:"id"`
	PayloadHex string `json:"payload_hex"`

	QName string `json:"qname"`

	Query struct {
		PacketHex string `json:"packet_hex"`
	} `json:"query"`

	ResponseOK *struct {
		PacketHex string `json:"packet_hex"`
	} `json:"response_ok"`

	ResponseNoData *struct {
		PacketHex string `json:"packet_hex"`
	} `json:"response_no_data"`

	ResponseError *struct {
		PacketHex string `json:"packet_hex"`
	} `json:"response_error"`
}

func TestVectors(t *testing.T) {
	blob, err := os.ReadFile(filepath.Join("..", "..", "testdata", "dns-vectors.json"))
	if err != nil {
		t.Fatalf("read vectors: %v", err)
	}

	var vf vectorFile
	if err := json.Unmarshal(blob, &vf); err != nil {
		t.Fatalf("unmarshal vectors: %v", err)
	}

	for _, tc := range vf.Vectors {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			queryPacket := mustHex(t, tc.Query.PacketHex)

			decoded, err := DecodeQuery(queryPacket, tc.Domain)
			if tc.ResponseOK != nil || tc.ResponseError != nil {
				if err != nil {
					reply, ok := err.(*QueryReplyError)
					if !ok {
						t.Fatalf("unexpected decode error: %T %v", err, err)
					}
					if tc.ResponseOK != nil {
						t.Fatalf("expected successful decode for %s", tc.Name)
					}

					resp := ResponseParams{
						ID:       reply.ID,
						RD:       reply.RD,
						CD:       reply.CD,
						Question: synthQuestionFromError(tc, reply.Question),
						RCode:    &reply.RCode,
					}
					got, err := EncodeResponse(resp)
					if err != nil {
						t.Fatalf("encode error response: %v", err)
					}
					want := mustHex(t, tc.ResponseError.PacketHex)
					assertHexEqual(t, got, want)
					return
				}

				if decoded.Question.Name != tc.QName {
					t.Fatalf("decoded qname = %q, want %q", decoded.Question.Name, tc.QName)
				}
				wantPayload := mustHex(t, tc.PayloadHex)
				if hex.EncodeToString(decoded.Payload) != hex.EncodeToString(wantPayload) {
					t.Fatalf("decoded payload mismatch")
				}
				gotResp, err := EncodeResponse(ResponseParams{
					ID:       decoded.ID,
					RD:       decoded.RD,
					CD:       decoded.CD,
					Question: decoded.Question,
					Payload:  decoded.Payload,
				})
				if err != nil {
					t.Fatalf("encode response: %v", err)
				}
				wantResp := mustHex(t, tc.ResponseOK.PacketHex)
				assertHexEqual(t, gotResp, wantResp)

				gotPayload := DecodeResponse(gotResp)
				if string(gotPayload) != string(decoded.Payload) {
					t.Fatalf("response payload mismatch")
				}

				gotQuery, err := EncodeQuery(QueryParams{
					ID:      decoded.ID,
					QName:   decoded.Question.Name,
					QType:   decoded.Question.Type,
					QClass:  decoded.Question.Class,
					RD:      decoded.RD,
					CD:      decoded.CD,
					QDCount: 1,
					IsQuery: true,
				})
				if err != nil {
					t.Fatalf("encode query: %v", err)
				}
				assertHexEqual(t, gotQuery, queryPacket)
			}
		})
	}
}

func TestBase32AndDots(t *testing.T) {
	if got := Dotify("A"); got != "A" {
		t.Fatalf("dotify single char = %q", got)
	}
	if got := Dotify("A" + repeat("B", 56)); got != "A"+repeat("B", 56) {
		t.Fatalf("dotify short = %q", got)
	}
	if got := Dotify(repeat("A", 114)); got != repeat("A", 57)+"."+repeat("A", 57) {
		t.Fatalf("dotify long = %q", got)
	}

	src := []byte("hello")
	enc := EncodeBase32(src)
	dec, err := DecodeBase32(enc)
	if err != nil {
		t.Fatalf("decode base32: %v", err)
	}
	if string(dec) != string(src) {
		t.Fatalf("base32 round trip mismatch")
	}
}

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("decode hex: %v", err)
	}
	return b
}

func assertHexEqual(t *testing.T, got, want []byte) {
	t.Helper()
	if hex.EncodeToString(got) != hex.EncodeToString(want) {
		t.Fatalf("hex mismatch\n got: %X\nwant: %X", got, want)
	}
}

func synthQuestion(tc vectorCase) Question {
	return Question{
		Name:  tc.QName,
		Type:  RRTXT,
		Class: ClassIN,
	}
}

func synthQuestionFromError(tc vectorCase, q *Question) Question {
	if q != nil {
		return *q
	}
	return synthQuestion(tc)
}

func repeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}
