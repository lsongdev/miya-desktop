import { createContext, useContext } from 'react'

export const NavigationContext = createContext({
  page: 'agents',
  params: {},
  navigate: () => {},
})

export function useNavigate() {
  const { navigate } = useContext(NavigationContext)
  return navigate
}
