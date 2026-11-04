import { Injectable } from '@angular/core';
import { ApiService } from 'api/api.service';
import { CONTEXT_RESOURCE } from 'api/api.service.factory';
import { Domain } from 'api/webApi/data/domain.interface';
import { BaseLandingService } from 'pages/dataPortal/services/baseLanding.service';
import { LoggingService } from 'services/logging.service';
import { Model } from 'services/model/model.service';
import { LocalStoragePersister } from 'services/model/persisters/localStoragePersister';
import { LocalStorageVariables } from 'services/model/persisters/localStorageVariables.enum';

/**
 * Service that is used to manage landing page..
 */
@Injectable()
export class LandingService extends BaseLandingService {

  constructor(
    protected readonly model: Model,
    protected readonly localStoragePersister: LocalStoragePersister,
    protected readonly apiService: ApiService,
    protected readonly loggingService: LoggingService,
  ) {
    super();
  }

  /**
   * The function retrieves ordered domains from an API service, adds ALL and Favourites, and updates
   * the source of the domains.
   */
  public getDomains(): void {
    void this.loggingService.logForPromise(
      this.apiService.getDomains(CONTEXT_RESOURCE),
      'List of domains'
    ).then((r: null | Array<Domain>) => {

      if (r !== null) {
        // add ALL and favourites before the ordered TCS domains
        r.unshift({
          id: '0',
          title: 'All data and services',
          code: 'ALL',
          linkUrl: '',
          imgUrl: 'assets/img/see_all.png',
          domain: false,
          color: '#fff',
        }, {
          id: '100',
          title: 'Favourites',
          code: 'FAV',
          linkUrl: '',
          imgUrl: 'assets/img/favourites.png',
          domain: false,
          color: '#fff',
        });

        this.domainsSrc.next(r);
      }

    });
  }

  /**
   * The setDomain function sets the active domain based on the provided value and updates the landing
   * page accordingly.
   * @param {string | boolean} value - The `value` parameter can be either a string or a boolean.
   */
  public setDomain(value: string | boolean): void {
    if (typeof value === 'string') {
      const selectedDomain: Domain = { code: value, isSelected: true };
      this.model.domainMI.set(selectedDomain);
      this.setActiveDomain(value);
      this.localStoragePersister.set(
        LocalStorageVariables.LS_CONFIGURABLES,
        selectedDomain.code,
        false,
        LocalStorageVariables.LS_DOMAIN_OPEN
      );
    } else {
      this.localStoragePersister.set(
        LocalStorageVariables.LS_CONFIGURABLES,
        '',
        false,
        LocalStorageVariables.LS_DOMAIN_OPEN
      );
      this.setActiveDomain(false);
    }

    this.showLanding((value === false) ? true : false);
  }

  public setDomainDuringTour(value: string): void {
    if (value != null && value !== this.model.domainMI.get().code) {
      const selectedDomain: Domain = { code: value, isSelected: true };
      this.model.domainMI.set(selectedDomain);
    }
    this.showLanding((value === null) ? true : false);
  }

}
