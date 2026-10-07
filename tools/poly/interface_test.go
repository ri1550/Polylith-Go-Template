package main

import (
	"strings"
	"testing"
)

func change(brick, status string) surfaceChange {
	return surfaceChange{Brick: brick, Status: status, Added: []string{"func X()"}}
}

func scope(number string, affects map[string][]string) adrScope {
	return adrScope{Number: number, Affects: affects}
}

func TestUnrecordedRequiresAnADRThatNamesEachBrick(t *testing.T) {
	changes := []surfaceChange{change("components/users", "new"), change("bases/api", "changed")}

	// No evidence at all: both bricks are missing.
	missing, _ := unrecorded(changes, recording{})
	if len(missing) != 2 {
		t.Fatalf("want 2 missing, got %v", missing)
	}

	// An ADR about something else does not count.
	missing, _ = unrecorded(changes, recording{ADRs: []adrScope{scope("0009", map[string][]string{"components": {"auth"}})}})
	if len(missing) != 2 {
		t.Errorf("unrelated ADR must not cover anything, got %v", missing)
	}

	// An ADR naming one brick leaves the other missing, and the message names it.
	missing, _ = unrecorded(changes, recording{ADRs: []adrScope{scope("0009", map[string][]string{"components": {"users"}})}})
	if len(missing) != 1 || missing[0].Brick != "bases/api" {
		t.Errorf("want only bases/api missing, got %v", missing)
	}

	// A wildcard covers every brick of that kind.
	missing, _ = unrecorded(changes, recording{ADRs: []adrScope{scope("0009", map[string][]string{"components": {"*"}, "bases": {"*"}})}})
	if len(missing) != 0 {
		t.Errorf("wildcard should cover all, got %v", missing)
	}
}

func TestUnrecordedMarkers(t *testing.T) {
	changes := []surfaceChange{change("components/users", "new")}
	for _, m := range []string{"none", "new"} {
		if missing, _ := unrecorded(changes, recording{Markers: []string{m}}); len(missing) != 0 {
			t.Errorf("[interface-impact: %s] should excuse the change, got %v", m, missing)
		}
	}
	missing, notes := unrecorded(changes, recording{Markers: []string{"breaking"}})
	if len(missing) != 1 {
		t.Errorf("breaking is not a marker; want the brick still missing, got %v", missing)
	}
	if len(notes) == 0 || !strings.Contains(notes[0], "needs an ADR file") {
		t.Errorf("want a note explaining breaking needs an ADR, got %v", notes)
	}
}

func TestMarkerAndReferenceParsing(t *testing.T) {
	msg := "Refactor the thing [interface-impact: NEW]\n\nSee ADR-0012 and adr-0003."
	if m := markerRE.FindStringSubmatch(msg); m == nil || strings.ToLower(m[1]) != "new" {
		t.Errorf("marker not parsed: %v", m)
	}
	refs := adrRefRE.FindAllStringSubmatch(msg, -1)
	if len(refs) != 2 || refs[0][1] != "0012" || refs[1][1] != "0003" {
		t.Errorf("refs = %v", refs)
	}
	if adrRefRE.MatchString("the badr-0001 thing") {
		t.Error("adr- inside another word must not count")
	}
}

func TestPairChangesFoldsRenames(t *testing.T) {
	before := map[string][]string{"bases/api": {"func Run()"}, "components/x": {"func X()"}}
	after := map[string][]string{"bases/greetapi": {"func Run()"}, "components/y": {"func Y()"}}
	renames := map[string]string{"bases/api": "bases/greetapi", "components/x": "components/y"}
	got := pairChanges(before, after, renames)
	if len(got) != 2 {
		t.Fatalf("want 2 changes, got %+v", got)
	}
	if got[0].Brick != "bases/greetapi" || got[0].From != "bases/api" || got[0].Status != "renamed" || len(got[0].Added) != 0 {
		t.Errorf("pure rename reported as %+v", got[0])
	}
	if got[1].Status != "renamed, changed" || len(got[1].Removed) != 1 || len(got[1].Added) != 1 {
		t.Errorf("rename with a surface change reported as %+v", got[1])
	}
	// Without git's rename signal the same trees are a removal and an addition.
	plain := pairChanges(before, after, nil)
	if len(plain) != 4 {
		t.Errorf("want removed+new for both bricks, got %+v", plain)
	}
}

func TestRenamedBrickIsCoveredByEitherName(t *testing.T) {
	c := surfaceChange{Brick: "bases/greetapi", From: "bases/api", Status: "renamed"}
	for _, name := range []string{"api", "greetapi"} {
		if missing, _ := unrecorded([]surfaceChange{c}, recording{ADRs: []adrScope{scope("0010", map[string][]string{"bases": {name}})}}); len(missing) != 0 {
			t.Errorf("ADR naming %q should cover the rename, got %v", name, missing)
		}
	}
}
