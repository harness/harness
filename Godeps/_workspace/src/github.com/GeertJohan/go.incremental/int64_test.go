package incremental

import (
	"testing"
)

type someInt64Struct struct {
	i Int64
}

func TestInt64Ptr(t *testing.T) {
	i := &Int64{}
	num := i.Next()
	if num != 1 {
		t.Fatal("expected 1, got %d", num)
	}
	num = i.Next()
	if num != 2 {
		t.Fatal("expected 2, got %d", num)
	}
	num = i.Last()
	if num != 2 {
		t.Fatal("expected last to be 2, got %d", num)
	}

	i.Set(42)
	num = i.Last()
	if num != 42 {
		t.Fatal("expected last to be 42, got %d", num)
	}
	num = i.Next()
	if num != 43 {
		t.Fatal("expected 43, got %d", num)
	}
}

func TestInt64AsField(t *testing.T) {
	s := someInt64Struct{}
	num := s.i.Next()
	if num != 1 {
		t.Fatal("expected 1, got %d", num)
	}
	num = s.i.Next()
	if num != 2 {
		t.Fatal("expected 2, got %d", num)
	}
	num = s.i.Last()
	if num != 2 {
		t.Fatal("expected last to be 2, got %d", num)
	}

	useSomeInt64Struct(&s, t)

	num = s.i.Last()
	if num != 3 {
		t.Fatal("expected last to be 3, got %d", num)
	}

	s.i.Set(42)
	num = s.i.Last()
	if num != 42 {
		t.Fatal("expected last to be 42, got %d", num)
	}
	num = s.i.Next()
	if num != 43 {
		t.Fatal("expected 43, got %d", num)
	}
}

func useSomeInt64Struct(s *someInt64Struct, t *testing.T) {
	num := s.i.Next()
	if num != 3 {
		t.Fatal("expected 3, got %d", num)
	}
}
