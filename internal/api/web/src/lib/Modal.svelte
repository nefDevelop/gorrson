<script>
  import { t } from './i18n.js';
  import { X, AlertTriangle, AlertCircle, Info } from 'lucide-svelte';
  import { fade, fly } from 'svelte/transition';

  /** @type {string} */
  export let title = "";
  /** @type {string} */
  export let message = "";
  /** @type {'info'|'warning'|'danger'} */
  export let type = "info";
  /** @type {string} */
  export let confirmText = "";
  /** @type {string} */
  export let cancelText = "";
  /** @type {boolean} */
  export let show = false;

  export let onConfirm = () => {};
  export let onCancel = () => {};

  const icons = {
    info: Info,
    warning: AlertCircle,
    danger: AlertTriangle
  };

  $: themeClass = `type-${type}`;

  function handleCancel() {
    show = false;
    onCancel();
  }

  function handleConfirm() {
    onConfirm();
    show = false;
  }
</script>

{#if show}
  <div class="modal-overlay" on:click|self={handleCancel} transition:fade={{ duration: 200 }} role="presentation">
    <div class="modal" transition:fly={{ y: 20, duration: 300 }} class:type-danger={type === 'danger'}>
      <header class="modal-header">
        <div class="title-group">
          <svelte:component this={icons[type]} size={20} class="type-icon" />
          <h2>{title}</h2>
        </div>
        <button class="close-btn" on:click={handleCancel}><X size={18} /></button>
      </header>

      <div class="modal-body">
        {#if message}
          <p class="message">{message}</p>
        {/if}
        <slot />
      </div>

      <footer class="modal-footer">
        {#if cancelText || onCancel}
          <button class="btn-text" on:click={handleCancel}>
            {cancelText || $t('cancel')}
          </button>
        {/if}
        <button class="btn-action" on:click={handleConfirm}>
          {confirmText || $t('ok')}
        </button>
      </footer>
    </div>
  </div>
{/if}

<style>
  .modal-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.65);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 5000;
    backdrop-filter: blur(8px);
    padding: 20px;
  }

  .modal {
    background: var(--md-surface);
    width: 100%;
    max-width: 440px;
    border-radius: 28px;
    border: 1px solid var(--md-outline);
    box-shadow: 0 24px 64px rgba(0, 0, 0, 0.5);
    overflow: hidden;
    position: relative;
  }

  .modal-header {
    padding: 24px 24px 12px 24px;
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .title-group {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .title-group h2 {
    font: var(--m3-headline-small);
    margin: 0;
    color: var(--md-on-surface);
    font-weight: 700;
  }

  .title-group :global(svg) {
    color: var(--md-primary);
  }

  .type-danger .title-group :global(svg) {
    color: #ff5252;
  }

  .close-btn {
    background: var(--md-surface-variant);
    border: none;
    color: var(--md-on-surface-variant);
    cursor: pointer;
    padding: 8px;
    border-radius: 50%;
    display: flex;
    transition: all 0.2s;
  }

  .modal-body {
    padding: 0 24px 32px 24px;
    color: var(--md-on-surface-variant);
    font: var(--m3-body-large);
    line-height: 1.5;
  }

  .message {
    margin: 0;
    white-space: pre-wrap;
  }

  .modal-footer {
    padding: 16px 24px;
    display: flex;
    justify-content: flex-end;
    gap: 12px;
    background: var(--md-surface-variant);
    border-top: 1px solid var(--md-outline);
  }

  .btn-text {
    background: transparent;
    border: none;
    color: var(--md-on-surface-variant);
    padding: 10px 20px;
    cursor: pointer;
    font: var(--m3-label-large);
    font-weight: 700;
    border-radius: var(--radius-full);
  }

  .btn-action {
    background: var(--md-primary);
    color: var(--md-on-primary);
    border: none;
    padding: 10px 24px;
    border-radius: var(--radius-full);
    font: var(--m3-label-large);
    font-weight: 700;
    cursor: pointer;
    transition: all 0.2s;
  }

  .type-danger .btn-action {
    background: #ff5252;
    color: white;
  }

  .btn-action:hover {
    filter: brightness(1.1);
    box-shadow: 0 4px 12px rgba(0,0,0,0.2);
  }
</style>
