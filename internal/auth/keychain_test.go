package auth

import (
	"errors"
	"testing"

	"github.com/99designs/keyring"
)

type stubKeyring struct {
	items map[string]keyring.Item
}

func newStubKeyring() *stubKeyring {
	return &stubKeyring{items: map[string]keyring.Item{}}
}

func (s *stubKeyring) Get(key string) (keyring.Item, error) {
	item, ok := s.items[key]
	if !ok {
		return keyring.Item{}, keyring.ErrKeyNotFound
	}
	return item, nil
}

func (s *stubKeyring) GetMetadata(key string) (keyring.Metadata, error) {
	_, err := s.Get(key)
	if err != nil {
		return keyring.Metadata{}, err
	}
	return keyring.Metadata{}, nil
}

func (s *stubKeyring) Set(item keyring.Item) error {
	s.items[item.Key] = item
	return nil
}

func (s *stubKeyring) Remove(key string) error {
	if _, ok := s.items[key]; !ok {
		return keyring.ErrKeyNotFound
	}
	delete(s.items, key)
	return nil
}

func (s *stubKeyring) Keys() ([]string, error) {
	out := make([]string, 0, len(s.items))
	for key := range s.items {
		out = append(out, key)
	}
	return out, nil
}

func withStubKeyring(t *testing.T) *stubKeyring {
	t.Helper()

	stub := newStubKeyring()
	prev := keyringOpener
	keyringOpener = func() (keyring.Keyring, error) {
		return stub, nil
	}
	t.Cleanup(func() {
		keyringOpener = prev
	})
	return stub
}

func TestShouldBypassKeychain(t *testing.T) {
	if ShouldBypassKeychain(func(string) string { return "true" }) != true {
		t.Fatal("expected truthy bypass")
	}
	if ShouldBypassKeychain(func(string) string { return "1" }) != true {
		t.Fatal("expected truthy bypass")
	}
	if ShouldBypassKeychain(func(string) string { return "false" }) != false {
		t.Fatal("expected false bypass")
	}
}

func TestStoreAndLoadProfileCredential(t *testing.T) {
	withStubKeyring(t)

	if err := StoreProfileCredential("default", []byte(`{"type":"service_account"}`)); err != nil {
		t.Fatalf("store failed: %v", err)
	}

	got, err := LoadProfileCredential("default")
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if string(got) != `{"type":"service_account"}` {
		t.Fatalf("unexpected payload: %s", string(got))
	}
}

func TestLoadProfileCredential_NotFound(t *testing.T) {
	withStubKeyring(t)

	_, err := LoadProfileCredential("missing")
	if !errors.Is(err, ErrCredentialNotFound) {
		t.Fatalf("expected ErrCredentialNotFound, got %v", err)
	}
}

func TestListAndRemoveProfiles(t *testing.T) {
	stub := withStubKeyring(t)

	stub.items[keyringKey("b")] = keyring.Item{Key: keyringKey("b"), Data: []byte(`{"a":1}`)}
	stub.items[keyringKey("a")] = keyring.Item{Key: keyringKey("a"), Data: []byte(`{"a":1}`)}
	stub.items["other-key"] = keyring.Item{Key: "other-key", Data: []byte("x")}

	profiles, err := ListKeychainProfiles()
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(profiles) != 2 || profiles[0] != "a" || profiles[1] != "b" {
		t.Fatalf("unexpected profiles: %#v", profiles)
	}

	if err := RemoveProfileCredential("a"); err != nil {
		t.Fatalf("remove failed: %v", err)
	}
	if _, ok := stub.items[keyringKey("a")]; ok {
		t.Fatal("profile should be removed")
	}

	if err := RemoveAllProfileCredentials(); err != nil {
		t.Fatalf("remove all failed: %v", err)
	}
	if _, ok := stub.items[keyringKey("b")]; ok {
		t.Fatal("profile should be removed")
	}
	if _, ok := stub.items["other-key"]; !ok {
		t.Fatal("non-profile key should remain")
	}
}

func TestKeychainAvailabilityUnavailable(t *testing.T) {
	prev := keyringOpener
	keyringOpener = func() (keyring.Keyring, error) {
		return nil, keyring.ErrNoAvailImpl
	}
	t.Cleanup(func() {
		keyringOpener = prev
	})

	ok, err := KeychainAvailable(func(string) string { return "" })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected unavailable")
	}
}
