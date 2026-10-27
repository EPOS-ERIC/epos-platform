import { Component, OnInit, inject } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';

import { AuthService } from '../../services/auth.service';

@Component({
  selector: 'app-login-page',
  templateUrl: './login-page.component.html',
  styleUrl: './login-page.component.css'
})
export class LoginPageComponent implements OnInit {
  protected readonly authService = inject(AuthService);
  private readonly activatedRoute = inject(ActivatedRoute);
  private readonly router = inject(Router);

  ngOnInit(): void {
    if (this.authService.authenticated()) {
      void this.router.navigateByUrl(this.authService.consumeReturnUrl());
    }
  }

  protected login(): void {
    this.authService.login(this.activatedRoute.snapshot.queryParamMap.get('returnUrl') ?? '/data');
  }
}
