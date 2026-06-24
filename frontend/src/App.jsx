import { useState } from 'react';
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
  SidebarTrigger,
  SidebarInset,
} from './components/ui/sidebar';
import { LayoutDashboard, Info, Settings as SettingsIcon, MoonIcon, SunIcon } from 'lucide-react';
import { ThemeProvider } from './context/ThemeContext';
import Home from './pages/Home';
import Settings from './pages/Settings';
import About from './pages/About';

const sidebarGroups = [
  {
    label: 'Pages',
    items: [
      { title: 'Home', icon: LayoutDashboard, page: 'home' },
      { title: 'About', icon: Info, page: 'about' },
    ],
  },
];

const pageComponents = {
  home: Home,
  settings: Settings,
  about: About,
};

function AppSidebar({ activePage, onNavigate }) {
  return (
    <Sidebar collapsible="icon" className="pt-0">
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
              <SidebarMenuButton size="lg">
                <div className="flex aspect-square size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
                  <LayoutDashboard className="size-4" />
                </div>
                <span className="font-semibold">Wails App</span>
              </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>

      <SidebarContent>
        {sidebarGroups.map((group) => (
          <SidebarGroup key={group.label}>
            <SidebarGroupLabel>{group.label}</SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarMenu>
                {group.items.map((item) => (
                  <SidebarMenuItem key={item.title}>
                    <SidebarMenuButton
                      isActive={activePage === item.page}
                      onClick={() => onNavigate(item.page)}
                    >
                      <item.icon />
                      <span>{item.title}</span>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                ))}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        ))}
      </SidebarContent>

      <SidebarFooter>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton
              isActive={activePage === 'settings'}
              onClick={() => onNavigate('settings')}
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
  const [activePage, setActivePage] = useState('home');
  const ActivePage = pageComponents[activePage];

  return (
    <ThemeProvider>
      <SidebarProvider>
        <AppSidebar activePage={activePage} onNavigate={setActivePage} />
        <SidebarInset>
          <header className="flex h-14 shrink-0 items-center justify-between border-b pl-3 pr-4" style={{ WebkitAppRegion: 'drag' }}>
            <div style={{ WebkitAppRegion: 'no-drag' }}>
              <SidebarTrigger />
            </div>
          </header>
          <main className="flex flex-1 flex-col overflow-auto p-6" style={{ WebkitAppRegion: 'no-drag' }}>
            {ActivePage && <ActivePage />}
          </main>
        </SidebarInset>
      </SidebarProvider>
    </ThemeProvider>
  );
}