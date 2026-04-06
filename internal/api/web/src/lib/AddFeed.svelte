<script>
  import { t } from './i18n.js';
  import { addFeed } from './api.js';
  import { Plus, X, Globe, MessageSquare, Link } from 'lucide-svelte';

  /** @type {(title: string) => void} */
  export let onSave = (_title) => {};
  export let onCancel = () => {};

  let input = "";
  let category = "";
  let loading = false;
  let error = "";

  // Auto-detection logic
  $: detectedType = (() => {
    const val = input.trim();
    if (val.startsWith('r/') || val.startsWith('/r/')) return 'reddit';
    if (val.startsWith('@')) return 'mastodon';
    if (val.startsWith('http') || val.includes('.')) return 'rss';
    return 'rss'; // Default
  })();

  async function handleAdd() {
    if (!input) return;
    loading = true;
    error = "";
    
    let url = input.trim();
    
    if (detectedType === 'mastodon') {
      const parts = url.replace(/^@/, '').split('@');
      if (parts.length === 2) {
        url = `https://${parts[1]}/@${parts[0]}.rss`;
      } else {
        error = "Formato Mastodon: @usuario@instancia";
        loading = false;
        return;
      }
    } else if (detectedType === 'reddit') {
      const sub = url.replace(/^\/?r\//, '');
      url = `https://www.reddit.com/r/${sub}.rss`;
    }

    try {
      const data = await addFeed(url, category);
      onSave(data?.title || url);
    } catch (e) {
      error = e.message.trim();
    } finally {
      loading = false;
    }
  }
</script>

<div class="modal-overlay" on:click|self={onCancel} role="presentation">
  <div class="modal animate-in">
    <header class="modal-header">
      <div class="header-left">
        <Plus size={20} color="var(--md-primary)" />
        <h2>{$t('add_feed')}</h2>
      </div>
      <button class="close-btn" on:click={onCancel}><X size={18} /></button>
    </header>

    <div class="input-area">
      <div class="field">
        <label for="feed-input">
          URL, Subreddit o Usuario
        </label>
        <div class="input-wrapper">
          <div class="detect-badge {detectedType}">
            {#if detectedType === 'reddit'}Reddit
            {:else if detectedType === 'mastodon'}Mastodon
            {:else if input.includes('.') }RSS/Web
            {:else}Auto
            {/if}
          </div>
          <input 
            id="feed-input"
            type="text" 
            placeholder="r/technology, @user@instance o https://..." 
            bind:value={input}
            disabled={loading}
            on:keydown={(e) => e.key === 'Enter' && handleAdd()}
          />
        </div>
        {#if error}
          <div class="error-msg">{error}</div>
        {/if}
      </div>
      <div class="field" style="margin-top: 24px;">
        <label for="category-input">Categoría <span style="opacity:0.5">(opcional)</span></label>
        <input id="category-input" type="text" placeholder="Noticias, Tech, Ocio..." bind:value={category} disabled={loading} />
      </div>
    </div>

    <footer class="modal-footer">
      <button class="btn-text" on:click={onCancel}>{$t('cancel')}</button>
      <button class="btn-primary" on:click={handleAdd} disabled={loading || !input}>
        {#if loading}Adding...{:else}{$t('save')}{/if}
      </button>
    </footer>
  </div>
</div>

<style>
  .modal-overlay { position: fixed; inset: 0; background: rgba(0, 0, 0, 0.6); display: flex; align-items: center; justify-content: center; z-index: 3000; backdrop-filter: blur(4px); }
  .modal { background: var(--md-surface); width: 100%; max-width: 460px; border-radius: var(--radius-xl); border: 1px solid var(--md-outline); box-shadow: 0 24px 64px rgba(0, 0, 0, 0.6); overflow: hidden; }
  .modal-header { padding: 24px; display: flex; align-items: center; justify-content: space-between; border-bottom: 1px solid var(--md-outline); }
  .header-left { display: flex; align-items: center; gap: 12px; }
  .header-left h2 { font: var(--m3-headline-small); margin: 0; color: var(--md-on-surface); font-weight: 700; }
  .close-btn { background: transparent; border: none; color: var(--md-on-surface-variant); cursor: pointer; padding: 4px; border-radius: var(--radius-full); }
  
  .input-area { padding: 32px 24px; }
  .field label { display: block; font: var(--m3-label-medium); color: var(--md-on-surface-variant); margin-bottom: 8px; font-weight: 600; }
  
  .input-wrapper { position: relative; display: flex; align-items: center; }
  .detect-badge { position: absolute; right: 12px; font: var(--m3-label-small); padding: 4px 10px; border-radius: var(--radius-s); background: var(--md-surface-variant); color: var(--md-on-surface-variant); font-weight: 800; text-transform: uppercase; pointer-events: none; transition: all 0.2s; }
  .detect-badge.reddit { background: #FF4500; color: white; }
  .detect-badge.mastodon { background: #6364FF; color: white; }
  .detect-badge.rss { background: var(--md-primary-container); color: var(--md-on-primary-container); }

  .field input { width: 100%; background: var(--md-surface-variant); border: 1px solid var(--md-outline); padding: 14px 16px; border-radius: var(--radius-m); color: var(--md-on-surface); font: var(--m3-body-large); outline: none; transition: border-color 0.2s; }
  .field input:focus { border-color: var(--md-primary); }
  .error-msg { margin-top: 12px; color: #ff5252; font: var(--m3-label-medium); }
  .modal-footer { padding: 20px 24px; display: flex; justify-content: flex-end; gap: 12px; background: var(--md-surface-variant); border-top: 1px solid var(--md-outline); opacity: 0.9; }
  .btn-text { background: transparent; border: none; color: var(--md-on-surface-variant); padding: 10px 20px; cursor: pointer; font: var(--m3-label-large); font-weight: 700; }
  .btn-primary { background: var(--md-primary); color: var(--md-on-primary); border: none; padding: 10px 24px; border-radius: var(--radius-m); font: var(--m3-label-large); font-weight: 700; cursor: pointer; }
  .animate-in { animation: modalIn 0.2s ease-out; }
  @keyframes modalIn { from { opacity: 0; transform: scale(0.95); } to { opacity: 1; transform: scale(1); } }
</style>
