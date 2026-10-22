import { Component, Input } from '@angular/core';

export type IconName =
  | 'arrow-left-alt'
  | 'at-sign'
  | 'apps'
  | 'check'
  | 'chevron-left'
  | 'chevron-right'
  | 'chevron-down'
  | 'code'
  | 'content-copy'
  | 'drag-indicator'
  | 'menu-open'
  | 'menu-close'
  | 'inbox'
  | 'database'
  | 'settings'
  | 'building'
  | 'tags'
  | 'users'
  | 'user'
  | 'search'
  | 'bell'
  | 'block'
  | 'warning'
  | 'sun'
  | 'moon'
  | 'join-group';

@Component({
  selector: 'app-icon',
  templateUrl: './app-icon.component.html',
  styleUrl: './app-icon.component.css'
})
export class AppIconComponent {
  @Input() name: IconName = 'database';
  @Input() size = 18;
}
