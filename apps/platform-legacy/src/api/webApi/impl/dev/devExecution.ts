import { DistributionSummary } from 'api/webApi/data/distributionSummary.interface';
import { ExecutionApi } from 'api/webApi/classes/executionApi.interface';
import { BaseUrl } from 'api/webApi/classes/baseurl.interface';
import { Rest } from 'api/webApi/classes/rest.interface';
import { DistributionFormat } from 'api/webApi/data/distributionFormat.interface';
import { ParameterValue } from 'api/webApi/data/parameterValue.interface';
import { JSONDistributionFactory } from 'api/webApi/data/impl/jsonDistributionFactory';

/**
 * Responsible for triggering calls to the webApi module "execute" endpoint.
 * - Accepts criteria from caller
 * - Triggers the webApi call via the {@link Rest} class
 * - Processes response data into internal objects
 * - Returns the appropriate response to the caller
 */
export class DevExecutionApi implements ExecutionApi {

  // path
  private static readonly GET_ORIGINAL = 'getoriginalurl';
  // params
  private static readonly ID = 'id';
  private static readonly PARAMS = 'params';
  private readonly pendingResourceRequests = new Map<string, Promise<Record<string, unknown> | Blob>>();

  constructor(private readonly baseUrl: BaseUrl, private readonly rest: Rest) { }

  public executeAuthenticatedUrl(
    url: string,
  ): Promise<Blob> {
    return this.rest.get(url, false, true) as Promise<Blob>;
  }

  public executeUrl(
    url: string,
  ): Promise<Blob> {
    return this.rest.get(url, false, true) as Promise<Blob>;
  }

  public executeDistributionFormat(
    format: DistributionFormat,
    params: null | Array<ParameterValue>,
    asBlob: boolean,
  ): Promise<Record<string, unknown> | Blob> {
    if (format.resourceAccess != null) {
      if (format.resourceAccess.type !== 'HTTP' && format.resourceAccess.type !== 'WFS' && format.resourceAccess.type !== 'DYNAMIC') {
        return Promise.reject(new Error(`Distribution type ${format.resourceAccess.type} must be accessed directly.`));
      }
      const body = this.resourceRequest(format, params);
      const key = JSON.stringify([format.getUrl(), body, asBlob]);
      const pending = this.pendingResourceRequests.get(key);
      if (pending != null) {
        return pending;
      }
      const request = this.rest.post(format.getUrl(), body, false, {}, asBlob) as Promise<Record<string, unknown> | Blob>;
      this.pendingResourceRequests.set(key, request);
      void request.then(() => this.pendingResourceRequests.delete(key), () => this.pendingResourceRequests.delete(key));
      return request;
    }
    const url = this.getExecuteUrl(format, params);
    return this.rest.get(url, false, asBlob) as Promise<Record<string, unknown> | Blob>;
  }

  public getExecuteUrl(
    format: DistributionFormat,
    params: null | Array<ParameterValue>,
  ): string {
    if (format.resourceAccess?.type === 'WMS' || format.resourceAccess?.type === 'WMTS') {
      return format.resourceAccess.capabilitiesUrl!;
    }
    let url = format.getUrl();

    url += (url.indexOf('?') < 0) ? '?' : '';

    if (params != null) {
      const paramsValueString = encodeURIComponent(this.paramsToJSONString(params));
      url += `&${DevExecutionApi.PARAMS}=${paramsValueString}`;
    }
    return url;
  }

  public getOriginatorUrl(distribution: DistributionSummary, params: null | Array<ParameterValue>, selectedFormat?: DistributionFormat): Promise<string> {
    const format = selectedFormat ?? distribution.getFormats().find(item => item.resourceAccess != null && (item.getType() === 'ORIGINAL' || item.getType() === 'SERVICE'));
    if (format?.resourceAccess != null) {
      if (format.resourceAccess?.type === 'WMS' || format.resourceAccess?.type === 'WMTS') {
        return Promise.resolve(format.resourceAccess.capabilitiesUrl!);
      }
      if (format.getType() === 'CONVERTED') {
        return Promise.reject(new Error('Converted formats do not support provider URL preparation.'));
      }
      return this.rest.post(format.getUrl().replace('/access?', '/prepare?'), this.resourceRequest(format, params))
        .then((response: { url: string }) => response.url);
    }

    const urlBuilder = this.baseUrl.urlBuilder().addPathElements(DevExecutionApi.GET_ORIGINAL,distribution.getIdentifier());

    if (params != null) {
      const paramsValueString = this.paramsToJSONString(params);
      urlBuilder.addParameter(DevExecutionApi.PARAMS, paramsValueString);
    }

    const url = urlBuilder.build();

    return this.rest
      .get(url)
      .then((json: Record<string, unknown>) => {
        const optional = JSONDistributionFactory.jsonToOriginalUrl(json);

        if (optional.isEmpty()) {
          return Promise.reject();
        }

        return Promise.resolve(optional.get()!);
      });

  }

  private resourceRequest(format: DistributionFormat, params: null | Array<ParameterValue>): Record<string, unknown> {
    const access = format.resourceAccess!;
    const values = params ?? [];
    const coordinates: Record<string, number> = {};
    const parameters = values.flatMap(param => {
      const definition = access.parameters.find(item => (item.inputName ?? item.name) === param.name);
      if (definition == null) {
        throw new Error(`Unknown resource parameter: ${param.name}`);
      }
      if (access.type === 'DYNAMIC' && (definition.role === 'LONGITUDE' || definition.role === 'LATITUDE')) {
        coordinates[definition.role.toLowerCase()] = Number(param.value);
        return [];
      }
      if (param.value === (definition.defaultValue ?? '')) {
        return [];
      }
      const scalarValue = (rawValue: string): string | number | boolean => definition.kind === 'FLOAT' || definition.kind === 'INTEGER' ? Number(rawValue)
        : definition.kind === 'BOOLEAN' ? rawValue === 'true' : rawValue;
      const value = definition.multipleValues ? param.value.split(',').map(scalarValue) : scalarValue(param.value);
      return [{ id: definition.id, value }];
    });
    return { type: access.type, ...coordinates, parameters };
  }

  private paramsArrayToObject(params: Array<ParameterValue>): Record<string, unknown> {
    const obj = {};
    params.forEach((param: ParameterValue) => {
      obj[param.name] = param.value;
    });
    return obj;
  }

  private paramsToJSONString(params: Array<ParameterValue>) {
    const paramsValue = this.paramsArrayToObject(params);
    return JSON.stringify(paramsValue);
  }
}
