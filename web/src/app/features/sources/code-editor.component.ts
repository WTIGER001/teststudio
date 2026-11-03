import { AfterViewInit, ChangeDetectionStrategy, Component, ElementRef, input, OnChanges, OnDestroy, output, ViewChild } from '@angular/core'
import { basicSetup } from 'codemirror'
import { EditorState } from '@codemirror/state'
import { EditorView } from '@codemirror/view'
import { javascript } from '@codemirror/lang-javascript'
import { json } from '@codemirror/lang-json'
import { HighlightStyle, syntaxHighlighting } from '@codemirror/language'
import { tags } from '@lezer/highlight'

@Component({
  selector: 'ts-code-editor', standalone: true, changeDetection: ChangeDetectionStrategy.OnPush,
  template: `<div #host class="editor-host"></div>`,
  styles: [`:host { display: block; min-width: 0; } .editor-host { border: 1px solid var(--border); border-radius: 7px; overflow: hidden; }`]
})
export class CodeEditorComponent implements AfterViewInit, OnChanges, OnDestroy {
  readonly content = input.required<string>()
  readonly path = input.required<string>()
  readonly disabled = input(false)
  readonly changed = output<string>()
  @ViewChild('host', { static: true }) host!: ElementRef<HTMLElement>
  private view?: EditorView
  private documentPath = ''

  ngAfterViewInit(): void { this.mount() }
  ngOnChanges(): void {
    if (!this.view) return
    if (this.documentPath !== this.path() || this.view.state.readOnly !== this.disabled()) { this.mount(); return }
    const value = this.content()
    if (this.view.state.doc.toString() !== value) this.view.dispatch({ changes: { from: 0, to: this.view.state.doc.length, insert: value } })
  }
  ngOnDestroy(): void { this.view?.destroy() }

  private mount(): void {
    this.view?.destroy()
    this.documentPath = this.path()
    this.view = new EditorView({
      parent: this.host.nativeElement, doc: this.content(),
      extensions: [
        basicSetup, this.path().endsWith('.json') ? json() : javascript({ typescript: true }),
        EditorState.readOnly.of(this.disabled()), EditorView.editable.of(!this.disabled()),
        EditorView.contentAttributes.of({ 'aria-label': `Source code: ${this.path()}`, 'aria-describedby': 'playwright-code-hint', 'spellcheck': 'false' }),
        EditorView.updateListener.of(update => { if (update.docChanged) this.changed.emit(update.state.doc.toString()) }),
        syntaxHighlighting(HighlightStyle.define([
          { tag: tags.keyword, color: 'var(--accent)' }, { tag: [tags.string, tags.regexp], color: 'var(--success)' },
          { tag: [tags.number, tags.bool, tags.null], color: 'var(--warning)' }, { tag: tags.comment, color: 'var(--muted)' },
          { tag: [tags.propertyName, tags.function(tags.variableName)], color: 'var(--timeout)' }
        ])),
        EditorView.theme({
          '&': { color: 'var(--text)', backgroundColor: 'var(--surface)', fontSize: '13px' },
          '&.cm-focused': { outline: '2px solid var(--accent)', outlineOffset: '-2px' },
          '.cm-scroller': { overflow: 'auto', maxHeight: '36rem', minHeight: '20rem', fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace' },
          '.cm-content': { caretColor: 'var(--text)' }, '.cm-gutters': { backgroundColor: 'var(--surface-muted)', color: 'var(--muted)', borderColor: 'var(--border)' },
          '.cm-activeLine, .cm-activeLineGutter': { backgroundColor: 'var(--accent-soft)' },
          '.cm-selectionBackground, &.cm-focused .cm-selectionBackground': { backgroundColor: 'var(--accent-soft)' },
          '.cm-panels, .cm-tooltip': { backgroundColor: 'var(--surface-muted)', color: 'var(--text)', borderColor: 'var(--border)' },
          '.cm-cursor': { borderLeftColor: 'var(--text)' }
        })
      ]
    })
  }
}
