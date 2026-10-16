package winproc

import "testing"

func TestClassifyMSFSExecutable(t *testing.T) {
	tests := []struct {
		name string
		want string
		ok   bool
	}{
		{"FlightSimulator.exe", "Microsoft Flight Simulator", true},
		{"FLIGHTSIMULATOR2024.EXE", "MSFS 2024", true},
		{"X-Plane.exe", "", false},
	}
	for _, test := range tests {
		got, ok := ClassifyMSFSExecutable(test.name)
		if got != test.want || ok != test.ok {
			t.Fatalf("ClassifyMSFSExecutable(%q)=(%q,%t), want (%q,%t)", test.name, got, ok, test.want, test.ok)
		}
	}
}
