import logo from '@/assets/images/logo-universal.png'

export default function Home() {
  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Home</h1>
      <div className="rounded-lg border bg-card p-6">
        <img src={logo} alt="logo" className="w-48 mb-4" />
        <p className="text-muted-foreground">Welcome to Wails App</p>
      </div>
    </div>
  )
}
