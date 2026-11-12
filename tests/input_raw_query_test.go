package tests

import (
	"net/http/httptest"
	"testing"

	"github.com/awesome-goose/goose/io/input"
	"github.com/awesome-goose/goose/platforms/api"
	"github.com/awesome-goose/goose/platforms/web"
	test "github.com/awesome-goose/goose/testing"
	"github.com/awesome-goose/goose/types"
)

// reqContext is a mock context around a platform's real request.
type reqContext struct {
	*test.MockContext
	req types.Request
}

func (c reqContext) Request() types.Request { return c.req }

type rawQueryDto struct {
	Query       string `raw:"query"`
	SAMLRequest string `query:"SAMLRequest"`
}

// A SAML HTTP-Redirect binding signature covers the query string exactly as
// the sender encoded it (lower-case escapes, its own parameter order), so a
// handler that checks one needs the raw query, not the decoded map.
func TestInput_RawQueryTag_DeliversTheQueryAsSent(t *testing.T) {
	raw := "SAMLRequest=a%2fb%2Bc&RelayState=x%20y&SigAlg=s&Signature=z%3d"
	for name, req := range map[string]types.Request{
		"api": api.NewRequest(httptest.NewRequest("GET", "/slo?"+raw, nil)),
		"web": web.NewRequest(httptest.NewRequest("GET", "/slo?"+raw, nil)),
	} {
		if got := types.RawQuery(req); got != raw {
			t.Fatalf("%s: RawQuery = %q, want %q", name, got, raw)
		}
		ctx := reqContext{test.NewMockContext(), req}
		var dto rawQueryDto
		if err := input.NewInput(ctx).Populate(&dto); err != nil {
			t.Fatal(err)
		}
		if dto.Query != raw || dto.SAMLRequest != "a/b+c" {
			t.Fatalf("%s: got %q / %q", name, dto.Query, dto.SAMLRequest)
		}
	}
}

// A platform without a raw query (the mock, the CLI) gets the parameters
// encoded again.
func TestInput_RawQueryTag_FallsBackToTheEncodedQueries(t *testing.T) {
	ctx := test.NewMockContext()
	ctx.MockRequest().WithQueries(map[string]string{"b": "2", "a": "x y"})
	var dto rawQueryDto
	if err := input.NewInput(ctx).Populate(&dto); err != nil {
		t.Fatal(err)
	}
	if dto.Query != "a=x+y&b=2" {
		t.Fatalf("got %q", dto.Query)
	}
}
