import { Rest } from 'api/webApi/classes/rest.interface';
import { BaseUrl } from 'api/webApi/classes/baseurl.interface';
import { ItemSummary } from 'api/webApi/data/itemSummary.interface';
import { SearchApi } from 'api/webApi/classes/searchApi.interface';
import { SearchCriteria } from 'api/webApi/classes/searchCriteria.enum';
import { Organization } from 'api/webApi/data/organization.interface';
import { UrlBuilder } from 'api/webApi/classes/urlBuilder.interface';
import { JSONDistributionFactory } from 'api/webApi/data/impl/jsonDistributionFactory';

/**
 * **NOT CURRENTLY IN USE. It was originally created when we had an infrastructure search**
 * Responsible for triggering calls to the webApi module "search" endpoint.
 * - Accepts criteria from caller
 * - Triggers the webApi call via the {@link Rest} class
 * - Processes response data into internal objects
 * - Returns the appropriate response to the caller
 */
export class DevSearchApi implements SearchApi {

  constructor(private readonly baseUrl: BaseUrl, private readonly rest: Rest) { }

  doSearch(searchCriteriaMap: Map<SearchCriteria, unknown>): Promise<Array<ItemSummary>> {
    throw new Error('API 1.3 - Method not implemented.');
  }

  getOrganizations(_type: string): Promise<Organization[] | null> {

    const builder: UrlBuilder = this.baseUrl.urlBuilder();
    builder.addPathElements('organizations');

    return this.rest
      .get(builder.build()).then((json: unknown) => {
        const organizations = Array.isArray(json) ? json.map((organization: Record<string, unknown>) => {
          const address = organization.address as Record<string, unknown>;
          return {
            ...organization,
            url: organization.page_url,
            country: address.country,
            logo: organization.logo_url,
          };
        }) : json;
        return JSONDistributionFactory.jsonToOrganizations(organizations);
      });
  }

  getOrganizationById(organizationId: string): Promise<Organization | null> {
      const builder: UrlBuilder = this.baseUrl.urlBuilder();
      builder.addPathElements('organizations');
      builder.addPathElements(organizationId);

      return this.rest
        .get(builder.build()).then((json: Record<string, unknown>) => {
          const address = json.address as Record<string, unknown>;
          return JSONDistributionFactory.jsonToOrganization([{
            ...json,
            url: json.page_url,
            country: address.country,
            logo: json.logo_url,
          }]);
        });
  }
}
