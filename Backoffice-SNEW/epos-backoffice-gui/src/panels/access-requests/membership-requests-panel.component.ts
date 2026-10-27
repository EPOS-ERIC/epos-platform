import { DatePipe } from '@angular/common';
import { Component, EventEmitter, Input, Output } from '@angular/core';

import type { MembershipRole } from '../../../generated/backofficeSchemas';

export interface MembershipRequestRole {
  id: string;
  role: MembershipRole;
}

export interface MembershipRequestItem {
  key: string;
  groupId: string;
  groupName: string;
  groupLogoPath: string | undefined;
  userId: string;
  userEmail: string;
  userName: string;
  requestedAt: string;
  requestedRoles: MembershipRole[];
  selectedRoles: MembershipRole[];
  pendingRoleRequests: MembershipRequestRole[];
}

export interface MembershipRequestDecision {
  item: MembershipRequestItem;
  decision: 'accept' | 'reject';
}

export interface MembershipRequestRoleToggle {
  item: MembershipRequestItem;
  role: MembershipRole;
}

@Component({
  selector: 'app-membership-requests-panel',
  imports: [DatePipe],
  templateUrl: './membership-requests-panel.component.html',
  styleUrl: './membership-requests-panel.component.css'
})
export class MembershipRequestsPanelComponent {
  @Input() requests: MembershipRequestItem[] = [];
  @Input() loading = false;
  @Input() error = '';
  @Input() decisionKey = '';
  @Input() context = 'Membership Requests';
  @Input() eyebrow = 'Administration';
  @Input() subtitle = '';
  @Input() canSelectRoles = false;
  @Input() availableRoles: MembershipRole[] = [];

  @Output() readonly decision = new EventEmitter<MembershipRequestDecision>();
  @Output() readonly roleToggle = new EventEmitter<MembershipRequestRoleToggle>();
}
