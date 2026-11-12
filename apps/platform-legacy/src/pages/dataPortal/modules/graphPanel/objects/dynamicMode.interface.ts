export interface DynamicModePoint {
  age: number;
  lat: number;
  lowerbound?: number;
  upperbound?: number;
}

export interface DynamicModeResponse {
  plate?: {
    id: string;
    name: string;
  };
  paleolatitude?: Array<DynamicModePoint>;
  error?: string;
  message?: string;
}

export const DYNAMIC_MODE_CONFIG_ID = 'dynamic-mode-map-tool';
export const DYNAMIC_MODE_TRACE_ID = 'dynamic-mode-map-tool-trace';
