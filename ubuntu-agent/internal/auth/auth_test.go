package auth

import "testing"

func TestIssueValidateRevoke(t *testing.T) {
	a, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	token, err := a.IssueToken("client-1", "Test Phone")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if len(token) != 64 {
		t.Fatalf("token length = %d, want 64", len(token))
	}
	dev, ok := a.Validate(token)
	if !ok || dev.ClientID != "client-1" {
		t.Fatalf("validate failed: %+v %v", dev, ok)
	}
	if _, ok := a.Validate("bogus"); ok {
		t.Fatal("bogus token validated")
	}
	if !a.Revoke(token) {
		t.Fatal("revoke returned false")
	}
	if _, ok := a.Validate(token); ok {
		t.Fatal("revoked token still valid")
	}
}

func TestPersistence(t *testing.T) {
	dir := t.TempDir()
	a, _ := New(dir)
	token, _ := a.IssueToken("client-9", "Persist Phone")
	b, err := New(dir) // reload from disk
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := b.Validate(token); !ok {
		t.Fatal("token not persisted across reload")
	}
}

func TestRevokeByClientID(t *testing.T) {
	a, _ := New(t.TempDir())
	t1, _ := a.IssueToken("c1", "One")
	t2, _ := a.IssueToken("c1", "One")
	if n := a.RevokeByClientID("c1"); n != 2 {
		t.Fatalf("revoked %d, want 2", n)
	}
	if _, ok := a.Validate(t1); ok {
		t.Fatal("t1 still valid")
	}
	if _, ok := a.Validate(t2); ok {
		t.Fatal("t2 still valid")
	}
}

func TestListHidesTokens(t *testing.T) {
	a, _ := New(t.TempDir())
	a.IssueToken("c1", "One")
	list := a.List()
	if len(list) != 1 || list[0].ClientID != "c1" {
		t.Fatalf("unexpected list: %+v", list)
	}
}
