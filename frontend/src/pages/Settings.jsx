import { useTheme } from '../context/ThemeContext'
import { Button } from '../components/ui/button'
import { MoonIcon, SunIcon } from 'lucide-react'

export default function Settings() {
  const { theme, toggleTheme } = useTheme()

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Settings</h1>
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
