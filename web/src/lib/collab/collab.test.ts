import { describe, expect, it } from 'vitest'
import * as Y from 'yjs'
import { createThread, deleteThread, listThreads, mentions, reply, setResolved } from './comments'
import { Channel, MsgType, frame, parseControl, parseFrame } from './protocol'
import { applyTextDiff, minimalChange, rebaseChange } from './textDiff'

describe('protocol', () => {
  it('round-trips frames and control JSON', () => {
    const f = frame(MsgType.Update, Channel.Comments, new Uint8Array([7, 8]))
    expect([...f]).toEqual([0, 1, 7, 8])
    expect(parseFrame(f)).toMatchObject({ type: 0, channel: 1 })
    expect(parseFrame(new Uint8Array([1]))).toBeNull()
    const ctl = new TextEncoder().encode('{"type":"external","text":"x"}')
    expect(parseControl(ctl)).toEqual({ type: 'external', text: 'x' })
    expect(parseControl(new TextEncoder().encode('{'))).toBeNull()
  })
})

describe('textDiff', () => {
  it('replaces only the changed span', () => {
    expect(minimalChange('abc', 'abc')).toBeNull()
    expect(minimalChange('hello world', 'hello brave world')).toEqual({ from: 6, to: 6, insert: 'brave ' })
    expect(minimalChange('aaa', 'aa')).toEqual({ from: 2, to: 3, insert: '' })
  })

  it('never splits a surrogate pair', () => {
    const c = minimalChange('x👍y', 'x👎y')!
    expect(c.insert).toBe('👎')
    expect('x👍y'.slice(c.from, c.to)).toBe('👍')
  })

  it('keeps a collaborator edit made elsewhere in the text', () => {
    const a = new Y.Doc()
    const b = new Y.Doc()
    a.getText('content').insert(0, 'line one\nline two\n')
    Y.applyUpdate(b, Y.encodeStateAsUpdate(a))
    // b types at the start while a folds in an outside edit at the end.
    b.getText('content').insert(0, '> ')
    applyTextDiff(a.getText('content'), 'line one\nline 2\n')
    Y.applyUpdate(a, Y.encodeStateAsUpdate(b))
    Y.applyUpdate(b, Y.encodeStateAsUpdate(a))
    expect(a.getText('content').toString()).toBe('> line one\nline 2\n')
    expect(b.getText('content').toString()).toBe(a.getText('content').toString())
  })
})

describe('comments', () => {
  const setup = () => {
    const textDoc = new Y.Doc()
    const text = textDoc.getText('content')
    text.insert(0, 'Khai cuộc Ý là một khai cuộc cổ điển.')
    return { textDoc, text, comments: new Y.Doc() }
  }
  const alice = { user: 'alice', name: 'Alice' }

  it('anchors follow edits made before them', () => {
    const { textDoc, text, comments } = setup()
    const from = text.toString().indexOf('khai cuộc cổ')
    const id = createThread(comments, text, from, from + 9, alice, 'Nên viết hoa?')
    text.insert(0, '= Chương 1\n')
    const [t] = listThreads(comments, textDoc)
    expect(t.id).toBe(id)
    expect(text.toString().slice(t.from!, t.to!)).toBe('khai cuộc')
    expect(t.quote).toBe('khai cuộc')
    expect(t.messages.map((m) => m.text)).toEqual(['Nên viết hoa?'])
  })

  it('typing right after the span does not grow it', () => {
    const { textDoc, text, comments } = setup()
    createThread(comments, text, 0, 4, alice, 'x')
    text.insert(4, 'XX')
    const [t] = listThreads(comments, textDoc)
    expect(text.toString().slice(t.from!, t.to!)).toBe('Khai')
  })

  it('replies, resolves, deletes and orphans', () => {
    const { textDoc, text, comments } = setup()
    const id = createThread(comments, text, 0, 4, alice, 'a')
    reply(comments, id, { user: 'bob', name: 'Bob' }, 'b @alice')
    setResolved(comments, id, true)
    let [t] = listThreads(comments, textDoc)
    expect(t.resolved).toBe(true)
    expect(t.messages.map((m) => m.authorName)).toEqual(['Alice', 'Bob'])
    text.delete(0, 10)
    ;[t] = listThreads(comments, textDoc)
    expect(t.from).toBeNull()
    deleteThread(comments, id)
    expect(listThreads(comments, textDoc)).toEqual([])
  })

  it('syncs between collaborators like any Yjs document', () => {
    const { textDoc, text, comments } = setup()
    createThread(comments, text, 0, 4, alice, 'hi')
    const textCopy = new Y.Doc()
    const commentsCopy = new Y.Doc()
    Y.applyUpdate(textCopy, Y.encodeStateAsUpdate(textDoc))
    Y.applyUpdate(commentsCopy, Y.encodeStateAsUpdate(comments))
    const [t] = listThreads(commentsCopy, textCopy)
    expect(textCopy.getText('content').toString().slice(t.from!, t.to!)).toBe('Khai')
  })

  it('finds mentions', () => {
    expect(mentions('@alice xem giúp, cc @bob.k')).toEqual(['alice', 'bob.k'])
    expect(mentions('email@example.com')).toEqual([])
  })
})

describe('rebaseChange', () => {
  const base = 'đầu\ngiữa\ncuối\n'
  it('applies an outside edit before or after local edits', () => {
    // local edit at the end, outside edit at the start
    const cur = base + 'offline\n'
    expect(rebaseChange(base, 'ĐẦU\ngiữa\ncuối\n', cur)).toEqual({ kind: 'apply', from: 0, to: 3, insert: 'ĐẦU' })
    // local edit at the start shifts an outside edit at the end
    const cur2 = '>> ' + base
    const r = rebaseChange(base, 'đầu\ngiữa\nCUỐI\n', cur2)
    expect(r.kind).toBe('apply')
    if (r.kind === 'apply') {
      const merged = cur2.slice(0, r.from) + r.insert + cur2.slice(r.to)
      expect(merged).toBe('>> đầu\ngiữa\nCUỐI\n')
    }
  })

  it('keeps offline edits when the disk did not really change', () => {
    expect(rebaseChange(base, base, base + 'offline')).toEqual({ kind: 'none' })
  })

  it('reports overlapping edits as a conflict', () => {
    // Both rewrite the end of "giữa".
    expect(rebaseChange(base, 'đầu\nGIỮA\ncuối\n', 'đầu\ngiữ nguyên\ncuối\n')).toEqual({ kind: 'conflict' })
    // Typing right after a word changed outside is not a conflict.
    expect(rebaseChange(base, 'đầu\nGIỮA\ncuối\n', 'đầu\ngiữa đã sửa\ncuối\n')).toEqual({
      kind: 'apply',
      from: 4,
      to: 8,
      insert: 'GIỮA',
    })
  })

  it('applies directly when there are no local edits', () => {
    expect(rebaseChange(base, base + 'x', base)).toEqual({ kind: 'apply', from: base.length, to: base.length, insert: 'x' })
  })
})
