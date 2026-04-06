import { writable } from 'svelte/store';

export const toasts = writable([]);

let id = 0;

/** @param {'success'|'error'|'info'} type */
export function toast(message, type = 'info', duration = 4000) {
  const toastId = ++id;
  toasts.update(t => [...t, { id: toastId, message, type }]);
  setTimeout(() => {
    toasts.update(t => t.filter(x => x.id !== toastId));
  }, duration);
}

export const success = (msg) => toast(msg, 'success');
export const error   = (msg) => toast(msg, 'error');
export const info    = (msg) => toast(msg, 'info');
