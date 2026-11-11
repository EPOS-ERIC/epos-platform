import { Injectable } from '@angular/core';
import { ApiService } from 'api/api.service';
import { NotificationService } from 'components/notification/notification.service';
import { InteractiveVisualisationService } from 'pages/dataPortal/services/interactiveVisualisation.service';
import { DataConfigurableI } from 'utility/configurables/dataConfigurableI.interface';
import { DataConfigurableDataSearchI } from 'utility/configurablesDataSearch/dataConfigurableDataSearchI.interface';
import { DataConfigurableDataSearchLoading } from 'utility/configurablesDataSearch/dataConfigurableDataSearchLoading';
import { DataSearchConfigurablesServiceResource } from '../../dataPanel/services/dataSearchConfigurables.service';
import { PaleolatitudePoint, PaleolatitudeResponse, PALEOLATITUDE_CONFIG_ID, PALEOLATITUDE_TRACE_ID } from '../objects/paleolatitude.interface';
import { Trace } from '../objects/trace';
import { YAxisDisplayType } from '../objects/yAxisDisplayType.enum';
import { DistributionFormat } from 'api/webApi/data/distributionFormat.interface';
import { ParameterValue } from 'api/webApi/data/parameterValue.interface';

export interface PaleolatitudeGraphRequest {
  normalizedLon: number;
  url: string;
  format?: DistributionFormat;
  parameters?: Array<ParameterValue>;
}

export interface PaleolatitudeGraphData {
  response: PaleolatitudeResponse;
  traces: Array<Trace>;
}

@Injectable({
  providedIn: 'root',
})
export class PaleolatitudeGraphService {

  constructor(
    private readonly apiService: ApiService,
    private readonly configurables: DataSearchConfigurablesServiceResource,
    private readonly interactiveVisualisations: InteractiveVisualisationService,
    private readonly notificationService: NotificationService,
  ) {
  }

  public supports(configurables: Array<DataConfigurableI>): boolean {
    return this.getConfigurable(configurables) != null;
  }

  public async request(id: string, lat: number, lon: number): Promise<void> {
    const request = this.createRequest(lat, lon, this.configurables.getAll());
    if (request == null) {
      this.interactiveVisualisations.removeSource(id);
      this.sendWarning(id);
      return;
    }

    const serviceName = this.getConfigurable(this.configurables.getAll())?.name ?? 'Paleolatitude';
    const pendingName = `Paleolatitude (${request.normalizedLon.toFixed(4)}, ${lat.toFixed(4)})`;
    this.interactiveVisualisations.setSource({
      kind: 'interactive',
      id,
      name: pendingName,
      groupId: PALEOLATITUDE_CONFIG_ID,
      traces: null,
    });
    this.interactiveVisualisations.viewSourceOnGraph(id);

    try {
      const result = await this.executeRequest(request, id, `${PALEOLATITUDE_TRACE_ID}-${id}`);
      if (this.interactiveVisualisations.getSource(id) == null) {
        return;
      }
      if (result.traces.length === 0) {
        this.interactiveVisualisations.removeSource(id);
        this.sendWarning(id);
        return;
      }

      const plateLabel = result.response.plate?.name != null ? ` - ${result.response.plate.name}` : '';
      const resultName = `Paleolatitude${plateLabel} (${request.normalizedLon.toFixed(4)}, ${lat.toFixed(4)})`;
      result.traces.forEach((trace: Trace) => {
        trace.axisGroup = PALEOLATITUDE_CONFIG_ID;
        trace.persistSelection = false;
      });
      const resultDetails = {
        id,
        name: resultName,
        selectedLat: lat,
        selectedLon: request.normalizedLon,
        plateId: result.response.plate?.id,
        plateName: result.response.plate?.name,
      };
      this.interactiveVisualisations.setSource({
        kind: 'interactive',
        id,
        name: resultName,
        groupId: PALEOLATITUDE_CONFIG_ID,
        traces: result.traces,
        table: {
          groupId: PALEOLATITUDE_CONFIG_ID,
          groupName: serviceName,
          rows: (result.response.paleolatitude ?? []).map((point: PaleolatitudePoint) => ({
            ...resultDetails,
            ...point,
          })),
          isMappable: true,
        },
        autoSelect: true,
        preferredDisplayType: YAxisDisplayType.OVERLAY,
        metadata: resultDetails,
      });
    } catch {
      if (this.interactiveVisualisations.getSource(id) != null) {
        this.interactiveVisualisations.removeSource(id);
        this.sendWarning(id);
      }
    }
  }

  public remove(id: string): void {
    this.interactiveVisualisations.removeSource(id);
  }

  public clear(): void {
    this.interactiveVisualisations.clearSources(PALEOLATITUDE_CONFIG_ID);
  }

  public createRequest(
    lat: number,
    lon: number,
    configurables: Array<DataConfigurableDataSearchI>,
  ): null | PaleolatitudeGraphRequest {
    const configurable = this.getConfigurable(configurables);
    if (configurable == null) {
      return null;
    }

    const normalizedLon = this.normalizeLongitude(lon);
    const format = configurable.getDistributionDetails().getFormats().find(item => item.resourceAccess?.type === 'DYNAMIC');
    if (format == null) {
      return null;
    }
    const coordinates = format.resourceAccess!.parameters.filter(item => item.role === 'LATITUDE' || item.role === 'LONGITUDE');
    return {
      normalizedLon,
      url: format.getUrl(),
      format,
      parameters: configurable.getNewParameterValues().filter(item => !coordinates.some(coordinate => coordinate.name === item.name))
        .concat(coordinates.map(item => ({ name: item.name, value: String(item.role === 'LATITUDE' ? lat : normalizedLon) }))),
    };
  }

  public async executeRequest(
    request: PaleolatitudeGraphRequest,
    configurableId: string,
    traceId: string,
  ): Promise<PaleolatitudeGraphData> {
    if (request.format == null || request.format.resourceAccess?.type !== 'DYNAMIC') {
      throw new Error('Paleolatitude requires a dynamic distribution format.');
    }
    const data = await this.apiService.executeDistributionFormat(request.format, request.parameters ?? [], true) as Blob;
    const response = JSON.parse(await data.text()) as PaleolatitudeResponse;

    return {
      response,
      traces: this.createTrace(response, configurableId, traceId),
    };
  }

  private createTrace(response: PaleolatitudeResponse, configurableId: string, traceId: string): Array<Trace> {
    const points = response.paleolatitude ?? [];
    if (points.length === 0 || response.error != null || response.message != null) {
      return [];
    }

    const trace = new Trace(
      configurableId,
      traceId,
      'scatter',
      'Paleolatitude',
      'Paleolatitude from selected map point',
      'deg',
      'Paleolatitude',
      points.map((point: PaleolatitudePoint) => String(point.lat)),
      points.map((point: PaleolatitudePoint) => String(point.age)),
      'lines+markers',
    );

    if (points.every((point: PaleolatitudePoint) => typeof point.lowerbound === 'number' && typeof point.upperbound === 'number')) {
      trace.yErrorMinValues = points.map((point: PaleolatitudePoint) => String(Math.max(point.lat - point.lowerbound!, 0)));
      trace.yErrorMaxValues = points.map((point: PaleolatitudePoint) => String(Math.max(point.upperbound! - point.lat, 0)));
    }

    return [trace];
  }

  private normalizeLongitude(lon: number): number {
    return ((((lon + 180) % 360) + 360) % 360) - 180;
  }

  private getConfigurable(configurables: Array<DataConfigurableI>): DataConfigurableI | undefined {
    return configurables.find((item: DataConfigurableI) => {
      return !(item instanceof DataConfigurableDataSearchLoading)
        && item.getDistributionDetails().getFormats().some(format => format.resourceAccess?.type === 'DYNAMIC');
    });
  }

  private sendWarning(id: string): void {
    this.notificationService.sendDistributionNotification({
      id,
      title: 'Warning',
      message: NotificationService.MESSAGE_NO_DATA,
      type: NotificationService.TYPE_WARNING as string,
      showAgain: false,
    });
  }
}
