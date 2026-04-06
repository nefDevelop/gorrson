<script>
  import { t } from './i18n.js';
  import { addFeed } from './api.js';
  import { Plus, X, Globe, MessageSquare, Link } from 'lucide-svelte';

  /** @type {(title: string) => void} */
  export let onSave = (_title) => {};
  export let onCancel = () => {};

  let activeTab = 'rss';
  let input = "";
  let category = "";
  let loading = false;
  let error = "";

  async function handleAdd() {
    if (!input) return;
    loading = true;
    error = "";
    
    let url = input;
    
    if (activeTab === 'mastodon') {
      const parts = input.replace(/^@/, '').split('@');
      if (parts.length === 2) {
        url = `https://${parts[1]}/@${parts[0]}.rss`;
      } else {
        error = "Format: @user@instance";
        loading = false;
        return;
      }
    } else if (activeTab === 'reddit') {
      const sub = input.replace(/^\/?r\//, '');
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

    <div class="tabs">
      <button class="tab {activeTab === 'rss' ? 'active' : ''}" on:click={() => activeTab = 'rss'}>
        <Globe size={14} /> RSS
      </button>
      <button class="tab {activeTab === 'mastodon' ? 'active' : ''}" on:click={() => activeTab = 'mastodon'}>
        Mastodon
      </button>
      <button class="tab {activeTab === 'reddit' ? 'active' : ''}" on:click={() => activeTab = 'reddit'}>
        Reddit
      </button>
    </div>

    <div class="input-area">
      <div class="field">
        <label for="feed-input">
          {#if activeTab === 'rss'}URL (RSS/Atom)
          {:else if activeTab === 'mastodon'}Mastodon User (@user@instance)
          {:else}Subreddit (/r/subreddit)
          {/if}
        </label>
        <input 
          id="feed-input"
          type="text" 
          placeholder={activeTab === 'rss' ? 'https://example.com/rss' : activeTab === 'mastodon' ? '@user@instance' : 'technology'} 
          bind:value={input}
          disabled={loading}
          on:keydown={(e) => e.key === 'Enter' && handleAdd()}
        />
        {#if error}
          <div class="error-msg">{error}</div>
        {/if}
      </div>
      <div class="field" style="margin-top: 16px;">
        <label for="category-input">Category <span style="opacity:0.5">(optional)</span></label>
        <input id="category-input" type="text" placeholder="e.g. Tech, Sports, News" bind:value={category} disabled={loading} />
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
  .modal { background: var(--md-surface); width: 100%; max-width: 440px; border-radius: var(--radius-xl); border: 1px solid var(--md-outline); box-shadow: 0 24px 64px rgba(0, 0, 0, 0.6); }
  .modal-header { padding: 20px 24px; display: flex; align-items: center; justify-content: space-between; border-bottom: 1px solid var(--md-outline); }
  .header-left { display: flex; align-items: center; gap: 12px; }
  .header-left h2 { font: var(--m3-headline-small); margin: 0; color: var(--md-on-surface); }
  .close-btn { background: transparent; border: none; color: var(--md-on-surface-variant); cursor: pointer; padding: 4px; border-radius: var(--radius-full); }
  .tabs { display: flex; padding: 12px 24px; gap: 8px; background: var(--md-background); border-bottom: 1px solid var(--md-outline); }
  .tab { background: transparent; border: 1px solid var(--md-outline); padding: 6px 16px; border-radius: var(--radius-full); font: var(--m3-label-large); color: var(--md-on-surface-variant); cursor: pointer; display: flex; align-items: center; gap: 6px; }
  .tab.active { background: var(--md-primary-container); color: var(--md-on-primary-container); border-color: var(--md-primary); }
  .input-area { padding: 32px 24px; }
  .field label { display: block; font: var(--m3-label-medium); color: var(--md-on-surface-variant); margin-bottom: 8px; }
  .field input { width: 100%; background: var(--md-surface-variant); border: 1px solid var(--md-outline); padding: 14px 16px; border-radius: var(--radius-m); color: var(--md-on-surface); font: var(--m3-body-large); outline: none; }
  .error-msg { margin-top: 12px; color: #ff5252; font: var(--m3-label-medium); }
  .modal-footer { padding: 16px 24px; display: flex; justify-content: flex-end; gap: 12px; background: var(--md-background); border-top: 1px solid var(--md-outline); }
  .btn-text { background: transparent; border: none; color: var(--md-on-surface-variant); padding: 10px 20px; cursor: pointer; font: var(--m3-label-large); font-weight: 700; }
  .btn-primary { background: var(--md-primary); color: var(--md-on-primary); border: none; padding: 10px 24px; border-radius: var(--radius-m); font: var(--m3-label-large); font-weight: 700; cursor: pointer; }
  .animate-in { animation: modalIn 0.2s ease-out; }
  @keyframes modalIn { from { opacity: 0; transform: scale(0.95); } to { opacity: 1; transform: scale(1); } }
</style>
