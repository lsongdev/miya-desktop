import { memo } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { Browser } from '@wailsio/runtime'

function LinkRenderer({ href, children }) {
  const handleClick = (e) => {
    e.preventDefault()
    if (href) Browser.OpenURL(href)
  }
  return (
    <a href={href} onClick={handleClick} target="_blank" rel="noreferrer">
      {children}
    </a>
  )
}

function MarkdownContent({ content, className = '' }) {
  if (!content) return null

  return (
    <div className={`markdown-content ${className}`}>
      <ReactMarkdown remarkPlugins={[remarkGfm]} components={{ a: LinkRenderer }}>
        {content}
      </ReactMarkdown>
    </div>
  )
}

export default memo(MarkdownContent)
