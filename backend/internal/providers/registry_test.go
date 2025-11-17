package providers

import "testing"

func TestDefinitionsExposeDescriptors(t *testing.T) {
	defs := DefaultDefinitions()
	if len(defs) == 0 {
		t.Fatalf("expected default definitions")
	}
	var openaiDef *Definition
	for i := range defs {
		if defs[i].Name == "openai" {
			openaiDef = &defs[i]
			break
		}
	}
	if openaiDef == nil {
		t.Fatalf("expected openai definition registered")
	}
	if openaiDef.Descriptor.Summary == "" {
		t.Fatalf("expected descriptor summary for openai")
	}
	if len(openaiDef.Descriptor.ConfigInputs) == 0 {
		t.Fatalf("expected config inputs documented")
	}
}
