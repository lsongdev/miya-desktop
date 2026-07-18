import { useCallback, useEffect, useState } from 'react'
import { ArrowLeft, ArrowRight, Check, Loader2, RefreshCw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useMiyaConfig } from '@/context/MiyaConfigContext'
import { FetchProviderModelsFromConfig } from '../../bindings/wails-app/app'
import miyaIcon from '@/assets/images/miya-icon.png'

export default function Welcome() {
  const { saveConfig, saving, error } = useMiyaConfig()
  const [step, setStep] = useState(0)
  const [providerName, setProviderName] = useState('openai')
  const [apiBase, setApiBase] = useState('https://api.openai.com/v1')
  const [apiKey, setApiKey] = useState('')
  const [model, setModel] = useState('gpt-5')
  const [formError, setFormError] = useState('')

  const providerValid = Boolean(providerName.trim()) && Boolean(apiBase.trim())
  const modelValid = Boolean(model.trim())

  const next = (event) => {
    event.preventDefault()
    if (step === 1 && !providerValid) {
      setFormError('Enter a valid provider name and URL.')
      return
    }
    setFormError('')
    setStep((current) => Math.min(current + 1, 2))
  }

  const initialize = async (event) => {
    event.preventDefault()
    if (!providerValid || !modelValid) return
    setFormError('')
    const providerID = providerName.trim()
    try {
      await saveConfig({
        agents: [],
        providers: {
          [providerID]: {
            type: 'openai',
            apiKey: apiKey.trim(),
            apiBase: apiBase.trim(),
          },
        },
        profiles: {
          default: {
            provider: providerID,
            model: model.trim(),
            workspace: '~/.miya/workspace',
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
      <div className="mx-auto flex w-full max-w-xl flex-col justify-center px-7 py-10">
        {step === 0 ? (
          <WelcomeStep onContinue={() => setStep(1)} />
        ) : (
          <form onSubmit={step === 1 ? next : initialize} className="w-full">
            <div className="mb-10 flex items-center justify-between">
              <div className="flex items-center gap-2.5">
                <img src={miyaIcon} alt="" className="size-8 rounded-md object-cover" />
                <span className="text-sm font-semibold">Miya</span>
              </div>
              <StepIndicator step={step} />
            </div>

            {step === 1 ? (
              <ProviderStep
                providerName={providerName}
                apiBase={apiBase}
                apiKey={apiKey}
                onProviderNameChange={setProviderName}
                onApiBaseChange={setApiBase}
                onApiKeyChange={setApiKey}
              />
            ) : (
              <ModelStep
                model={model}
                providerName={providerName}
                apiBase={apiBase}
                apiKey={apiKey}
                onModelChange={setModel}
              />
            )}

            {(formError || error) && <p className="mt-5 text-sm text-destructive">{formError || error}</p>}

            <div className="mt-8 flex items-center justify-between border-t pt-5">
              <Button type="button" variant="ghost" onClick={() => setStep((current) => current - 1)}>
                <ArrowLeft className="size-4" />
                Back
              </Button>
              <Button type="submit" disabled={saving || (step === 1 ? !providerValid : !modelValid)}>
                {saving ? <Loader2 className="size-4 animate-spin" /> : step === 1 ? <ArrowRight className="size-4" /> : <Check className="size-4" />}
                {step === 1 ? 'Continue' : 'Start using Miya'}
              </Button>
            </div>
          </form>
        )}
      </div>
    </main>
  )
}

function WelcomeStep({ onContinue }) {
  return (
    <div className="flex flex-col items-center text-center">
      <img src={miyaIcon} alt="" className="mb-7 size-20 rounded-xl object-cover shadow-sm" />
      <h1 className="text-3xl font-semibold">Welcome to Miya</h1>
      <p className="mt-3 max-w-sm text-sm leading-6 text-muted-foreground">
        Connect your model provider and start your first conversation.
      </p>
      <Button size="lg" className="mt-8 min-w-36" onClick={onContinue}>
        Get started
        <ArrowRight className="size-4" />
      </Button>
    </div>
  )
}

function ProviderStep({ providerName, apiBase, apiKey, onProviderNameChange, onApiBaseChange, onApiKeyChange }) {
  return (
    <section>
      <p className="text-xs font-medium text-muted-foreground">Step 1 of 2</p>
      <h1 className="mt-2 text-2xl font-semibold">Connect a provider</h1>
      <div className="mt-7 space-y-5">
        <Field label="Provider name">
          <Input
            value={providerName}
            onChange={(event) => onProviderNameChange(event.target.value)}
            placeholder="openai"
            autoFocus
          />
        </Field>
        <Field label="Base URL">
          <Input value={apiBase} onChange={(event) => onApiBaseChange(event.target.value)} placeholder="https://api.openai.com/v1" />
        </Field>
        <Field label="API key">
          <Input type="password" value={apiKey} onChange={(event) => onApiKeyChange(event.target.value)} placeholder="sk-..." />
        </Field>
      </div>
    </section>
  )
}

function ModelStep({ model, providerName, apiBase, apiKey, onModelChange }) {
  const [modelOptions, setModelOptions] = useState([])
  const [modelsLoading, setModelsLoading] = useState(false)
  const [modelsError, setModelsError] = useState('')

  const fetchModels = useCallback(async () => {
    if (!apiKey.trim()) {
      setModelsError('Enter an API key to fetch available models, or type a model name.')
      return
    }
    setModelsLoading(true)
    setModelsError('')
    try {
      const models = await FetchProviderModelsFromConfig(providerName.trim(), {
        type: 'openai',
        apiKey: apiKey.trim(),
        apiBase: apiBase.trim(),
      })
      setModelOptions(models || [])
    } catch (err) {
      setModelsError(err?.toString?.() || String(err))
    } finally {
      setModelsLoading(false)
    }
  }, [apiBase, apiKey, providerName])

  useEffect(() => {
    fetchModels()
  }, [fetchModels])

  return (
    <section>
      <p className="text-xs font-medium text-muted-foreground">Step 2 of 2</p>
      <h1 className="mt-2 text-2xl font-semibold">Choose your default model</h1>
      <p className="mt-2 text-sm text-muted-foreground">Provider: {providerName}</p>
      <div className="mt-7">
        <Field label="Model">
          <div className="flex gap-2">
            <Input
              list="welcome-model-options"
              value={model}
              onChange={(event) => onModelChange(event.target.value)}
              placeholder="Select or enter a model"
              autoFocus
            />
            <Button type="button" variant="outline" size="icon" onClick={fetchModels} disabled={modelsLoading} title="Refresh models">
              {modelsLoading ? <Loader2 className="size-4 animate-spin" /> : <RefreshCw className="size-4" />}
              <span className="sr-only">Refresh models</span>
            </Button>
          </div>
          <datalist id="welcome-model-options">
            {modelOptions.map((option) => <option key={option} value={option} />)}
          </datalist>
          {modelsError && <p className="text-xs text-destructive">{modelsError}</p>}
        </Field>
      </div>
    </section>
  )
}

function StepIndicator({ step }) {
  return (
    <div className="flex items-center gap-1.5" aria-label={`Step ${step} of 2`}>
      {[1, 2].map((item) => (
        <span key={item} className={`h-1.5 w-8 rounded-full ${item <= step ? 'bg-foreground' : 'bg-muted'}`} />
      ))}
    </div>
  )
}

function Field({ label, children }) {
  return (
    <label className="block space-y-1.5">
      <span className="block text-sm font-medium">{label}</span>
      {children}
    </label>
  )
}
