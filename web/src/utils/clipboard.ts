export async function copyText(value: string): Promise<void> {
  if (navigator.clipboard && window.isSecureContext) {
    await navigator.clipboard.writeText(value)
    return
  }

  // 非セキュアコンテキスト向けフォールバック
  const ta = document.createElement('textarea')
  ta.value = value
  ta.setAttribute('readonly', '')
  ta.style.position = 'fixed'
  ta.style.top = '0'
  ta.style.left = '-9999px'
  ta.style.width = '1px'
  ta.style.height = '1px'
  ta.style.padding = '0'
  ta.style.border = 'none'
  ta.style.outline = 'none'

  const parent =
    (document.activeElement?.closest('[role="dialog"]') as HTMLElement | null) ?? document.body
  parent.appendChild(ta)

  const prevFocus = document.activeElement as HTMLElement | null
  try {
    ta.focus({ preventScroll: true })
    ta.select()
    ta.setSelectionRange(0, ta.value.length)
    const ok = document.execCommand('copy')
    if (!ok) throw new Error('execCommand("copy") returned false')
  } finally {
    parent.removeChild(ta)
    prevFocus?.focus?.()
  }
}
