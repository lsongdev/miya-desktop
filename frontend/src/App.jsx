import { useCallback, useEffect, useState } from 'react';
import { Events } from '@wailsio/runtime'
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
  SidebarInset,
} from './components/ui/sidebar';
import { MessagesSquare, Settings as SettingsIcon } from 'lucide-react'
import { ThemeProvider } from './context/ThemeContext';
import { AgentProvider } from './context/AgentContext';
import { ProviderProvider } from './context/ProviderContext';
import { MiyaConfigProvider } from './context/MiyaConfigContext';
import { NavigationContext } from './hooks/useNavigate';
import useDesktopNotifications from './hooks/useDesktopNotifications';
import miyaIcon from './assets/images/miya-icon.png'
import Chat from './pages/Chat';
import Settings from './pages/Settings';

const pageComponents = {
  chat: Chat,
  settings: Settings,
};

function updateViewportHeight() {
  document.documentElement.style.setProperty('--app-height', `${window.innerHeight}px`)
}

function repairViewport() {
  const viewport = document.querySelector('meta[name="viewport"]')
  viewport?.setAttribute('content', 'width=device-width, initial-scale=1.0')
  document.documentElement.style.zoom = ''
  document.body.style.zoom = ''
  updateViewportHeight()
  window.dispatchEvent(new Event('resize'))
  requestAnimationFrame(() => {
    updateViewportHeight()
    window.dispatchEvent(new Event('resize'))
  })
}

function AppSidebar({ activePage, onNavigate }) {
  return (
    <Sidebar collapsible="icon" className="pt-0">
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton size="lg" className="group-data-[collapsible=icon]:justify-center">
              <div className="flex aspect-square size-8 shrink-0 items-center justify-center overflow-hidden rounded-lg bg-muted">
                <img src={miyaIcon} alt="" className="size-full object-cover" />
              </div>
              <span className="font-semibold">Miya Desktop</span>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>

      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupContent>
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton
                  isActive={activePage === 'chat'}
                  onClick={() => onNavigate('chat')}
                  tooltip="Chat"
                >
                  <MessagesSquare />
                  <span>Chat</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>

      <SidebarFooter>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton
              isActive={activePage === 'settings'}
              onClick={() => onNavigate('settings')}
              tooltip="Settings"
            >
              <SettingsIcon />
              <span>Settings</span>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarFooter>
    </Sidebar>
  );
}

export default function App() {
  const [activePage, setActivePage] = useState('chat');
  const [navParams, setNavParams] = useState({});
  const ActivePage = pageComponents[activePage];
  const navigate = useCallback((page, params = {}) => {
    setActivePage(page)
    setNavParams(params)
  }, [])
  useDesktopNotifications(navigate)

  useEffect(() => {
    repairViewport()
    const cleanup = Events.On('app:viewport-repair', () => {
      repairViewport()
      window.setTimeout(repairViewport, 100)
    })
    window.addEventListener('resize', updateViewportHeight)
    window.visualViewport?.addEventListener('resize', updateViewportHeight)
    return () => {
      cleanup()
      window.removeEventListener('resize', updateViewportHeight)
      window.visualViewport?.removeEventListener('resize', updateViewportHeight)
    }
  }, [])

  return (
    <ThemeProvider>
      <ProviderProvider>
        <MiyaConfigProvider>
          <AgentProvider>
            <NavigationContext.Provider value={{ page: activePage, params: navParams, navigate }}>
              <SidebarProvider defaultOpen={false}>
                <AppSidebar activePage={activePage} onNavigate={(page) => navigate(page)} />
                <SidebarInset className="min-w-0 overflow-hidden">
                  <main className="flex flex-1 flex-col min-w-0 overflow-hidden" style={{ WebkitAppRegion: 'no-drag' }}>
                    {ActivePage && <ActivePage />}
                  </main>
                </SidebarInset>
              </SidebarProvider>
            </NavigationContext.Provider>
          </AgentProvider>
        </MiyaConfigProvider>
      </ProviderProvider>
    </ThemeProvider>
  );
}
