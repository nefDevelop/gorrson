<script>
  import { toasts } from './toast.js';
  import { CheckCircle, XCircle, Info, X } from 'lucide-svelte';

  const icons = { success: CheckCircle, error: XCircle, info: Info };

  function dismiss(id) {
    toasts.update(t => t.filter(x => x.id !== id));
  }
</script>

<div class="toast-container">
  {#each $toasts as toast (toast.id)}
    <div class="toast toast-{toast.type}" role="alert">
      <svelte:component this={icons[toast.type]} size={18} class="toast-icon" />
      <span class="toast-msg">{toast.message}</span>
      <button class="toast-close" on:click={() => dismiss(toast.id)}>
        <X size={14} />
      </button>
    </div>
  {/each}
</div>

<style>
  .toast-container {
    position: fixed;
    bottom: 32px;
    right: 32px;
    z-index: 9999;
    display: flex;
    flex-direction: column;
    gap: 10px;
    pointer-events: none;
  }

  .toast {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 14px 18px;
    border-radius: var(--radius-m);
    backdrop-filter: blur(12px);
    border: 1px solid var(--md-outline);
    box-shadow: 0 8px 32px rgba(0,0,0,0.4);
    font: var(--m3-body-medium);
    pointer-events: all;
    animation: slideIn 0.25s cubic-bezier(0.2, 0, 0, 1);
    min-width: 280px;
    max-width: 440px;
  }

  .toast-success { background: rgba(20, 40, 20, 0.95); border-color: #4caf50; }
  .toast-error   { background: rgba(40, 15, 15, 0.95); border-color: #ff5252; }
  .toast-info    { background: rgba(10, 20, 40, 0.95); border-color: var(--md-primary); }

  .toast-msg { flex: 1; color: var(--md-on-surface); }

  .toast-close {
    background: transparent;
    border: none;
    cursor: pointer;
    color: var(--md-on-surface-variant);
    padding: 2px;
    display: flex;
    align-items: center;
    border-radius: var(--radius-s);
  }
  .toast-close:hover { background: rgba(255,255,255,0.1); }

  @keyframes slideIn {
    from { opacity: 0; transform: translateX(40px); }
    to   { opacity: 1; transform: translateX(0); }
  }
</style>
