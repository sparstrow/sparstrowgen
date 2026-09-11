package auth

import (
	"strings"
	"testing"
)

func TestAPasswordVerifiesAgainstItsOwnHashAndNothingElse(t *testing.T) {
	hash, err := HashPassword("the right password")
	if err != nil {
		t.Fatal(err)
	}

	ok, err := VerifyPassword(hash, "the right password")
	if err != nil || !ok {
		t.Fatalf("the correct password was rejected: ok=%v err=%v", ok, err)
	}

	for _, wrong := range []string{
		"the right passwore", // one character out
		"the right password ",
		"The Right Password",
		"",
		"the right passwor",
	} {
		ok, err := VerifyPassword(hash, wrong)
		if err != nil {
			t.Fatalf("verifying %q: %v", wrong, err)
		}
		if ok {
			t.Errorf("%q was accepted", wrong)
		}
	}
}

// Two hashes of the same password must differ, or the hash is a fingerprint:
// anyone holding two deployments' configuration could tell they share a
// password, and a precomputed table would work against both.
func TestTheSamePasswordHashesDifferentlyEveryTime(t *testing.T) {
	first, err := HashPassword("same password")
	if err != nil {
		t.Fatal(err)
	}
	second, err := HashPassword("same password")
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("two hashes of one password came out identical — the salt is not random")
	}
	for _, h := range []string{first, second} {
		if ok, err := VerifyPassword(h, "same password"); err != nil || !ok {
			t.Errorf("a hash did not verify its own password: %v", err)
		}
	}
}

// The parameters live in the hash, not in a constant read at verify time. This
// is what lets the cost be raised later without locking the owner out of the
// password he already set — so it is worth a test rather than a comment.
func TestAHashWrittenWithOtherParametersStillVerifies(t *testing.T) {
	hash, err := HashPassword("portable")
	if err != nil {
		t.Fatal(err)
	}
	// A hash carrying its own, different memory cost — as one written before a
	// future increase would.
	weaker := strings.Replace(hash, "m=19456", "m=9728", 1)
	if weaker == hash {
		t.Fatalf("the test did not find the parameters to change in %q", hash)
	}
	// It must not verify against the ORIGINAL password, because the key derived
	// under different parameters is a different key — what matters is that it
	// is read and used rather than ignored in favour of the current constants.
	ok, err := VerifyPassword(weaker, "portable")
	if err != nil {
		t.Fatalf("a hash with different parameters could not be read: %v", err)
	}
	if ok {
		t.Error("the encoded parameters were ignored: the current constants were used instead")
	}
}

// A misconfigured hash must be distinguishable from a wrong password. One means
// "fix the deployment", the other means "somebody is guessing", and answering
// the first as though it were the second hides an outage behind a login screen.
func TestAnUnreadableHashIsAnErrorNotAWrongPassword(t *testing.T) {
	for _, bad := range []string{
		"",
		"not a hash at all",
		"$argon2i$v=19$m=19456,t=2,p=1$c2FsdA$a2V5",  // wrong variant
		"$argon2id$v=19$m=19456,t=2,p=1$c2FsdA",      // truncated
		"$argon2id$v=99$m=19456,t=2,p=1$c2FsdA$a2V5", // version we do not write
		"$argon2id$v=19$m=notanumber,t=2,p=1$c2FsdA$a2V5",
	} {
		ok, err := VerifyPassword(bad, "anything")
		if err == nil {
			t.Errorf("%q was accepted as a readable hash", bad)
		}
		if ok {
			t.Errorf("%q verified something", bad)
		}
	}
}

// Codex's finding: the parameters are read from the hash, so the hash must not
// be allowed to claim anything it likes. A one-byte key would mean the password
// is verified on one byte — eight bits of guessing, however long the real
// password is — and t=0 or p=0 panics inside argon2 rather than erroring.
//
// None of these is reachable by a remote attacker; the hash comes from the
// deployment's own configuration. The point is that a typo there must stay a
// typo rather than becoming a bypass or an outage.
func TestAHashCannotTalkItsWayIntoBeingWeak(t *testing.T) {
	cases := []struct {
		name    string
		encoded string
	}{{
		name:    "a one-byte key verifies on one byte",
		encoded: "$argon2id$v=19$m=19456,t=2,p=1$AAAAAAAAAAAAAAAAAAAAAA$AA",
	}, {
		name:    "a short salt",
		encoded: "$argon2id$v=19$m=19456,t=2,p=1$AAAA$" + strings.Repeat("A", 43),
	}, {
		name:    "no iterations at all, which panics rather than failing",
		encoded: "$argon2id$v=19$m=19456,t=0,p=1$AAAAAAAAAAAAAAAAAAAAAA$" + strings.Repeat("A", 43),
	}, {
		name:    "no parallelism, same",
		encoded: "$argon2id$v=19$m=19456,t=2,p=0$AAAAAAAAAAAAAAAAAAAAAA$" + strings.Repeat("A", 43),
	}, {
		name:    "memory so low the hash is not worth the name",
		encoded: "$argon2id$v=19$m=8,t=2,p=1$AAAAAAAAAAAAAAAAAAAAAA$" + strings.Repeat("A", 43),
	}, {
		name:    "memory so high every attempt is an out-of-memory",
		encoded: "$argon2id$v=19$m=4294967295,t=2,p=1$AAAAAAAAAAAAAAAAAAAAAA$" + strings.Repeat("A", 43),
	}}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Must not panic, must not verify, must report the hash as unreadable.
			ok, err := VerifyPassword(c.encoded, "whatever")
			if ok {
				t.Error("it verified a password")
			}
			if err == nil {
				t.Error("it was accepted as a readable hash")
			}
		})
	}
}

// And the hash this program actually writes must survive all of that.
func TestOurOwnHashPassesTheBounds(t *testing.T) {
	hash, err := HashPassword("a perfectly ordinary password")
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := VerifyPassword(hash, "a perfectly ordinary password"); err != nil || !ok {
		t.Fatalf("the bounds rejected our own hash: ok=%v err=%v", ok, err)
	}
}
