// CardTask из API не содержит topic (см. openapi.yaml — поле есть только в
// CreateCardTaskRequest, эхо в ответе отсутствует). Тему задачи, которую сами
// создали в этой сессии, запоминаем на клиенте, чтобы список задач был
// читаемым — это не источник истины, только подсказка для UI.
const KEY = 'medflow-card-task-topics'

function readAll(): Record<string, string> {
  try {
    return JSON.parse(localStorage.getItem(KEY) ?? '{}')
  } catch {
    return {}
  }
}

export function rememberTaskTopic(taskId: string, topic: string) {
  const all = readAll()
  all[taskId] = topic
  localStorage.setItem(KEY, JSON.stringify(all))
}

export function getTaskTopic(taskId: string): string | undefined {
  return readAll()[taskId]
}
