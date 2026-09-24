// CodeMirror theme for the Typst editor. Every color is a CSS variable from
// index.css, so the editor follows the app's light/dark switch without
// rebuilding the EditorView.
import { HighlightStyle, syntaxHighlighting } from '@codemirror/language'
import { Prec, type Extension } from '@codemirror/state'
import { EditorView } from '@codemirror/view'
import { tags as t } from '@lezer/highlight'
import { typstTags } from 'codemirror-lang-typst/lezer'

const chrome = EditorView.theme({
  '&': {
    height: '100%',
    color: 'var(--text)',
    backgroundColor: 'var(--surface)',
    fontSize: '14px',
  },
  '.cm-scroller': {
    fontFamily: 'var(--font-mono)',
    lineHeight: '1.6',
  },
  '.cm-content': { caretColor: 'var(--gold)' },
  '.cm-cursor, .cm-dropCursor': { borderLeftColor: 'var(--gold)', borderLeftWidth: '2px' },
  '&.cm-focused .cm-selectionBackground, .cm-selectionBackground, .cm-content ::selection': {
    backgroundColor: 'var(--editor-selection)',
  },
  '.cm-activeLine': { backgroundColor: 'var(--editor-active-line)' },
  '.cm-gutters': {
    backgroundColor: 'var(--surface)',
    color: 'var(--text-dim)',
    borderRight: '1px solid var(--border)',
  },
  '.cm-activeLineGutter': { backgroundColor: 'var(--editor-active-line)', color: 'var(--text)' },
  '.cm-foldPlaceholder': {
    backgroundColor: 'var(--surface-2)',
    border: '1px solid var(--border)',
    color: 'var(--text-dim)',
  },
  '.cm-matchingBracket, &.cm-focused .cm-matchingBracket': {
    backgroundColor: 'var(--gold-soft)',
    outline: '1px solid var(--gold)',
  },
  '.cm-searchMatch': { backgroundColor: 'var(--gold-soft)' },
  '.cm-panels': { backgroundColor: 'var(--surface-2)', color: 'var(--text)' },
  '.cm-panels.cm-panels-bottom': { borderTop: '1px solid var(--border)' },
  '.cm-panels.cm-panels-top': { borderBottom: '1px solid var(--border)' },
  '.cm-tooltip': {
    backgroundColor: 'var(--surface)',
    color: 'var(--text)',
    border: '1px solid var(--border)',
    borderRadius: 'var(--radius)',
    boxShadow: 'var(--shadow)',
  },
  '.cm-tooltip-autocomplete > ul > li[aria-selected]': {
    backgroundColor: 'var(--brand)',
    color: 'var(--on-brand)',
  },
})

const highlight = HighlightStyle.define([
  { tag: t.keyword, color: 'var(--syn-keyword)', fontWeight: '500' },
  { tag: t.heading, color: 'var(--syn-heading)', fontWeight: '700' },
  { tag: t.strong, fontWeight: '700' },
  { tag: t.emphasis, fontStyle: 'italic' },
  { tag: t.function(t.variableName), color: 'var(--syn-function)' },
  { tag: t.variableName, color: 'var(--text)' },
  { tag: t.propertyName, color: 'var(--syn-property)' },
  { tag: t.string, color: 'var(--syn-string)' },
  { tag: [t.number, t.bool, t.null, t.atom], color: 'var(--syn-number)' },
  { tag: t.comment, color: 'var(--syn-comment)', fontStyle: 'italic' },
  { tag: [t.escape, t.labelName], color: 'var(--syn-escape)' },
  { tag: t.monospace, color: 'var(--syn-raw)' },
  { tag: t.link, color: 'var(--accent)', textDecoration: 'underline' },
  { tag: [t.operator, t.punctuation, t.paren], color: 'var(--syn-operator)' },
  { tag: t.invalid, color: 'var(--error)' },
  { tag: typstTags.interpolated, color: 'var(--syn-keyword)' },
  { tag: typstTags.listMarker, color: 'var(--syn-heading)', fontWeight: '700' },
  { tag: typstTags.mathDelimiter, color: 'var(--syn-string)' },
])

// typst_lezer() installs its own (light-only) highlight styles as
// non-fallback highlighters; Prec.highest mounts ours first so its rules
// take precedence over theirs.
export const typstifyTheme: Extension = [chrome, Prec.highest(syntaxHighlighting(highlight))]
