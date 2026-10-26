import { describe, expect, it } from 'vitest'
import { importInsertion, universeThumbnailUrl } from './universe'

const apply = (doc: string, name = 'cetz', version = '0.4.2') => {
  const ins = importInsertion(doc, name, version)
  return ins ? doc.slice(0, ins.at) + ins.text + doc.slice(ins.at) : null
}

describe('importInsertion', () => {
  it('adds the import below the imports the file starts with', () => {
    expect(apply('#import "@local/chessbook:0.1.0": *\n\n= Title\n')).toBe(
      '#import "@local/chessbook:0.1.0": *\n#import "@preview/cetz:0.4.2": *\n\n= Title\n',
    )
  })

  it('puts it on top when there is no import, and ends an unterminated last line', () => {
    expect(apply('= Title\n')).toBe('#import "@preview/cetz:0.4.2": *\n= Title\n')
    expect(apply('#import "a.typ"')).toBe('#import "a.typ"\n#import "@preview/cetz:0.4.2": *\n')
  })

  it('goes below an import whose item list spans lines', () => {
    expect(apply('#import "@local/chessbook:0.1.0": (\n  chess-board,\n)\n= Title\n')).toBe(
      '#import "@local/chessbook:0.1.0": (\n  chess-board,\n)\n#import "@preview/cetz:0.4.2": *\n= Title\n',
    )
    // Unbalanced: the very top.
    expect(apply('#import "x.typ": (\n  a,\n= Title\n')?.startsWith('#import "@preview/cetz:0.4.2": *\n')).toBe(true)
  })

  it('ignores a commented-out import', () => {
    expect(importInsertion('// #import "@preview/cetz:0.3.0": *\n= T\n', 'cetz', '0.4.2')).not.toBeNull()
  })

  it('does nothing when the package is already imported in any version', () => {
    expect(importInsertion('#import "@preview/cetz:0.3.0": canvas\n', 'cetz', '0.4.2')).toBeNull()
    // A package whose name merely starts the same is a different one.
    expect(importInsertion('#import "@preview/cetz-plot:0.1.0": *\n', 'cetz', '0.4.2')).not.toBeNull()
  })
})

describe('universeThumbnailUrl', () => {
  it('builds the proxied thumbnail URL', () => {
    expect(universeThumbnailUrl({ name: 'a-b', version: '1.0.0' })).toBe('/api/universe/thumbnail/a-b/1.0.0?size=small')
    expect(universeThumbnailUrl({ name: 'a', version: '1.0.0' }, false)).toBe('/api/universe/thumbnail/a/1.0.0')
  })
})
