import { createContext, useContext } from 'react'

export const NavigationContext = createContext({
  page: 'chat',
  params: {},
  navigate: () => {},
})

export function useNavigate() {
  const { navigate } = useContext(NavigationContext)
  return navigate
}
