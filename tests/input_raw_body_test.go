package tests

import (
	"bytes"
	"testing"

	"github.com/awesome-goose/goose/io/input"
	test "github.com/awesome-goose/goose/testing"
)

type rawBodyDto struct {
	ID   string `param:"id"`
	Part []byte `raw:"body"`
}

type rawBodyStringDto struct {
	Body string `raw:"body"`
}

// A binary upload part (PLAN M1-15) must reach the handler byte-for-byte: it
// is not JSON or form data, and the binder used to have no way to hand over
// the body unparsed.
func TestInput_RawBodyTag_DeliversBytesUnparsed(t *testing.T) {
	payload := []byte{0x00, 0xff, '{', '"', 0x80, 'a', '=', 'b', 0x0a, 0x00}
	ctx := test.NewMockContext()
	ctx.MockRequest().WithParams(map[string]string{"id": "u1"}).WithBody(payload)

	var dto rawBodyDto
	if err := input.NewInput(ctx).Populate(&dto); err != nil {
		t.Fatal(err)
	}
	if dto.ID != "u1" || !bytes.Equal(dto.Part, payload) {
		t.Fatalf("got id=%q part=%v, want the exact bytes %v", dto.ID, dto.Part, payload)
	}
}

func TestInput_RawBodyTag_JSONLookingBodyIsStillRaw(t *testing.T) {
	body := []byte(`{"a":1}`)
	ctx := test.NewMockContext()
	ctx.MockRequest().WithBody(body)
	var dto rawBodyStringDto
	if err := input.NewInput(ctx).Populate(&dto); err != nil {
		t.Fatal(err)
	}
	if dto.Body != string(body) {
		t.Fatalf("got %q", dto.Body)
	}
}

func TestInput_RawBodyTag_EmptyBodyLeavesFieldEmpty(t *testing.T) {
	ctx := test.NewMockContext()
	var dto rawBodyDto
	if err := input.NewInput(ctx).Populate(&dto); err != nil {
		t.Fatal(err)
	}
	if len(dto.Part) != 0 {
		t.Fatalf("got %v", dto.Part)
	}
}
