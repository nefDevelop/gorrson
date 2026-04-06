<script>
  import { onMount } from 'svelte';
  import { fetchFilters, addFilter, deleteFilter } from './api.js';
  import { t } from './i18n.js';
  import { Trash2, Plus, Shield, Search, Globe } from 'lucide-svelte';

  /** @type {Array<any>} */
  export let feeds = [];

  let filters = [];
  let loading = true;
  let showAddModal = false;

  let newFilter = {
    pattern: "",
    action: "BLOCK",
    target: "both",
    replacement: "",
    feed_id: ""
  };

  async function loadFilters() {
    try {
      loading = true;
      filters = (await fetchFilters()) || [];
    } catch (e) {
      console.error(e);
    } finally {
      loading = false;
    }
  }

  async function handleAdd() {
    if (!newFilter.pattern) return;
    const ok = await addFilter(newFilter);
    if (ok) {
      showAddModal = false;
      newFilter = { pattern: "", action: "BLOCK", target: "both", replacement: "", feed_id: "" };
      loadFilters();
    }
  }

  async function handleDelete(id) {
    if (confirm('Delete this filter?')) {
      const ok = await deleteFilter(id);
      if (ok) loadFilters();
    }
  }

  onMount(loadFilters);
</script>

<div class="view-container">
  <header class="header-bar">
    <div class="header-left">
      <Shield size={20} class="header-icon" />
      <h1 class="title">{$t('filters')}</h1>
    </div>
    <button class="add-btn-primary" on:click={() => showAddModal = true}>
      <Plus size={18} />
      <span>{$t('add_filter')}</span>
    </button>
  </header>

  <div class="content-scroll scroll-area">
    {#if loading}
      <div class="state-msg">Loading filters...</div>
    {:else if (filters || []).length === 0}
      <div class="state-msg">{$t('no_filters')}</div>
    {:else}
      <div class="filters-grid">
        {#each (filters || []) as filter}
          <div class="filter-card">
            <div class="filter-info">
              <div class="pattern-row">
                <span class="label">{$t('pattern')}:</span>
                <code class="pattern">{filter.pattern}</code>
              </div>
              <div class="tags">
                <span class="tag action-{filter.action.toLowerCase()}">{filter.action}</span>
                <span class="tag target">{filter.target}</span>
                {#if filter.feed_id}
                  <span class="tag feed-specific">
                    {feeds.find(f => f.id === filter.feed_id)?.title || 'Feed'}
                  </span>
                {:else}
                  <span class="tag global">Global</span>
                {/if}
              </div>
            </div>
            <button class="delete-btn" on:click={() => handleDelete(filter.id)}>
              <Trash2 size={16} />
            </button>
          </div>
        {/each}
      </div>
    {/if}
  </div>

  {#if showAddModal}
    <div class="modal-overlay">
      <div class="modal">
        <h3>{$t('add_filter')}</h3>
        
        <div class="form-group">
          <label for="pattern-input">{$t('pattern')}</label>
          <input id="pattern-input" type="text" bind:value={newFilter.pattern} placeholder="e.g. crypto|spam" />
          <div class="field-help">
            <strong>Examples:</strong> <code>crypto|nft</code> (block multiple), <code>(?i)linux</code> (case-insensitive)
          </div>
        </div>

        <div class="form-group">
          <label for="feed-select">Apply To</label>
          <select id="feed-select" bind:value={newFilter.feed_id}>
            <option value="">Global (All feeds)</option>
            {#each feeds as feed}
              <option value={feed.id}>{feed.title}</option>
            {/each}
          </select>
        </div>

        <div class="form-row">
          <div class="form-group">
            <label for="action-select">{$t('action')}</label>
            <select id="action-select" bind:value={newFilter.action}>
              <option value="BLOCK">{$t('block')}</option>
              <option value="REPLACE">{$t('replace')}</option>
            </select>
          </div>
          <div class="form-group">
            <label for="target-select">{$t('target')}</label>
            <select id="target-select" bind:value={newFilter.target}>
              <option value="title">Title</option>
              <option value="content">Content</option>
              <option value="both">Both</option>
            </select>
          </div>
        </div>

        {#if newFilter.action === 'REPLACE'}
          <div class="form-group">
            <label for="replacement-input">{$t('replacement')}</label>
            <input id="replacement-input" type="text" bind:value={newFilter.replacement} />
          </div>
        {/if}

        <div class="modal-actions">
          <button class="btn-text" on:click={() => showAddModal = false}>{$t('cancel')}</button>
          <button class="btn-primary" on:click={handleAdd}>{$t('save')}</button>
        </div>
      </div>
    </div>
  {/if}
</div>

<style>
  .view-container { display: flex; flex-direction: column; height: 100%; background: var(--md-background); flex: 1; width: 100%; }
  .header-bar { 
    height: var(--header-height); 
    padding: 0 24px; 
    display: flex; 
    align-items: center; 
    justify-content: space-between; 
    border-bottom: 1px solid var(--md-outline); 
    border-left: 1px solid var(--md-outline);
    background: var(--md-surface); 
    box-sizing: border-box;
  }
  .header-left { display: flex; align-items: center; gap: 12px; }
  .title { font: var(--m3-headline-small); font-weight: 700; color: var(--md-on-surface); }

  .content-scroll { padding: 24px; flex: 1; overflow-y: auto; }
  .filters-grid { display: grid; gap: 12px; grid-template-columns: repeat(auto-fill, minmax(320px, 1fr)); }
  
  .filter-card {
    background: var(--md-surface-variant);
    border: 1px solid var(--md-outline);
    border-radius: var(--radius-m);
    padding: 16px;
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
  }

  .pattern-row { margin-bottom: 12px; }
  .label { font: var(--m3-label-small); opacity: 0.6; margin-right: 8px; }
  .pattern { font-family: var(--font-mono); color: var(--md-primary); background: rgba(168, 199, 255, 0.1); padding: 2px 6px; border-radius: 4px; font-size: 13px; }

  .tags { display: flex; gap: 8px; flex-wrap: wrap; }
  .tag { font: var(--m3-label-small); font-weight: 700; padding: 2px 8px; border-radius: 4px; text-transform: uppercase; }
  .target { background: var(--md-outline); color: var(--md-on-surface-variant); }
  .global { border: 1px solid var(--md-primary); color: var(--md-primary); }
  .feed-specific { background: var(--md-secondary-container); color: var(--md-on-secondary-container); }
  .action-block { background: #ff525233; color: #ff5252; }
  .action-replace { background: #4caf5033; color: #4caf50; }

  .delete-btn { background: transparent; border: none; color: var(--md-on-surface-variant); cursor: pointer; padding: 8px; border-radius: var(--radius-s); transition: all 0.2s; }
  .delete-btn:hover { background: #ff525222; color: #ff5252; }

  .add-btn-primary { background: var(--md-primary); color: var(--md-on-primary); border: none; padding: 8px 16px; border-radius: var(--radius-m); font: var(--m3-label-large); font-weight: 700; cursor: pointer; display: flex; align-items: center; gap: 8px; }

  .modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.7); display: flex; align-items: center; justify-content: center; z-index: 2000; }
  .modal { background: var(--md-surface); border: 1px solid var(--md-outline); padding: 32px; border-radius: var(--radius-xl); width: 100%; max-width: 480px; box-shadow: 0 20px 40px rgba(0,0,0,0.4); }
  .modal h3 { font: var(--m3-headline-small); margin-bottom: 24px; color: var(--md-on-surface); }
  
  .form-group { margin-bottom: 20px; }
  .form-group label { display: block; font: var(--m3-label-medium); margin-bottom: 8px; opacity: 0.8; }
  .field-help { font: var(--m3-label-small); color: var(--md-on-surface-variant); margin-top: 8px; opacity: 0.7; }
  .field-help code { background: var(--md-outline); padding: 2px 4px; border-radius: 4px; color: var(--md-primary); }
  .form-group input, .form-group select { width: 100%; padding: 12px; border-radius: var(--radius-s); background: var(--md-surface-variant); border: 1px solid var(--md-outline); color: var(--md-on-surface); font: var(--m3-body-medium); }

  .form-row { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }

  .modal-actions { display: flex; justify-content: flex-end; gap: 12px; margin-top: 32px; }
  .btn-text { background: transparent; border: none; color: var(--md-on-surface-variant); padding: 10px 20px; cursor: pointer; font: var(--m3-label-large); }
  .btn-primary { background: var(--md-primary); color: var(--md-on-primary); border: none; padding: 10px 24px; border-radius: var(--radius-m); font: var(--m3-label-large); font-weight: 700; cursor: pointer; }

  .state-msg { text-align: center; margin-top: 60px; color: var(--md-on-surface-variant); font: var(--m3-body-large); opacity: 0.6; }
</style>
