package mcp

import "testing"

func TestClientListResourcesReturnsCopy(t *testing.T) {
	c := &Client{
		resources: []Resource{{Name: "r1", URI: "file://a"}},
	}
	out := c.ListResources()
	if len(out) != 1 || out[0].Name != "r1" {
		t.Fatalf("unexpected resources: %+v", out)
	}
	out[0].Name = "mutated"
	if c.resources[0].Name != "r1" {
		t.Fatal("ListResources should return copy")
	}
}

func TestIsUnauthorized(t *testing.T) {
	c := &Client{}
	if !c.isUnauthorized(assertErr("status 401 unauthorized")) {
		t.Fatal("expected unauthorized=true")
	}
	if c.isUnauthorized(assertErr("timeout")) {
		t.Fatal("expected unauthorized=false")
	}
}

type assertErr string

func (e assertErr) Error() string { return string(e) }
