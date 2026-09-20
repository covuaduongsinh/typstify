import { useState } from 'react'

/** Pre-canned prompts for common Typst-editing tasks, sent through the same
 * agent.prompt() the free-form chat box uses (AgentChat.tsx). Each action
 * just builds a prompt telling the agent which of its existing MCP tools
 * (service/mcp/apis.go: getActiveDocument, queryDiagnostics, searchPackages)
 * to use -- no new MCP tool or backend plumbing needed. */

interface QuickAction {
  id: string
  label: string
  /** When set, a small text field is shown and its value is interpolated
   * into the prompt before sending; placeholder doubles as the field hint. */
  input?: { placeholder: string }
  buildPrompt: (value: string) => string
}

const ACTIONS: QuickAction[] = [
  {
    id: 'fix-errors',
    label: 'Fix compile errors',
    buildPrompt: () =>
      'Check the currently active Typst document for compile/diagnostic errors ' +
      '(use the getActiveDocument and queryDiagnostics tools), then fix them directly in the file.',
  },
  {
    id: 'summarize',
    label: 'Summarize document',
    buildPrompt: () =>
      'Summarize the currently active Typst document (use getActiveDocument to find it) ' +
      'in a few concise bullet points.',
  },
  {
    id: 'translate',
    label: 'Translate document',
    input: { placeholder: 'Target language, e.g. Vietnamese' },
    buildPrompt: (lang) =>
      `Translate the currently active Typst document into ${lang || 'Vietnamese'}, ` +
      'preserving all Typst markup and commands unchanged, and write the translation to a new file next to the original.',
  },
  {
    id: 'suggest-package',
    label: 'Suggest a Typst package',
    input: { placeholder: 'What do you need? e.g. "a resume template"' },
    buildPrompt: (need) =>
      `Search the Typst package registry (use the searchPackages tool) for something matching this need: "${need}". ` +
      'Suggest the best matching package(s) and explain why.',
  },
  {
    id: 'add-citation',
    label: 'Add bibliography citation',
    input: { placeholder: 'Paste the source (title, author, URL, etc.)' },
    buildPrompt: (source) =>
      'Add a bibliography citation for this source to the currently active document, ' +
      `using correct Typst citation syntax (a bibliography file plus a #cite/@key reference): \n\n${source}`,
  },
]

export function QuickActions({ onPrompt, disabled = false }: { onPrompt: (text: string) => void; disabled?: boolean }) {
  const [openId, setOpenId] = useState<string | null>(null)
  const [value, setValue] = useState('')

  const run = (action: QuickAction) => {
    if (disabled) return
    if (action.input && openId !== action.id) {
      setOpenId(action.id)
      setValue('')
      return
    }
    onPrompt(action.buildPrompt(value))
    setOpenId(null)
    setValue('')
  }

  return (
    <div className="quick-actions">
      <div className="quick-actions-row">
        {ACTIONS.map((a) => (
          <button key={a.id} className="quick-action-btn" onClick={() => run(a)} disabled={disabled}>
            {a.label}
          </button>
        ))}
      </div>
      {openId && (
        <div className="quick-action-input">
          <input
            autoFocus
            value={value}
            placeholder={ACTIONS.find((a) => a.id === openId)?.input?.placeholder}
            onChange={(e) => setValue(e.target.value)}
            disabled={disabled}
            onKeyDown={(e) => {
              if (e.key === 'Enter') run(ACTIONS.find((a) => a.id === openId)!)
              if (e.key === 'Escape') setOpenId(null)
            }}
          />
          <button onClick={() => run(ACTIONS.find((a) => a.id === openId)!)} disabled={disabled}>
            Send
          </button>
          <button onClick={() => setOpenId(null)}>Cancel</button>
        </div>
      )}
    </div>
  )
}
