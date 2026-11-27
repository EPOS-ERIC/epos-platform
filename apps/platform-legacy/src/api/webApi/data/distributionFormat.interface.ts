import { Usable } from './usable';

export interface ResourceParameter {
  id: string;
  name: string;
  kind: string;
  inputName?: string;
  role?: string;
  defaultValue?: string;
  required?: boolean;
  readOnly?: boolean;
  multipleValues?: boolean;
  explode?: boolean;
  template?: string;
  parameters?: Array<ResourceParameter>;
}

/**
 * A (DDSS) distrubution can support multiple formats, those formats are represented by this interface.
 */
export interface DistributionFormat extends Usable {
  resourceAccess?: {
    type: string;
    parameters: Array<ResourceParameter>;
    conversionId?: string;
    plugin?: { id: string; name: string; description: string };
    capabilitiesUrl?: string;
    defaultLayer?: string;
    defaultStyle?: string;
    defaultTransparency?: boolean;
    transparencyEditable?: boolean;
    layers?: Array<{ name: string; defaultStyle?: string; styles: Array<string> }>;
    operations?: { tile: Array<ResourceParameter>; featureInfo: Array<ResourceParameter> };
  };
  /**
   * Display label for the format.
   */
  getLabel(): string;

  /**
   * Identifier for the format.
   */
  getFormat(): string;

  /**
   * Identifier for the original format.
   */
  getOriginalFormat(): string;

  /**
   * The HREF (link) to access this format from a paticular distribition.
   */
  getUrl(): string;

  /**
   * The errr 'type' of the format whether its 'original' or 'converted'.
   */
  getType(): string;
}
