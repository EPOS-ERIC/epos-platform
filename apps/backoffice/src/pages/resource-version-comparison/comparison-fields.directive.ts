import { AfterViewInit, Directive, ElementRef, Input, OnChanges, OnDestroy, inject } from '@angular/core';

const FIELD_SELECTOR = 'input:not([type="button"]):not([type="submit"]):not(.parameter-search-input), textarea, select';
const READONLY_ROW_SELECTOR = '.contact-point-row, .provider-row, .identifier-item, .category-row';
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

    this.compareControls(this.controls(leftPane), this.controls(rightPane));
    this.compareReadonlyRows(this.readonlyRows(leftPane), this.readonlyRows(rightPane));
    this.compareSelectedDistributions(leftPane, rightPane);
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
    return this.controlsWithin(pane).filter(
      (control) => !control.closest('.parameter-editor, .distribution-body')
    );
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
    const leftEditors = Array.from(
      leftPane.querySelectorAll<HTMLElement>('.parameter-editor[data-comparison-key]')
    );
    const rightEditors = Array.from(
      rightPane.querySelectorAll<HTMLElement>('.parameter-editor[data-comparison-key]')
    );
    const keyedEditors = (editors: HTMLElement[]): Map<string, HTMLElement> => new Map(
      editors.map((editor) => [editor.dataset['comparisonKey']!, editor])
    );
    const leftKeyed = keyedEditors(leftEditors);
    const rightKeyed = keyedEditors(rightEditors);

    leftEditors.forEach((editor) => this.toggleComparisonClass(this.controlsWithin(editor), LEFT_CLASS, false));
    rightEditors.forEach((editor) => this.toggleComparisonClass(this.controlsWithin(editor), RIGHT_CLASS, false));

    new Set([...leftKeyed.keys(), ...rightKeyed.keys()]).forEach((key) => {
      const leftEditor = leftKeyed.get(key);
      const rightEditor = rightKeyed.get(key);
      const leftControls = leftEditor ? this.controlsWithin(leftEditor) : [];
      const rightControls = rightEditor ? this.controlsWithin(rightEditor) : [];
      if (leftEditor && rightEditor) {
        this.compareControls(leftControls, rightControls);
        return;
      }

      this.toggleComparisonClass(leftControls, LEFT_CLASS, leftEditor?.dataset['comparisonStatus'] === 'removed');
      this.toggleComparisonClass(rightControls, RIGHT_CLASS, rightEditor?.dataset['comparisonStatus'] === 'added');
    });
  }

  private compareSelectedDistributions(leftPane: HTMLElement, rightPane: HTMLElement): void {
    const leftDistribution = leftPane.querySelector<HTMLElement>('.distribution-body[data-comparison-key]');
    const rightDistribution = rightPane.querySelector<HTMLElement>('.distribution-body[data-comparison-key]');
    const leftControls = leftDistribution ? this.controlsWithin(leftDistribution) : [];
    const rightControls = rightDistribution ? this.controlsWithin(rightDistribution) : [];

    this.toggleComparisonClass(leftControls, LEFT_CLASS, false);
    this.toggleComparisonClass(rightControls, RIGHT_CLASS, false);

    const leftKey = leftDistribution?.dataset['comparisonKey'];
    const rightKey = rightDistribution?.dataset['comparisonKey'];
    if (leftKey && leftKey === rightKey) {
      this.compareControls(leftControls, rightControls);
      return;
    }

    this.toggleComparisonClass(leftControls, LEFT_CLASS, leftDistribution?.dataset['comparisonStatus'] === 'removed');
    this.toggleComparisonClass(rightControls, RIGHT_CLASS, rightDistribution?.dataset['comparisonStatus'] === 'added');
  }

  private toggleComparisonClass(elements: HTMLElement[], className: string, enabled: boolean): void {
    elements.forEach((element) => element.classList.toggle(className, enabled));
  }

  private compareControls(
    leftControls: Array<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>,
    rightControls: Array<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>
  ): void {
    const leftKeyed = this.keyedControls(leftControls);
    const rightKeyed = this.keyedControls(rightControls);
    const keyedControls = new Set([...leftKeyed.keys(), ...rightKeyed.keys()]);

    this.toggleComparisonClass(leftControls, LEFT_CLASS, false);
    this.toggleComparisonClass(rightControls, RIGHT_CLASS, false);

    keyedControls.forEach((key) => {
      const left = leftKeyed.get(key);
      const right = rightKeyed.get(key);
      const changed = !left || !right || this.controlValue(left) !== this.controlValue(right);
      left?.classList.toggle(LEFT_CLASS, changed);
      right?.classList.toggle(RIGHT_CLASS, changed);
    });

    this.compareElements(
      leftControls.filter((control) => !this.controlComparisonKey(control)),
      rightControls.filter((control) => !this.controlComparisonKey(control)),
      (element) => this.controlValue(element)
    );
  }

  private keyedControls(
    controls: Array<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>
  ): Map<string, HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement> {
    const result = new Map<string, HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>();
    controls.forEach((control) => {
      const key = this.controlComparisonKey(control);
      if (key) {
        result.set(key, control);
      }
    });
    return result;
  }

  private controlComparisonKey(
    control: HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement
  ): string | null {
    const item = control.closest<HTMLElement>('[data-comparison-resource-id]');
    const resourceId = item?.dataset['comparisonResourceId'];
    if (!resourceId) {
      return null;
    }

    const label = this.controlLabel(control);
    const matchingControls = Array.from(
      item.querySelectorAll<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>(FIELD_SELECTOR)
    ).filter((candidate) =>
      candidate.closest('[data-comparison-resource-id]') === item
      && this.controlLabel(candidate) === label
    );
    return `${resourceId}:${label}:${matchingControls.indexOf(control)}`;
  }

  private controlLabel(control: HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement): string {
    const label = control.closest('label');
    const labelText = label?.querySelector<HTMLElement>('.detail-label, span')?.textContent
      ?? label?.textContent
      ?? control.getAttribute('aria-label')
      ?? control.type;
    return this.normalizedTextValue(labelText);
  }

  private readonlyRows(pane: HTMLElement): HTMLElement[] {
    return Array.from(pane.querySelectorAll<HTMLElement>(READONLY_ROW_SELECTOR))
      .filter((row) => !row.querySelector(FIELD_SELECTOR));
  }

  private compareReadonlyRows(leftRows: HTMLElement[], rightRows: HTMLElement[]): void {
    const leftKeyed = new Map(
      leftRows.flatMap((row) => row.dataset['comparisonKey'] ? [[row.dataset['comparisonKey'], row] as const] : [])
    );
    const rightKeyed = new Map(
      rightRows.flatMap((row) => row.dataset['comparisonKey'] ? [[row.dataset['comparisonKey'], row] as const] : [])
    );

    this.toggleComparisonClass(leftRows, LEFT_CLASS, false);
    this.toggleComparisonClass(rightRows, RIGHT_CLASS, false);

    new Set([...leftKeyed.keys(), ...rightKeyed.keys()]).forEach((key) => {
      const left = leftKeyed.get(key);
      const right = rightKeyed.get(key);
      const changed = !left || !right || this.normalizedText(left) !== this.normalizedText(right);
      left?.classList.toggle(LEFT_CLASS, changed);
      right?.classList.toggle(RIGHT_CLASS, changed);
    });

    this.compareElements(
      leftRows.filter((row) => !row.dataset['comparisonKey']),
      rightRows.filter((row) => !row.dataset['comparisonKey']),
      (element) => this.normalizedText(element)
    );
  }

  private controlValue(control: HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement): string {
    if (control instanceof HTMLInputElement && (control.type === 'checkbox' || control.type === 'radio')) {
      return String(control.checked);
    }
    return control.value;
  }

  private normalizedText(element: HTMLElement): string {
    return this.normalizedTextValue(element.textContent ?? '');
  }

  private normalizedTextValue(value: string): string {
    return value.replace(/\s+/g, ' ').replace('*', '').trim().toLocaleLowerCase();
  }
}
