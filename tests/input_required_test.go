package tests

import (
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/awesome-goose/goose"
	"github.com/awesome-goose/goose/errors"
	"github.com/awesome-goose/goose/io/input"
	"github.com/awesome-goose/goose/io/output"
	"github.com/awesome-goose/goose/platforms/api"
	test "github.com/awesome-goose/goose/testing"
	"github.com/awesome-goose/goose/types"
)

// binding:"required" used to be read for OpenAPI only: a request without the
// field reached the handler with its zero value (identity's login accepted a
// missing clientId). The binder now refuses it.

type requiredLoginDto struct {
	Email    string  `json:"email"    binding:"required"`
	Password string  `json:"password" binding:"required,min=8"`
	ClientId string  `json:"clientId" binding:"required"`
	Remember bool    `json:"remember"`
	Note     *string `json:"note"`
}

type requiredMixedDto struct {
	ID       string            `param:"id"         binding:"required"`
	Tenant   string            `header:"X-Tenant"  binding:"required"`
	Page     int               `query:"page"       binding:"required"`
	Flag     bool              `json:"flag"        binding:"required"`
	Count    int               `json:"count"       binding:"required"`
	Tags     []string          `json:"tags"        binding:"required"`
	Ref      *string           `json:"ref"         binding:"required"`
	Server   string            `context:"user"     binding:"required"`
	Ignored  string            `json:"-"           binding:"required"`
	ReadOnly string            `json:"readOnly"    binding:"-"`
	All      map[string]string `queries:"all"      binding:"required"`
}

type requiredFormDto struct {
	Name string `form:"name" binding:"required"`
}

type requiredRawDto struct {
	Part []byte `raw:"body" binding:"required"`
}

type requiredEitherDto struct {
	Key string `query:"key" json:"key" binding:"required"`
}

func missingFields(t *testing.T, err error) []string {
	t.Helper()
	if err == nil {
		t.Fatal("want a missing-field error, got nil")
	}
	if !stderrors.Is(err, errors.ErrMissingRequiredFields) {
		t.Fatalf("want ErrMissingRequiredFields, got %v", err)
	}
	fields, ok := input.MissingFields(err)
	if !ok {
		t.Fatalf("MissingFields(%v) found no fields", err)
	}
	return fields
}

func TestInput_Required_AllPresentPasses(t *testing.T) {
	ctx := test.NewMockContext()
	ctx.MockRequest().WithJSONBody(map[string]any{"email": "a@b.c", "password": "longenough", "clientId": "web"})
	var dto requiredLoginDto
	if err := input.NewInput(ctx).Populate(&dto); err != nil {
		t.Fatal(err)
	}
	if dto.ClientId != "web" || dto.Remember || dto.Note != nil {
		t.Fatalf("got %+v", dto)
	}
}

func TestInput_Required_MissingJSONFieldIsNamed(t *testing.T) {
	ctx := test.NewMockContext()
	ctx.MockRequest().WithJSONBody(map[string]any{"email": "a@b.c", "password": "longenough"})
	var dto requiredLoginDto
	err := input.NewInput(ctx).Populate(&dto)
	if got := missingFields(t, err); strings.Join(got, ",") != "clientId" {
		t.Fatalf("missing %v, want [clientId]", got)
	}
	// The rest is still bound, so a caller that chooses to ignore the error sees the request.
	if dto.Email != "a@b.c" || dto.Password != "longenough" {
		t.Fatalf("got %+v", dto)
	}
	if !strings.Contains(err.Error(), "clientId") {
		t.Fatalf("error does not name the field: %v", err)
	}
}

func TestInput_Required_NullAndEmptyStringAreMissing(t *testing.T) {
	ctx := test.NewMockContext()
	ctx.MockRequest().WithBody([]byte(`{"email":"","password":null,"clientId":"web"}`))
	var dto requiredLoginDto
	got := missingFields(t, input.NewInput(ctx).Populate(&dto))
	if strings.Join(got, ",") != "email,password" {
		t.Fatalf("missing %v, want [email password]", got)
	}
}

func TestInput_Required_NoBodyNamesEveryField(t *testing.T) {
	ctx := test.NewMockContext()
	var dto requiredLoginDto
	got := missingFields(t, input.NewInput(ctx).Populate(&dto))
	if strings.Join(got, ",") != "email,password,clientId" {
		t.Fatalf("missing %v, want [email password clientId] in field order", got)
	}
}

func TestInput_Required_FalseZeroAndEmptyListAreGiven(t *testing.T) {
	ctx := test.NewMockContext()
	ctx.MockRequest().
		WithParams(map[string]string{"id": "x1"}).
		WithHeader("X-Tenant", "t1").
		WithQueries(map[string]string{"page": "0"}).
		WithBody([]byte(`{"flag":false,"count":0,"tags":[],"ref":"r"}`))
	var dto requiredMixedDto
	if err := input.NewInput(ctx).Populate(&dto); err != nil {
		t.Fatalf("an explicit false, 0 and [] are values: %v", err)
	}
}

func TestInput_Required_EachSourceIsChecked(t *testing.T) {
	ctx := test.NewMockContext()
	ctx.MockRequest().WithBody([]byte(`{"ref":null}`))
	var dto requiredMixedDto
	got := missingFields(t, input.NewInput(ctx).Populate(&dto))
	// context-filled, json:"-", binding:"-" and queries:"all" fields are not the caller's to send.
	if strings.Join(got, ",") != "id,X-Tenant,page,flag,count,tags,ref" {
		t.Fatalf("missing %v", got)
	}
}

func TestInput_Required_FormAndRawBody(t *testing.T) {
	ctx := test.NewMockContext()
	ctx.MockRequest().WithBody([]byte("other=1"))
	var form requiredFormDto
	if got := missingFields(t, input.NewInput(ctx).Populate(&form)); strings.Join(got, ",") != "name" {
		t.Fatalf("missing %v", got)
	}
	ctx = test.NewMockContext()
	ctx.MockRequest().WithBody([]byte("name=n"))
	if err := input.NewInput(ctx).Populate(&form); err != nil || form.Name != "n" {
		t.Fatalf("form: %v %+v", err, form)
	}

	var raw requiredRawDto
	if got := missingFields(t, input.NewInput(test.NewMockContext()).Populate(&raw)); strings.Join(got, ",") != "body" {
		t.Fatalf("missing %v", got)
	}
}

func TestInput_Required_AnySourceSatisfiesAField(t *testing.T) {
	ctx := test.NewMockContext()
	ctx.MockRequest().WithQueries(map[string]string{"key": "k"})
	var dto requiredEitherDto
	if err := input.NewInput(ctx).Populate(&dto); err != nil || dto.Key != "k" {
		t.Fatalf("query: %v %+v", err, dto)
	}
	ctx = test.NewMockContext()
	ctx.MockRequest().WithJSONBody(map[string]any{"key": "j"})
	dto = requiredEitherDto{}
	if err := input.NewInput(ctx).Populate(&dto); err != nil || dto.Key != "j" {
		t.Fatalf("json: %v %+v", err, dto)
	}
}

// Through the kernel: the handler never runs and the caller gets a 400 in goose's
// JSON shape ({success:false, message, data}) naming the fields.

type requiredModule struct{ ran *bool }

func (m *requiredModule) Imports() []types.Module { return nil }
func (m *requiredModule) Exports() []any          { return nil }
func (m *requiredModule) Declarations() []any     { return nil }
func (m *requiredModule) Boot(k types.Kernel) error {
	_, err := k.AppendRoutes(types.Route{Method: types.POST, Path: "required-login", Handler: func(p *requiredLoginDto) types.Output {
		*m.ran = true
		return output.OK(p.ClientId)
	}})
	return err
}

func TestKernel_RequiredFieldMissingIs400(t *testing.T) {
	port := 18313
	ran := false
	stopCh := make(chan func() error, 1)
	errCh := make(chan error, 1)
	go func() {
		stop, err := goose.Start(goose.API(api.NewPlatform(api.WithHost("127.0.0.1"), api.WithPort(port)), &requiredModule{ran: &ran}, nil))
		stopCh <- stop
		errCh <- err
	}()
	defer func() {
		_ = syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		select {
		case <-errCh:
		case <-time.After(5 * time.Second):
		}
		select {
		case stop := <-stopCh:
			_ = stop()
		default:
		}
	}()

	url := fmt.Sprintf("http://127.0.0.1:%d/required-login", port)
	post := func(body string) (int, string) {
		var resp *http.Response
		var err error
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			if resp, err = http.Post(url, "application/json", strings.NewReader(body)); err == nil {
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return resp.StatusCode, string(b)
	}

	code, body := post(`{"email":"a@b.c","password":"longenough"}`)
	if code != http.StatusBadRequest {
		t.Fatalf("HTTP %d %s, want 400", code, body)
	}
	for _, want := range []string{`"success":false`, `clientId`, `"fields":["clientId"]`, `"code":"MISSING_REQUIRED_FIELDS"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("body lacks %s: %s", want, body)
		}
	}
	if ran {
		t.Fatal("the handler ran without a required field")
	}

	code, body = post(`{"email":"a@b.c","password":"longenough","clientId":"web"}`)
	if code != http.StatusOK || !ran || !strings.Contains(body, "web") {
		t.Fatalf("complete request: HTTP %d %s (handler ran: %v)", code, body, ran)
	}
}
