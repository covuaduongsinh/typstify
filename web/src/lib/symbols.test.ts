import { describe, expect, it } from 'vitest'
import { filterSymbols } from './symbols'

const list = [
  { name: 'arrow.r', char: '→', category: 'arrow' },
  { name: 'arrow.r.long', char: '⟶', category: 'arrow' },
  { name: 'harpoon.rt', char: '⇀', category: 'arrow' },
  { name: 'alpha', char: 'α', category: 'greek' },
]

describe('filterSymbols', () => {
  it('ranks prefix matches before substring and category matches', () => {
    expect(filterSymbols(list, 'arrow').map((s) => s.name)).toEqual(['arrow.r', 'arrow.r.long', 'harpoon.rt'])
    expect(filterSymbols(list, '#sym.alp').map((s) => s.name)).toEqual(['alpha'])
  })

  it('finds a symbol by its character', () => {
    expect(filterSymbols(list, '⟶')[0].name).toBe('arrow.r.long')
  })

  it('returns everything (up to the limit) for an empty query', () => {
    expect(filterSymbols(list, '', 2)).toHaveLength(2)
  })
})
