<script>
  import { onMount } from 'svelte';
  import { fetchFilters, addFilter, deleteFilter } from './api.js';
  import { t } from './i18n.js';
  import { Trash2, Plus, Shield, Search, Globe } from 'lucide-svelte';

  /** @type {Array<any>} */
  export let feeds = [];
  export let headless = false;

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

<div class="view-container {headless ? 'headless' : ''}">
  {#if !headless}
    <header class="header-bar">
      <div class="header-left">
        <Shield size={20} class="header-icon" />
        <h1 class="title">{$t('filters')}</h1>
      </div>
    </header>

    <!-- Sharp edge fix for unibody rounding -->
    <div class="corner-bg"></div>
    <div class="corner-arc"></div>
  {/if}

  <div class="content-scroll scroll-area {headless ? 'headless-scroll' : ''}">
    <section class="add-filter-section">
      <div class="inline-form">
        <input class="modern-input pattern-input" type="text" bind:value={newFilter.pattern} placeholder="Pattern (e.g. crypto|spam)" />
        
        <select class="modern-select" bind:value={newFilter.action}>
          <option value="HIDE">Hide (No sale)</option>
          <option value="SILENT">Silent (Desactivada)</option>
          <option value="HIGHLIGHT">Highlight (Destacar)</option>
          <option value="IMPORTANT">Important (Prioridad)</option>
          <option value="NOTIFY">Notify (Campana)</option>
          <option value="REPLACE">{$t('replace')}</option>
        </select>

        {#if newFilter.action === 'REPLACE'}
          <input class="modern-input replacement-input" type="text" bind:value={newFilter.replacement} placeholder="Replacement" />
        {/if}

        <select class="modern-select" bind:value={newFilter.target}>
          <option value="title">Title</option>
          <option value="content">Content</option>
          <option value="both">Both</option>
        </select>

        <select class="modern-select" bind:value={newFilter.feed_id}>
          <option value="">Global</option>
          {#each feeds as feed}
            <option value={feed.id}>{feed.title}</option>
          {/each}
        </select>

        <button class="btn-primary" on:click={handleAdd} disabled={!newFilter.pattern}>
          <Plus size={18} />
          <span>Add</span>
        </button>
      </div>
      <div class="field-help">
        <strong>Tip:</strong> Use <code>(?i)word</code> for case-insensitive or <code>word1|word2</code> for multiple matches.
      </div>
    </section>

    {#if loading}
      <div class="state-msg">Loading filters...</div>
    {:else if (filters || []).length === 0}
      <div class="state-msg">{$t('no_filters')}</div>
    {:else}
      <div class="filters-list">
        {#each (filters || []) as filter}
          <div class="filter-item">
            <div class="filter-main">
              <code class="pattern">{filter.pattern}</code>
              <div class="filter-meta">
                <span class="tag action-{filter.action.toLowerCase()}">{filter.action}</span>
                {#if filter.action === 'REPLACE'}
                  <span class="replacement-arrow">→</span>
                  <code class="replacement-text">{filter.replacement}</code>
                {/if}
                <span class="tag target">{filter.target}</span>
                <span class="tag feed-badge {filter.feed_id ? 'specific' : 'global'}">
                  {filter.feed_id ? (feeds.find(f => f.id === filter.feed_id)?.title || 'Feed') : 'Global'}
                </span>
              </div>
            </div>
            <button class="delete-btn" on:click={() => handleDelete(filter.id)} title="Delete filter">
              <Trash2 size={18} />
            </button>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>

<style>
  .view-container { 
    display: flex; 
    flex-direction: column; 
    height: 100%; 
    background: var(--md-surface); 
    flex: 1; 
    width: 100%; 
    position: relative;
    overflow: hidden;
  }
  .view-container.headless { background: transparent; }
  .header-bar { 
    height: var(--header-height); 
    padding: 0 40px; 
    display: flex; 
    align-items: center; 
    justify-content: space-between; 
    background: var(--md-surface); 
    box-sizing: border-box;
    position: relative;
    z-index: 20;
  }

  /* Eraser for the straight line crossing the curved gap */
  .header-bar::before {
    content: "";
    position: absolute;
    left: 0px; 
    bottom: -1px; 
    width: 32px; 
    height: 2px;
    background: var(--md-surface);
    z-index: 15;
  }

  /* Solid background to color the gap outside the content radius */
  .corner-bg {
    position: absolute;
    top: var(--header-height);
    left: 0;
    width: 32px;
    height: 32px;
    background: var(--md-surface);
    z-index: 0;
  }

  /* Overlaid pure arc to draw the crisp 1px unibody curve without clipping */
  .corner-arc {
    position: absolute;
    top: var(--header-height);
    left: 0;
    width: 32px;
    height: 32px;
    border-top-left-radius: 32px;
    border-top: 1px solid var(--md-outline);
    border-left: 1px solid var(--md-outline);
    z-index: 20;
    pointer-events: none;
  }

  .header-left { display: flex; align-items: center; gap: 12px; }
  .title { font: var(--m3-headline-small); font-weight: 800; color: var(--md-on-surface); }

  .content-scroll { 
    padding: 40px; 
    flex: 1; 
    overflow-y: auto; 
    background: var(--md-background);
    border-top-left-radius: 32px;
    border-left: 1px solid var(--md-outline);
    position: relative;
    z-index: 10;
  }
  .content-scroll.headless-scroll {
    padding: 0;
    background: transparent;
    border: none;
    overflow: visible;
  }
  
  .filters-list { display: flex; flex-direction: column; gap: 12px; }
  .filter-item {
    background: var(--md-surface);
    border: 1px solid var(--md-outline);
    border-radius: var(--radius-xl);
    padding: 16px 24px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    transition: all 0.2s;
  }
  .filter-item:hover { border-color: var(--md-primary); transform: translateX(4px); }
  
  .filter-main { display: flex; flex-direction: column; gap: 8px; overflow: hidden; }
  .filter-meta { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }

  .add-filter-section { margin-bottom: 32px; }
  .inline-form { display: flex; gap: 12px; flex-wrap: wrap; background: var(--md-surface); padding: 16px; border-radius: var(--radius-xl); border: 1px solid var(--md-outline); align-items: center; }
  
  .modern-input, .modern-select {
    background: var(--md-background);
    border: 1px solid var(--md-outline);
    padding: 8px 16px;
    border-radius: var(--radius-full);
    color: var(--md-on-surface);
    font: var(--m3-body-medium);
    outline: none;
    transition: all 0.2s;
    height: 40px;
  }
  .modern-input:focus, .modern-select:focus { border-color: var(--md-primary); }
  .pattern-input { flex: 2; min-width: 200px; }
  .replacement-input { flex: 1; min-width: 150px; }

  .btn-primary { 
    background: var(--md-primary); 
    color: var(--md-on-primary); 
    border: none; 
    padding: 0 24px; 
    border-radius: var(--radius-full); 
    font: var(--m3-label-large); 
    font-weight: 800; 
    cursor: pointer; 
    display: flex; 
    align-items: center; 
    gap: 8px;
    height: 40px;
    transition: all 0.2s;
  }
  .btn-primary:hover { transform: translateY(-1px); box-shadow: 0 4px 12px var(--md-primary-container); }
  .btn-primary:disabled { opacity: 0.5; cursor: not-allowed; transform: none; box-shadow: none; }

  .pattern { font-family: var(--font-mono); color: var(--md-primary); font-size: 15px; font-weight: 800; }
  .replacement-text { font-family: var(--font-mono); color: #4caf50; font-weight: 700; }
  .replacement-arrow { color: var(--md-on-surface-variant); opacity: 0.5; font-weight: 900; }

  .tags { display: flex; gap: 4px; }
  .tag { font: var(--m3-label-small); font-weight: 800; padding: 2px 8px; border-radius: var(--radius-full); text-transform: uppercase; letter-spacing: 0.5px; border: 1px solid transparent; }
  .target { background: var(--md-outline); color: var(--md-on-surface-variant); }
  .feed-badge { max-width: 150px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .feed-badge.global { border-color: var(--md-primary); color: var(--md-primary); }
  .feed-badge.specific { background: var(--md-secondary-container); color: var(--md-on-secondary-container); }
  
  .tag.action-hide, .tag.action-block { background: #ff525222; color: #ff5252; border-color: #ff525244; }
  .tag.action-replace { background: #4caf5022; color: #4caf50; border-color: #4caf5044; }
  .tag.action-silent { background: var(--md-outline); color: var(--md-on-surface-variant); opacity: 0.6; }
  .tag.action-highlight { background: var(--md-secondary-container); color: var(--md-secondary); border-color: var(--md-secondary); }
  .tag.action-important { background: #ffd70033; color: #b8860b; border-color: #ffd70066; font-weight: 900; }
  .tag.action-notify { background: #e91e6322; color: #e91e63; border-color: #e91e6344; }

  .delete-btn { background: transparent; border: none; color: var(--md-on-surface-variant); cursor: pointer; padding: 10px; border-radius: var(--radius-full); transition: all 0.2s; display: flex; align-items: center; justify-content: center; }
  .delete-btn:hover { background: #ff525222; color: #ff5252; }

  .field-help { font: var(--m3-label-small); color: var(--md-on-surface-variant); margin-top: 12px; opacity: 0.7; padding-left: 16px; }
  .field-help code { background: var(--md-outline); padding: 2px 6px; border-radius: 4px; color: var(--md-primary); }
  .state-msg { text-align: center; margin-top: 80px; color: var(--md-on-surface-variant); font: var(--m3-body-large); opacity: 0.6; }
</style>
