import { Navigate, createBrowserRouter } from 'react-router-dom'
import { AppShell } from '@/components/shared/app-shell'
import { RequireAuthOutlet } from '@/components/shared/require-auth-outlet'
import { LoginPage } from '@/features/auth/login-page'
import { RegisterPage } from '@/features/auth/register-page'
import { LibraryCatalogPage } from '@/features/library/catalog-page'
import { TextbookDetailsPage } from '@/features/library/textbook-details-page'
import { LibraryUploadPage } from '@/features/library/upload-page'
import { NavigatorPage } from '@/features/navigator/navigator-page'
import { CardsHomePage } from '@/features/cards/cards-home-page'
import { CreateCardTaskPage } from '@/features/cards/create-task-page'
import { TaskDetailPage } from '@/features/cards/task-detail-page'
import { ReviewPage } from '@/features/cards/review-page'
import { ForumFeedPage } from '@/features/forum/forum-feed-page'
import { CreateThreadPage } from '@/features/forum/create-thread-page'
import { ThreadDetailPage } from '@/features/forum/thread-detail-page'
import { ProfilePage } from '@/features/profile/profile-page'

export const router = createBrowserRouter([
  { path: '/login', element: <LoginPage /> },
  { path: '/register', element: <RegisterPage /> },
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
          { index: true, element: <LibraryCatalogPage /> },
          {
            path: 'upload',
            element: (
              <RequireAuthOutlet
                title="Загрузка материалов требует входа"
                description="Чтобы прикрепить свой PDF и подготовить из него карточки, войдите или зарегистрируйтесь."
              />
            ),
            children: [{ index: true, element: <LibraryUploadPage /> }],
          },
          { path: ':id', element: <TextbookDetailsPage /> },
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
          { index: true, element: <CardsHomePage /> },
          { path: 'create', element: <CreateCardTaskPage /> },
          { path: 'review', element: <ReviewPage /> },
          { path: 'tasks/:id', element: <TaskDetailPage /> },
        ],
      },
      { path: 'navigator', element: <NavigatorPage /> },
      {
        path: 'forum',
        // Форум целиком закрыт для гостя - в отличие от Library/Map, ни у
        // одного /threads или /comments эндпоинта нет security: [] в
        // openapi.yaml (как и у Cards, см. подтверждённое решение "как в ТЗ").
        element: (
          <RequireAuthOutlet
            title="Форум — только для авторизованных"
            description="Войдите, чтобы читать и создавать треды, комментировать и ставить реакции."
          />
        ),
        children: [
          { index: true, element: <ForumFeedPage /> },
          { path: 'create', element: <CreateThreadPage /> },
          { path: ':id', element: <ThreadDetailPage /> },
        ],
      },
      {
        path: 'profile',
        element: (
          <RequireAuthOutlet title="Профиль" description="Войдите, чтобы посмотреть и отредактировать профиль." />
        ),
        children: [{ index: true, element: <ProfilePage /> }],
      },
    ],
  },
  { path: '*', element: <Navigate to="/library" replace /> },
])
