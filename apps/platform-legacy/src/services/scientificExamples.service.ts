/* eslint-disable @typescript-eslint/member-ordering */
/* eslint-disable @typescript-eslint/no-floating-promises */

import { Injectable } from '@angular/core';
import { Observable, BehaviorSubject, throwError } from 'rxjs';
import { catchError, map } from 'rxjs/operators';
import { HttpClient } from '@angular/common/http';
import { UserNotificationService } from 'components/userNotifications/userNotifications.service';
import { Examples } from 'components/dialog/scientificExamplesDialog/scientificExamplesDialog.component';

interface OfficialConfigurationSummary {
  id: string;
  title: string;
  description: string | null;
  // eslint-disable-next-line @typescript-eslint/naming-convention
  list_of_services: string[];
}

@Injectable({
  providedIn: 'root',
})


export class ScientificExamplesService {
  private readonly endpoint = new URL('api/v1/official-configurations', document.baseURI).toString();
  private examplesSubject = new BehaviorSubject<Examples[]>([]);
  public examples$ = this.examplesSubject.asObservable();

  constructor(
    private readonly http: HttpClient,
    private readonly userNotificationService: UserNotificationService,

  ) {
    this.fetchScientificExamples();
  }

  getscientificExamples(): Observable<object> {
    return this.http.get<OfficialConfigurationSummary[]>(this.endpoint
    ).pipe(
      map((response) => response.map((configuration): Examples => {
        const sharingUrl = new URL(window.location.href);
        sharingUrl.searchParams.set('share', configuration.id);
        return {
          example: configuration.id,
          title: configuration.title,
          description: configuration.description ?? '',
          listOfServices: configuration.list_of_services,
          sharingLinkUrl: sharingUrl.toString(),
        };
      })),
      catchError((error: unknown) => {
        this.userNotificationService.sendErrorNotification(
          'Error fetching scientific examples.',
          6000,
        );
        return throwError(() => error);
      })
    );
  }

  private fetchScientificExamples(): void {
    this.getscientificExamples().subscribe({
      next: (examples: Examples[]) => {
        this.examplesSubject.next(examples);
      },
      error: (error) => {
        console.error('Error fetching examples:', error);
      },
    });
  }
}
