package db

import (
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/metadata"
)

func TestSpatialJSONRoundTrip(t *testing.T) {
	t.Parallel()

	input := &metadata.Spatial{Polygon: []metadata.Coordinate{{Latitude: 45.1, Longitude: 12.2}, {Latitude: 45.2, Longitude: 12.3}, {Latitude: 45.1, Longitude: 12.2}}}
	point, polygon, err := spatialJSON(input)
	if err != nil {
		t.Fatal(err)
	}
	if point != nil || polygon == nil {
		t.Fatalf("point = %v, polygon = %v", point, polygon)
	}

	output, err := spatialFromJSON("", *polygon)
	if err != nil {
		t.Fatal(err)
	}
	if len(output.Polygon) != len(input.Polygon) || output.Polygon[1] != input.Polygon[1] {
		t.Fatalf("polygon = %#v", output.Polygon)
	}
}
