import { AfterContentInit, Component, EventEmitter, HostListener, OnInit, Output, ViewChild } from '@angular/core';
import { MatPaginator } from '@angular/material/paginator';
import { MatTableDataSource } from '@angular/material/table';
import { DiscoverResponse } from 'api/webApi/classes/discoverApi.interface';
import { DistributionSummary } from 'api/webApi/data/distributionSummary.interface';
import { FacetModel } from 'api/webApi/data/facetModel.interface';
import { Domain } from 'api/webApi/data/domain.interface';
import { BehaviorSubject, of, Subscription , from } from 'rxjs';
import { DialogService } from 'components/dialog/dialog.service';
import { LoadingService } from 'services/loading.service';
import { DataConfigurableI } from 'utility/configurables/dataConfigurableI.interface';
import { DataConfigurable } from 'utility/configurables/dataConfigurable.abstract';
import { PanelsEmitterService } from 'services/panelsEventEmitter.service';
import { DistributionLevel } from 'api/webApi/data/distributionLevel.interface';
import { ResultsPanelService } from 'pages/dataPortal/services/resultsPanel.service';
import { DataConfigurableDataSearchI } from 'utility/configurablesDataSearch/dataConfigurableDataSearchI.interface';
import { DataConfigurableDataSearchLoading } from 'utility/configurablesDataSearch/dataConfigurableDataSearchLoading';
import { DataConfigurableDataSearch } from 'utility/configurablesDataSearch/dataConfigurableDataSearch';
import { LocalStoragePersister } from 'services/model/persisters/localStoragePersister';
import { NotificationMessage, NotificationService } from 'services/notification.service';
import { DistributionNotificationText } from 'pages/dataPortal/enums/distributionNotification.enum';
import { Unsubscriber } from 'decorators/unsubscriber.decorator';
import { LocalStorageVariables } from 'services/model/persisters/localStorageVariables.enum';
import { DataConfigurationType } from 'utility/configurables/dataConfigurationType.enum';
import { FacetFlatNode } from 'components/facetDropdown/facetDropdown.component';
import { DistributionItem } from 'api/webApi/data/distributionItem.interface';
import { DataSearchConfigurablesService } from 'pages/dataPortal/services/dataSearchConfigurables.service';
import { DomainMI } from 'services/model/modelItems/domainMI';
import { BaseLandingService } from 'pages/dataPortal/services/baseLanding.service';
import { TypeDataMI } from 'services/model/modelItems/typeDataMI';
import { SearchService } from '../../services/search.service';
import { LeafletLoadingService } from '../../utility/eposLeaflet/services/leafletLoading.service';
import { NotificationSnackComponent } from '../notificationSnack/notificationSnack.component';
import { MatSnackBarRef } from '@angular/material/snack-bar';
import { environment } from 'environments/environment';
import { MetaDataStatusService } from 'services/metaDataStatus.service';
import { Model } from 'services/model/model.service';
import { AaaiService } from 'api/aaai.service';
import { AAAIUser } from 'api/aaai/aaaiUser.interface';
import { CONTEXT_FACILITY, CONTEXT_RESOURCE } from 'api/api.service.factory';
import { DatasetSummary } from 'api/webApi/data/datasetSummary.interface';
import { ViewType } from 'api/webApi/data/viewType.enum';

@Unsubscriber(['domainSubscription', 'subscriptions'])
@Component({
  selector: 'app-base-results-panel',
  template: '',
  styleUrls: ['./baseResultsPanel.component.scss'],
})
export class BaseResultsPanelComponent implements OnInit, AfterContentInit {
  @ViewChild(MatPaginator, { static: true }) paginator: MatPaginator;
  @Output() basicItemList = new EventEmitter();
  @Output() domainResultsCounterEmitter: EventEmitter<Array<number>> = new EventEmitter<Array<number>>();

  /** The above code is declaring a public variable named "selectedConfigurableSource" of type
   BehaviorSubject. This variable can hold either a null value or an object that implements the
   DataConfigurableI interface. The BehaviorSubject is a type of Observable that emits the most recent
   value to its subscribers and also has a method to retrieve the current value. */
  public selectedConfigurableSource = new BehaviorSubject<null | DataConfigurableI>(null);
  public parameterDialogFormType = new BehaviorSubject<DataConfigurationType>(DataConfigurationType.DATA);

  public dataSource: MatTableDataSource<DistributionItem>;
  public columnsToDisplay = ['name'];
  public expandedElement: DistributionItem | null;

  public rawTreeData: FacetModel<DistributionSummary> | null;
  public totalFacetsSelected = 0;

  public activeDomainData: Domain = { id: '0', code: 'ALL' };
  public domainResultsCounter: Array<number> = [];
  public showLoading = false;

  public facetExpansionHidden = true;

  public messageId: string;
  public messageText: string;
  public messageShow = false;
  public messageTitle = 'Service data informations';
  public messageType = NotificationService.TYPE_INFO;
  public messageCheckShowAgain = false;


  public hasAllowedValues: boolean = true;

  public metadataStatusModeActive: boolean = false;

  public metadataSelectedStatuses: Array<string> = [];

  public showStatusChipsOnCards: boolean = false;

  protected timeout: NodeJS.Timeout;
  protected favTimeout: NodeJS.Timeout;

  protected readonly subscriptions: Array<Subscription> = new Array<Subscription>();
  protected readonly domainSubscription: Array<Subscription> = new Array<Subscription>();

  protected data: Array<DistributionItem>;

  protected domains: Array<Domain>;

  protected context: string;

  private filteredFacetData: Array<DistributionItem> = [];
  private activeDomain: Domain;
  private dataConfigurablesArraySource = new BehaviorSubject<null | Array<DataConfigurableI>>([]);

  private lastGoodSelectedConfigurableId: string;
  private wasAuthenticatedUser = false;
  private datasets: DatasetSummary[] | null = null;
  private selectedDatasetFacets = new Set<string>();
  private discoverResponse: DiscoverResponse | null = null;
  private readonly providerOrganizationUrls = new Map<string, Promise<string>>();

  constructor(
    public readonly configurables: DataSearchConfigurablesService,
    protected readonly dialogService: DialogService,
    protected readonly loadingService: LoadingService,
    protected readonly landingService: BaseLandingService,
    protected readonly panelsEvent: PanelsEmitterService,
    protected readonly resultPanelService: ResultsPanelService,
    protected readonly localStoragePersister: LocalStoragePersister,
    protected readonly notification: NotificationService,
    protected readonly searchService: SearchService,
    protected readonly leafletLoadingService: LeafletLoadingService,
    protected readonly metadataStatusService: MetaDataStatusService,
    protected readonly model: Model,
    protected readonly aaaiService: AaaiService
  ) {
    this.expandedElement = null;
    this.wasAuthenticatedUser = this.aaaiService.getUser() !== null;

  }

  public get hasVersionedResources(): boolean {
    return this.datasets !== null || this.data?.some(item => item.distSummary.resourceVersion != null) === true;
  }

  public get resultCount(): number {
    return this.datasets !== null ? this.data.filter(item => !item.hideToResult).length : this.data.length;
  }

  public get displayedResultCount(): number {
    return this.dataSource?.filteredData.length ?? 0;
  }

  @HostListener('window:beforeunload', ['$event'])
  onBeforeUnload(): void {
    this.configurables.setSelected(null, true);
  }

  public onChangeDomain(selectedDomain: Domain): void {
    this.resultPanelService.clearFacetSelection();
    this.activeDomain = selectedDomain;
    this.selectedDatasetFacets.clear();
    this.refresh(this.activeDomain);
    this.activeDomainData = this.configurables.getDomainInfoBy(this.domains, this.activeDomain.code, 'code');
    this.totalFacetsSelected = 0;
    if (!this.facetExpansionHidden) {
      this.loadFacetCategories();
    }
    setTimeout(() => {
      this.panelsEvent.dataPanelToggle();
    }, 100);
  }

  /**
   * The function `onChangeResponse` processes a discover response, updates data display based on
   * filters, emits events, and performs other related tasks.
   * @param {DiscoverResponse} discoverResponse - The `discoverResponse` parameter is of type
   * `DiscoverResponse`, which likely contains the results of a discovery operation.
   * @param {DomainMI} domainMI - The `domainMI` parameter in the `onChangeResponse` method is of type
   * `DomainMI`. It seems to be used to get the active domain for further processing within the method.
   * @param {TypeDataMI | null} filterType - The `filterType` parameter in the `onChangeResponse`
   * method is of type `TypeDataMI | null`. This means it can either hold a value of type `TypeDataMI`
   * or `null`. If it holds a value of `TypeDataMI`, it will be used to filter the
   */
  public onChangeResponse(discoverResponse: DiscoverResponse, domainMI: DomainMI, filterType: TypeDataMI | null): void {
    this.discoverResponse = discoverResponse;
    let results: null | FacetModel<DistributionSummary> = null;
    if (discoverResponse != null) {
      results = discoverResponse.results();
    }

    this.activeDomain = domainMI.get();

    const filterTypeArray = filterType !== null ? filterType.get() ?? [] : [];

    this.datasets = this.context === CONTEXT_RESOURCE || this.context === CONTEXT_FACILITY ? discoverResponse?.datasets?.() ?? null : null;
    if (this.datasets !== null) {
      const groupIds = new Set(this.domains.map(domain => domain.code));
      this.datasets.forEach(dataset => {
        if (!groupIds.has(dataset.group.id)) {
          this.domains.push({ id: dataset.group.id, code: dataset.group.id, title: dataset.group.name, color: '#fff' });
          groupIds.add(dataset.group.id);
        }
      });
      const groupLogos = new Map<string, string>([
        ['Seismology', 'SEISM_logo.png'],
        ['Near Fault Observatories', 'NFO_logo.png'],
        ['GNSS Data and Products', 'GNSS_logo.png'],
        ['Volcano Observations', 'VOLCANO_logo.png'],
        ['Satellite Data', 'SATELLITE_logo.png'],
        ['Geomagnetic Observations', 'GEOMAG_logo.png'],
        ['Anthropogenic Hazards', 'ANTHROPO_logo.png'],
        ['Geological Information and Modeling', 'MODELING_logo.png'],
        ['Multi-scale Laboratories', 'LABS_logo.png'],
        ['Laboratories', 'LABS_logo.png'],
        ['Tsunami', 'TSU_logo.png'],
      ]);
      this.domains.forEach(domain => {
        const logo = groupLogos.get(domain.title ?? '');
        if (logo !== undefined && !domain.imgUrl) {
          domain.imgUrl = `assets/img/logo/${logo}`;
        }
      });
    }
    this.activeDomainData = this.configurables.getDomainInfoBy(this.domains, this.activeDomain.code, 'code');
    this.createItemsDisplay(results, filterTypeArray as Array<string>);
    this.selectedDatasetFacets.clear();
    this.rawTreeData = results;
    if (!this.facetExpansionHidden) {
      this.loadFacetCategories();
    }
    this.data.filter(item => !item.hideToResult && item.distSummary.resourceVersion != null).forEach(item => {
      item.levels = item.levels.map(levels => levels.slice(0, -1));
      const resourceVersion = item.distSummary.resourceVersion!;
      const dataset = this.datasets?.find(candidate => candidate.resourceId === resourceVersion.resourceId
        && candidate.versionId === resourceVersion.versionId);
      item.cardProviderName = dataset?.distributions.find(distribution => distribution.id === item.distId)?.provider?.name;
    });

    this.refresh(this.activeDomain);
    this.totalFacetsSelected = 0;

    this.basicItemList.emit(this.data);

    if (this.domains !== null && this.domains !== undefined) {
      this.domains.forEach(element => {
        if (element.code === 'ALL') {
          this.domainResultsCounter[element.code] = this.data?.filter(o => o.hideToResult === false).length;
        } else if (element.code === 'FAV') {
          this.domainResultsCounter[element.code] = this.configurables.getAllPinned().length;
        } else {
          this.domainResultsCounter[element.code] = this.data?.filter(o => o.code === element.code && o.hideToResult === false).length;
        }
      });
      this.domainResultsCounterEmitter.emit(this.domainResultsCounter);
    }

    // reset pagination after each search
    this.resetPagination();

    this.panelsEvent.dataPanelToggle();

    setTimeout(() => {
      // wait a few seconds to remove the loading
      this.loadingService.showLoading(false);
    }, 500);
  }

  /**
   * The `ngOnInit` function initializes the component by subscribing to various observables and
   * performing some initial setup tasks.
   */
  ngOnInit(): void {
    this.domainSubscription.push(

      // waiting list of domains
      this.landingService.domainObs.subscribe((domains) => {

        if (domains !== null) {
          this.domains = domains;

          this.subscriptions.push(

            this.landingService.showLandingObs.subscribe((landing: boolean) => {

              // if in landing page remove selected item
              if (landing) {
                this.configurables.setSelected(null, true);
              }
            }),

            this.configurables.watchAll().subscribe(() => {
              this.updateConfigs();
            }),

            // setting variable to hold MetadataPreview Mode status
            this.model.metadataPreviewMode.valueObs.subscribe((active)=>{
              // both dataPanel and registryPanel check this variable from their template
              if(active != null){
                this.metadataStatusModeActive = active;
              }
            }),
            // establish if to show chips or not (when metadata Feature on and 0 statuses selected don't show, when feature on and selected statuses > 0 show)
            this.model.metadataPreviewModeStatuses.valueObs.subscribe((statuses)=>{
              if(this.metadataStatusModeActive === true && statuses != null){
                if(statuses.length > 0){
                  this.showStatusChipsOnCards = true;
                }
                else{
                  this.showStatusChipsOnCards = false;
                }
              }
            }),
            // when user gets logged out (automatically, timer from last interaction has expired) and metadataStatusModeActive variable still true
            this.aaaiService.watchUser().subscribe((user: null | AAAIUser)=>{
              const isAuthenticated = user !== null;
              const isRealLogout = this.wasAuthenticatedUser && !isAuthenticated;

              if(isRealLogout && this.metadataStatusModeActive === true){
                this.metadataStatusModeActive = false;
                this.showStatusChipsOnCards = false;
                this.metadataStatusService.metadataStatusModeActive.next(false);
                this.metadataStatusService.metadataSelectedStatuses.next([]);
              }

              this.wasAuthenticatedUser = isAuthenticated;
            }),

            // notification by data service
            this.notification.distributionNotificationObs.subscribe((message: NotificationMessage) => {
              if (message !== null) {
                this.formatNotificationMessage(message);
              }
            }),

            this.dataConfigurablesArraySource.subscribe((dataConfigurables: Array<DataConfigurable>) => {
              this.updateSelectedConfig();
            }),

            this.selectedConfigurableSource.subscribe((selectedConfigurable: null | DataConfigurableI) => {
              this.hasAllowedValues = selectedConfigurable?.getParameterDefinitions().getOtherParameters()?.some((param) => {
                return param.hasAllowedValues;
              }) ?? false;
            }),

            this.panelsEvent.invokeSelectItem.subscribe((id: string) => {
              const item = this.data.find((i: DistributionItem) => { return i.id === id; });
              if (item !== null && item !== undefined) {
                this.select(item);
              }
            }),

            this.resultPanelService.openFacetSelectionObs.subscribe((open: boolean) => {
              if (open) {
                this.facetExpansionHidden = false;
                this.loadFacetCategories();
              }
            }),
          );
        }
      }),
    );
  }

  ngAfterContentInit(): void {
    setTimeout(() => {
      this.subscriptions.push(
        this.loadingService.showLoadingObs.subscribe((showLoading: boolean) => {
          this.showLoading = showLoading;
        })
      );
    });

    setTimeout(() => {
      this.subscriptions.push(
        this.landingService.showLandingObs.subscribe((landing: boolean) => {
          // if in landing page remove selected item
          if (landing) {
            this.configurables.setSelected(null, true);
          }
        }),
      );
    }, 1000);
  }

  /**
   * When the user clicks on an item, the item is selected
   * @param {DistributionItem} expandedElement - The element that was expanded.
   */
  public select(expandedElement: DistributionItem, event: Event | null = null): void {

    if (event !== null) {
      // reset the version (if it was set by sharing the URL)
      this.resetVersion();
    }

    /* IF (contains): dialog show up conditions */

    // If (there's an element expanded and new element is not the current one) AND (this current element is not pinned and new element is pinned OR this current element is not pinned and new is not pinned)
    if(((this.expandedElement !== null) && !this.isDistributionExpanded(expandedElement)) && ((!this.configurables.isPinned(this.expandedElement!.id) && this.configurables.isPinned(expandedElement!.id)) || (!this.configurables.isPinned(this.expandedElement!.id) && !this.configurables.isPinned(expandedElement!.id)))){
      // If dialogcheck == null (first time) OR dialogcheck == false, show dialog
      if(this.localStoragePersister.getValue(LocalStorageVariables.LS_SWITCH_DISTRIBUTION_ITEM_CHECK) === null || this.localStoragePersister.getValue(LocalStorageVariables.LS_SWITCH_DISTRIBUTION_ITEM_CHECK as string) === 'false'){
        this.openSwitchItemConfirmationDialog(expandedElement);
      }
      else{
        void this.toggleExpandItem(expandedElement);
      }
    }
    // If nothing is expanded, expand the element; OR if it is the same element, close it; OR If previous is pinned and new element is pinned; OR If previous is pinned and new element is not pinned; OR If previous is pinned and new element is not pinned
    else{
      void this.toggleExpandItem(expandedElement);
    }
  }

  /* The 'toggleExpandItem' function executes the actual instructions for the select(), which performs some controls, based on whom, will either call this function or not.  */
  public toggleExpandItem(expandedElement: DistributionItem): Promise<void>{

    this.expandedElement = this.isDistributionExpanded(expandedElement) ? null : expandedElement;

    this.messageShow = false;

    if (this.expandedElement !== null) {
      if (this.expandedElement.distSummary.resourceVersion != null && !this.expandedElement.metadataLoaded) {
        void this.loadCardMetadata(this.expandedElement).catch(error => console.error('Unable to load resource card metadata', error));
      }
      if(!this.configurables.isPinned(this.expandedElement.distId)){
        // Send the notification
        this.loadingNotification(this.expandedElement!.distId);
      }

      // eslint-disable-next-line max-len
      this.configurables.setSelected(this.expandedElement!.distId, true);

      // add levels information
      this.addLevelToElement(expandedElement);

      // refine data
      void this.localStoragePersister.get(LocalStorageVariables.LS_TOUR_ACTIVE).then((active: string) => {
        const tourActive = active === 'true' ? true : false;
        void this.localStoragePersister.get(DistributionNotificationText.LS_DISTRIBUTION_MESSAGE_ID).then((val) => {

          let messageShow = (val || val === null) ? true : false;

          setTimeout(() => {
            const distributionSelected = this.configurables.getSelected();

            if (distributionSelected !== null && distributionSelected instanceof DataConfigurableDataSearch) {
              // if only downloadable => no message
              const distributionDetails = distributionSelected?.getDistributionDetails();
              // eslint-disable-next-line max-len
              if (distributionDetails!.isOnlyDownloadable) {
                messageShow = false;
              }

              if (messageShow && !this.configurables.isPinned(expandedElement!.id) && !tourActive ) {
                this.notification.sendDistributionNotification({
                  id: DistributionNotificationText.LS_DISTRIBUTION_MESSAGE_ID,
                  title: 'Information',
                  message: DistributionNotificationText.MESSAGE_REFINE_DATA,
                  type: NotificationService.TYPE_INFO,
                  showAgain: true
                });

              }
            }
          }, 1000);

        });
      });
    }else{
      this.configurables.setSelected(null, true);
    }
    return Promise.resolve();
  }

  /** The 'openSwitchItemConfirmationDialog' function opens up the confirmation dialog
   *  when switching between distribution items (under certain conditions established in the select()) */
  public openSwitchItemConfirmationDialog(expandedElement: DistributionItem){

    const expandedElementName = this.configurables.getSelected()!.name;


    void this.dialogService.openSwitchItemConfirmationDialog({expandedElementName: expandedElementName}).then((confirmed)=>{

      if(confirmed !== true){
        void this.toggleExpandItem(expandedElement);
        this.dialogService.setSwitchItemDialogConfirmation(confirmed);
        return;
      }else{
        void this.awaitFavouriteThenToggleExpandItem(expandedElement);
      }
    });
  }

  /** This function awaits that the currently expanded item is set to favourite,
   *  then switches to the newly clicked item.
   */
  public async awaitFavouriteThenToggleExpandItem(expandedElement: DistributionItem){
    await this.setExpandedToFavourite();
    await this.toggleExpandItem(expandedElement);
    this.dialogService.setSwitchItemDialogConfirmation(true);
  }

  public async setExpandedToFavourite(): Promise<void> {
    return new Promise((resolve) => {
      this.favourite(this.expandedElement as DistributionItem);
      resolve();
    });
  }


  /**
   * The `favourite` function toggles the pinned status of a distribution item and performs additional
   * actions based on the active domain.
   * @param {DistributionItem} element - The `element` parameter in the `favourite` function represents
   * an item that is being favorited or pinned. It is of type `DistributionItem`, which likely contains
   * information about the item being favorited, such as its ID, name, and other relevant data.
   * @param {Event | null} [event=null] - The `event` parameter in the `favourite` function is optional
   * and can accept an `Event` object or `null` as its value. If an `Event` object is provided, the
   * function will reset the version. If `null` is passed, this step will be skipped.
   */
  public favourite(element: DistributionItem, event: Event | null = null): void {

    if (event !== null) {
      // reset the version (if it was set by sharing the URL)
      this.resetVersion();
    }

    void this.configurables.togglePinned(element.distId, true).then(pinned => {
      element.isPinned = pinned;
      // If the element is expanded, no loading notification
      if (pinned && this.configurables.isSelected(element.distId)) {
        // add levels information
        this.addLevelToElement(element);
      }
      // If the element is not expanded, loading notification
      else if (pinned && !this.configurables.isSelected(element.distId)) {
        this.addLevelToElement(element);
        this.loadingNotification(element.distId);
      }

      if (this.activeDomain.code === 'FAV') {
        // only in the FAV sections: if element is selected => unselected
        if (this.configurables.isSelected(element.distId)) {
          this.configurables.setSelected(null, true);
        }

        if (this.configurables.getAllPinned().length === 0) {
          this.landingService.showLanding(true);
        }

        this.refresh(this.activeDomain);
      }

      // eslint-disable-next-line @typescript-eslint/dot-notation
      this.domainResultsCounter['FAV'] = this.configurables.getAllPinned().length;

    });
  }


  /**
   * The function `openDialog` opens a details dialog for a given element.
   * @param {DistributionItem} element - The parameter `element` is of type `DistributionItem`.
   */
  public openDialog(element: DistributionItem): void {
    const resourceVersion = element.distSummary.resourceVersion;
    void this.dialogService.openDetailsDialog(
      element.id,
      this.context,
      this.domains,
      '50vw',
      resourceVersion != null ? {
        ...resourceVersion,
        summary: element.distSummary,
      } : undefined,
    );
  }

  public getConfigByIndex(index: number): DataConfigurableI | null {
    const configurables = this.dataConfigurablesArraySource.getValue();
    let selectedConfigurable: null | DataConfigurableI = null;
    if (configurables != null) {
      if ((index >= 0) && (index < configurables.length)) {
        selectedConfigurable = configurables[index];
        this.lastGoodSelectedConfigurableId = selectedConfigurable!.id;
      }
    }
    return selectedConfigurable;
  }

  public selectConfigIndex(newIndex: number): void {
    const selectedConfigurable: null | DataConfigurableI = this.getConfigByIndex(newIndex);

    // we're using null to trigger "loading"
    this.selectedConfigurableSource.next(selectedConfigurable);
  }

  /**
   * The function `toggleDomain` sets the domain value based on the input value and a check for closing.
   * @param {string | boolean | undefined} value - The value parameter can be of type string, boolean, or
   * undefined. It represents the value that will be passed to the setDomain method of the
   * landingService.
   * @param [checkForClose=true] - The checkForClose parameter is a boolean value that determines whether
   * to check for the close condition. If it is set to true, the code will check if the value is
   * undefined, false, or equal to the active domain code before setting the domain. If it is set to
   * false, the code will
   */
  public toggleDomain(value: string | boolean | undefined, checkForClose = true): void {

    if (checkForClose === false) {
      this.landingService.setDomain((value === undefined || value === false) ? false : value);
    } else {
      this.landingService.setDomain((value === undefined || value === false || value === this.activeDomain.code) ? false : value);
    }
  }

  /**
   * Called on press of the displayed "Clear" button and triggers the display of a confirmation
   * to remove all of the items that are pinned.
   */
  public removeAllFavourites(autoConfirm: boolean = false): void {
    const confirmation$ = autoConfirm
      ? of(true)  // Automatically confirm when triggered via observable
      : from(this.dialogService.openConfirmationDialog(
          `Do you want to clear all (${this.configurables.getAllPinned().length}) favourite items?`,
          false,
          'Ok',
          'primary'
        ));  // Convert Promise to Observable
    confirmation$.subscribe((confirm: boolean) => {
      if (confirm) {
        this.data.filter(item => item.isPinned).forEach((item: DistributionItem) => {
          item.isPinned = false;
          this.favourite(item);
        });
      }
    });
  }

  /**
   * The function `onPaginateChange` is used to handle pagination changes by resetting the selected item,
   * showing a loading spinner, scrolling to the top of the results container, and hiding the loading
   * spinner after a delay.
   */
  public onPaginateChange(): void {
    this.configurables.setSelected(null, true);
    this.showLoading = true;
    setTimeout(() => {
      const scrollElem = document.querySelector('.results-container.nice-scrollbar');
      if (scrollElem) {
        scrollElem.scrollTo({ top: 0 });
      }
      this.showLoading = false;
    }, 300);
  }

  /**
   * It resets the paginator to the first page
   */
  public resetPagination(): void {
    if (this.paginator !== undefined) {
      this.paginator.firstPage();
    }
  }

  public toggleFacetExpansionPanel(): void {
    this.facetExpansionHidden = !this.facetExpansionHidden;
    if (!this.facetExpansionHidden) {
      this.loadFacetCategories();
    }
  }

  public setSelectedFacets(data: Array<FacetFlatNode>): void {
    if (this.hasVersionedResources) {
      this.selectedDatasetFacets = new Set(data.map(node => node.distId));
      this.refresh(this.activeDomain);
      this.resetPagination();
      return;
    }
    const facetSet = new Set<DistributionItem>();
    const domainFilteredData = this.data.filter(o => o.code === this.activeDomain.code);
    domainFilteredData.filter((item: DistributionItem) => {
      return data.some((node: FacetFlatNode) => {
        return item.levels.forEach((levels: DistributionLevel[]) => {
          return levels.forEach((level: DistributionLevel, key, mylevels) => {
            if (level.distId === node.distId) {
              if (node.name === '') {
                if (Object.is(mylevels.length - 1, key)) {
                  facetSet.add(item);
                }
              } else {
                facetSet.add(item);
              }
            }
          });
        });
      });
    });

    clearTimeout(this.timeout);
    this.timeout = setTimeout(() => {
      this.filteredFacetData = Array.from(facetSet);
      this.refresh(this.activeDomain);
    }, 10);

  }

  public showMessage(showMessage: boolean): void {
    this.messageShow = showMessage;
  }

  public isDistributionExpanded(item: DistributionItem): boolean {
    return this.expandedElement !== null && this.expandedElement.id === item.id && this.expandedElement.datasetKey === item.datasetKey;
  }


  /**
   * The function `resetVersion` checks a value in local storage and updates it if necessary.
   */
  protected resetVersion(): void {
    void this.localStoragePersister.get(LocalStorageVariables.LS_CONF_FROM_SHARE).then((val: boolean) => {

      if (val === true) {
        this.localStoragePersister.set(LocalStorageVariables.LS_CONF_FROM_SHARE, false);
        // set new version on localstorage
        this.localStoragePersister.set(LocalStorageVariables.LS_CONFIGURABLES, environment.version, false, LocalStorageVariables.LS_VERSION);
      }
    });
  }


  /**
   * Update the configurables array source with the selected item, or an empty array if nothing is
   * selected
   */
  protected updateConfigs(): void {
    const allConfigurables = this.configurables.getAll().slice();
    if (this.datasets !== null) {
      this.data.forEach(item => { item.isPinned = this.configurables.isPinned(item.distId); });
    }

    this.ensureReloadFuncSet(allConfigurables);

    const selectedItemConfigurable = this.configurables.getSelected();

    let configurablesData: null | Array<DataConfigurableDataSearchI>;
    if (selectedItemConfigurable == null) {
      // nothing selected = empty
      configurablesData = [];
      this.expandedElement = null;
    } else if (selectedItemConfigurable instanceof DataConfigurableDataSearchLoading) {
      // loading state
      configurablesData = null;
    } else {
      // display selected item
      configurablesData = [selectedItemConfigurable];
    }
    this.dataConfigurablesArraySource.next(configurablesData);

  }

  private loadFacetCategories(): void {
    const response = this.discoverResponse;
    const domainCode = this.activeDomain.code;
    if (response?.loadCategories == null || domainCode === 'FAV') {
      return;
    }
    void response.loadCategories(domainCode === 'ALL' ? undefined : domainCode).then(results => {
      if (this.discoverResponse === response && this.activeDomain.code === domainCode && this.rawTreeData !== results) {
        this.rawTreeData = results;
      }
    }).catch(error => {
      if (this.discoverResponse === response && this.activeDomain.code === domainCode) {
        console.error('Unable to load categories', error);
        this.notification.sendErrorNotification('Close and reopen the filter to retry.', 'Unable to load categories');
      }
    });
  }

  private async loadCardMetadata(item: DistributionItem): Promise<void> {
    const resourceVersion = item.distSummary.resourceVersion!;
    const details = await this.searchService.getResourceVersionDistributionDetails(
      resourceVersion.resourceType, resourceVersion.resourceId, resourceVersion.versionId, item.distId, item.distSummary,
    );
    item.levels = details.getCategories()?.children.flatMap(group => {
      const root: DistributionLevel = { id: 0, value: group.name, distId: group.code, level: 0 };
      return group.children.length === 0 ? [[root]] : group.children.map(category => [
        root, { id: 1, value: category.name, distId: category.code, level: 1, parentName: group.name },
      ]);
    }) ?? [];
    const provider = details.getWebServiceProvider();
    item.cardProviderName = provider?.dataProviderLegalName ?? '';
    item.cardProviderUrl = provider?.dataProviderUrl ?? '';
    item.cardCreators = details.getCreator().map(creator => ({ ...creator }));
    item.visibility = [
      !(details.isMappable || details.isTabularable || details.isGraphable) ? 'Not viewable' : '',
      details.isMappable ? ViewType.MAP : '',
      details.isTabularable ? ViewType.TABLE : '',
      details.isGraphable ? ViewType.GRAPH : '',
    ];
    item.metadataLoaded = true;
    if (!item.cardProviderUrl && provider?.dataProviderId) {
      item.cardProviderUrl = await this.getProviderOrganizationUrl(provider.dataProviderId);
    }
    await Promise.all(item.cardCreators.map(async creator => {
      if (!creator.url && creator.uid) {
        creator.url = await this.getProviderOrganizationUrl(creator.uid);
      }
    }));
  }

  private getProviderOrganizationUrl(providerId: string): Promise<string> {
    let url = this.providerOrganizationUrls.get(providerId);
    if (url === undefined) {
      url = this.searchService.getOrganizationById(providerId).then(organization => organization?.getUrl() ?? '').catch(error => {
        this.providerOrganizationUrls.delete(providerId);
        throw error;
      });
      this.providerOrganizationUrls.set(providerId, url);
    }
    return url;
  }

  private refresh(domain: Domain): void {
    if (null != this.data) {
      let basicItemList = this.datasets !== null ? this.data.filter(item => !item.hideToResult) : this.data;
      if (domain.code === 'FAV') {

        basicItemList = [];

        this.configurables.getAllPinned().forEach(element => {
          basicItemList.push(this.data.filter(o => o.distId === element.id)[0]);
        });

        // sort favourite items
        basicItemList.sort((a, b) => {
          return this.configurables.sortData(a, b);
        });

      } else if (domain.code !== 'ALL') {
        if (this.filteredFacetData.length !== 0 && this.filteredFacetData[0].code === domain.code) {
          basicItemList = this.filteredFacetData;
        } else {
          basicItemList = basicItemList.filter((item: DistributionItem) => item.code === domain.code);
          this.filteredFacetData = [];
        }
      }

      this.totalFacetsSelected = this.filteredFacetData.length;
      this.dataSource = new MatTableDataSource<DistributionItem>(basicItemList);
      if (this.hasVersionedResources && domain.code !== 'FAV' && this.selectedDatasetFacets.size > 0) {
        this.dataSource.data = basicItemList.filter(item => {
          const resourceVersion = item.distSummary.resourceVersion;
          if (this.datasets !== null) {
            return this.datasets.some(dataset => dataset.distributions.some(distribution => distribution.id === item.distId)
              && (this.selectedDatasetFacets.has(`${dataset.resourceId}:${dataset.versionId}`)
                || this.selectedDatasetFacets.has(dataset.group.id)
                || dataset.categories.some(category => this.selectedDatasetFacets.has(category.id))));
          }
          return this.selectedDatasetFacets.has(`${resourceVersion?.resourceId}:${resourceVersion?.versionId}`)
            || item.levels.some(levels => levels.some(level => this.selectedDatasetFacets.has(level.distId ?? '')));
        });
        this.totalFacetsSelected = this.dataSource.data.length;
      }
      this.dataSource.paginator = this.paginator;
    }
  }

  private addLevelToElement(elem: DistributionItem | null): void {
    if (elem !== null) {
      setTimeout(() => {

        // add to selected extra information (level*)
        this.configurables.addLevels(
          elem.distId,
          elem.levels[0]
        );

      }, 500);
    }
  }

  /**
   * If there are any data configurables, try to reselect the previously selected one. If that fails,
   * select the first one
   */
  private updateSelectedConfig(): void {
    const dataConfigurables = this.dataConfigurablesArraySource.getValue();
    let newSelectedIndex = -1;

    if ((dataConfigurables != null) && (dataConfigurables.length > 0)) {
      // try to reselect previous selected
      newSelectedIndex = dataConfigurables.findIndex((configurable: DataConfigurable) => {
        return (this.lastGoodSelectedConfigurableId === configurable.id);
      });
      if (newSelectedIndex === -1) {
        newSelectedIndex = 0;
      }
    }
    this.selectConfigIndex(newSelectedIndex);

  }

  /**
   * It ensures that the triggerReloadFunc is set on the configurables.
   * @param configurables - Array<DataConfigurableDataSearchI>
   */
  private ensureReloadFuncSet(configurables: Array<DataConfigurableDataSearchI>): void {
    // ensure reset func set
    configurables.forEach((configurable: DataConfigurableDataSearchI) => {
      if (configurable instanceof DataConfigurableDataSearch) {
        configurable.setTriggerReloadFunc((configurableToUpdate: DataConfigurableDataSearch) => {
          this.configurables.replaceOrAdd(configurableToUpdate, true);
        });
      }
    });
  }

  /**
   * The function creates items to display in a MatTable based on a distribution facet model and filter
   * type.
   * @param {FacetModel<DistributionSummary> | null} distFacetModelSource - The `distFacetModelSource`
   * parameter is of type `FacetModel<DistributionSummary> | null`, which means it can either be a
   * `FacetModel` containing `DistributionSummary` objects or `null`.
   * @param filterType - The `filterType` parameter in the `createItemsDisplay` method is an array of
   * strings that specifies the type of filters to be applied when creating items to display. It is
   * used as input to the `createItemsToDisplay` method along with `distFacetModelSource` and `domains
   */
  private createItemsDisplay(distFacetModelSource: FacetModel<DistributionSummary> | null, filterType: Array<string>): void {
    this.data = this.configurables.createItemsToDisplay(distFacetModelSource, this.domains, filterType);

    this.dataSource = new MatTableDataSource<DistributionItem>(this.data);
    this.dataSource.paginator = this.paginator;
  }

  private formatNotificationMessage(message: NotificationMessage): void {
    this.messageId = message.id ?? '';
    this.messageShow = true;
    this.messageTitle = message.title ?? this.messageTitle;
    this.messageType = message.type;
    this.messageText = message.message;
    this.messageCheckShowAgain = message.showAgain;
  }

  /**
   * Display a notification message while loading the distribution.
   * @param distId - The distribution ID to load the notification for. Used to load the details of the
   * provider.
   * @param duration - The duration of the notification in milliseconds.
   */
  private loadingNotification(distId: string, duration: number = 4000): void {
    // Get the details of the provider
    this.getProviderDetails(distId).then(providerDetails => {
      // If no provider details, no notification
      if (!providerDetails) {
        return;
      }

      // Get the organization logo and show the notification
      this.getOrganizationLogo(providerDetails.providerId).then(logoUrl => {
        // Create the notification message
        const notificationMessage = this.createNotificationMessage(providerDetails);
        // Show the loading notification and keep a reference to it
        const notificationRef = this.notification.sendLoadingNotification(notificationMessage, logoUrl);
        // Handle the duration of the notification based on the loading state of the data
        this.handleNotificationDuration(notificationRef, duration);
      }).catch(error => console.error(error));
    }).catch(error => console.error(error));
  }

  /**
   * Get the logo of the organization with the given ID.
   * @param organizationId - The organization ID to get the logo for.
   * @returns A promise that resolves to the logo URL of the organization or null if no logo is found.
   */
  private async getOrganizationLogo(organizationId: string): Promise<string | null> {
    const organization = await this.searchService.getOrganizationById(organizationId);
    return organization?.getLogoUrl() ?? null;
  }

  /**
   * Get the provider details of a distribution.
   * @param distId - The distribution ID to get the provider details for.
   * @returns A promise that resolves to the provider name and url or null if no web service provider
   * is found.
   */
  private async getProviderDetails(distId: string): Promise<ProviderDetails | null> {
    const dataset = this.datasets?.find(item => item.distributions.some(distribution => distribution.id === distId));
    if (dataset != null) {
      const provider = dataset.distributions.find(distribution => distribution.id === distId)?.provider;
      if (!provider?.id || !provider.name) {
        return null;
      }
      return { providerName: provider.name, providerUrl: null, providerId: provider.id };
    }

    // Get the details
    const summary = this.data.find(item => item.distId === distId)?.distSummary;
    const resourceVersion = summary?.resourceVersion;
    const details = resourceVersion != null
      ? await this.searchService.getResourceVersionDistributionDetails(
        resourceVersion.resourceType, resourceVersion.resourceId, resourceVersion.versionId, distId, summary!,
      ) : await this.searchService.getDetailsById(distId, this.context);

    const providerName = details.getWebServiceProvider()?.dataProviderLegalName;
    const providerUrl = details.getWebServiceProvider()?.dataProviderUrl;
    const providerId = details.getWebServiceProvider()?.dataProviderId;

    // If no provider name or ID, return null
    if (!providerName || !providerId) {
      return null;
    }

    return { providerName, providerUrl, providerId };
  }

  /**
   * Create the notification message for the loading notification.
   * @param providerDetails - The provider details to create the notification message for.
   * @returns A reference to the notification.
   */
  private createNotificationMessage(providerDetails: ProviderDetails): string {
    return `<b>Connecting to service provider<br>
        ${providerDetails.providerUrl // If there is a provider URL, make the provider name a link
        ? `<a href="${providerDetails.providerUrl}" target="_blank" rel="noopener noreferrer">${providerDetails.providerName}</a>`
        : providerDetails.providerName
        }</b>`;
  }

  /**
   * Handle the duration of the notification based on the loading state of the data.
   * @param notificationRef - The reference to the notification to handle the duration for.
   * @param duration - The duration of the notification in milliseconds.
   */
  private handleNotificationDuration(notificationRef: MatSnackBarRef<NotificationSnackComponent>, duration: number): void {
    // Minimum time given to the system to start the loading of the data, it is necessary for when the loading is instant
    const minTimeToStartLoading = 2000;

    // Start the timer
    const startTime = new Date().getTime();
    // Flag to check if the loading has started
    let loadingStarted = false;

    // If the loading has not started after the minimum time, dismiss the notification after the duration
    setTimeout(() => {
      if (!loadingStarted) {
        setTimeout(() => notificationRef.dismiss(), duration - minTimeToStartLoading);
      }
    }, minTimeToStartLoading);

    // Subscribe to the leaflet loading service to check if the loading has started
    const subscription = this.leafletLoadingService.showLoadingObs.subscribe(showLoading => {
      // If the loading has started, set the flag to true
      if (showLoading) {
        loadingStarted = true;
        return;
      }

      // If the loading has stopped and the notification is still showing, dismiss it
      if (!showLoading && loadingStarted) {
        const timeElapsed = new Date().getTime() - startTime;
        const remainingTime = duration - timeElapsed;

        setTimeout(() => notificationRef.dismiss(), remainingTime > 0 ? remainingTime : 0);
        subscription.unsubscribe();
      }
    });
  }
}

class ProviderDetails {
  providerName: string;
  providerUrl: string | null | undefined;
  providerId: string;
}
