import { Component, Input } from '@angular/core';

export type IconName =
  | 'chevron-left'
  | 'chevron-right'
  | 'chevron-down'
  | 'menu-open'
  | 'menu-close'
  | 'database'
  | 'settings'
  | 'building'
  | 'tags'
  | 'users'
  | 'user'
  | 'search'
  | 'bell'
  | 'warning'
  | 'sun'
  | 'moon';

@Component({
  selector: 'app-icon',
  templateUrl: './app-icon.component.html',
  styleUrl: './app-icon.component.css'
})
export class AppIconComponent {
  @Input() name: IconName = 'database';
  @Input() size = 18;
}
