import { DistributionFormat, ResourceParameter } from '../distributionFormat.interface';
import { ParameterValue } from '../parameterValue.interface';
import { Confirm } from 'api/webApi/utility/preconditions';
import { DistributionFormatType } from '../distributionFormatType';

export class SimpleDistributionFormat implements DistributionFormat {
  public readonly isDownloadable: boolean;
  public readonly isMappable: boolean;
  public readonly isGraphable: boolean;
  public readonly isTabularable: boolean;
  public readonly isOnlyDownloadable: boolean;

  protected constructor( //
    protected readonly label: string, //
    protected readonly format: string, //
    protected readonly originalFormat: string, //
    protected readonly url: string, //
    protected readonly type: string
  ) {
    this.isDownloadable = this.type !== 'SERVICE' && DistributionFormatType.isDownloadable(this.format);
    this.isMappable = DistributionFormatType.isMappable(this.format);
    this.isGraphable = DistributionFormatType.isGraphable(this.format);
    this.isTabularable = DistributionFormatType.isTabularable(this.format);
    this.isOnlyDownloadable = false;
  }

  public static make(
    label: string,
    format: string,
    originalFormat: string,
    url: string,
    type: string
  ): DistributionFormat {
    Confirm.requiresValidString(label);
    Confirm.requiresValidString(format);
    Confirm.requiresValidString(originalFormat);
    Confirm.requiresValidString(type);
    Confirm.requiresValidString(url);
    return new SimpleDistributionFormat(label, format, originalFormat, url, type);
  }

  public static getOperationParameters(
    format: DistributionFormat,
    values: Array<ParameterValue>,
    operation: 'tile' | 'featureInfo',
  ): URLSearchParams {
    const access = format.resourceAccess!;
    const result = new URL(access.capabilitiesUrl!).searchParams;
    Array.from(result.keys()).forEach(key => {
      if (['service', 'request', 'version'].includes(key.toLowerCase())) {
        result.delete(key);
      }
    });
    const resolve = (parameter: ResourceParameter): string | undefined => {
      if (parameter.readOnly) {
        return parameter.defaultValue;
      }
      const supplied = values.find(value => value.name === (parameter.inputName ?? parameter.name))?.value;
      return supplied === '' && parameter.defaultValue === undefined ? undefined : supplied ?? parameter.defaultValue;
    };
    (access.operations?.[operation] ?? []).forEach(parameter => {
      let value: string;
      if (parameter.kind === 'TEMPLATE') {
        const children = parameter.parameters ?? [];
        const missing = children.some(child => resolve(child) === undefined);
        if (!parameter.required && missing) {
          if (children.some(child => values.some(item => item.name === child.inputName && item.value !== child.defaultValue && item.value !== ''))) {
            throw new Error(`Supply all inputs for template ${parameter.name}.`);
          }
          return;
        }
        const source = parameter.template ?? '';
        const substitution = /\{\{\s*\.([A-Za-z_][A-Za-z0-9_]*)\s*\}\}/g;
        if (/\{\{|\}\}/.test(source.replace(substitution, ''))) {
          throw new Error(`Unsupported provider template expression in ${parameter.name}.`);
        }
        value = source.replace(substitution, (_, name: string) => {
          const child = children.find(item => item.name === name);
          if (child == null || resolve(child) === undefined) {
            throw new Error(`Missing input ${name} for template ${parameter.name}.`);
          }
          return resolve(child)!;
        });
      } else {
        const resolved = resolve(parameter);
        if (resolved === undefined && !parameter.required) {
          return;
        }
        if (resolved === undefined) {
          throw new Error(`Missing required provider parameter ${parameter.name}.`);
        }
        value = resolved;
      }
      Array.from(result.keys()).forEach(key => {
        if (key.toLowerCase() === parameter.name.toLowerCase()) {
          result.delete(key);
        }
      });
      const items = parameter.explode && parameter.multipleValues ? value.split(',') : [value];
      items.forEach(item => result.append(parameter.name, item));
    });
    return result;
  }

  public static withParameters(url: string, parameters: URLSearchParams): string {
    const placeholders: Array<string> = [];
    let marker = '__eposURLPlaceholder';
    while (url.includes(marker) || parameters.toString().includes(marker)) {
      marker += '_';
    }
    // Protect existing URL placeholders; braces in new parameter values must stay encoded.
    const source = url.replace(/\{[^{}]+\}/g, placeholder => `${marker}${placeholders.push(placeholder) - 1}__`);
    const result = new URL(source);
    const names = new Set(Array.from(parameters.keys()).map(key => key.toLowerCase()));
    Array.from(result.searchParams.keys()).forEach(key => {
      if (names.has(key.toLowerCase())) {
        result.searchParams.delete(key);
      }
    });
    parameters.forEach((value, key) => result.searchParams.append(key, value));
    return result.toString().replace(new RegExp(`${marker}(\\d+)__`, 'g'), (_, index: string) => placeholders[Number(index)]);
  }

  getUrl(): string {
    return this.url;
  }
  getLabel(): string {
    return this.label;
  }
  getFormat(): string {
    return this.format;
  }
  getOriginalFormat(): string {
    return this.originalFormat;
  }
  getType(): string {
    return this.type;
  }
}
