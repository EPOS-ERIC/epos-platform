import { Component, Inject, OnInit } from '@angular/core';
import { FormBuilder, FormGroup, FormControl, Validators } from '@angular/forms';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { NotificationService } from 'services/notification.service';
import { DialogData } from '../baseDialogService.abstract';
import { MAT_DIALOG_DATA } from '@angular/material/dialog';
import { DistributionDetails } from 'api/webApi/data/distributionDetails.interface';
import { BaseUrlImpl } from 'api/webApi/impl/baseUrl';
import { AaaiService } from 'api/aaai.service';
import { LocalStoragePersister } from 'services/model/persisters/localStoragePersister';
import { LocalStorageVariables } from 'services/model/persisters/localStorageVariables.enum';
import { Tracker } from 'utility/tracker/tracker.service';
import { TrackerAction, TrackerCategory } from 'utility/tracker/tracker.enum';

export interface DetailsDataIn {
  resourceType: NonNullable<DistributionDetails['resourceVersion']>['resourceType'];
  resourceId: string;
  details: DistributionDetails;
}

@Component({
  selector: 'app-contact-form-dialog',
  templateUrl: './contactFormDialog.component.html',
  styleUrls: ['./contactFormDialog.component.scss']
})
export class ContactFormDialogComponent implements OnInit {
  public readonly defaultAlert = 'This field is required';

  public show: 'loader' | 'form' | 'error' = 'loader';
  public submitDisabled = false;
  public detailsData: DistributionDetails;
  public itemName: string | null;

  public formGroup: FormGroup;
  public feedback = false;
  public isSoftware: boolean;
  public submissionResult = '';

  constructor(
    @Inject(MAT_DIALOG_DATA) public data: DialogData<DetailsDataIn>,
    private readonly formBuilder: FormBuilder,
    private readonly http: HttpClient,
    private readonly notificationService: NotificationService,
    private readonly aaai: AaaiService,
    private readonly localStoragePersister: LocalStoragePersister,
    private readonly tracker: Tracker,
  ) { }

  /**
   * FormControl of the message field.
   * @return {FormControl}
   */
  get message(): FormControl {
    return this.formGroup.get('message') as FormControl;
  }

  get type(): FormControl {
    return this.formGroup.get('type') as FormControl;
  }

  public ngOnInit(): void {
    this.detailsData = this.data.dataIn.details;
    this.itemName = this.detailsData.getName();
    this.isSoftware = this.data.dataIn.resourceType !== 'dataset';

    this.createForm();
    this.show = 'form';
  }

  /**
   * Sends the message to the resource's contact points through Gateway.
   * @param post {object} Data object to be submitted.
   */
  public async onSubmit(post: Record<string, string>): Promise<void> {
    if (this.submitDisabled || !this.formGroup.valid) {
      return;
    }
    this.submitDisabled = true;
    this.submissionResult = '';

    const body = {
      message: post.message.trim(),
    };

    const { resourceType, resourceId } = this.data.dataIn;
    const targets: Array<{ label: string; url: string }> = [];
    if (post.type === 'DATA' || post.type === 'BOTH') {
      targets.push({
        label: this.isSoftware ? 'Software' : 'Data',
        url: new BaseUrlImpl('/api/v1').urlBuilder()
          .addPathElements('resources', resourceType, resourceId, 'contact-requests').build(),
      });
    }
    if (post.type === 'SERVICE' || post.type === 'BOTH') {
      targets.push({
        label: 'Service',
        url: new BaseUrlImpl('/api/v1').urlBuilder()
          .addPathElements('distributions', this.detailsData.getIdentifier(), 'contact-requests').build(),
      });
    }
    const user = this.aaai.getUser();

    const httpOptions = {
      headers: new HttpHeaders({
        Authorization: 'Bearer ' + user?.getToken(),
        // eslint-disable-next-line @typescript-eslint/naming-convention
        'Content-Type': 'application/json',
      }),
    };

    const results = await Promise.allSettled(targets.map(target =>
      this.http.post<void>(target.url, body, httpOptions).toPromise()
    ));
    const sent = targets.filter((_, index) => results[index].status === 'fulfilled');
    const failed = targets.filter((_, index) => results[index].status === 'rejected');

    if (sent.length > 0) {
      this.tracker.trackEvent(TrackerCategory.DISTRIBUTION, TrackerAction.CONTACT_US, this.detailsData.getDomainCode() + Tracker.TARCKER_DATA_SEPARATION + this.detailsData.getName());
    }
    if (failed.length === 0) {
      this.feedback = true;
    } else {
      this.submissionResult = (sent.length > 0 ? `Message sent to ${sent.map(target => target.label).join(' and ')}. ` : '')
        + `Sending to ${failed.map(target => target.label).join(' and ')} failed. Choose where to send again.`;
      this.notificationService.sendNotification(this.submissionResult, 'Close', 'error');
      this.submitDisabled = false;
    }
  }

  public close(): void {
    this.localStoragePersister.set(LocalStorageVariables.LS_CONFIGURABLES, '', false, LocalStorageVariables.LS_LAST_DETAIL_DIALOG_ID);

    this.data.close();
  }

  private createForm() {
    this.formGroup = this.formBuilder.group({
      // eslint-disable-next-line @typescript-eslint/unbound-method
      type: [this.isSoftware ? 'DATA' : null, Validators.required],
      // eslint-disable-next-line @typescript-eslint/unbound-method
      message: [null, [Validators.required, this.noWhitespaceValidator]],
    });
  }

  private noWhitespaceValidator(control: FormControl) {
    // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access, @typescript-eslint/no-unsafe-call
    return (control.value || '').trim().length ? null : { whitespace: true };
  }
}
