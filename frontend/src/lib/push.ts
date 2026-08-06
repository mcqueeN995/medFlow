import { deletePushUnsubscribe, postPushSubscribe } from '@/api/generated/medFlowAPI'

export function isPushSupported(): boolean {
  return 'serviceWorker' in navigator && 'PushManager' in window && 'Notification' in window
}

// urlBase64ToUint8Array - VAPID applicationServerKey приходит в
// URL-safe base64 (см. webpush.GenerateVAPIDKeys на бэкенде), PushManager
// ожидает Uint8Array.
function urlBase64ToUint8Array(base64: string): Uint8Array<ArrayBuffer> {
  const padding = '='.repeat((4 - (base64.length % 4)) % 4)
  const base64Safe = (base64 + padding).replace(/-/g, '+').replace(/_/g, '/')
  const raw = atob(base64Safe)
  const bytes = new Uint8Array(new ArrayBuffer(raw.length))
  for (let i = 0; i < raw.length; i++) bytes[i] = raw.charCodeAt(i)
  return bytes
}

export async function getCurrentPushSubscription(): Promise<PushSubscription | null> {
  if (!isPushSupported()) return null
  const registration = await navigator.serviceWorker.ready
  return registration.pushManager.getSubscription()
}

export async function subscribeToPush(): Promise<void> {
  const vapidPublicKey = import.meta.env.VITE_VAPID_PUBLIC_KEY
  if (!vapidPublicKey) throw new Error('VAPID public key is not configured')

  const permission = await Notification.requestPermission()
  if (permission !== 'granted') throw new Error('permission denied')

  const registration = await navigator.serviceWorker.ready
  const subscription = await registration.pushManager.subscribe({
    userVisibleOnly: true,
    applicationServerKey: urlBase64ToUint8Array(vapidPublicKey),
  })

  const json = subscription.toJSON()
  await postPushSubscribe({
    endpoint: json.endpoint!,
    p256dh: json.keys!.p256dh!,
    auth: json.keys!.auth!,
  })
}

export async function unsubscribeFromPush(): Promise<void> {
  const subscription = await getCurrentPushSubscription()
  if (!subscription) return
  await deletePushUnsubscribe({ endpoint: subscription.endpoint })
  await subscription.unsubscribe()
}
