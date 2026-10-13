package provider

import("strings";"testing")
func TestResponsesReasoningAliasDoesNotLeak(t *testing.T){
 raw:=[]byte(`{"reasoning_effort":"low"}`)
 wire,e:=normalizeFactoryParameters("gpt-5.3-codex","openai-response","openai-response",raw,raw)
 if e!=nil{t.Fatal(e)}
 if strings.Contains(string(wire),"reasoning_effort")||!strings.Contains(string(wire),`"effort":"low"`){t.Fatalf("source alias leaked to upstream: %s",wire)}
}
