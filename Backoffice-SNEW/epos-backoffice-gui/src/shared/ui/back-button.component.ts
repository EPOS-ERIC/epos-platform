import { Component, Input, inject } from '@angular/core';
import { Router, RouterLink } from '@angular/router';

import { AppIconComponent } from './app-icon.component';

@Component({
  selector: 'app-back-button',
  imports: [AppIconComponent, RouterLink],
  templateUrl: './back-button.component.html',
  styleUrl: './back-button.component.css'
})
export class BackButtonComponent {
  private readonly router = inject(Router);

  @Input() route = '/data';
  @Input() label = 'Back';

  protected get routeTree() {
    return this.router.parseUrl(this.route);
  }
}
