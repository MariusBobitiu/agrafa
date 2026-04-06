package services

import "testing"

func TestPasswordServiceHashAndVerify(t *testing.T) {
	t.Parallel()

	service := NewPasswordService()

	hash, err := service.Hash("correct horse battery staple")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	ok, err := service.Verify("correct horse battery staple", hash)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !ok {
		t.Fatal("Verify() = false, want true")
	}

	ok, err = service.Verify("wrong password", hash)
	if err != nil {
		t.Fatalf("Verify() with wrong password error = %v", err)
	}
	if ok {
		t.Fatal("Verify() = true for wrong password, want false")
	}
}
