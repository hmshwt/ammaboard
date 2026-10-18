package tuikit

import "testing"

// The kit's generated keymaps are documentation: every binding carries Help,
// and no helper ever touches the reserved global verbs.
func TestScrollKeysAreDocumentedAndTame(t *testing.T) {
	var s Scroll
	km := ScrollKeys(&s)
	for _, b := range km {
		if b.Help == "" {
			t.Errorf("scroll binding %v carries no Help", b.Keys)
		}
	}
	if violations := NoShadowing(nil, km); len(violations) != 0 {
		t.Errorf("scroll keys violate scope discipline: %v", violations)
	}
}

func TestReservedKeysAreTheGlobalVerbs(t *testing.T) {
	want := []string{"q", "ctrl+c", "esc"}
	if len(ReservedKeys) != len(want) {
		t.Fatalf("ReservedKeys = %v, want %v", ReservedKeys, want)
	}
	for i, k := range want {
		if ReservedKeys[i] != k {
			t.Fatalf("ReservedKeys = %v, want %v", ReservedKeys, want)
		}
	}
}
