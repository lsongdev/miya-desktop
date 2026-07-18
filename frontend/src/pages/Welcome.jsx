import { useState } from 'react'
import { Bot, Check, Loader2, Sparkles } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useMiyaConfig } from '@/context/MiyaConfigContext'
import miyaIcon from '@/assets/images/miya-icon.png'

const providerDefaults = {
  openai: {
    id: 'openai',
    baseUrl: 'https://api.openai.com/v1',
    model: 'gpt-5',
  },
  compatible: {
    id: 'provider',
    baseUrl: '',
    model: '',
  },
}

const validId = /^[A-Za-z0-9][A-Za-z0-9._-]*$/

export default function Welcome() {
  const { path, saveConfig, saving, error } = useMiyaConfig()
  const [providerKind, setProviderKind] = useState('openai')
  const [providerId, setProviderId] = useState(providerDefaults.openai.id)
  const [apiBase, setApiBase] = useState(providerDefaults.openai.baseUrl)
  const [apiKey, setApiKey] = useState('')
  const [profileId, setProfileId] = useState('default')
  const [model, setModel] = useState(providerDefaults.openai.model)
  const [workspace, setWorkspace] = useState('~/.miya/workspace')
  const [formError, setFormError] = useState('')

  const canSubmit = validId.test(providerId.trim()) && validId.test(profileId.trim()) && Boolean(model.trim())

  const selectProviderKind = (kind) => {
    const defaults = providerDefaults[kind]
    setProviderKind(kind)
    setProviderId(defaults.id)
    setApiBase(defaults.baseUrl)
    setModel(defaults.model)
  }

  const initialize = async (event) => {
    event.preventDefault()
    if (!canSubmit) {
      setFormError('Provider and profile IDs may only contain letters, numbers, dots, underscores, and dashes.')
      return
    }
    setFormError('')
    const normalizedProvider = providerId.trim()
    const normalizedProfile = profileId.trim()
    try {
      await saveConfig({
        agents: [{
          id: `miya-${normalizedProfile}`,
          name: normalizedProfile === 'default' ? 'Miya Default' : normalizedProfile,
          enabled: true,
          type: 'builtin',
          profile: normalizedProfile,
          command: 'miya-agent',
          args: ['acp'],
        }],
        providers: {
          [normalizedProvider]: {
            type: 'openai',
            apiKey: apiKey.trim(),
            apiBase: apiBase.trim(),
          },
        },
        profiles: {
          [normalizedProfile]: {
            provider: normalizedProvider,
            model: model.trim(),
            workspace: workspace.trim(),
            maxTokens: 8192,
            contextWindowTokens: 128000,
            contextWarnRatio: 0.9,
          },
        },
        mcpServers: {},
        channels: [],
        channelsEnabled: false,
        logging: { enabled: true, level: 'info' },
      })
    } catch (err) {
      setFormError(err?.toString() || 'Failed to initialize Miya')
    }
  }

  return (
    <main className="flex h-full min-h-0 w-full overflow-y-auto bg-background">
      <div className="mx-auto flex w-full max-w-4xl flex-col justify-center px-6 py-8 lg:px-10">
        <div className="mb-6 flex items-center gap-3">
          <div className="size-11 overflow-hidden rounded-lg bg-muted">
            <img src={miyaIcon} alt="" className="size-full object-cover" />
          </div>
          <div>
            <h1 className="text-2xl font-semibold">Welcome to Miya</h1>
            <p className="text-sm text-muted-foreground">Set up your first agent profile.</p>
          </div>
        </div>

        <form onSubmit={initialize} className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_18rem]">
          <section className="min-w-0 space-y-5">
            <div className="space-y-2">
              <label className="text-sm font-medium">Provider</label>
              <div className="grid grid-cols-2 gap-2" role="radiogroup" aria-label="Provider type">
                <button
                  type="button"
                  role="radio"
                  aria-checked={providerKind === 'openai'}
                  onClick={() => selectProviderKind('openai')}
                  className={`rounded-md border px-3 py-2 text-left text-sm ${providerKind === 'openai' ? 'border-foreground bg-muted' : 'hover:bg-muted/50'}`}
                >
                  <span className="block font-medium">OpenAI</span>
                  <span className="block text-xs text-muted-foreground">OpenAI API</span>
                </button>
                <button
                  type="button"
                  role="radio"
                  aria-checked={providerKind === 'compatible'}
                  onClick={() => selectProviderKind('compatible')}
                  className={`rounded-md border px-3 py-2 text-left text-sm ${providerKind === 'compatible' ? 'border-foreground bg-muted' : 'hover:bg-muted/50'}`}
                >
                  <span className="block font-medium">Compatible API</span>
                  <span className="block text-xs text-muted-foreground">Custom OpenAI endpoint</span>
                </button>
              </div>
            </div>

            <div className="grid gap-4 sm:grid-cols-2">
              <Field label="Provider ID">
                <Input value={providerId} onChange={(event) => setProviderId(event.target.value)} placeholder="openai" pattern="[A-Za-z0-9][A-Za-z0-9._-]*" />
              </Field>
              <Field label="API key">
                <Input type="password" value={apiKey} onChange={(event) => setApiKey(event.target.value)} placeholder="Optional for local endpoints" />
              </Field>
              <Field label="API base" className="sm:col-span-2">
                <Input value={apiBase} onChange={(event) => setApiBase(event.target.value)} placeholder="https://api.openai.com/v1" />
              </Field>
            </div>

            <div className="grid gap-4 border-t pt-5 sm:grid-cols-2">
              <Field label="Profile ID">
                <Input value={profileId} onChange={(event) => setProfileId(event.target.value)} placeholder="default" pattern="[A-Za-z0-9][A-Za-z0-9._-]*" />
              </Field>
              <Field label="Model">
                <Input value={model} onChange={(event) => setModel(event.target.value)} placeholder="gpt-5" />
              </Field>
              <Field label="Workspace" className="sm:col-span-2">
                <Input value={workspace} onChange={(event) => setWorkspace(event.target.value)} placeholder="~/.miya/workspace" />
              </Field>
            </div>

            {(formError || error) && <p className="text-sm text-destructive">{formError || error}</p>}
            <Button type="submit" disabled={!canSubmit || saving} className="min-w-36">
              {saving ? <Loader2 className="size-4 animate-spin" /> : <Sparkles className="size-4" />}
              Initialize Miya
            </Button>
          </section>

          <aside className="border-l pl-6">
            <h2 className="mb-3 text-sm font-semibold">Ready on first launch</h2>
            <div className="space-y-3 text-sm">
              <ReadyItem icon={Bot} title="Default profile" detail="Provider, model, and workspace" />
              <ReadyItem icon={Check} title="Built-in skills" detail="Miya configuration skill included" />
              <ReadyItem icon={Check} title="Private config" detail={path || '~/.miya/config.json'} />
            </div>
          </aside>
        </form>
      </div>
    </main>
  )
}

function Field({ label, className = '', children }) {
  return (
    <label className={`min-w-0 space-y-1.5 ${className}`}>
      <span className="block text-sm font-medium">{label}</span>
      {children}
    </label>
  )
}

function ReadyItem({ icon: Icon, title, detail }) {
  return (
    <div className="flex items-start gap-2.5">
      <Icon className="mt-0.5 size-4 shrink-0 text-muted-foreground" />
      <div className="min-w-0">
        <p className="font-medium">{title}</p>
        <p className="break-words text-xs text-muted-foreground">{detail}</p>
      </div>
    </div>
  )
}
