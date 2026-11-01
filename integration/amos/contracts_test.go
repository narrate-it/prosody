package amos

import (
	"context"
	"strings"
	"testing"

	"github.com/ajent-social/amos/app/operation"
)

func TestFirstOperationDescriptor(t *testing.T) {
	registry, api, mapping, effects, err := Contracts(func(context.Context, operation.InvocationContext, string) ([]byte, error) {
		return []byte(`{"testOnly":true}`), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	definitions := registry.Definitions()
	if len(definitions) != 1 || len(effects) != 1 {
		t.Fatal("unexpected operation inventory")
	}
	m := definitions[0].Metadata()
	if m.OperationID != OperationID || m.IdempotencyField != "/idempotencyKey" || m.SuccessKind != operation.ResultAccepted || m.Resource == nil || m.Resource.Field != "/feedId" {
		t.Fatal("descriptor lost admission bindings")
	}
	if !strings.Contains(string(api), `"202"`) || !strings.Contains(string(mapping), `"workspaceSelection":"active_principal"`) {
		t.Fatal("transport mapping mismatch")
	}
}
func TestMissingAdmissionResolverFails(t *testing.T) {
	if _, _, _, _, err := Contracts(nil); err == nil {
		t.Fatal("accepted absent admission resolver")
	}
}
func TestBodyKeyUnicode(t *testing.T) {
	schema, err := operation.NewJSONSchema([]byte(InputSchema))
	if err != nil {
		t.Fatal(err)
	}
	inner, err := operation.NewJSONCodec[Input](schema)
	if err != nil {
		t.Fatal(err)
	}
	c := codec{inner}
	for _, tc := range []struct {
		key   string
		valid bool
	}{
		{`" \u0000\r\n界 "`, true}, {`"\ud800"`, false}, {`"` + strings.Repeat("界", 86) + `"`, false},
	} {
		raw := []byte(`{"feedId":"01900000-0000-7000-8000-000000000001","idempotencyKey":` + tc.key + `,"text":"source"}`)
		_, _, err = c.DecodeCanonical(operation.RawInput{Body: raw})
		if (err == nil) != tc.valid {
			t.Fatalf("valid=%t err=%v", tc.valid, err)
		}
	}
}
