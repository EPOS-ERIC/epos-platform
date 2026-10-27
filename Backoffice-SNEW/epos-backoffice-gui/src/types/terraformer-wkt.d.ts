declare module '@terraformer/wkt' {
  export type WktGeoJsonGeometry = {
    type: string;
    coordinates?: unknown;
    geometries?: WktGeoJsonGeometry[];
  };

  export function wktToGeoJSON(value: string): WktGeoJsonGeometry;
}
