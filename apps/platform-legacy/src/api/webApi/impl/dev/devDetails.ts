import { Rest } from 'api/webApi/classes/rest.interface';
import { BaseUrl } from 'api/webApi/classes/baseurl.interface';
import { DetailsApi } from 'api/webApi/classes/detailsApi.interface';
import { DistributionDetails } from '../../data/distributionDetails.interface';
import { DistributionSummary } from '../../data/distributionSummary.interface';
import { JSONDistributionFactory } from 'api/webApi/data/impl/jsonDistributionFactory';

/**
 * Responsible for triggering calls to the webApi module "details" endpoint.
 * - Accepts criteria from caller
 * - Triggers the webApi call via the {@link Rest} class
 * - Processes response data into internal objects
 * - Returns the appropriate response to the caller
 */
export class DevDetailsApi implements DetailsApi {
  private readonly resourceVersionResponses = new Map<string, Promise<unknown>>();

  constructor(private readonly baseUrl: BaseUrl, private readonly rest: Rest) { }


  public getDetails(summary: DistributionSummary, context: string): Promise<null | DistributionDetails> {
    return this.getDetailsById(summary.getIdentifier(), context);
  }

  public getDetailsById(idIn: string, context: string): Promise<null | DistributionDetails> {

    const url = this.buildDetailsURL(idIn, context);

    return this.rest
      .get(url)
      .then((json: Record<string, unknown>) => {
        const details = JSONDistributionFactory.jsonToDistributionDetails(json, context);

      if (details.isEmpty()) {
        return Promise.reject();
      }

      return Promise.resolve(details.get());
    });
  }

  public getResourceVersionDistributionDetails(
    resourceType: string,
    resourceId: string,
    versionId: string,
    distributionId: string,
    summary: DistributionSummary,
  ): Promise<DistributionDetails> {
    const url = this.baseUrl.urlBuilder()
      .addPathElements('resources', resourceType, resourceId, 'versions', versionId)
      .build();

    let response = this.resourceVersionResponses.get(url);
    if (response === undefined) {
      response = this.rest.get(url).catch(error => {
        this.resourceVersionResponses.delete(url);
        throw error;
      });
      this.resourceVersionResponses.set(url, response);
    }
    return response
      .then(json => {
        if (typeof json !== 'object' || json === null || Array.isArray(json)) {
          throw new Error(`Invalid resource version response for ${versionId}`);
        }
        const accessUrl = this.baseUrl.urlBuilder().addPathElements('distributions', distributionId, 'access').build();
        const details = JSONDistributionFactory.jsonToResourceVersionDistributionDetails(json as Record<string, unknown>, distributionId, summary, accessUrl);
        if (details.isEmpty()) {
          return Promise.reject(new Error(`Distribution ${distributionId} not found in resource version ${versionId}`));
        }
        return details.get()!;
      });
  }

  private buildDetailsURL(id: string, context: string): string {

    return this.baseUrl.urlBuilder().addPathElements(context, 'details', id).build();
  }
}
