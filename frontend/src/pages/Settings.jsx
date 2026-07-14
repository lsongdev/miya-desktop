import { useState } from 'react'
import { useTheme } from '../context/ThemeContext'
import { useAgent } from '../context/AgentContext'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import {
  MoonIcon, SunIcon, Settings as SettingsIcon, Bot, Puzzle, Palette, Info,
  Loader2, Plus, Trash2, Pencil, Check, X,
} from 'lucide-react'

const settingsItems = [
  { id: 'general', label: 'General', icon: SettingsIcon },
  { id: 'agents', label: 'Agents', icon: Bot },
  { id: 'mcp', label: 'MCP Servers', icon: Puzzle },
  { id: 'appearance', label: 'Appearance', icon: Palette },
  { id: 'about', label: 'About', icon: Info },
]

function GeneralSettings() {
  const { agents, selectedAgent, selectedAgentId, setSelectedAgentId, connected, connecting, agentInfo, error, connect, disconnect } = useAgent()

  return (
    <div className="space-y-4">
      <h2 className="text-lg font-semibold">General</h2>
      <p className="text-sm text-muted-foreground">
        Select the active agent for chat conversations.
      </p>

      <div className="rounded-lg border bg-card p-6 space-y-4">
        <div className="space-y-2">
          <label className="text-sm font-medium">Active Agent</label>
          <div className="flex gap-2">
            <select
              value={selectedAgentId}
              onChange={(e) => setSelectedAgentId(e.target.value)}
              className="flex-1 h-9 rounded-md border bg-transparent px-3 text-sm outline-none focus:ring-2 focus:ring-ring"
            >
              {agents.map((a) => (
                <option key={a.id} value={a.id}>{a.name}</option>
              ))}
            </select>
            {connected ? (
              <Button variant="outline" onClick={disconnect} disabled={connecting}>
                Disconnect
              </Button>
            ) : (
              <Button onClick={() => connect()} disabled={connecting || !selectedAgent}>
                {connecting ? <Loader2 className="size-4 animate-spin" /> : 'Connect'}
              </Button>
            )}
          </div>
        </div>

        {selectedAgent && (
          <div className="flex items-center gap-3 text-sm">
            <span className="font-medium text-muted-foreground">Command:</span>
            <code className="flex-1 rounded bg-muted px-2 py-1 font-mono text-xs">
              {selectedAgent.command}
            </code>
          </div>
        )}

        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <span className={`size-2 rounded-full ${connected ? 'bg-green-500' : 'bg-muted-foreground/30'}`} />
            <span className="text-sm">
              {connected
                ? `Connected${agentInfo?.name ? `: ${agentInfo.name}` : ''}`
                : 'Disconnected'}
            </span>
          </div>
        </div>

        {error && (
          <div className="rounded-md bg-destructive/10 border border-destructive/20 p-3">
            <p className="text-sm text-destructive">{error}</p>
          </div>
        )}
      </div>
    </div>
  )
}

function AgentsSettings() {
  const { agents, addAgent, updateAgent, removeAgent } = useAgent()
  const [editing, setEditing] = useState(null)
  const [adding, setAdding] = useState(false)
  const [form, setForm] = useState({ name: '', command: '' })

  const startAdd = () => {
    setAdding(true)
    setEditing(null)
    setForm({ name: '', command: '' })
  }

  const startEdit = (agent) => {
    setAdding(false)
    setEditing(agent.id)
    setForm({ name: agent.name, command: agent.command })
  }

  const handleSave = () => {
    if (!form.name.trim() || !form.command.trim()) return
    if (adding) {
      const id = form.name.toLowerCase().replace(/\s+/g, '-') + '-' + Date.now()
      addAgent({ id, name: form.name.trim(), command: form.command.trim() })
    } else if (editing) {
      updateAgent(editing, { name: form.name.trim(), command: form.command.trim() })
    }
    setAdding(false)
    setEditing(null)
    setForm({ name: '', command: '' })
  }

  const handleCancel = () => {
    setAdding(false)
    setEditing(null)
    setForm({ name: '', command: '' })
  }

  const handleDelete = (id) => {
    removeAgent(id)
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold">Agents</h2>
          <p className="text-sm text-muted-foreground">Manage ACP agent configurations.</p>
        </div>
        <Button size="sm" onClick={startAdd} disabled={adding || editing !== null}>
          <Plus className="size-3.5 mr-1" />
          Add Agent
        </Button>
      </div>

      <div className="rounded-lg border bg-card divide-y">
        {agents.map((agent) => (
          <div key={agent.id} className="px-4 py-3">
            {editing === agent.id ? (
              <div className="space-y-2">
                <Input
                  value={form.name}
                  onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
                  placeholder="Agent name"
                  className="h-8 text-sm"
                />
                <Input
                  value={form.command}
                  onChange={(e) => setForm((f) => ({ ...f, command: e.target.value }))}
                  placeholder="e.g. opencode acp"
                  className="h-8 font-mono text-sm"
                />
                <div className="flex gap-1.5">
                  <Button size="sm" onClick={handleSave} disabled={!form.name.trim() || !form.command.trim()}>
                    <Check className="size-3.5 mr-1" /> Save
                  </Button>
                  <Button size="sm" variant="ghost" onClick={handleCancel}>
                    <X className="size-3.5 mr-1" /> Cancel
                  </Button>
                </div>
              </div>
            ) : (
              <div className="flex items-center justify-between">
                <div className="min-w-0">
                  <p className="font-medium text-sm">{agent.name}</p>
                  <p className="text-xs text-muted-foreground font-mono truncate">{agent.command}</p>
                </div>
                <div className="flex items-center gap-0.5 shrink-0 ml-2">
                  <Button variant="ghost" size="icon-xs" onClick={() => startEdit(agent)}>
                    <Pencil className="size-3" />
                  </Button>
                  <Button variant="ghost" size="icon-xs" onClick={() => handleDelete(agent.id)}>
                    <Trash2 className="size-3" />
                  </Button>
                </div>
              </div>
            )}
          </div>
        ))}

        {agents.length === 0 && (
          <div className="px-4 py-8 text-center text-xs text-muted-foreground">
            No agents configured
          </div>
        )}

        {(adding || (editing === null && false)) && (
          <div className="px-4 py-3">
            <div className="space-y-2">
              <Input
                value={form.name}
                onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
                placeholder="Agent name"
                className="h-8 text-sm"
                autoFocus
              />
              <Input
                value={form.command}
                onChange={(e) => setForm((f) => ({ ...f, command: e.target.value }))}
                placeholder="e.g. opencode acp"
                className="h-8 font-mono text-sm"
              />
              <div className="flex gap-1.5">
                <Button size="sm" onClick={handleSave} disabled={!form.name.trim() || !form.command.trim()}>
                  <Check className="size-3.5 mr-1" /> Save
                </Button>
                <Button size="sm" variant="ghost" onClick={handleCancel}>
                  <X className="size-3.5 mr-1" /> Cancel
                </Button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

function McpSettings() {
  return (
    <div className="space-y-4">
      <h2 className="text-lg font-semibold">MCP Servers</h2>
      <p className="text-sm text-muted-foreground">
        Manage Model Context Protocol servers for tool integrations.
      </p>
      <div className="rounded-lg border bg-card p-6">
        <p className="text-sm text-muted-foreground">Coming soon...</p>
      </div>
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
  mcp: McpSettings,
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
