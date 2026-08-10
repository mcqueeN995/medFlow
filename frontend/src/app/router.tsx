import { lazy, Suspense, type ComponentType } from 'react'
import { Navigate, createBrowserRouter } from 'react-router-dom'
import { AppShell } from '@/components/shared/app-shell'
import { RequireAuthOutlet } from '@/components/shared/require-auth-outlet'
import { RequireRoleOutlet } from '@/components/shared/require-role-outlet'
import { Skeleton } from '@/components/ui/skeleton'
import { UserRole } from '@/api/generated'
import { LoginPage } from '@/features/auth/login-page'
import { RegisterPage } from '@/features/auth/register-page'
import { TermsPage } from '@/features/auth/terms-page'
import { PrivacyPage } from '@/features/auth/privacy-page'

function PageFallback() {
  return (
    <div className="mx-auto flex max-w-xl flex-col gap-3 p-6">
      <Skeleton className="h-8 w-48" />
      <Skeleton className="h-64 rounded-2xl" />
    </div>
  )
}

// lazyPage - маршруты форума/карточек/админки/библиотеки тянут за собой
// заметную часть общего бандла (988кБ единым чанком до code-splitting'а -
// см. Lighthouse Performance). React.lazy разбивает их на отдельные чанки,
// загружаемые по факту перехода на маршрут, а не все разом при первой
// отрисовке приложения - без этого FCP/LCP на холодном старте были <90.
function lazyPage(loader: () => Promise<{ default: ComponentType<Record<string, never>> }>) {
  const LazyComponent = lazy(loader)
  return (
    <Suspense fallback={<PageFallback />}>
      <LazyComponent />
    </Suspense>
  )
}

export const router = createBrowserRouter([
  { path: '/login', element: <LoginPage /> },
  { path: '/register', element: <RegisterPage /> },
  { path: '/terms', element: <TermsPage /> },
  { path: '/privacy', element: <PrivacyPage /> },
  {
    // AppShell больше не требует входа целиком — гость видит навигацию и
    // публичные разделы (библиотека, навигатор), см. роль guest в ТЗ.
    // Разделы, которым нужен аккаунт, гейтятся точечно через RequireAuthOutlet.
    element: <AppShell />,
    children: [
      { index: true, element: <Navigate to="/library" replace /> },
      {
        path: 'library',
        children: [
          {
            index: true,
            element: lazyPage(() => import('@/features/library/catalog-page').then((m) => ({ default: m.LibraryCatalogPage }))),
          },
          {
            path: 'upload',
            element: (
              <RequireAuthOutlet
                title="Загрузка материалов требует входа"
                description="Чтобы прикрепить свой PDF и подготовить из него карточки, войдите или зарегистрируйтесь."
              />
            ),
            children: [
              {
                index: true,
                element: lazyPage(() => import('@/features/library/upload-page').then((m) => ({ default: m.LibraryUploadPage }))),
              },
            ],
          },
          {
            path: ':id',
            element: lazyPage(() =>
              import('@/features/library/textbook-details-page').then((m) => ({ default: m.TextbookDetailsPage })),
            ),
          },
        ],
      },
      {
        path: 'cards',
        element: (
          <RequireAuthOutlet
            title="Карточки — только для авторизованных"
            description="Войдите, чтобы генерировать карточки из своих материалов и повторять их по SM-2 — прогресс повторения привязан к аккаунту."
          />
        ),
        children: [
          {
            index: true,
            element: lazyPage(() => import('@/features/cards/cards-home-page').then((m) => ({ default: m.CardsHomePage }))),
          },
          {
            path: 'create',
            element: lazyPage(() => import('@/features/cards/create-task-page').then((m) => ({ default: m.CreateCardTaskPage }))),
          },
          {
            path: 'review',
            element: lazyPage(() => import('@/features/cards/review-page').then((m) => ({ default: m.ReviewPage }))),
          },
          {
            path: 'tasks/:id',
            element: lazyPage(() => import('@/features/cards/task-detail-page').then((m) => ({ default: m.TaskDetailPage }))),
          },
        ],
      },
      {
        path: 'navigator',
        element: lazyPage(() => import('@/features/navigator/navigator-page').then((m) => ({ default: m.NavigatorPage }))),
      },
      {
        path: 'forum',
        // Форум целиком закрыт для гостя - в отличие от Library/Map, ни у
        // одного /threads или /comments эндпоинта нет security: [] в
        // openapi.yaml (как и у Cards, см. подтверждённое решение "как в ТЗ").
        element: (
          <RequireAuthOutlet
            title="Треды — только для авторизованных"
            description="Войдите, чтобы читать и создавать треды, комментировать и ставить реакции."
          />
        ),
        children: [
          {
            index: true,
            element: lazyPage(() => import('@/features/forum/forum-feed-page').then((m) => ({ default: m.ForumFeedPage }))),
          },
          {
            path: 'create',
            element: lazyPage(() => import('@/features/forum/create-thread-page').then((m) => ({ default: m.CreateThreadPage }))),
          },
          {
            path: ':id',
            element: lazyPage(() => import('@/features/forum/thread-detail-page').then((m) => ({ default: m.ThreadDetailPage }))),
          },
        ],
      },
      {
        path: 'profile',
        element: (
          <RequireAuthOutlet title="Профиль" description="Войдите, чтобы посмотреть и отредактировать профиль." />
        ),
        children: [
          {
            index: true,
            element: lazyPage(() => import('@/features/profile/profile-page').then((m) => ({ default: m.ProfilePage }))),
          },
        ],
      },
      {
        path: 'admin',
        element: <RequireRoleOutlet roles={[UserRole.moderator, UserRole.admin]} />,
        children: [
          {
            element: lazyPage(() => import('@/features/admin/admin-layout').then((m) => ({ default: m.AdminLayout }))),
            children: [
              { index: true, element: <Navigate to="/admin/reports" replace /> },
              {
                path: 'reports',
                element: lazyPage(() => import('@/features/admin/reports-page').then((m) => ({ default: m.AdminReportsPage }))),
              },
              {
                path: 'users',
                element: lazyPage(() => import('@/features/admin/users-page').then((m) => ({ default: m.AdminUsersPage }))),
              },
              {
                path: 'stats',
                element: lazyPage(() => import('@/features/admin/stats-page').then((m) => ({ default: m.AdminStatsPage }))),
              },
              {
                path: 'audit-log',
                element: lazyPage(() => import('@/features/admin/audit-log-page').then((m) => ({ default: m.AdminAuditLogPage }))),
              },
            ],
          },
        ],
      },
    ],
  },
  { path: '*', element: <Navigate to="/library" replace /> },
])
