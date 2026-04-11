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

  // Validation and detection logic
  $: isValid = (() => {
    const val = input.trim();
    if (!val) return false;
    // Protocol based
    if (val.startsWith('http://') || val.startsWith('https://')) return true;
    // Special formats
    if (val.startsWith('r/') || val.startsWith('/r/')) return true;
    if (val.startsWith('@') && val.includes('@', 1)) return true;
    if (val.startsWith('#') && val.length > 1) return true;
    if (val.endsWith('.bsky.social')) return true;
    return false;
  })();

  $: detectedType = (() => {
    const val = input.trim();
    if (val.startsWith('r/') || val.startsWith('/r/')) return 'reddit';
    if (val.startsWith('@') || val.startsWith('#')) return 'mastodon';
    if (val.endsWith('.bsky.social') || val.includes('bsky.app')) return 'bluesky';
    if (val.startsWith('http')) return 'rss';
    return 'none';
  })();

  async function handleAdd() {
    if (!isValid || loading) return;
    loading = true;
    error = "";
    
    let url = input.trim();
    
    if (detectedType === 'mastodon') {
      if (url.startsWith('#')) {
        const instance = localStorage.getItem('gorrson-mastodon-instance') || 'mastodon.social';
        const tag = url.replace(/^#/, '');
        url = `https://${instance}/tags/${tag}.rss`;
      } else {
        const parts = url.replace(/^@/, '').split('@');
        if (parts.length === 2) {
          url = `https://${parts[1]}/@${parts[0]}.rss`;
        } else {
          error = "Formato Mastodon: @usuario@instancia o #hashtag";
          loading = false;
          return;
        }
      }
    } else if (detectedType === 'reddit') {
      const sub = url.replace(/^\/?r\//, '');
      url = `https://www.reddit.com/r/${sub}.rss`;
    } else if (detectedType === 'bluesky') {
      if (url.includes('bsky.app/profile/')) {
        if (!url.endsWith('/rss')) {
          url = url.replace(/\/$/, '') + '/rss';
        }
      } else if (url.endsWith('.bsky.social')) {
         url = `https://bsky.app/profile/${url}/rss`;
      }
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
            {:else if detectedType === 'bluesky'}Bluesky
            {:else if detectedType === 'rss'}RSS/URL
            {:else}Inválido
            {/if}
          </div>
          <input 
            id="feed-input"
            type="text" 
            placeholder="r/technology, @user@instance, #hashtag o https://..." 
            bind:value={input}
            disabled={loading}
            on:keydown={(e) => e.key === 'Enter' && isValid && handleAdd()}
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
      <button class="btn-primary" on:click={handleAdd} disabled={loading || !isValid}>
        {#if loading}Adding...{:else}{$t('save')}{/if}
      </button>
    </footer>
  </div>
</div>

<style>
  .modal-overlay { position: fixed; inset: 0; background: rgba(0, 0, 0, 0.7); display: flex; align-items: center; justify-content: center; z-index: 3000; backdrop-filter: blur(8px); }
  .modal { background: var(--md-surface); width: 100%; max-width: 480px; border-radius: 32px; border: 1px solid var(--md-outline); box-shadow: 0 24px 64px rgba(0, 0, 0, 0.4); overflow: hidden; }
  .modal-header { padding: 32px 32px 12px 32px; display: flex; align-items: center; justify-content: space-between; }
  .header-left { display: flex; align-items: center; gap: 12px; }
  .header-left h2 { font: var(--m3-headline-small); margin: 0; color: var(--md-on-surface); font-weight: 800; }
  .close-btn { background: var(--md-surface-variant); border: none; color: var(--md-on-surface-variant); cursor: pointer; padding: 10px; border-radius: var(--radius-full); display: flex; align-items: center; justify-content: center; transition: all 0.2s; }
  .close-btn:hover { background: var(--md-outline); color: var(--md-on-surface); }
  
  .input-area { padding: 24px 32px 40px 32px; }
  .field label { display: block; font: var(--m3-label-large); color: var(--md-primary); margin-bottom: 12px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px; }
  
  .input-wrapper { position: relative; display: flex; align-items: center; }
  .detect-badge { position: absolute; right: 12px; font: var(--m3-label-small); padding: 4px 12px; border-radius: var(--radius-full); background: var(--md-surface-variant); color: var(--md-on-surface-variant); font-weight: 800; text-transform: uppercase; pointer-events: none; transition: all 0.2s; border: 1px solid var(--md-outline); }
  .detect-badge.reddit { background: #FF4500; color: white; border: none; }
  .detect-badge.mastodon { background: #6364FF; color: white; border: none; }
  .detect-badge.bluesky { background: #0085FF; color: white; border: none; }
  .detect-badge.rss { background: var(--md-primary-container); color: var(--md-on-primary-container); border: 1px solid var(--md-primary); }
  .detect-badge.none { background: #555; opacity: 0.5; }

  .field input { width: 100%; background: var(--md-surface); border: 1px solid var(--md-outline); padding: 16px 20px; border-radius: var(--radius-full); color: var(--md-on-surface); font: var(--m3-body-large); outline: none; transition: all 0.2s; }
  .field input:focus { border-color: var(--md-primary); box-shadow: 0 0 0 2px var(--md-primary-container); }
  .error-msg { margin-top: 12px; color: #ff5252; font: var(--m3-label-medium); padding: 8px 16px; background: #ff525211; border-radius: var(--radius-m); }
  .modal-footer { padding: 24px 32px; display: flex; justify-content: flex-end; gap: 16px; background: var(--md-surface-variant); border-top: 1px solid var(--md-outline); }
  .btn-text { background: transparent; border: none; color: var(--md-on-surface-variant); padding: 12px 24px; cursor: pointer; font: var(--m3-label-large); font-weight: 700; border-radius: var(--radius-full); }
  .btn-text:hover { background: var(--md-outline); }
  .btn-primary { background: var(--md-primary); color: var(--md-on-primary); border: none; padding: 12px 32px; border-radius: var(--radius-full); font: var(--m3-label-large); font-weight: 700; cursor: pointer; transition: all 0.2s; }
  .btn-primary:hover { transform: translateY(-1px); box-shadow: 0 4px 12px var(--md-primary-container); }
  .btn-primary:disabled { opacity: 0.5; cursor: not-allowed; transform: none; box-shadow: none; }
  .animate-in { animation: modalIn 0.3s cubic-bezier(0.16, 1, 0.3, 1); }
  @keyframes modalIn { from { opacity: 0; transform: translateY(20px) scale(0.98); } to { opacity: 1; transform: translateY(0) scale(1); } }
</style>
