import { FacetModel } from '../../data/facetModel.interface';
import { ObjectAccessUtility } from '../../utility/objectAccessUtility';
import { DistributionSummary } from '../../data/distributionSummary.interface';
import { Rest } from 'api/webApi/classes/rest.interface';
import { BaseUrl } from 'api/webApi/classes/baseurl.interface';
import { DiscoverApi, DiscoverResponse, DiscoverRequest } from 'api/webApi/classes/discoverApi.interface';
import { UrlBuilder } from 'api/webApi/classes/urlBuilder.interface';
import { Facet } from 'api/webApi/data/facet.interface';
import { SimpleFacet } from 'api/webApi/data/impl/simpleFacet';
import { SimpleFacetModel } from 'api/webApi/data/impl/simpleFacetModel';
import { JSONDistributionFactory } from 'api/webApi/data/impl/jsonDistributionFactory';
import { Optional } from 'api/webApi/utility/optional';
import { Domain } from 'api/webApi/data/domain.interface';
import { CONTEXT_FACILITY, CONTEXT_RESOURCE, CONTEXT_SOFTWARE } from 'api/api.service.factory';
import { DatasetDistribution, DatasetReference, DatasetSummary } from 'api/webApi/data/datasetSummary.interface';

interface GroupCategory {
  id: string;
  name: string;
  parentId: string | null;
}

interface GroupMetadata {
  id: string;
  position: number;
  concept: 'DATA' | 'FACILITIES' | 'SOFTWARE';
}

/**
 * Responsible for triggering calls to the webApi module "discover" endpoint.
 * - Accepts criteria from caller
 * - Triggers the webApi call via the {@link Rest} class
 * - Processes response data into internal objects
 * - Returns the appropriate response to the caller
 */
export class DevDiscoverApi implements DiscoverApi {

  constructor(
    protected readonly baseUrl: BaseUrl,
    protected readonly rest: Rest,
  ) { }

  getDomains(context: string): Promise<null | Array<Domain>> {
    const builder: UrlBuilder = this.baseUrl.urlBuilder();
    builder.addPathElements('groups');

    const url = builder.build();

    return this.rest.get(url)
      .then((groups: Record<string, unknown>) => {
        if (!Array.isArray(groups)) {
          throw new Error('Invalid groups response');
        }
        const domains = this.parseJSONDomains(groups).orNull();
        if (domains === null) {
          return null;
        }
        return this.orderDomainsByGroupPosition(domains, groups as GroupMetadata[], context);
      });
  }

  getFilters(context: string): Promise<null | DiscoverResponse> {
    const builder: UrlBuilder = this.baseUrl.urlBuilder();
    builder.addPathElements('search');
    this.addResourceFilters(builder, context);

    const url = builder.build();

    return this.fetchAndParse(url);
  }

  discover(request: DiscoverRequest): Promise<null | DiscoverResponse> {
    const builder: UrlBuilder = this.baseUrl.urlBuilder();
    builder.addPathElements('search');

    const context = request.getContext();
    if (context != null) {
      this.addResourceFilters(builder, context);
    }

    // QUERY
    const query = request.getQuery();
    if (query != null && query.trim().length > 0) {
      builder.addParameter('q', query);
    }

    if (request.hasTemporalRange()) {
      const momentFormat = 'YYYY-MM-DDTHH:mm:ss[Z]';

      // START
      const start = request.getStartDate();
      if (start != null) {
        builder.addParameter('dateFrom', start.utc().format(momentFormat));
      }

      // END
      const end = request.getEndDate();
      if (end != null) {
        builder.addParameter('dateTo', end.utc().format(momentFormat));
      }

    }

    // BBOX
    const bbox = request.getBBox();
    if (bbox.isBounded()) {
      // order: epos:northernmostLatitude , epos:easternmostLongitude, epos:southernmostLatitude, epos:westernmostLongitude
      // (north,east,south,west)
      // const value: string = '' + bbox[0] + ',' + bbox[1] + ',' + bbox[2] + ',' + bbox[3] + '';
      const [north, east, south, west] = bbox.asArray();
      const value = `POLYGON((${west} ${south}, ${east} ${south}, ${east} ${north}, ${west} ${north}, ${west} ${south}))`;
      builder.addParameter('spatialFilter', value);
    }

    // keywords
    const keywords = request.getKeywordIds();
    if (keywords != null && keywords.length > 0) {
      const value: string = keywords.join(',');
      builder.addParameter('keywords', value);
    }

    // organisations
    const organisations = request.getOrganisationIds();
    if (organisations != null && organisations.length > 0) {
      const value: string = organisations.join(',');
      builder.addParameter('provider', value);
    }

    // versioning Status ("Metadata Status" feature)
    const versioningStatus = request.getVersioningStatus();
    if(versioningStatus != null && versioningStatus.length > 0){
      const value: string = versioningStatus.map(status => status.toUpperCase()).join(',');
      builder.addParameter('versioningStatus', value);
    }


    // the search URL
    const url = builder.build();

    return this.fetchAndParse(url);
  }

  protected fetchAndParse(url: string): Promise<null | DiscoverResponse> {

    return this.rest.get(url)
      .then((json: Record<string, unknown>) => {

        const categoriesByGroup = new Map<string, GroupCategory[]>();
        const response = this.parseJSON(json, categoriesByGroup).orNull();
        if (Array.isArray(json)) {
          const groupIds = new Set(json
            .filter((item: Record<string, unknown>) => item.type === 'dataset'
              || item.type === 'software-application' || item.type === 'software-source-code')
            .map((item: Record<string, unknown>) => this.parseDatasetGroup(item).id));
          const categoryResults = new Map<string, Promise<FacetModel<DistributionSummary>>>();
          const categoryRequests = new Map<string, Promise<void>>();
          if (response !== null && groupIds.size > 0) {
            response.loadCategories = (selectedGroupId?: string) => {
              const scope = selectedGroupId ?? 'ALL';
              let results = categoryResults.get(scope);
              if (results === undefined) {
                const requestedGroupIds = selectedGroupId === undefined
                  ? Array.from(groupIds) : Array.from(groupIds).filter(groupId => groupId === selectedGroupId);
                results = Promise.all(requestedGroupIds.map(groupId => {
                  let categoryRequest = categoryRequests.get(groupId);
                  if (categoryRequest === undefined) {
                    const builder = this.baseUrl.urlBuilder();
                    builder.addPathElements('groups', groupId, 'categories');
                    categoryRequest = this.rest.get(builder.build()).then(categories => {
                      if (!Array.isArray(categories)) {
                        throw new Error(`Invalid categories response for group: ${groupId}`);
                      }
                      categoriesByGroup.set(groupId, categories.map((category: Record<string, unknown>) => ({
                        id: ObjectAccessUtility.getObjectValueString(category, 'id', true),
                        name: ObjectAccessUtility.getObjectValueString(category, 'name', true),
                        parentId: ObjectAccessUtility.getObjectValue<string>(category, 'parent_category_id'),
                      })));
                    }).catch(error => {
                      categoryRequests.delete(groupId);
                      throw error;
                    });
                    categoryRequests.set(groupId, categoryRequest);
                  }
                  return categoryRequest;
                })).then(() => {
                  const categoryResponse = this.parseJSON(json, categoriesByGroup).orNull();
                  if (categoryResponse === null) {
                    throw new Error('Unable to parse category results');
                  }
                  return categoryResponse.results();
                }).catch(error => {
                  categoryResults.delete(scope);
                  throw error;
                });
                categoryResults.set(scope, results);
              }
              return results;
            };
          }
        }

        return response;
      });
  }

  private addResourceFilters(builder: UrlBuilder, context: string): void {
    if (context === CONTEXT_RESOURCE || context === CONTEXT_FACILITY) {
      builder.addParameter('type', 'dataset');
      builder.addParameter('concept', context === CONTEXT_FACILITY ? 'FACILITIES' : 'DATA');
    } else if (context === CONTEXT_SOFTWARE) {
      builder.addParameter('type', 'software-application,software-source-code');
      builder.addParameter('concept', 'SOFTWARE');
    }
  }

  private orderDomainsByGroupPosition(domains: Domain[], groups: GroupMetadata[], context: string): Domain[] {
    const conceptsByContext: Record<string, GroupMetadata['concept']> = {
      [CONTEXT_RESOURCE]: 'DATA',
      [CONTEXT_FACILITY]: 'FACILITIES',
      [CONTEXT_SOFTWARE]: 'SOFTWARE',
    };
    const concept = conceptsByContext[context] ?? null;
    const positionsByGroupId = new Map<string, number>(
      groups
        .filter(group => concept === null || group.concept === concept)
        .map(group => [group.id, group.position] as const)
    );

    return domains
      .filter(domain => positionsByGroupId.has(domain.id ?? domain.code))
      .map((domain, index) => {
        const position = positionsByGroupId.get(domain.id ?? domain.code);
        return {
          domain,
          index,
          position,
        };
      })
      .sort((left, right) => {
        if (left.position === undefined && right.position !== undefined) {
          return 1;
        }
        if (right.position === undefined && left.position !== undefined) {
          return -1;
        }
        return (left.position ?? 0) - (right.position ?? 0) || left.index - right.index;
      })
      .map(({ domain }) => domain);
  }

  private parseJSONDomains(json: Record<string, unknown>): Optional<null | Array<Domain>> {

    const domainsAppendTo: Array<Domain> = [];

    if (json == null) {
      return Optional.empty();
    }

    if (Array.isArray(json)) {
      const domainsById = new Map<string, Domain>();
      json.forEach((item: Record<string, unknown>) => {
        const references = [{
          id: ObjectAccessUtility.getObjectValueString(item, 'id', true),
          name: ObjectAccessUtility.getObjectValueString(item, 'name', true),
        }];
        references.forEach(reference => {
          const { id, name } = reference;
          if (id && name && !domainsById.has(id)) {
            domainsById.set(id, {
              id, code: id, title: name, color: ObjectAccessUtility.getObjectValueString(item, 'color', false, '#fff'),
              imgUrl: ObjectAccessUtility.getObjectValueString(item, 'logo', false),
              linkUrl: ObjectAccessUtility.getObjectValueString(item, 'page', false),
            });
          }
        });
      });
      domainsAppendTo.push(...domainsById.values());
    } else {
      const resultsRoot = ObjectAccessUtility.getObjectValue<Record<string, unknown>>(json, 'results', true);
      if (resultsRoot != null) {

        const childrenJSONArray = ObjectAccessUtility.getObjectArray<Record<string, unknown>>(resultsRoot, 'children', false);

        childrenJSONArray.forEach(domainJson => {
          domainsAppendTo.push({
            code: ObjectAccessUtility.getObjectValueString(domainJson, 'code', false),
            color: ObjectAccessUtility.getObjectValueString(domainJson, 'color', false),
            id: ObjectAccessUtility.getObjectValueString(domainJson, 'id', false),
            imgUrl: ObjectAccessUtility.getObjectValueString(domainJson, 'imgUrl', false),
            linkUrl: ObjectAccessUtility.getObjectValueString(domainJson, 'linkUrl', false),
            title: ObjectAccessUtility.getObjectValueString(domainJson, 'name', false),
          });

        });
      }
    }

    return Optional.ofNullable(domainsAppendTo);
  }

  private parseJSON(json: Record<string, unknown>, categoriesByGroup: Map<string, GroupCategory[]>): Optional<null | DiscoverResponse> {

    if (json == null) {
      return Optional.empty();
    }

    // FILTERS
    const idToFilterFacets: Map<string, Facet<void>> = new Map();

    // filters as facets
    const filters: FacetModel<void> = new SimpleFacetModel<void>(idToFilterFacets);

    // ----------------------------------------------------------------------------------

    // RESULTS
    const idToDistributionFacets: Map<string, Facet<DistributionSummary>> = new Map();
    const datasets = Array.isArray(json) ? json
      .filter((item: Record<string, unknown>) => item.type === 'dataset')
      .map((item: Record<string, unknown>) => this.parseDataset(item)) : [];
    const softwareResources = Array.isArray(json) ? json
      .filter((item: Record<string, unknown>) => item.type === 'software-application' || item.type === 'software-source-code')
      .map((item: Record<string, unknown>) => {
        const resourceId = ObjectAccessUtility.getObjectValueString(item, 'resource_id', true);
        const versionId = ObjectAccessUtility.getObjectValueString(item, 'version_id', true);
        const name = ObjectAccessUtility.getObjectValueString(item, 'name', true);
        const summary = JSONDistributionFactory.jsonToDistributionSummary({
          id: resourceId, title: name, availableFormats: [], versioningStatus: item.status,
          changeDate: item.updated_at, editorFullName: (item.created_by as DatasetReference | undefined)?.name,
        }).get();
        if (summary === null) {
          throw new Error(`Invalid software summary: ${resourceId}`);
        }
        summary.resourceVersion = {
          resourceType: item.type as 'software-application' | 'software-source-code', resourceId, versionId,
        };
        return {
          resourceId, versionId, name, group: this.parseDatasetGroup(item),
          categories: ObjectAccessUtility.getObjectArray<DatasetReference>(item, 'categories', false) ?? [],
          distributions: [{ summary }],
        };
      }) : [];
    const resources = [...datasets, ...softwareResources];
    if (resources.length > 0) {
      const root = SimpleFacet.make<DistributionSummary>(null, 'discovery', 'Discovery');
      const groupsById = new Map<string, Facet<DistributionSummary>>();
      const categoriesById = new Map<string, Facet<DistributionSummary>>();
      resources.forEach(dataset => {
        let group = groupsById.get(dataset.group.id);
        if (group == null) {
          group = SimpleFacet.make(root, dataset.group.id, dataset.group.name);
          groupsById.set(dataset.group.id, group);
          root.addChild(group);
          const categoryGroup = group;
          const groupCategories = new Map((categoriesByGroup.get(dataset.group.id) ?? []).map(category => [category.id, category]));
          const createCategoryFacet = (category: GroupCategory): Facet<DistributionSummary> => {
            const categoryKey = `${dataset.group.id}:${category.id}`;
            const existingFacet = categoriesById.get(categoryKey);
            if (existingFacet != null) {
              return existingFacet;
            }
            let parent = categoryGroup;
            if (category.parentId !== null) {
              const parentCategory = groupCategories.get(category.parentId);
              if (parentCategory == null) {
                throw new Error(`Missing parent category: ${category.parentId}`);
              }
              parent = createCategoryFacet(parentCategory);
            }
            const facet = SimpleFacet.make<DistributionSummary>(parent, category.id, category.name);
            categoriesById.set(categoryKey, facet);
            parent.addChild(facet);
            return facet;
          };
          groupCategories.forEach(category => createCategoryFacet(category));
        }
        const groupFacet = group;
        const parents: Array<Facet<DistributionSummary>> = dataset.categories.map(category => {
          const categoryKey = `${dataset.group.id}:${category.id}`;
          let facet = categoriesById.get(categoryKey);
          if (facet == null) {
            facet = SimpleFacet.make(groupFacet, category.id, category.name);
            categoriesById.set(categoryKey, facet);
            groupFacet.addChild(facet);
          }
          return facet;
        });
        if (parents.length === 0) {
          parents.push(group);
        }
        parents.forEach(facet => {
          facet.addChild(SimpleFacet.make(facet, `${dataset.resourceId}:${dataset.versionId}`, dataset.name,
            dataset.distributions.map(distribution => distribution.summary)));
        });
      });
      idToDistributionFacets.set(root.getIdentifier(), root);
    } else if (Array.isArray(json)) {
      const distributions: DistributionSummary[] = [];
      json.forEach((item: Record<string, unknown>) => {
        const rawDistributions = ObjectAccessUtility.getObjectArray<Record<string, unknown>>(item, 'distributions', false) ?? [];
        rawDistributions.forEach(distribution => {
          const id = ObjectAccessUtility.getObjectValueString(distribution, 'id', false);
          const name = ObjectAccessUtility.getObjectValueString(distribution, 'name', false);
          if (id && name) {
            const summary = JSONDistributionFactory.jsonToDistributionSummary({ id, title: name, availableFormats: [] }).orNull();
            if (summary != null) {
              distributions.push(summary);
            }
          }
        });
      });
      if (distributions.length > 0) {
        const root = SimpleFacet.make<DistributionSummary>(null, 'discovery', 'Discovery', distributions);
        idToDistributionFacets.set(root.getIdentifier(), root);
      }
    }

    // results as facets
    const results: FacetModel<DistributionSummary> = new SimpleFacetModel<DistributionSummary>(idToDistributionFacets);

    // ----------------------------------------------------------------------------------

    // Anon DiscoverResponse
    const response: DiscoverResponse = {
      filters: (): FacetModel<void> => filters,
      results: (): FacetModel<DistributionSummary> => results
    };
    if (Array.isArray(json) && json.every((item: Record<string, unknown>) => item.type === 'dataset')) {
      response.datasets = () => datasets;
    }


    return Optional.ofNullable(response);
  }

  private parseDatasetGroup(item: Record<string, unknown>): DatasetReference {
    const group = ObjectAccessUtility.getObjectValue<DatasetReference>(item, 'group', true);
    if (group === null) {
      throw new Error('Dataset search result has no group');
    }
    return group;
  }

  private parseDataset(item: Record<string, unknown>): DatasetSummary {
    const rawDistributions = ObjectAccessUtility.getObjectArray<Record<string, unknown>>(item, 'distributions', false) ?? [];
    return {
      group: this.parseDatasetGroup(item),
      resourceId: ObjectAccessUtility.getObjectValueString(item, 'resource_id', true),
      versionId: ObjectAccessUtility.getObjectValueString(item, 'version_id', true),
      name: ObjectAccessUtility.getObjectValueString(item, 'name', true),
      status: ObjectAccessUtility.getObjectValueString(item, 'status', true),
      updatedAt: ObjectAccessUtility.getObjectValueString(item, 'updated_at', true),
      createdBy: ObjectAccessUtility.getObjectValue<DatasetReference>(item, 'created_by', false),
      providers: ObjectAccessUtility.getObjectArray<DatasetReference>(item, 'providers', false) ?? [],
      categories: ObjectAccessUtility.getObjectArray<DatasetReference>(item, 'categories', false) ?? [],
      distributions: rawDistributions.map(distribution => {
        const id = ObjectAccessUtility.getObjectValueString(distribution, 'id', true);
        const name = ObjectAccessUtility.getObjectValueString(distribution, 'name', true);
        const type = ObjectAccessUtility.getObjectValueString(distribution, 'type', true) as DatasetDistribution['type'];
        const visibleOn = ObjectAccessUtility.getObjectArray<'MAP' | 'TABLE' | 'GRAPH'>(distribution, 'visible_on', false);
        const summary = JSONDistributionFactory.jsonToDistributionSummary({
          id, title: name, availableFormats: [], versioningStatus: item.status,
          changeDate: item.updated_at, editorFullName: (item.created_by as DatasetReference | undefined)?.name,
        }).get();
        if (summary === null) {
          throw new Error(`Invalid distribution summary: ${id}`);
        }
        summary.resourceVersion = {
          resourceType: 'dataset',
          resourceId: ObjectAccessUtility.getObjectValueString(item, 'resource_id', true),
          versionId: ObjectAccessUtility.getObjectValueString(item, 'version_id', true),
        };
        summary.isMappable = visibleOn.includes('MAP');
        summary.isGraphable = visibleOn.includes('GRAPH');
        summary.isTabularable = visibleOn.includes('TABLE');
        summary.isDownloadable = type === 'DOWNLOADABLE_FILE';
        summary.isOnlyDownloadable = type === 'DOWNLOADABLE_FILE';
        return {
          id, name, type, summary,
          visibleOn: visibleOn,
          webServiceType: ObjectAccessUtility.getObjectValueString(distribution, 'web_service_type', false),
          provider: ObjectAccessUtility.getObjectValue<DatasetReference>(distribution, 'provider', false),
        };
      }),
    };
  }

  /**
   *  Recurse through the filters
   * @param parent
   * @param filter
   */
  private recurseThroughFiltersJSON(parent: null | Facet<void>, filterJSON: Record<string, unknown>): Optional<null | Facet<void>> {
    const facetName = ObjectAccessUtility.getObjectValueString(filterJSON, 'name', false);
    const facetID = ObjectAccessUtility.getObjectValueString(filterJSON, 'id', false, facetName);


    // If there is no facet name or id - abort
    if (facetName == null || facetID == null) {
      return Optional.empty();
    } else {

      // ... continue to create facet and children

      const facet: Facet<void> = SimpleFacet.make(parent, facetID, facetName);

      if (parent != null) {
        parent.addChild(facet);
      }

      // If there are child filters
      const childrenJSONArray = ObjectAccessUtility.getObjectArray<Record<string, unknown>>(filterJSON, 'children', false);
      if (childrenJSONArray != null) {
        childrenJSONArray.forEach(childJSON => {
          if (childJSON != null) {
            this.recurseThroughFiltersJSON(facet, childJSON);
          }
        });
      }


      return Optional.ofNonNullable(facet);
    }
  }

  private appendToByName(resultJSON, distributionsAppendTo: Array<DistributionSummary>, name: string): void {

    const distributionsJSONArray = ObjectAccessUtility.getObjectArray<Record<string, unknown>>(resultJSON as Record<string, unknown>, name, false);

    // If there is a distributions array iterate over it
    if (distributionsJSONArray != null) {
      distributionsJSONArray.forEach(distJSON => {
        if (distJSON != null) {
          const distSummary = JSONDistributionFactory.jsonToDistributionSummary(distJSON);
          if (distSummary != null) {
            distSummary.ifPresent((dist: DistributionSummary) => distributionsAppendTo.push(dist));
          }
        }
      });
    }
  }

  private recurseThroughResultsJSON(
    parent: null | Facet<DistributionSummary>,
    resultJSON: Record<string, unknown>,
  ): Optional<null | Facet<DistributionSummary>> {

    const facetName = ObjectAccessUtility.getObjectValueString(resultJSON, 'name', false);

    // if there is no id - use name as id
    const facetID = ObjectAccessUtility.getObjectValueString(resultJSON, 'id', false, facetName);

    const distributionsAppendTo: Array<DistributionSummary> = [];

    // DISTRIBUTIONS
    this.appendToByName(resultJSON, distributionsAppendTo, 'distributions');

    // FACILITIES
    this.appendToByName(resultJSON, distributionsAppendTo, 'facilities');

    // If there is no facet name or id - abort
    if (facetName == null || facetID == null) {
      return Optional.empty();
    } else {

      // ... continue to create facet and children

      // Create the facet - on success
      const facet: Facet<DistributionSummary> = SimpleFacet.make(parent, facetID, facetName, distributionsAppendTo);

      // Add facet to parent if there is a parent (roots have null parent)
      if (parent != null) {
        parent.addChild(facet);
      }

      // If there are child facets - recurse into them
      const childrenJSONArray = ObjectAccessUtility.getObjectArray<Record<string, unknown>>(resultJSON, 'children', false);
      if (childrenJSONArray != null) {
        childrenJSONArray.forEach(childJSON => {
          if (childJSON != null) {
            this.recurseThroughResultsJSON(facet, childJSON);
          }
        });
      }

      return Optional.ofNonNullable(facet);
    }
  }
}

