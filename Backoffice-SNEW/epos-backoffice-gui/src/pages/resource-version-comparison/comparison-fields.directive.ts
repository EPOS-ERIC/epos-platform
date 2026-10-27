import { AfterViewInit, Directive, ElementRef, Input, OnChanges, OnDestroy, inject } from '@angular/core';

const FIELD_SELECTOR = 'input:not([type="button"]):not([type="submit"]):not(.parameter-search-input), textarea, select';
const READONLY_ROW_SELECTOR = '.contact-point-row, .provider-row, .identifier-item';
const LEFT_CLASS = 'comparison-value--left';
const RIGHT_CLASS = 'comparison-value--right';

@Directive({
  selector: '[appComparisonFields]'
})
export class ComparisonFieldsDirective implements AfterViewInit, OnChanges, OnDestroy {
  private readonly elementRef = inject<ElementRef<HTMLElement>>(ElementRef);
  private observer: MutationObserver | null = null;
  private scheduled = false;
  private initialized = false;
  private destroyed = false;
  private readonly onInteraction = (): void => this.scheduleComparison();

  @Input() appComparisonFields = 0;

  ngAfterViewInit(): void {
    const element = this.elementRef.nativeElement;
    this.initialized = true;
    this.observer = new MutationObserver(() => this.scheduleComparison());
    this.observer.observe(element, {
      childList: true,
      subtree: true,
      characterData: true,
      attributes: true,
      attributeFilter: ['data-comparison-key', 'data-comparison-status']
    });
    element.addEventListener('click', this.onInteraction, true);
    element.addEventListener('change', this.onInteraction, true);
    this.scheduleComparison();
  }

  ngOnChanges(): void {
    if (this.initialized) {
      this.scheduleComparison();
    }
  }

  ngOnDestroy(): void {
    this.destroyed = true;
    this.observer?.disconnect();
    const element = this.elementRef.nativeElement;
    element.removeEventListener('click', this.onInteraction, true);
    element.removeEventListener('change', this.onInteraction, true);
  }

  private scheduleComparison(): void {
    if (this.scheduled) {
      return;
    }

    this.scheduled = true;
    queueMicrotask(() => {
      this.scheduled = false;
      if (!this.destroyed) {
        this.compareRenderedValues();
      }
    });
  }

  private compareRenderedValues(): void {
    const host = this.elementRef.nativeElement;
    const leftPane = host.querySelector<HTMLElement>('.version-section-pane--left');
    const rightPane = host.querySelector<HTMLElement>('.version-section-pane--right');
    if (!leftPane || !rightPane) {
      return;
    }

    this.compareElements(
      this.controls(leftPane),
      this.controls(rightPane),
      (element) => this.controlValue(element)
    );
    this.compareElements(
      this.readonlyRows(leftPane),
      this.readonlyRows(rightPane),
      (element) => this.normalizedText(element)
    );
    this.compareSelectedParameters(leftPane, rightPane);
  }

  private compareElements<T extends HTMLElement>(
    leftElements: T[],
    rightElements: T[],
    value: (element: T) => string
  ): void {
    const length = Math.max(leftElements.length, rightElements.length);
    for (let index = 0; index < length; index += 1) {
      const left = leftElements[index];
      const right = rightElements[index];
      const changed = !left || !right || value(left) !== value(right);
      left?.classList.toggle(LEFT_CLASS, changed);
      right?.classList.toggle(RIGHT_CLASS, changed);
    }
  }

  private controls(pane: HTMLElement): Array<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement> {
    return this.controlsWithin(pane).filter((control) => !control.closest('.parameter-editor'));
  }

  private controlsWithin(element: HTMLElement): Array<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement> {
    return Array.from(
      element.querySelectorAll<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>(FIELD_SELECTOR)
    ).filter((control) => {
      const label = control.closest('label')?.querySelector<HTMLElement>('.detail-label');
      return label?.textContent?.replace('*', '').trim().toLowerCase() !== 'id';
    });
  }

  private compareSelectedParameters(leftPane: HTMLElement, rightPane: HTMLElement): void {
    const leftEditor = leftPane.querySelector<HTMLElement>('.parameter-editor[data-comparison-key]');
    const rightEditor = rightPane.querySelector<HTMLElement>('.parameter-editor[data-comparison-key]');
    const leftControls = leftEditor ? this.controlsWithin(leftEditor) : [];
    const rightControls = rightEditor ? this.controlsWithin(rightEditor) : [];

    this.toggleComparisonClass(leftControls, LEFT_CLASS, false);
    this.toggleComparisonClass(rightControls, RIGHT_CLASS, false);

    const leftKey = leftEditor?.dataset['comparisonKey'];
    const rightKey = rightEditor?.dataset['comparisonKey'];
    if (leftKey && leftKey === rightKey) {
      this.compareElements(leftControls, rightControls, (element) => this.controlValue(element));
      return;
    }

    this.toggleComparisonClass(leftControls, LEFT_CLASS, leftEditor?.dataset['comparisonStatus'] === 'removed');
    this.toggleComparisonClass(rightControls, RIGHT_CLASS, rightEditor?.dataset['comparisonStatus'] === 'added');
  }

  private toggleComparisonClass(elements: HTMLElement[], className: string, enabled: boolean): void {
    elements.forEach((element) => element.classList.toggle(className, enabled));
  }

  private readonlyRows(pane: HTMLElement): HTMLElement[] {
    return Array.from(pane.querySelectorAll<HTMLElement>(READONLY_ROW_SELECTOR))
      .filter((row) => !row.querySelector(FIELD_SELECTOR));
  }

  private controlValue(control: HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement): string {
    if (control instanceof HTMLInputElement && (control.type === 'checkbox' || control.type === 'radio')) {
      return String(control.checked);
    }
    return control.value;
  }

  private normalizedText(element: HTMLElement): string {
    return (element.textContent ?? '').replace(/\s+/g, ' ').trim();
  }
}
