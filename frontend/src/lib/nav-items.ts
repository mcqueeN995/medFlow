import { BookOpen, Layers, MapPin, MessageSquare, User } from 'lucide-react'

export const navItems = [
  { to: '/library', label: 'Библиотека', icon: BookOpen },
  { to: '/cards', label: 'Карточки', icon: Layers },
  { to: '/navigator', label: 'Навигатор', icon: MapPin },
  { to: '/forum', label: 'Треды', icon: MessageSquare },
  { to: '/profile', label: 'Профиль', icon: User },
] as const
