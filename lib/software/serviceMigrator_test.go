package software

import (
	"reflect"
	"testing"
)

// Ubuntu's apache2 package ships both apache2.service and the template
// apache2@.service. The template name itself is not a unit systemctl can
// query ("systemctl show apache2@" exits 1), so it must never reach the
// service list; a running instance of it must take its place instead.
func TestSelectServiceUnitsReplacesTemplateWithRunningInstances(t *testing.T) {
	found := []string{"apache2", "apache2@"}
	running := map[string][]string{"apache2@": {"apache2@site1"}}

	got := selectServiceUnits(found, running)
	want := []string{"apache2", "apache2@site1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("selectServiceUnits(%v, %v) = %v, want %v", found, running, got, want)
	}
}

func TestSelectServiceUnitsDropsTemplateWithoutRunningInstance(t *testing.T) {
	found := []string{"apache2", "apache2@"}

	got := selectServiceUnits(found, nil)
	want := []string{"apache2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("selectServiceUnits(%v, nil) = %v, want %v", found, got, want)
	}
}

func TestSelectServiceUnitsKeepsPlainUnitsAndDedups(t *testing.T) {
	found := []string{"nginx", "nginx", "nginx-debug"}

	got := selectServiceUnits(found, nil)
	want := []string{"nginx", "nginx-debug"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("selectServiceUnits(%v, nil) = %v, want %v", found, got, want)
	}
}

func TestTemplateServiceUnits(t *testing.T) {
	got := templateServiceUnits([]string{"apache2", "apache2@", "getty@", "apache2@site1"})
	want := []string{"apache2@", "getty@"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("templateServiceUnits = %v, want %v", got, want)
	}
}

// Output of
// systemctl list-units --type=service --state=running --no-legend --plain 'apache2@*.service'
func TestParseRunningTemplateInstances(t *testing.T) {
	out := "apache2@site1.service loaded active running The Apache HTTP Server\n" +
		"  apache2@site2.service loaded active running The Apache HTTP Server\n" +
		"\n" +
		"apache2.service loaded active running The Apache HTTP Server\n"

	got := parseRunningTemplateInstances(out, "apache2@")
	want := []string{"apache2@site1", "apache2@site2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseRunningTemplateInstances = %v, want %v", got, want)
	}

	if got := parseRunningTemplateInstances("", "apache2@"); len(got) != 0 {
		t.Fatalf("parseRunningTemplateInstances(empty) = %v, want none", got)
	}
}
