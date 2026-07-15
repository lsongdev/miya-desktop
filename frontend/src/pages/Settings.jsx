import { useEffect, useState } from 'react'
import { useTheme } from '../context/ThemeContext'
import { useMiyaConfig } from '../context/MiyaConfigContext'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { ChannelsServiceStatus, StartChannelsService, StopChannelsService } from '../../wailsjs/go/main/App'
import {
  MoonIcon, SunIcon, Settings as SettingsIcon, Bot, Puzzle, Palette, Info,
  Plus, Trash2, Pencil, Check, X, Key, Radio, Loader2,
} from 'lucide-react'

const settingsItems = [
  { id: 'general', label: 'General', icon: SettingsIcon },
  { id: 'agents', label: 'Agents', icon: Bot },
  { id: 'profiles', label: 'Profiles', icon: Bot },
  { id: 'providers', label: 'Providers', icon: Key },
  { id: 'mcp', label: 'MCP Servers', icon: Puzzle },
  { id: 'channels', label: 'Channels', icon: Radio },
  { id: 'appearance', label: 'Appearance', icon: Palette },
  { id: 'about', label: 'About', icon: Info },
]

const inputClass = 'h-8 text-sm'
const monoInputClass = 'h-8 font-mono text-sm'
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

function ConfigError({ error }) {
  if (!error) return null
  return <p className="text-xs text-destructive">{error}</p>
}

function GeneralSettings() {
  return (
    <div className="space-y-4">
      <h2 className="text-lg font-semibold">General</h2>
      <p className="text-sm text-muted-foreground">Application preferences.</p>
    </div>
  )
}

function AgentsSettings() {
  const { config, path, saveConfig, saving, error } = useMiyaConfig()
  const agents = Array.isArray(config.agents) ? config.agents : []
  const [editing, setEditing] = useState(null)
  const [adding, setAdding] = useState(false)
  const [form, setForm] = useState({ id: '', name: '', enabled: true, type: 'stdio', command: '', args: '', url: '', headers: '' })

  const reset = () => setForm({ id: '', name: '', enabled: true, type: 'stdio', command: '', args: '', url: '', headers: '' })
  const startAdd = () => { setAdding(true); setEditing(null); reset() }
  const startEdit = (agent) => {
    setAdding(false)
    setEditing(agent.id)
    setForm({
      id: agent.id || '',
      name: agent.name || '',
      enabled: agent.enabled !== false,
      type: agent.type || 'stdio',
      command: agent.command || '',
      args: (agent.args || []).join(' '),
      url: agent.url || '',
      headers: keyValuesToText(agent.headers),
    })
  }
  const handleCancel = () => { setAdding(false); setEditing(null); reset() }
  const handleSave = async () => {
    const id = form.id.trim()
    if (!id) return
    await saveConfig((prev) => {
      const agents = (prev.agents || []).filter((agent) => agent.id !== (editing || id))
      agents.push({
        id,
        name: form.name.trim(),
        enabled: form.enabled,
        type: form.type.trim() || 'stdio',
        command: form.command.trim(),
        args: parseList(form.args),
        url: form.url.trim(),
        headers: parseKeyValues(form.headers),
      })
      return { ...prev, agents }
    })
    handleCancel()
  }
  const handleDelete = async (id) => {
    await saveConfig((prev) => ({ ...prev, agents: (prev.agents || []).filter((agent) => agent.id !== id) }))
  }

  const editor = (
    <div className="space-y-2">
      <div className="grid gap-2 md:grid-cols-2">
        <Input value={form.id} onChange={(e) => setForm((f) => ({ ...f, id: e.target.value }))} placeholder="Agent ID" className={inputClass} autoFocus />
        <Input value={form.name} onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))} placeholder="Display name" className={inputClass} />
        <select value={form.type} onChange={(e) => setForm((f) => ({ ...f, type: e.target.value }))} className="h-8 rounded-md border bg-transparent px-3 text-sm outline-none focus:ring-2 focus:ring-ring">
          <option value="stdio">stdio</option>
          <option value="http">http</option>
          <option value="sse">sse</option>
        </select>
        <Input value={form.command} onChange={(e) => setForm((f) => ({ ...f, command: e.target.value }))} placeholder="Command" className={monoInputClass} />
        <Input value={form.args} onChange={(e) => setForm((f) => ({ ...f, args: e.target.value }))} placeholder="Args" className={monoInputClass} />
        <Input value={form.url} onChange={(e) => setForm((f) => ({ ...f, url: e.target.value }))} placeholder="HTTP/SSE URL" className={monoInputClass} />
      </div>
      <label className="flex items-center gap-2 text-sm">
        <input
          type="checkbox"
          checked={form.enabled}
          onChange={(e) => setForm((f) => ({ ...f, enabled: e.target.checked }))}
          className="size-4"
        />
        Enable for new chat sessions
      </label>
      <textarea value={form.headers} onChange={(e) => setForm((f) => ({ ...f, headers: e.target.value }))} placeholder="Header-Name=value" className={textAreaClass} />
      <div className="flex gap-1.5">
        <Button size="sm" onClick={handleSave} disabled={!form.id.trim() || saving}><Check className="size-3.5 mr-1" /> Save</Button>
        <Button size="sm" variant="ghost" onClick={handleCancel}><X className="size-3.5 mr-1" /> Cancel</Button>
      </div>
    </div>
  )

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold">Agents</h2>
          <p className="text-sm text-muted-foreground">Manage ACP agent endpoints in {path || '~/.miya/config.json'}.</p>
        </div>
        <Button size="sm" onClick={startAdd} disabled={adding || editing !== null}>
          <Plus className="size-3.5 mr-1" /> Add Agent
        </Button>
      </div>
      <div className="rounded-lg border bg-card divide-y">
        {agents.map((agent) => (
          <div key={agent.id} className="px-4 py-3">
            {editing === agent.id ? editor : (
              <div className="flex items-center justify-between">
                <div className="min-w-0">
                  <p className="font-medium text-sm">{agent.name || agent.id}</p>
                  <p className="text-xs text-muted-foreground font-mono truncate">
                    {agent.enabled === false ? 'disabled' : 'enabled'} / {agent.type || 'stdio'} / {commandFromAgent(agent)}
                  </p>
                </div>
                <div className="flex items-center gap-0.5 shrink-0 ml-2">
                  <Button variant="ghost" size="icon-xs" onClick={() => startEdit(agent)}><Pencil className="size-3" /></Button>
                  <Button variant="ghost" size="icon-xs" onClick={() => handleDelete(agent.id)}><Trash2 className="size-3" /></Button>
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
  const { config, path, saveConfig, saving, error } = useMiyaConfig()
  const profiles = entriesOf(config.profiles)
  const [editing, setEditing] = useState(null)
  const [adding, setAdding] = useState(false)
  const [form, setForm] = useState({
    id: '', provider: '', model: '', workspace: '', maxTokens: '', temperature: '',
    contextWindowTokens: '', contextWarnRatio: '',
  })

  const reset = () => setForm({ id: '', provider: '', model: '', workspace: '', maxTokens: '', temperature: '', contextWindowTokens: '', contextWarnRatio: '' })
  const startAdd = () => { setAdding(true); setEditing(null); reset() }
  const startEdit = (profile) => {
    setAdding(false)
    setEditing(profile.id)
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
      <div className="grid gap-2 md:grid-cols-2">
        <Input value={form.id} onChange={(e) => setForm((f) => ({ ...f, id: e.target.value }))} placeholder="Profile ID" className={inputClass} autoFocus />
        <Input value={form.provider} onChange={(e) => setForm((f) => ({ ...f, provider: e.target.value }))} placeholder="Provider ID" className={inputClass} />
        <Input value={form.model} onChange={(e) => setForm((f) => ({ ...f, model: e.target.value }))} placeholder="Model" className={monoInputClass} />
        <Input value={form.workspace} onChange={(e) => setForm((f) => ({ ...f, workspace: e.target.value }))} placeholder="Workspace" className={monoInputClass} />
        <Input value={form.maxTokens} onChange={(e) => setForm((f) => ({ ...f, maxTokens: e.target.value }))} placeholder="Max tokens" className={monoInputClass} />
        <Input value={form.temperature} onChange={(e) => setForm((f) => ({ ...f, temperature: e.target.value }))} placeholder="Temperature" className={monoInputClass} />
        <Input value={form.contextWindowTokens} onChange={(e) => setForm((f) => ({ ...f, contextWindowTokens: e.target.value }))} placeholder="Context window tokens" className={monoInputClass} />
        <Input value={form.contextWarnRatio} onChange={(e) => setForm((f) => ({ ...f, contextWarnRatio: e.target.value }))} placeholder="Context warn ratio" className={monoInputClass} />
      </div>
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
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold">Profiles</h2>
          <p className="text-sm text-muted-foreground">Manage miya-agents runtime profiles in {path || '~/.miya/config.json'}.</p>
        </div>
        <Button size="sm" onClick={startAdd} disabled={adding || editing !== null}>
          <Plus className="size-3.5 mr-1" /> Add Profile
        </Button>
      </div>
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
  const { config, path, saveConfig, saving, error } = useMiyaConfig()
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
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold">Providers</h2>
          <p className="text-sm text-muted-foreground">Manage providers in {path || '~/.miya/config.json'}.</p>
        </div>
        <Button size="sm" onClick={startAdd} disabled={adding || editing !== null}>
          <Plus className="size-3.5 mr-1" /> Add Provider
        </Button>
      </div>
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
  const { config, path, saveConfig, saving, error } = useMiyaConfig()
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
    const id = form.id.trim()
    if (!id) return
    await saveConfig((prev) => {
      const mcpServers = { ...(prev.mcpServers || {}) }
      if (editing && editing !== id) delete mcpServers[editing]
      mcpServers[id] = {
        type: form.type.trim() || 'stdio',
        command: form.command.trim(),
        args: parseList(form.args),
        env: parseKeyValues(form.env),
        url: form.url.trim(),
        headers: parseKeyValues(form.headers),
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
      <div className="grid gap-2 md:grid-cols-2">
        <Input value={form.id} onChange={(e) => setForm((f) => ({ ...f, id: e.target.value }))} placeholder="Server ID" className={inputClass} autoFocus />
        <select value={form.type} onChange={(e) => setForm((f) => ({ ...f, type: e.target.value }))} className="h-8 rounded-md border bg-transparent px-3 text-sm outline-none focus:ring-2 focus:ring-ring">
          <option value="stdio">stdio</option>
          <option value="http">http</option>
          <option value="sse">sse</option>
        </select>
        <Input value={form.command} onChange={(e) => setForm((f) => ({ ...f, command: e.target.value }))} placeholder="Command" className={monoInputClass} />
        <Input value={form.args} onChange={(e) => setForm((f) => ({ ...f, args: e.target.value }))} placeholder="Args" className={monoInputClass} />
        <Input value={form.url} onChange={(e) => setForm((f) => ({ ...f, url: e.target.value }))} placeholder="HTTP/SSE URL" className={monoInputClass} />
      </div>
      <textarea value={form.env} onChange={(e) => setForm((f) => ({ ...f, env: e.target.value }))} placeholder="ENV_KEY=value" className={textAreaClass} />
      <textarea value={form.headers} onChange={(e) => setForm((f) => ({ ...f, headers: e.target.value }))} placeholder="Header-Name=value" className={textAreaClass} />
      <div className="flex gap-1.5">
        <Button size="sm" onClick={handleSave} disabled={!form.id.trim() || saving}><Check className="size-3.5 mr-1" /> Save</Button>
        <Button size="sm" variant="ghost" onClick={handleCancel}><X className="size-3.5 mr-1" /> Cancel</Button>
      </div>
    </div>
  )

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold">MCP Servers</h2>
          <p className="text-sm text-muted-foreground">Manage MCP servers in {path || '~/.miya/config.json'}.</p>
        </div>
        <Button size="sm" onClick={startAdd} disabled={adding || editing !== null}>
          <Plus className="size-3.5 mr-1" /> Add Server
        </Button>
      </div>
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
  const { config, path, saveConfig, saving, error } = useMiyaConfig()
  const channels = entriesOf(config.channels)
  const [editing, setEditing] = useState(null)
  const [adding, setAdding] = useState(false)
  const [localError, setLocalError] = useState(null)
  const [service, setService] = useState({ running: false })
  const [serviceBusy, setServiceBusy] = useState(false)
  const [form, setForm] = useState({ id: '', json: '{}' })

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

  const reset = () => { setForm({ id: '', json: '{}' }); setLocalError(null) }
  const startAdd = () => { setAdding(true); setEditing(null); reset() }
  const startEdit = (channel) => {
    setAdding(false)
    setEditing(channel.id)
    setLocalError(null)
    const { id, ...value } = channel
    setForm({ id, json: rawToText(value) })
  }
  const handleCancel = () => { setAdding(false); setEditing(null); reset() }
  const handleSave = async () => {
    const id = form.id.trim()
    if (!id) return
    let value
    try {
      value = JSON.parse(form.json || '{}')
    } catch (err) {
      setLocalError(err.toString())
      return
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
      <Input value={form.id} onChange={(e) => setForm((f) => ({ ...f, id: e.target.value }))} placeholder="Channel ID, e.g. telegram" className={inputClass} autoFocus />
      <textarea value={form.json} onChange={(e) => setForm((f) => ({ ...f, json: e.target.value }))} placeholder='{"botToken":"...","chatId":"..."}' className="min-h-48 w-full rounded-md border bg-transparent px-3 py-2 font-mono text-xs outline-none focus:ring-2 focus:ring-ring" />
      <div className="flex gap-1.5">
        <Button size="sm" onClick={handleSave} disabled={!form.id.trim() || saving}><Check className="size-3.5 mr-1" /> Save</Button>
        <Button size="sm" variant="ghost" onClick={handleCancel}><X className="size-3.5 mr-1" /> Cancel</Button>
      </div>
      <ConfigError error={localError} />
    </div>
  )

  return (
    <div className="space-y-4">
      <div className="space-y-1">
        <div className="flex items-center justify-between gap-3">
          <h2 className="text-lg font-semibold">Channels</h2>
          <div className="flex items-center gap-2">
            <Button size="sm" variant={service.running ? 'outline' : 'default'} onClick={toggleService} disabled={serviceBusy}>
              {serviceBusy && <Loader2 className="size-3.5 mr-1 animate-spin" />}
              {service.running ? 'Stop Service' : 'Start Service'}
            </Button>
            <Button size="sm" onClick={startAdd} disabled={adding || editing !== null}>
              <Plus className="size-3.5 mr-1" /> Add Channel
            </Button>
          </div>
        </div>
        <p className="text-sm text-muted-foreground">Manage remote-control channel config in {path || '~/.miya/config.json'}.</p>
      </div>
      <div className="rounded-lg border bg-card px-4 py-3 text-xs">
        <div className="flex items-center gap-2">
          <span className={`size-2 rounded-full ${service.running ? 'bg-green-500' : 'bg-muted-foreground/30'}`} />
          <span className="font-medium">{service.running ? 'Service running' : 'Service stopped'}</span>
          {service.pid ? <span className="text-muted-foreground">PID {service.pid}</span> : null}
        </div>
        {service.command && <p className="mt-1 truncate font-mono text-muted-foreground">{service.command}</p>}
        {service.error && <p className="mt-1 text-destructive">{service.error}</p>}
      </div>
      <div className="rounded-lg border bg-card divide-y">
        {channels.map((channel) => (
          <div key={channel.id} className="px-4 py-3">
            {editing === channel.id ? editor : (
              <div className="flex items-center justify-between">
                <div className="min-w-0">
                  <p className="font-medium text-sm">{channel.id}</p>
                  <p className="text-xs text-muted-foreground font-mono truncate">{rawToText(channel).replace(/\s+/g, ' ')}</p>
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

function AppearanceSettings() {
  const { theme, toggleTheme } = useTheme()

  return (
    <div className="space-y-4">
      <h2 className="text-lg font-semibold">Appearance</h2>
      <p className="text-sm text-muted-foreground">
        Customize the look and feel of the application.
      </p>
      <div className="rounded-lg border bg-card p-6 space-y-4">
        <div className="flex items-center justify-between">
          <div>
            <p className="font-medium">Dark Mode</p>
            <p className="text-sm text-muted-foreground">Toggle between light and dark theme</p>
          </div>
          <Button variant="outline" size="icon" onClick={toggleTheme}>
            {theme === 'light' ? <MoonIcon className="h-5 w-5" /> : <SunIcon className="h-5 w-5" />}
          </Button>
        </div>
      </div>
    </div>
  )
}

function AboutSettings() {
  return (
    <div className="space-y-4">
      <h2 className="text-lg font-semibold">About</h2>
      <div className="rounded-lg border bg-card p-6 space-y-2">
        <p className="font-medium">Miya Desktop</p>
        <p className="text-sm text-muted-foreground">Version 0.1.0</p>
        <p className="text-sm text-muted-foreground">
          An AI Agent chat application built with Wails, React, and shadcn/ui.
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
  appearance: AppearanceSettings,
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
