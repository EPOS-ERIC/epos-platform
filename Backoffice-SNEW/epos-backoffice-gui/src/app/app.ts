import { Component, DOCUMENT, effect, inject, signal } from '@angular/core';
import { RouterOutlet } from '@angular/router';

@Component({
  selector: 'app-root',
  imports: [RouterOutlet],
  templateUrl: './app.html',
  styleUrl: './app.css'
})
export class App {
  private readonly document = inject(DOCUMENT);
  private readonly useScalarVariant = false;

  protected readonly title = signal('epos-backoffice-gui');

  constructor() {
    effect(() => {
      const variant = this.useScalarVariant ? 'scalar' : 'current';
      this.document.documentElement.setAttribute('data-ui-variant', variant);
    });
  }
}
