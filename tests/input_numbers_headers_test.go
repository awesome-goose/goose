package tests

import (
	"encoding/json"
	"testing"

	"github.com/awesome-goose/goose/io/input"
	test "github.com/awesome-goose/goose/testing"
)

type moneyDto struct {
	Minor  int64           `json:"minor"`
	Big    uint64          `json:"big"`
	AsText string          `json:"asText"`
	Raw    json.RawMessage `json:"raw"`
	Nested struct {
		Wei int64 `json:"wei"`
	} `json:"nested"`
	Num    json.Number    `json:"num"`
	Loose  any            `json:"loose"`
	LooseM map[string]any `json:"looseM"`
	Ratio  float64        `json:"ratio"`
	Small  int            `json:"small"`
}

// Lane 2 (Yan): JSON numbers were decoded through float64, so anything above
// 2^53 was rounded before it reached the DTO — money in minor units, wei.
func TestInput_LargeJSONNumbersArriveExactly(t *testing.T) {
	body := []byte(`{"minor": 9007199254740993, "big": 18446744073709551615, "asText": 12345678901234567890,
		"raw": {"amount": 123456789012345678901234567890}, "nested": {"wei": 9223372036854775807},
		"num": 1e400, "loose": 3, "looseM": {"n": 2.5}, "ratio": 0.1, "small": 7}`)
	ctx := test.NewMockContext()
	ctx.MockRequest().WithBody(body)
	var d moneyDto
	if err := input.NewInput(ctx).Populate(&d); err != nil {
		t.Fatal(err)
	}
	if d.Minor != 9007199254740993 || d.Big != 18446744073709551615 || d.AsText != "12345678901234567890" || d.Nested.Wei != 9223372036854775807 {
		t.Fatalf("rounded: %+v", d)
	}
	if string(d.Raw) != `{"amount":123456789012345678901234567890}` {
		t.Fatalf("raw = %s", d.Raw)
	}
	if d.Num.String() != "1e400" || d.Ratio != 0.1 || d.Small != 7 {
		t.Fatalf("num/ratio/small = %v %v %v", d.Num, d.Ratio, d.Small)
	}
	// Fields typed any keep getting float64, as before.
	if f, ok := d.Loose.(float64); !ok || f != 3 {
		t.Fatalf("loose = %#v", d.Loose)
	}
	if f, ok := d.LooseM["n"].(float64); !ok || f != 2.5 {
		t.Fatalf("looseM = %#v", d.LooseM)
	}
}

func TestInput_TrailingDataIsNotJSON(t *testing.T) {
	ctx := test.NewMockContext()
	ctx.MockRequest().WithBody([]byte(`{"small": 1} {"small": 2}`))
	var d moneyDto
	_ = input.NewInput(ctx).Populate(&d)
	if d.Small != 0 {
		t.Fatalf("a body with trailing data was read as JSON: %+v", d)
	}
}

type allHeadersDto struct {
	One  map[string]string   `headers:"all"`
	Many map[string][]string `headers:"all"`
}

func TestInput_HeadersAllTag(t *testing.T) {
	ctx := test.NewMockContext()
	ctx.MockRequest().WithHeader("X-Alchemy-Signature", "abc").WithHeader("X-Multi", "a", "b")
	var d allHeadersDto
	if err := input.NewInput(ctx).Populate(&d); err != nil {
		t.Fatal(err)
	}
	if d.One["X-Alchemy-Signature"] != "abc" || len(d.Many["X-Multi"]) != 2 {
		t.Fatalf("%+v", d)
	}
	d.Many["X-Multi"][0] = "changed"
	if ctx.Request().Headers()["X-Multi"][0] != "a" {
		t.Fatal("the request's headers were not copied")
	}
}
