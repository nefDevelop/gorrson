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
    bottom: 40px;
    left: 50%;
    transform: translateX(-50%);
    z-index: 9999;
    display: flex;
    flex-direction: column-reverse;
    gap: 8px;
    pointer-events: none;
  }

  .toast {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 6px 16px;
    min-height: 48px;
    border-radius: var(--radius-m);
    background: var(--md-surface-variant);
    border: 1px solid var(--md-outline);
    box-shadow: 0 4px 12px rgba(0,0,0,0.25);
    font: var(--m3-body-medium);
    pointer-events: all;
    animation: toast-in 0.4s cubic-bezier(0, 0, 0.2, 1);
    min-width: 320px;
    max-width: 560px;
    color: var(--md-on-surface);
  }

  .toast-success { 
    background: var(--md-primary-container); 
    border-color: var(--md-primary); 
    color: var(--md-on-primary-container);
  }
  .toast-error { 
    background: #ffecf0; 
    border-color: #ff5252; 
    color: #8c1d1d; 
  }
  .toast-info { 
    background: var(--md-surface); 
    border-color: var(--md-outline);
  }

  .toast-icon {
    flex-shrink: 0;
    opacity: 0.8;
  }

  .toast-msg { 
    flex: 1; 
  }

  .toast-close {
    background: transparent;
    border: none;
    cursor: pointer;
    color: inherit;
    padding: 8px;
    display: flex;
    align-items: center;
    border-radius: 50%;
    opacity: 0.6;
    transition: opacity 0.2s;
  }
  .toast-close:hover { opacity: 1; background: rgba(0,0,0,0.05); }

  @keyframes toast-in {
    from { opacity: 0; transform: translateY(20px) scale(0.95); }
    to   { opacity: 1; transform: translateY(0) scale(1); }
  }

  @media (max-width: 600px) {
    .toast-container {
      width: 100%;
      padding: 0 16px;
      box-sizing: border-box;
    }
    .toast {
      min-width: 0;
      width: 100%;
    }
  }
</style>
