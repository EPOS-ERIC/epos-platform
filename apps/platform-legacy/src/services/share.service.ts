/* eslint-disable @typescript-eslint/member-ordering */
/* eslint-disable @typescript-eslint/no-floating-promises */
import {  EventEmitter, Injectable } from '@angular/core';
import { LocalStoragePersister } from './model/persisters/localStoragePersister';
import { LocalStorageVariables } from './model/persisters/localStorageVariables.enum';
import { NotificationService } from './notification.service';
import { ApiService } from 'api/api.service';

/**
 * This service knows whether this is the live deployment
 * and may run scripts based on this info.
 */
@Injectable()
export class ShareService {
  private readonly triggerRemoveAllFavorites = new EventEmitter<void>();
  public triggerRemoveAllFavoritesObservable = this.triggerRemoveAllFavorites.asObservable();

  constructor(
    private readonly localStorage: LocalStoragePersister,
    private readonly notificationService: NotificationService,
    private readonly apiService: ApiService,
  ) { }


  /**
   * The function creates a data portal URL by retrieving data from local storage before
   * copying a citation to the clipboard.
   */
  public createDataPortalUrl(title?: string, description?: string): void {

    const url = new URL(window.location.href);

    // retrieve all localstorage info
    void this.localStorage.get(LocalStorageVariables.LS_DATA_SEARCH_CONFIGURABLES).then((dataSearchConfigurables: string) => {

      void this.localStorage.get(LocalStorageVariables.LS_DATA_SEARCH_CONFIGURABLES_REG).then((dataSearchConfigurablesReg: string) => {

        void this.localStorage.get(LocalStorageVariables.LS_CONFIGURABLES).then((configurables) => {

          void this.localStorage.get(LocalStorageVariables.LS_EXTERNAL_LAYERS).then((externalLayers: string) => {

          const shareConfiguration: ShareConfigurables = {
            dataSearchConfigurables: dataSearchConfigurables,
            dataSearchConfigurablesReg: dataSearchConfigurablesReg,
            configurables: JSON.stringify(configurables),
            externalLayers: externalLayers
          };

          // save on DB
          void this.apiService.saveConfigurables(JSON.stringify(shareConfiguration), title, description).then(key => {
            url.searchParams.set('share', key);
            this.copyCitationToClipboard(url.toString());
          }).catch((_) => {
            // Show an error notification
            this.notificationService.sendNotification('Failed to copy EPOS Platform URL to clipboard', 'x', NotificationService.TYPE_ERROR, 5000);
          });

          });

        });

      });

    });

  }

  public triggerRemoveFavorites(): void {
    this.triggerRemoveAllFavorites.next();
  }

  private copyCitationToClipboard(data: string): void {

    navigator.clipboard.writeText(data).then(() => {
      // Show a success notification
      this.notificationService.sendNotification('EPOS Platform URL copied to clipboard', 'x', NotificationService.TYPE_SUCCESS, 5000);
    }).catch((_) => {
      // Show an error notification
      this.notificationService.sendNotification('Failed to copy EPOS Platform URL to clipboard', 'x', NotificationService.TYPE_ERROR, 5000);
    });
  }
}

export interface ShareConfigurables {
  dataSearchConfigurables: string;
  dataSearchConfigurablesReg: string;
  configurables: string;
  externalLayers: string;
}
