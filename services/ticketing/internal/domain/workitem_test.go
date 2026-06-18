package domain

import "testing"

func TestTypeValidAndPrefix(t *testing.T) {
	if !TypeIncident.Valid() || !TypeServiceRequest.Valid() {
		t.Fatal("known types must be valid")
	}
	if Type("problem").Valid() {
		t.Fatal("unknown type must be invalid")
	}
	if TypeIncident.NumberPrefix() != "INC" {
		t.Fatalf("incident prefix = %q", TypeIncident.NumberPrefix())
	}
	if TypeServiceRequest.NumberPrefix() != "REQ" {
		t.Fatalf("request prefix = %q", TypeServiceRequest.NumberPrefix())
	}
}

func TestFormatNumber(t *testing.T) {
	if got := FormatNumber(TypeIncident, 1001); got != "INC0001001" {
		t.Fatalf("got %q", got)
	}
	if got := FormatNumber(TypeServiceRequest, 42); got != "REQ0000042" {
		t.Fatalf("got %q", got)
	}
}

func TestPriorityValid(t *testing.T) {
	for _, p := range []Priority{PriorityCritical, PriorityHigh, PriorityModerate, PriorityLow} {
		if !p.Valid() {
			t.Fatalf("%q should be valid", p)
		}
	}
	if Priority("urgent").Valid() {
		t.Fatal("unknown priority must be invalid")
	}
}

func TestCanTransition_Legal(t *testing.T) {
	legal := [][2]Status{
		{StatusNew, StatusInProgress},
		{StatusNew, StatusCancelled},
		{StatusInProgress, StatusOnHold},
		{StatusOnHold, StatusInProgress},
		{StatusInProgress, StatusResolved},
		{StatusResolved, StatusClosed},
		{StatusResolved, StatusInProgress}, // reopen
	}
	for _, tc := range legal {
		if !CanTransition(tc[0], tc[1]) {
			t.Errorf("expected %s -> %s to be allowed", tc[0], tc[1])
		}
	}
}

func TestCanTransition_Illegal(t *testing.T) {
	illegal := [][2]Status{
		{StatusNew, StatusClosed},        // must resolve first
		{StatusClosed, StatusInProgress}, // terminal
		{StatusCancelled, StatusNew},     // terminal
		{StatusClosed, StatusClosed},     // no self-loop
		{StatusResolved, StatusOnHold},   // not allowed
	}
	for _, tc := range illegal {
		if CanTransition(tc[0], tc[1]) {
			t.Errorf("expected %s -> %s to be rejected", tc[0], tc[1])
		}
	}
}

func TestIsTerminal(t *testing.T) {
	if !StatusClosed.IsTerminal() || !StatusCancelled.IsTerminal() {
		t.Fatal("closed/cancelled must be terminal")
	}
	if StatusNew.IsTerminal() || StatusResolved.IsTerminal() {
		t.Fatal("new/resolved must not be terminal")
	}
	for _, term := range []Status{StatusClosed, StatusCancelled} {
		if len(AllowedTransitions(term)) != 0 {
			t.Errorf("%s should have no outgoing transitions", term)
		}
	}
}
