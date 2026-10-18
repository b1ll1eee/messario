package domain

import "testing"

func TestDirectKeyIsOrderIndependent(t *testing.T) {
	if DirectKey(7, 3) != "3:7" || DirectKey(3, 7) != "3:7" {
		t.Fatalf("DirectKey not canonical: %q %q", DirectKey(7, 3), DirectKey(3, 7))
	}
}

func TestMemberCanPost(t *testing.T) {
	cases := []struct {
		role MemberRole
		typ  ChatType
		want bool
	}{
		{MemberMember, ChatDirect, true},
		{MemberMember, ChatGroup, true},
		{MemberMember, ChatChannel, false},
		{MemberAdmin, ChatChannel, true},
		{MemberOwner, ChatChannel, true},
	}
	for _, c := range cases {
		m := Member{Role: c.role}
		if got := m.CanPost(c.typ); got != c.want {
			t.Errorf("role %s in %s: CanPost=%v want %v", c.role, c.typ, got, c.want)
		}
	}
}

func TestInvalidIsDetectable(t *testing.T) {
	err := Invalid("bad %s", "thing")
	if !IsInvalid(err) || err.Error() != "bad thing" {
		t.Fatalf("unexpected: %v", err)
	}
	if IsInvalid(ErrNotFound) {
		t.Fatal("ErrNotFound must not be a validation error")
	}
}
