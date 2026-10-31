package deepopt

import "testing"

func TestDeepControlWireIntactRejectsToolsDelete(t *testing.T) {
	t.Parallel()
	in := []byte(`{"tools":[{"name":"x"}],"messages":[{"role":"user","content":"hello"}]}`)
	out := []byte(`{"messages":[{"role":"user","content":"hi"}]}`)
	if DeepControlWireIntact(in, out) {
		t.Fatal("expected tools delete to fail integrity")
	}
	if !DeepExpanded(in, out, 10, 5) {
		t.Fatal("DeepExpanded must reject tools delete")
	}
}

func TestDeepControlWireIntactKeepsValidRewrite(t *testing.T) {
	t.Parallel()
	in := []byte(`{"tools":[{"name":"x"}],"messages":[{"role":"system","content":"S"},{"role":"user","content":"long filler prose here"}]}`)
	out := []byte(`{"tools":[{"name":"x"}],"messages":[{"role":"system","content":"S"},{"role":"user","content":"short"}]}`)
	if !DeepControlWireIntact(in, out) {
		t.Fatal("valid last-user rewrite should pass")
	}
}
