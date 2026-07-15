import { useState } from 'react';
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
import { Bot, MessagesSquare, SlidersHorizontal, Sparkles } from 'lucide-react'
import { ThemeProvider } from './context/ThemeContext';
import { AgentProvider } from './context/AgentContext';
import { ProviderProvider } from './context/ProviderContext';
import { MiyaConfigProvider } from './context/MiyaConfigContext';
import Chat from './pages/Chat';
import Settings from './pages/Settings';

const pageComponents = {
  chat: Chat,
  settings: Settings,
};

function AppSidebar({ activePage, onNavigate }) {
  return (
    <Sidebar collapsible="icon" className="pt-0">
      <SidebarHeader className="p-1.5">
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton size="default" className="h-9">
              <div className="flex aspect-square size-7 items-center justify-center rounded-md bg-primary text-primary-foreground">
                <Sparkles className="size-3.5" />
              </div>
              <span className="font-semibold">Miya</span>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>

      <SidebarContent>
        <SidebarGroup className="p-1.5">
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

      <SidebarFooter className="p-1.5">
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton
              isActive={activePage === 'settings'}
              onClick={() => onNavigate('settings')}
              tooltip="Settings"
            >
              <SlidersHorizontal />
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
  const ActivePage = pageComponents[activePage];

  return (
    <ThemeProvider>
      <ProviderProvider>
        <MiyaConfigProvider>
          <AgentProvider>
            <SidebarProvider defaultOpen={false}>
              <AppSidebar activePage={activePage} onNavigate={setActivePage} />
              <SidebarInset className="min-w-0 overflow-hidden">
                <main className="flex flex-1 flex-col min-w-0 overflow-hidden" style={{ WebkitAppRegion: 'no-drag' }}>
                  {ActivePage && <ActivePage />}
                </main>
              </SidebarInset>
            </SidebarProvider>
          </AgentProvider>
        </MiyaConfigProvider>
      </ProviderProvider>
    </ThemeProvider>
  );
}
