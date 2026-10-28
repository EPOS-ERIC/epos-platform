import { Component, EventEmitter, Input, Output } from '@angular/core';
import { FormsModule } from '@angular/forms';

@Component({
  selector: 'app-keyword-chips',
  imports: [FormsModule],
  templateUrl: './keyword-chips.component.html',
  styleUrl: './keyword-chips.component.css'
})
export class KeywordChipsComponent {
  @Input() keywords: string[] = [];
  @Input() disabled = false;
  @Output() keywordsChange = new EventEmitter<string[]>();

  protected draft = '';

  protected handleKeydown(event: KeyboardEvent): void {
    if (event.key !== 'Enter' && event.key !== ',') return;
    event.preventDefault();
    this.commitDraft();
  }

  protected commitDraft(): void {
    if (this.disabled) return;

    const existing = new Set(this.keywords.map((keyword) => keyword.toLowerCase()));
    const additions = this.draft
      .split(',')
      .map((keyword) => keyword.trim())
      .filter((keyword) => {
        const normalized = keyword.toLowerCase();
        if (!normalized || existing.has(normalized)) return false;
        existing.add(normalized);
        return true;
      });

    this.draft = '';
    if (additions.length) {
      this.keywordsChange.emit([...this.keywords, ...additions]);
    }
  }

  protected removeKeyword(index: number): void {
    if (this.disabled) return;
    this.keywordsChange.emit(this.keywords.filter((_, keywordIndex) => keywordIndex !== index));
  }
}
