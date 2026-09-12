// Українські назви статусів, результатів заявок і подій — для всього UI.

export const STATUS_LABELS = {
  new: 'нове',
  seen: 'побачено',
  applied: 'подано заявку',
  won: 'виграно',
  lost: 'втрачено',
  archived: 'в архіві',
};

export const APP_RESULT_LABELS = {
  pending: 'очікує',
  accepted: 'прийнято',
  declined: 'відмовлено',
};

export const EVENT_LABELS = {
  new: 'нове замовлення',
  removed: 'видалено з джерела',
  status_changed: 'зміна статусу',
  applied: 'заявку подано',
};

export function statusLabel(status) {
  return STATUS_LABELS[status] || status;
}
