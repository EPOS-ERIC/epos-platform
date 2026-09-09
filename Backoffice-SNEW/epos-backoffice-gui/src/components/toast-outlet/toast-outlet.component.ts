import { Component, inject } from '@angular/core';

import { ToastService } from '../../services/toast.service';

@Component({
  selector: 'app-toast-outlet',
  templateUrl: './toast-outlet.component.html',
  styleUrl: './toast-outlet.component.css'
})
export class ToastOutletComponent {
  protected readonly toastService = inject(ToastService);
}
