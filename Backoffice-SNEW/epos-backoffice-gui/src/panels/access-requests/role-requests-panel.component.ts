import { DatePipe } from '@angular/common';
import { Component, EventEmitter, Input, Output } from '@angular/core';

import type { MembershipRole } from '../../../generated/backofficeSchemas';

export interface RoleRequestItem {
  key: string;
  groupId: string;
  groupName: string;
  groupLogoPath: string | undefined;
  userId: string;
  userEmail: string;
  userName: string;
  requestId: string;
  role: MembershipRole;
  requestedAt: string;
}

export interface RoleRequestDecision {
  item: RoleRequestItem;
  decision: 'accept' | 'reject';
}

@Component({
  selector: 'app-role-requests-panel',
  imports: [DatePipe],
  templateUrl: './role-requests-panel.component.html',
  styleUrl: './role-requests-panel.component.css'
})
export class RoleRequestsPanelComponent {
  @Input() requests: RoleRequestItem[] = [];
  @Input() loading = false;
  @Input() error = '';
  @Input() decisionKey = '';
  @Input() context = 'Role Requests';
  @Input() eyebrow = 'Administration';
  @Input() subtitle = '';

  @Output() readonly decision = new EventEmitter<RoleRequestDecision>();
}
