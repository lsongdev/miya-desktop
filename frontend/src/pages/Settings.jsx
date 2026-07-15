import { useEffect, useState } from 'react'
import { useTheme } from '../context/ThemeContext'
import { useMiyaConfig } from '../context/MiyaConfigContext'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { ChannelsServiceStatus, FetchProviderModels, StartChannelsService, StopChannelsService } from '../../wailsjs/go/main/App'
import {
  MoonIcon, SunIcon, Settings as SettingsIcon, Bot, Puzzle, Info,
  Plus, Trash2, Pencil, Check, X, Key, Radio, Loader2, ExternalLink,
} from 'lucide-react'
import miyaIcon from '../assets/images/miya-icon.png'

const settingsItems = [
  { id: 'general', label: 'General', icon: SettingsIcon },
  { id: 'agents', label: 'Agents', icon: Bot },
  { id: 'profiles', label: 'Profiles', icon: Bot },
  { id: 'providers', label: 'Providers', icon: Key },
  { id: 'mcp', label: 'MCP Servers', icon: Puzzle },
  { id: 'channels', label: 'Channels', icon: Radio },
  { id: 'about', label: 'About', icon: Info },
]

const channelTypes = [
  { id: 'telegram', label: 'Telegram' },
  { id: 'feishu', label: 'Feishu' },
  { id: 'wechat', label: 'WeChat' },
  { id: 'wecom', label: 'WeCom' },
]

const inputClass = 'h-8 text-sm'
const monoInputClass = 'h-8 font-mono text-sm'
const selectClass = 'h-8 rounded-md border bg-transparent px-3 text-sm outline-none focus:ring-2 focus:ring-ring'
const textAreaClass = 'min-h-24 w-full rounded-md border bg-transparent px-3 py-2 font-mono text-xs outline-none focus:ring-2 focus:ring-ring'

function entriesOf(obj) {
  return Object.entries(obj || {}).map(([id, value]) => ({ id, ...(value || {}) }))
}

function commandFromAgent(agent) {
  if (!agent?.command) return agent?.url || 'not configured'
  return [agent.command, ...(agent.args || [])].join(' ')
}

function parseList(value) {
  return value.split(/\s+/).map((s) => s.trim()).filter(Boolean)
}

function slugify(value) {
  return value
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
}

function basename(value) {
  const normalized = value.trim().replace(/\\/g, '/')
  return normalized.split('/').filter(Boolean).pop() || ''
}

function uniqueAgentId(form, agents, editing) {
  if (editing) return editing
  const base = slugify(form.name || basename(form.command) || 'agent') || 'agent'
  const used = new Set((agents || []).map((agent) => agent.id).filter(Boolean))
  if (!used.has(base)) return base
  let index = 2
  while (used.has(`${base}-${index}`)) index += 1
  return `${base}-${index}`
}

function uniqueMcpServerId(form, servers, editing) {
  if (editing) return editing
  const source = form.command || form.url || 'mcp-server'
  const base = slugify(basename(source) || source || 'mcp-server') || 'mcp-server'
  const used = new Set(Object.keys(servers || {}))
  if (!used.has(base)) return base
  let index = 2
  while (used.has(`${base}-${index}`)) index += 1
  return `${base}-${index}`
}

function parseKeyValues(value) {
  const out = {}
  value.split('\n').map((s) => s.trim()).filter(Boolean).forEach((line) => {
    const idx = line.indexOf('=')
    if (idx > 0) out[line.slice(0, idx).trim()] = line.slice(idx + 1).trim()
  })
  return out
}

function keyValuesToText(value) {
  return Object.entries(value || {}).map(([k, v]) => `${k}=${v}`).join('\n')
}

function rawToText(value) {
  if (!value) return '{}'
  try {
    return JSON.stringify(value, null, 2)
  } catch {
    return '{}'
  }
}

function channelSummary(channel) {
  if (channel.id === 'telegram') {
    return channel.token ? 'bot token configured' : 'bot token missing'
  }
  return rawToText(channel).replace(/\s+/g, ' ')
}

function ConfigError({ error }) {
  if (!error) return null
  return <p className="text-xs text-destructive">{error}</p>
}

function SectionHeader({ title, description, action }) {
  return (
    <div className="flex items-start justify-between gap-4">
      <div className="min-w-0">
        <h2 className="text-lg font-semibold">{title}</h2>
        {description && <p className="text-sm text-muted-foreground">{description}</p>}
      </div>
      {action && <div className="shrink-0">{action}</div>}
    </div>
  )
}

function Switch({ checked, onChange, disabled, label }) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      aria-label={label}
      disabled={disabled}
      onClick={() => onChange?.(!checked)}
      className={`relative inline-flex h-5 w-9 shrink-0 items-center rounded-full transition-colors disabled:opacity-50 ${
        checked ? 'bg-primary' : 'bg-muted-foreground/30'
      }`}
    >
      <span
        className={`inline-block size-4 rounded-full bg-background shadow transition-transform ${
          checked ? 'translate-x-4' : 'translate-x-0.5'
        }`}
      />
    </button>
  )
}

function GitHubMark({ className = 'size-4' }) {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true" className={className} fill="currentColor">
      <path d="M12 .5A12 12 0 0 0 8.2 23.9c.6.1.8-.3.8-.6v-2.1c-3.3.7-4-1.4-4-1.4-.5-1.3-1.3-1.7-1.3-1.7-1.1-.7.1-.7.1-.7 1.2.1 1.9 1.2 1.9 1.2 1.1 1.8 2.9 1.3 3.5 1 .1-.8.4-1.3.8-1.6-2.6-.3-5.4-1.3-5.4-5.9 0-1.3.5-2.4 1.2-3.2-.1-.3-.5-1.6.1-3.2 0 0 1-.3 3.3 1.2a11.4 11.4 0 0 1 6 0c2.3-1.5 3.3-1.2 3.3-1.2.6 1.6.2 2.9.1 3.2.8.8 1.2 1.9 1.2 3.2 0 4.6-2.8 5.6-5.4 5.9.4.4.8 1.1.8 2.2v3.1c0 .3.2.7.8.6A12 12 0 0 0 12 .5Z" />
    </svg>
  )
}

function GeneralSettings() {
  const { theme, toggleTheme } = useTheme()

  return (
    <div className="space-y-4">
      <SectionHeader title="General" description="Application preferences and appearance." />
      <div className="rounded-lg border bg-card p-4">
        <div className="flex items-center justify-between gap-4">
          <div>
            <p className="text-sm font-medium">Theme</p>
            <p className="text-sm text-muted-foreground">Switch between light and dark mode.</p>
          </div>
          <Button variant="outline" size="icon" onClick={toggleTheme}>
            {theme === 'light' ? <MoonIcon className="h-5 w-5" /> : <SunIcon className="h-5 w-5" />}
          </Button>
        </div>
      </div>
    </div>
  )
}

function AgentsSettings() {
  const { config, saveConfig, saving, error } = useMiyaConfig()
  const builtinAgent = { id: 'miya', name: 'Miya Agents', type: 'builtin', command: 'miya-agent', args: ['acp'], builtin: true, enabled: true }
  const configuredAgents = Array.isArray(config.agents) ? config.agents : []
  const agents = [builtinAgent, ...configuredAgents.filter((agent) => agent.id !== builtinAgent.id)]
  const [editing, setEditing] = useState(null)
  const [adding, setAdding] = useState(false)
  const [form, setForm] = useState({ name: '', enabled: true, command: '', args: '' })

  const reset = () => setForm({ name: '', enabled: true, command: '', args: '' })
  const startAdd = () => { setAdding(true); setEditing(null); reset() }
  const startEdit = (agent) => {
    setAdding(false)
    setEditing(agent.id)
    setForm({
      name: agent.name || '',
      enabled: agent.enabled !== false,
      command: agent.command || '',
      args: (agent.args || []).join(' '),
    })
  }
  const handleCancel = () => { setAdding(false); setEditing(null); reset() }
  const handleSave = async () => {
    if (!form.command.trim()) return
    await saveConfig((prev) => {
      const id = uniqueAgentId(form, prev.agents || [], editing)
      const agents = (prev.agents || []).filter((agent) => agent.id !== id)
      agents.push({
        id,
        name: form.name.trim(),
        enabled: form.enabled,
        type: 'stdio',
        command: form.command.trim(),
        args: parseList(form.args),
      })
      return { ...prev, agents }
    })
    handleCancel()
  }
  const handleDelete = async (id) => {
    await saveConfig((prev) => ({ ...prev, agents: (prev.agents || []).filter((agent) => agent.id !== id) }))
  }
  const handleToggleEnabled = async (id, enabled) => {
    await saveConfig((prev) => ({
      ...prev,
      agents: (prev.agents || []).map((agent) => (agent.id === id ? { ...agent, enabled } : agent)),
    }))
  }

  const editor = (
    <div className="space-y-2">
      <div className="grid gap-2">
        <Input value={form.name} onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))} placeholder="Display name" className={inputClass} autoFocus />
        <div className="grid gap-2 md:grid-cols-2">
          <Input value={form.command} onChange={(e) => setForm((f) => ({ ...f, command: e.target.value }))} placeholder="Command" className={monoInputClass} />
          <Input value={form.args} onChange={(e) => setForm((f) => ({ ...f, args: e.target.value }))} placeholder="Args" className={monoInputClass} />
        </div>
      </div>
      <div className="flex gap-1.5">
        <Button size="sm" onClick={handleSave} disabled={!form.command.trim() || saving}><Check className="size-3.5 mr-1" /> Save</Button>
        <Button size="sm" variant="ghost" onClick={handleCancel}><X className="size-3.5 mr-1" /> Cancel</Button>
      </div>
    </div>
  )

  return (
    <div className="space-y-4">
      <SectionHeader
        title="Agents"
        description="Manage ACP agent endpoints."
        action={(
          <Button size="sm" onClick={startAdd} disabled={adding || editing !== null}>
            <Plus className="size-3.5 mr-1" /> Add Agent
          </Button>
        )}
      />
      <div className="rounded-lg border bg-card divide-y">
        {agents.map((agent) => (
          <div key={agent.id} className="px-4 py-3">
            {editing === agent.id ? editor : (
              <div className="flex items-center justify-between">
                <div className="min-w-0">
                  <div className="flex items-center gap-2">
                    <p className="font-medium text-sm">{agent.name || agent.id}</p>
                    {agent.builtin && <span className="rounded border px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-muted-foreground">Built in</span>}
                  </div>
                  <p className="text-xs text-muted-foreground font-mono truncate">
                    {agent.type || 'stdio'} / {commandFromAgent(agent)}
                  </p>
                </div>
                <div className="flex items-center gap-2 shrink-0 ml-2">
                  {!agent.builtin && (
                    <>
                      <Button variant="ghost" size="icon-xs" onClick={() => startEdit(agent)}><Pencil className="size-3" /></Button>
                      <Button variant="ghost" size="icon-xs" onClick={() => handleDelete(agent.id)}><Trash2 className="size-3" /></Button>
                    </>
                  )}
                  <Switch
                    checked={agent.enabled !== false}
                    onChange={(enabled) => handleToggleEnabled(agent.id, enabled)}
                    disabled={saving || agent.builtin}
                    label={`Enable ${agent.name || agent.id}`}
                  />
                </div>
              </div>
            )}
          </div>
        ))}
        {agents.length === 0 && <div className="px-4 py-8 text-center text-xs text-muted-foreground">No agents configured</div>}
        {adding && <div className="px-4 py-3">{editor}</div>}
      </div>
      <ConfigError error={error} />
    </div>
  )
}

function ProfilesSettings() {
  const { config, saveConfig, saving, error } = useMiyaConfig()
  const profiles = entriesOf(config.profiles)
  const providers = entriesOf(config.providers)
  const defaultProvider = providers[0]?.id || ''
  const [editing, setEditing] = useState(null)
  const [adding, setAdding] = useState(false)
  const [modelOptions, setModelOptions] = useState([])
  const [modelsLoading, setModelsLoading] = useState(false)
  const [modelsError, setModelsError] = useState(null)
  const [form, setForm] = useState({
    id: 'default', provider: defaultProvider, model: '', workspace: '', maxTokens: '', temperature: '',
    contextWindowTokens: '', contextWarnRatio: '',
  })

  const reset = () => {
    setForm({ id: 'default', provider: providers[0]?.id || '', model: '', workspace: '', maxTokens: '', temperature: '', contextWindowTokens: '', contextWarnRatio: '' })
    setModelOptions([])
    setModelsError(null)
  }
  const startAdd = () => { setAdding(true); setEditing(null); reset() }
  const startEdit = (profile) => {
    setAdding(false)
    setEditing(profile.id)
    setModelOptions([])
    setModelsError(null)
    setForm({
      id: profile.id,
      provider: profile.provider || '',
      model: profile.model || '',
      workspace: profile.workspace || '',
      maxTokens: profile.maxTokens ? String(profile.maxTokens) : '',
      temperature: profile.temperature ? String(profile.temperature) : '',
      contextWindowTokens: profile.contextWindowTokens ? String(profile.contextWindowTokens) : '',
      contextWarnRatio: profile.contextWarnRatio ? String(profile.contextWarnRatio) : '',
    })
  }
  const handleCancel = () => { setAdding(false); setEditing(null); reset() }
  const handleProviderChange = (provider) => {
    setForm((f) => ({ ...f, provider }))
    setModelOptions([])
    setModelsError(null)
  }
  const handleFetchModels = async () => {
    if (!form.provider.trim()) return
    setModelsLoading(true)
    setModelsError(null)
    try {
      const models = await FetchProviderModels(form.provider.trim())
      setModelOptions(models || [])
    } catch (err) {
      setModelsError(err?.toString?.() || String(err))
    } finally {
      setModelsLoading(false)
    }
  }
  const handleSave = async () => {
    const id = form.id.trim()
    if (!id || !form.provider.trim()) return
    await saveConfig((prev) => {
      const profiles = { ...(prev.profiles || {}) }
      if (editing && editing !== id) delete profiles[editing]
      profiles[id] = {
        provider: form.provider.trim(),
        model: form.model.trim(),
        workspace: form.workspace.trim(),
        maxTokens: Number(form.maxTokens) || 0,
        temperature: Number(form.temperature) || 0,
        contextWindowTokens: Number(form.contextWindowTokens) || 0,
        contextWarnRatio: Number(form.contextWarnRatio) || 0,
      }
      return { ...prev, profiles }
    })
    handleCancel()
  }
  const handleDelete = async (id) => {
    await saveConfig((prev) => {
      const profiles = { ...(prev.profiles || {}) }
      delete profiles[id]
      return { ...prev, profiles }
    })
  }

  const editor = (
    <div className="space-y-2">
      <Input value={form.id} onChange={(e) => setForm((f) => ({ ...f, id: e.target.value }))} placeholder="Profile ID" className={inputClass} autoFocus />
      <select value={form.provider} onChange={(e) => handleProviderChange(e.target.value)} className={selectClass}>
        <option value="" disabled>{providers.length ? 'Select provider' : 'No providers configured'}</option>
        {providers.map((provider) => (
          <option key={provider.id} value={provider.id}>{provider.id}</option>
        ))}
      </select>
      <div className="flex gap-2">
        <Input
          value={form.model}
          onChange={(e) => setForm((f) => ({ ...f, model: e.target.value }))}
          placeholder="Model"
          className={monoInputClass}
          list="profile-model-options"
        />
        <datalist id="profile-model-options">
          {modelOptions.map((model) => <option key={model} value={model} />)}
        </datalist>
        <Button size="sm" variant="outline" onClick={handleFetchModels} disabled={!form.provider.trim() || modelsLoading}>
          {modelsLoading && <Loader2 className="size-3.5 mr-1 animate-spin" />}
          Fetch
        </Button>
      </div>
      {modelsError && <p className="text-xs text-destructive">{modelsError}</p>}
      <Input value={form.workspace} onChange={(e) => setForm((f) => ({ ...f, workspace: e.target.value }))} placeholder="Workspace" className={monoInputClass} />
      <Input value={form.maxTokens} onChange={(e) => setForm((f) => ({ ...f, maxTokens: e.target.value }))} placeholder="Max tokens" className={monoInputClass} />
      <Input value={form.temperature} onChange={(e) => setForm((f) => ({ ...f, temperature: e.target.value }))} placeholder="Temperature" className={monoInputClass} />
      <Input value={form.contextWindowTokens} onChange={(e) => setForm((f) => ({ ...f, contextWindowTokens: e.target.value }))} placeholder="Context window tokens" className={monoInputClass} />
      <Input value={form.contextWarnRatio} onChange={(e) => setForm((f) => ({ ...f, contextWarnRatio: e.target.value }))} placeholder="Context warn ratio" className={monoInputClass} />
      <div className="flex gap-1.5">
        <Button size="sm" onClick={handleSave} disabled={!form.id.trim() || !form.provider.trim() || saving}>
          <Check className="size-3.5 mr-1" /> Save
        </Button>
        <Button size="sm" variant="ghost" onClick={handleCancel}>
          <X className="size-3.5 mr-1" /> Cancel
        </Button>
      </div>
    </div>
  )

  return (
    <div className="space-y-4">
      <SectionHeader
        title="Profiles"
        description="Manage miya-agents runtime profiles."
        action={(
          <Button size="sm" onClick={startAdd} disabled={adding || editing !== null}>
            <Plus className="size-3.5 mr-1" /> Add Profile
          </Button>
        )}
      />
      <div className="rounded-lg border bg-card divide-y">
        {profiles.map((profile) => (
          <div key={profile.id} className="px-4 py-3">
            {editing === profile.id ? editor : (
              <div className="flex items-center justify-between">
                <div className="min-w-0">
                  <p className="font-medium text-sm">{profile.id}</p>
                  <p className="text-xs text-muted-foreground font-mono truncate">{profile.provider || 'no provider'} / {profile.model || 'no model'}</p>
                </div>
                <div className="flex items-center gap-0.5 shrink-0 ml-2">
                  <Button variant="ghost" size="icon-xs" onClick={() => startEdit(profile)}><Pencil className="size-3" /></Button>
                  <Button variant="ghost" size="icon-xs" onClick={() => handleDelete(profile.id)}><Trash2 className="size-3" /></Button>
                </div>
              </div>
            )}
          </div>
        ))}
        {profiles.length === 0 && <div className="px-4 py-8 text-center text-xs text-muted-foreground">No profiles configured</div>}
        {adding && <div className="px-4 py-3">{editor}</div>}
      </div>
      <ConfigError error={error} />
    </div>
  )
}

function ProvidersSettings() {
  const { config, saveConfig, saving, error } = useMiyaConfig()
  const providers = entriesOf(config.providers)
  const [editing, setEditing] = useState(null)
  const [adding, setAdding] = useState(false)
  const [form, setForm] = useState({ id: '', type: 'openai', apiKey: '', apiBase: '' })

  const reset = () => setForm({ id: '', type: 'openai', apiKey: '', apiBase: '' })
  const startAdd = () => { setAdding(true); setEditing(null); reset() }
  const startEdit = (provider) => {
    setAdding(false)
    setEditing(provider.id)
    setForm({ id: provider.id, type: provider.type || 'openai', apiKey: provider.apiKey || '', apiBase: provider.apiBase || '' })
  }
  const handleCancel = () => { setAdding(false); setEditing(null); reset() }
  const handleSave = async () => {
    const id = form.id.trim()
    if (!id) return
    await saveConfig((prev) => {
      const providers = { ...(prev.providers || {}) }
      if (editing && editing !== id) delete providers[editing]
      providers[id] = { type: form.type.trim() || 'openai', apiKey: form.apiKey.trim(), apiBase: form.apiBase.trim() }
      return { ...prev, providers }
    })
    handleCancel()
  }
  const handleDelete = async (id) => {
    await saveConfig((prev) => {
      const providers = { ...(prev.providers || {}) }
      delete providers[id]
      return { ...prev, providers }
    })
  }

  const editor = (
    <div className="space-y-2">
      <Input value={form.id} onChange={(e) => setForm((f) => ({ ...f, id: e.target.value }))} placeholder="Provider ID" className={inputClass} autoFocus />
      <Input value={form.type} onChange={(e) => setForm((f) => ({ ...f, type: e.target.value }))} placeholder="Type: openai or anthropic" className={monoInputClass} />
      <Input value={form.apiKey} onChange={(e) => setForm((f) => ({ ...f, apiKey: e.target.value }))} placeholder="API Key" type="password" className={monoInputClass} />
      <Input value={form.apiBase} onChange={(e) => setForm((f) => ({ ...f, apiBase: e.target.value }))} placeholder="API Base URL" className={monoInputClass} />
      <div className="flex gap-1.5">
        <Button size="sm" onClick={handleSave} disabled={!form.id.trim() || saving}><Check className="size-3.5 mr-1" /> Save</Button>
        <Button size="sm" variant="ghost" onClick={handleCancel}><X className="size-3.5 mr-1" /> Cancel</Button>
      </div>
    </div>
  )

  return (
    <div className="space-y-4">
      <SectionHeader
        title="Providers"
        description="Manage model provider credentials and endpoints."
        action={(
          <Button size="sm" onClick={startAdd} disabled={adding || editing !== null}>
            <Plus className="size-3.5 mr-1" /> Add Provider
          </Button>
        )}
      />
      <div className="rounded-lg border bg-card divide-y">
        {providers.map((provider) => (
          <div key={provider.id} className="px-4 py-3">
            {editing === provider.id ? editor : (
              <div className="flex items-center justify-between">
                <div className="min-w-0">
                  <p className="font-medium text-sm">{provider.id}</p>
                  <p className="text-xs text-muted-foreground font-mono truncate">{provider.type || 'openai'} / {provider.apiKey ? 'configured key' : 'no key'} / {provider.apiBase || 'default endpoint'}</p>
                </div>
                <div className="flex items-center gap-0.5 shrink-0 ml-2">
                  <Button variant="ghost" size="icon-xs" onClick={() => startEdit(provider)}><Pencil className="size-3" /></Button>
                  <Button variant="ghost" size="icon-xs" onClick={() => handleDelete(provider.id)}><Trash2 className="size-3" /></Button>
                </div>
              </div>
            )}
          </div>
        ))}
        {providers.length === 0 && <div className="px-4 py-8 text-center text-xs text-muted-foreground">No providers configured</div>}
        {adding && <div className="px-4 py-3">{editor}</div>}
      </div>
      <ConfigError error={error} />
    </div>
  )
}

function McpSettings() {
  const { config, saveConfig, saving, error } = useMiyaConfig()
  const servers = entriesOf(config.mcpServers)
  const [editing, setEditing] = useState(null)
  const [adding, setAdding] = useState(false)
  const [form, setForm] = useState({ id: '', type: 'stdio', command: '', args: '', env: '', url: '', headers: '' })

  const reset = () => setForm({ id: '', type: 'stdio', command: '', args: '', env: '', url: '', headers: '' })
  const startAdd = () => { setAdding(true); setEditing(null); reset() }
  const startEdit = (server) => {
    setAdding(false)
    setEditing(server.id)
    setForm({
      id: server.id,
      type: server.type || 'stdio',
      command: server.command || '',
      args: (server.args || []).join(' '),
      env: keyValuesToText(server.env),
      url: server.url || '',
      headers: keyValuesToText(server.headers),
    })
  }
  const handleCancel = () => { setAdding(false); setEditing(null); reset() }
  const handleSave = async () => {
    if (form.type === 'stdio' && !form.command.trim()) return
    if (form.type !== 'stdio' && !form.url.trim()) return
    await saveConfig((prev) => {
      const id = uniqueMcpServerId(form, prev.mcpServers || {}, editing)
      const mcpServers = { ...(prev.mcpServers || {}) }
      if (editing) delete mcpServers[editing]
      mcpServers[id] = {
        type: form.type.trim() || 'stdio',
        command: form.type === 'stdio' ? form.command.trim() : '',
        args: form.type === 'stdio' ? parseList(form.args) : [],
        env: form.type === 'stdio' ? parseKeyValues(form.env) : {},
        url: form.type === 'stdio' ? '' : form.url.trim(),
        headers: form.type === 'stdio' ? {} : parseKeyValues(form.headers),
      }
      return { ...prev, mcpServers }
    })
    handleCancel()
  }
  const handleDelete = async (id) => {
    await saveConfig((prev) => {
      const mcpServers = { ...(prev.mcpServers || {}) }
      delete mcpServers[id]
      return { ...prev, mcpServers }
    })
  }

  const editor = (
    <div className="space-y-2">
      <select value={form.type} onChange={(e) => setForm((f) => ({ ...f, type: e.target.value }))} className={selectClass} autoFocus>
          <option value="stdio">stdio</option>
          <option value="http">http</option>
          <option value="sse">sse</option>
      </select>
      {form.type === 'stdio' ? (
        <>
          <div className="grid gap-2 md:grid-cols-2">
            <Input value={form.command} onChange={(e) => setForm((f) => ({ ...f, command: e.target.value }))} placeholder="Command" className={monoInputClass} />
            <Input value={form.args} onChange={(e) => setForm((f) => ({ ...f, args: e.target.value }))} placeholder="Args" className={monoInputClass} />
          </div>
          <textarea value={form.env} onChange={(e) => setForm((f) => ({ ...f, env: e.target.value }))} placeholder="ENV_KEY=value" className={textAreaClass} />
        </>
      ) : (
        <>
          <Input value={form.url} onChange={(e) => setForm((f) => ({ ...f, url: e.target.value }))} placeholder={`${form.type.toUpperCase()} URL`} className={monoInputClass} />
          <textarea value={form.headers} onChange={(e) => setForm((f) => ({ ...f, headers: e.target.value }))} placeholder="Header-Name=value" className={textAreaClass} />
        </>
      )}
      <div className="flex gap-1.5">
        <Button size="sm" onClick={handleSave} disabled={(form.type === 'stdio' ? !form.command.trim() : !form.url.trim()) || saving}><Check className="size-3.5 mr-1" /> Save</Button>
        <Button size="sm" variant="ghost" onClick={handleCancel}><X className="size-3.5 mr-1" /> Cancel</Button>
      </div>
    </div>
  )

  return (
    <div className="space-y-4">
      <SectionHeader
        title="MCP Servers"
        description="Manage tool server connections."
        action={(
          <Button size="sm" onClick={startAdd} disabled={adding || editing !== null}>
            <Plus className="size-3.5 mr-1" /> Add Server
          </Button>
        )}
      />
      <div className="rounded-lg border bg-card divide-y">
        {servers.map((server) => (
          <div key={server.id} className="px-4 py-3">
            {editing === server.id ? editor : (
              <div className="flex items-center justify-between">
                <div className="min-w-0">
                  <p className="font-medium text-sm">{server.id}</p>
                  <p className="text-xs text-muted-foreground font-mono truncate">{server.type || 'stdio'} / {server.url || [server.command, ...(server.args || [])].filter(Boolean).join(' ') || 'not configured'}</p>
                </div>
                <div className="flex items-center gap-0.5 shrink-0 ml-2">
                  <Button variant="ghost" size="icon-xs" onClick={() => startEdit(server)}><Pencil className="size-3" /></Button>
                  <Button variant="ghost" size="icon-xs" onClick={() => handleDelete(server.id)}><Trash2 className="size-3" /></Button>
                </div>
              </div>
            )}
          </div>
        ))}
        {servers.length === 0 && <div className="px-4 py-8 text-center text-xs text-muted-foreground">No MCP servers configured</div>}
        {adding && <div className="px-4 py-3">{editor}</div>}
      </div>
      <ConfigError error={error} />
    </div>
  )
}

function ChannelsSettings() {
  const { config, saveConfig, saving, error } = useMiyaConfig()
  const channels = entriesOf(config.channels)
  // TODO: Move this desktop-local preference out of shared ~/.miya/config.json.
  const channelsEnabled = config.channelsEnabled !== false
  const [editing, setEditing] = useState(null)
  const [adding, setAdding] = useState(false)
  const [localError, setLocalError] = useState(null)
  const [service, setService] = useState({ running: false })
  const [serviceBusy, setServiceBusy] = useState(false)
  const [form, setForm] = useState({ id: 'telegram', token: '', json: '{}' })

  useEffect(() => {
    let cancelled = false
    const refresh = () => {
      ChannelsServiceStatus()
        .then((status) => { if (!cancelled) setService(status) })
        .catch((err) => { if (!cancelled) setService({ running: false, error: err.toString() }) })
    }
    refresh()
    const timer = setInterval(refresh, 3000)
    return () => {
      cancelled = true
      clearInterval(timer)
    }
  }, [])

  const reset = () => { setForm({ id: 'telegram', token: '', json: '{}' }); setLocalError(null) }
  const startAdd = () => { setAdding(true); setEditing(null); reset() }
  const startEdit = (channel) => {
    setAdding(false)
    setEditing(channel.id)
    setLocalError(null)
    const { id, ...value } = channel
    setForm({ id, token: value.token || '', json: rawToText(value) })
  }
  const handleCancel = () => { setAdding(false); setEditing(null); reset() }
  const handleSave = async () => {
    const id = form.id.trim()
    if (!id) return
    let value
    if (id === 'telegram') {
      if (!form.token.trim()) return
      value = { token: form.token.trim() }
    } else {
      try {
        value = JSON.parse(form.json || '{}')
      } catch (err) {
        setLocalError(err.toString())
        return
      }
    }
    await saveConfig((prev) => {
      const channels = { ...(prev.channels || {}) }
      if (editing && editing !== id) delete channels[editing]
      channels[id] = value
      return { ...prev, channels }
    })
    handleCancel()
  }
  const handleDelete = async (id) => {
    await saveConfig((prev) => {
      const channels = { ...(prev.channels || {}) }
      delete channels[id]
      return { ...prev, channels }
    })
  }
  const toggleChannelsEnabled = async (enabled) => {
    await saveConfig((prev) => ({ ...prev, channelsEnabled: enabled }))
  }

  const toggleService = async () => {
    setServiceBusy(true)
    setLocalError(null)
    try {
      const next = service.running ? await StopChannelsService() : await StartChannelsService()
      setService(next)
    } catch (err) {
      setLocalError(err.toString())
      try {
        setService(await ChannelsServiceStatus())
      } catch {}
    } finally {
      setServiceBusy(false)
    }
  }

  const editor = (
    <div className="space-y-2">
      <select value={form.id} onChange={(e) => setForm((f) => ({ ...f, id: e.target.value }))} className={selectClass} autoFocus>
        {channelTypes.map((channel) => (
          <option key={channel.id} value={channel.id}>{channel.label}</option>
        ))}
      </select>
      {form.id === 'telegram' ? (
        <Input value={form.token} onChange={(e) => setForm((f) => ({ ...f, token: e.target.value }))} placeholder="Bot token" type="password" className={monoInputClass} />
      ) : (
        <textarea value={form.json} onChange={(e) => setForm((f) => ({ ...f, json: e.target.value }))} placeholder="Channel JSON config" className="min-h-48 w-full rounded-md border bg-transparent px-3 py-2 font-mono text-xs outline-none focus:ring-2 focus:ring-ring" />
      )}
      <div className="flex gap-1.5">
        <Button size="sm" onClick={handleSave} disabled={!form.id.trim() || (form.id === 'telegram' && !form.token.trim()) || saving}><Check className="size-3.5 mr-1" /> Save</Button>
        <Button size="sm" variant="ghost" onClick={handleCancel}><X className="size-3.5 mr-1" /> Cancel</Button>
      </div>
      <ConfigError error={localError} />
    </div>
  )

  return (
    <div className="space-y-4">
      <SectionHeader
        title="Channels"
        description="Manage remote-control channel config."
        action={(
          <div className="flex items-center gap-3">
            <Switch
              checked={channelsEnabled}
              onChange={toggleChannelsEnabled}
              disabled={saving}
              label="Enable channels"
            />
            <Button size="sm" onClick={startAdd} disabled={adding || editing !== null}>
              <Plus className="size-3.5 mr-1" /> Add Channel
            </Button>
          </div>
        )}
      />
      <div className="rounded-lg border bg-card px-4 py-3 text-xs">
        <div className="flex items-center justify-between gap-3">
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <span className={`size-2 rounded-full ${service.running ? 'bg-green-500' : 'bg-muted-foreground/30'}`} />
              <span className="font-medium">{service.running ? 'Service running' : 'Service stopped'}</span>
              {service.pid ? <span className="text-muted-foreground">PID {service.pid}</span> : null}
            </div>
            {service.command && <p className="mt-1 truncate font-mono text-muted-foreground">{service.command}</p>}
            {service.error && <p className="mt-1 text-destructive">{service.error}</p>}
          </div>
          <Button size="sm" variant={service.running ? 'outline' : 'default'} onClick={toggleService} disabled={serviceBusy || !channelsEnabled}>
            {serviceBusy && <Loader2 className="size-3.5 mr-1 animate-spin" />}
            {service.running ? 'Stop Service' : 'Start Service'}
          </Button>
        </div>
      </div>
      <div className="rounded-lg border bg-card divide-y">
        {channels.map((channel) => (
          <div key={channel.id} className="px-4 py-3">
            {editing === channel.id ? editor : (
              <div className="flex items-center justify-between">
                <div className="min-w-0">
                  <p className="font-medium text-sm">{channel.id}</p>
                  <p className="text-xs text-muted-foreground font-mono truncate">{channelSummary(channel)}</p>
                </div>
                <div className="flex items-center gap-0.5 shrink-0 ml-2">
                  <Button variant="ghost" size="icon-xs" onClick={() => startEdit(channel)}><Pencil className="size-3" /></Button>
                  <Button variant="ghost" size="icon-xs" onClick={() => handleDelete(channel.id)}><Trash2 className="size-3" /></Button>
                </div>
              </div>
            )}
          </div>
        ))}
        {channels.length === 0 && <div className="px-4 py-8 text-center text-xs text-muted-foreground">No channels configured</div>}
        {adding && <div className="px-4 py-3">{editor}</div>}
      </div>
      <ConfigError error={error} />
    </div>
  )
}

function AboutSettings() {
  return (
    <div className="space-y-4">
      <SectionHeader title="About" description="Product information and runtime scope." />
      <div className="rounded-lg border bg-card p-5 space-y-5">
        <div className="flex items-center gap-4">
          <img src={miyaIcon} alt="" className="size-14 rounded-xl object-cover" />
          <div>
            <p className="text-base font-semibold">Miya Desktop</p>
            <p className="text-sm text-muted-foreground">Version 0.1.0 Preview</p>
          </div>
        </div>
        <p className="max-w-2xl text-sm leading-6 text-muted-foreground">
          Miya Desktop is an ACP-native client for managing agent conversations, local agent runtime profiles,
          MCP tools, and remote-control channels from a single desktop workspace.
        </p>
        <div className="space-y-2 text-sm">
          <a className="flex w-full items-center justify-between rounded-md border px-3 py-2 hover:bg-muted" href="https://github.com/lsongdev/miya-desktop" target="_blank" rel="noreferrer">
            <span className="flex items-center gap-2"><GitHubMark /> miya-desktop</span>
            <ExternalLink className="size-3.5" />
          </a>
          <a className="flex w-full items-center justify-between rounded-md border px-3 py-2 hover:bg-muted" href="https://github.com/lsongdev/miya-agents" target="_blank" rel="noreferrer">
            <span className="flex items-center gap-2"><GitHubMark /> miya-agents</span>
            <ExternalLink className="size-3.5" />
          </a>
          <a className="flex w-full items-center justify-between rounded-md border px-3 py-2 hover:bg-muted" href="https://github.com/lsongdev/miya-channels" target="_blank" rel="noreferrer">
            <span className="flex items-center gap-2"><GitHubMark /> miya-channels</span>
            <ExternalLink className="size-3.5" />
          </a>
        </div>
        <p className="text-xs text-muted-foreground">
          License: MIT. Copyright © Lsong.
        </p>
      </div>
    </div>
  )
}

const settingsContent = {
  general: GeneralSettings,
  agents: AgentsSettings,
  profiles: ProfilesSettings,
  providers: ProvidersSettings,
  mcp: McpSettings,
  channels: ChannelsSettings,
  about: AboutSettings,
}

export default function Settings() {
  const [activeItem, setActiveItem] = useState('general')
  const ActiveContent = settingsContent[activeItem]

  return (
    <div className="flex h-full">
      <div className="w-56 shrink-0 border-r p-3">
        <h3 className="px-2 mb-2 text-xs font-medium text-muted-foreground uppercase tracking-wider">
          Settings
        </h3>
        <nav className="space-y-0.5">
          {settingsItems.map((item) => (
            <button
              key={item.id}
              onClick={() => setActiveItem(item.id)}
              className={`flex items-center gap-2.5 w-full px-2 py-1.5 rounded-md text-sm transition-colors ${
                activeItem === item.id
                  ? 'bg-muted font-medium text-foreground'
                  : 'text-muted-foreground hover:bg-muted/50 hover:text-foreground'
              }`}
            >
              <item.icon className="size-4 shrink-0" />
              <span>{item.label}</span>
            </button>
          ))}
        </nav>
      </div>
      <div className="flex-1 overflow-y-auto p-6">
        {ActiveContent && <ActiveContent />}
      </div>
    </div>
  )
}
