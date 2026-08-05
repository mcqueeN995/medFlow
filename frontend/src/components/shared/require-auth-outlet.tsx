import { Outlet } from 'react-router-dom'
import { useAuthStore } from '@/stores/auth-store'
import { AuthGate } from './auth-gate'

// В отличие от старого RequireAuth (жёсткий редирект на /login), этот гард
// используется как element родительского route: гость не выкидывается со
// страницы, а видит объяснение, зачем нужен аккаунт, и остаётся в AppShell
// (шапка/навигация не исчезают). AppShell теперь публичный — гостю доступны
// каталог библиотеки и навигатор без входа, см. openapi.yaml (security: []
// только у GET /library/* и GET /map/poi) и роль guest в ТЗ.
export function RequireAuthOutlet({ title, description }: { title: string; description: string }) {
  const accessToken = useAuthStore((s) => s.accessToken)
  if (!accessToken) return <AuthGate title={title} description={description} />
  return <Outlet />
}
