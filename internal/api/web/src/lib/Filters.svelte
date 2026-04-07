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

  <!-- Sharp edge fix for unibody rounding -->
  <div class="corner-bg"></div>
  <div class="corner-arc"></div>

  <div class="content-scroll scroll-area">
    {#if loading}
      <div class="state-msg">Loading filters...</div>
    {:else if (filters || []).length === 0}
      <div class="state-msg">{$t('no_filters')}</div>
    {:else}
      <div class="filters-table-wrapper">
        <table class="filters-table">
          <thead>
            <tr>
              <th class="th-pattern">{$t('pattern')}</th>
              <th class="th-action">{$t('action')}</th>
              <th class="th-target">{$t('target')}</th>
              <th class="th-apply">Apply To</th>
              <th class="th-actions"></th>
            </tr>
          </thead>
          <tbody>
            {#each (filters || []) as filter}
              <tr class="filter-row">
                <td class="td-pattern">
                  <code class="pattern">{filter.pattern}</code>
                </td>
                <td class="td-action">
                  <span class="tag action-{filter.action.toLowerCase()}">{filter.action}</span>
                </td>
                <td class="td-target">
                  <span class="tag target">{filter.target}</span>
                </td>
                <td class="td-apply">
                  {#if filter.feed_id}
                    <span class="tag feed-specific">
                      {feeds.find(f => f.id === filter.feed_id)?.title || 'Feed'}
                    </span>
                  {:else}
                    <span class="tag global">Global</span>
                  {/if}
                </td>
                <td class="td-actions">
                  <button class="delete-btn" on:click={() => handleDelete(filter.id)} title="Delete filter">
                    <Trash2 size={16} />
                  </button>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
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
  
  .filters-table-wrapper {
    background: var(--md-surface);
    border: 1px solid var(--md-outline);
    border-radius: var(--radius-xl);
    overflow: hidden;
    width: 100%;
    box-shadow: 0 4px 12px rgba(0,0,0,0.1);
  }

  .filters-table {
    width: 100%;
    border-collapse: collapse;
    text-align: left;
  }

  .filters-table th {
    background: var(--md-surface-variant);
    padding: 14px 20px;
    font: var(--m3-label-medium);
    color: var(--md-on-surface-variant);
    font-weight: 800;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    border-bottom: 1px solid var(--md-outline);
  }

  .filters-table td {
    padding: 12px 20px;
    font: var(--m3-body-medium);
    color: var(--md-on-surface);
    border-bottom: 1px solid var(--md-outline-variant);
    vertical-align: middle;
  }

  .filter-row:last-child td {
    border-bottom: none;
  }

  .filter-row {
    transition: background 0.2s;
  }

  .filter-row:hover {
    background: var(--md-surface-variant);
  }

  .th-pattern, .td-pattern { width: 35%; }
  .th-action, .td-action { width: 15%; }
  .th-target, .td-target { width: 15%; }
  .th-apply, .td-apply { width: 25%; }
  .th-actions, .td-actions { width: 10%; text-align: right; }

  .pattern { font-family: var(--font-mono); color: var(--md-primary); background: var(--md-primary-container); padding: 4px 10px; border-radius: var(--radius-full); font-size: 13px; font-weight: 600; display: inline-block; }

  .tags { display: flex; gap: 8px; flex-wrap: wrap; }
  .tag { font: var(--m3-label-small); font-weight: 800; padding: 4px 12px; border-radius: var(--radius-full); text-transform: uppercase; letter-spacing: 0.5px; }
  .target { background: var(--md-outline); color: var(--md-on-surface-variant); }
  .global { border: 1px solid var(--md-primary); color: var(--md-primary); }
  .feed-specific { background: var(--md-secondary-container); color: var(--md-on-secondary-container); }
  .action-block { background: #ff525222; color: #ff5252; border: 1px solid #ff525244; }
  .action-replace { background: #4caf5022; color: #4caf50; border: 1px solid #4caf5044; }

  .delete-btn { background: transparent; border: none; color: var(--md-on-surface-variant); cursor: pointer; padding: 10px; border-radius: var(--radius-full); transition: all 0.2s; display: flex; align-items: center; justify-content: center; }
  .delete-btn:hover { background: #ff525222; color: #ff5252; }

  .add-btn-primary { 
    background: var(--md-primary); 
    color: var(--md-on-primary); 
    border: none; 
    padding: 10px 24px; 
    border-radius: var(--radius-full); 
    font: var(--m3-label-large); 
    font-weight: 800; 
    cursor: pointer; 
    display: flex; 
    align-items: center; 
    gap: 8px;
    transition: all 0.2s;
  }
  .add-btn-primary:hover { transform: translateY(-1px); box-shadow: 0 4px 12px var(--md-primary-container); }

  .modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.7); display: flex; align-items: center; justify-content: center; z-index: 3000; backdrop-filter: blur(8px); }
  .modal { background: var(--md-surface); border: 1px solid var(--md-outline); padding: 40px; border-radius: 32px; width: 100%; max-width: 520px; box-shadow: 0 40px 80px rgba(0,0,0,0.5); }
  .modal h3 { font: var(--m3-headline-small); margin-bottom: 32px; color: var(--md-on-surface); font-weight: 800; }
  
  .form-group { margin-bottom: 24px; }
  .form-group label { display: block; font: var(--m3-label-large); color: var(--md-primary); margin-bottom: 12px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px; }
  .field-help { font: var(--m3-label-small); color: var(--md-on-surface-variant); margin-top: 10px; opacity: 0.7; }
  .field-help code { background: var(--md-outline); padding: 2px 6px; border-radius: 4px; color: var(--md-primary); }
  .form-group input, .form-group select { width: 100%; padding: 16px 20px; border-radius: var(--radius-full); background: var(--md-background); border: 1px solid var(--md-outline); color: var(--md-on-surface); font: var(--m3-body-large); outline: none; transition: all 0.2s; }
  .form-group input:focus, .form-group select:focus { border-color: var(--md-primary); box-shadow: 0 0 0 2px var(--md-primary-container); }

  .form-row { display: grid; grid-template-columns: 1fr 1fr; gap: 20px; }

  .modal-actions { display: flex; justify-content: flex-end; gap: 16px; margin-top: 40px; }
  .btn-text { background: transparent; border: none; color: var(--md-on-surface-variant); padding: 12px 24px; cursor: pointer; font: var(--m3-label-large); font-weight: 700; border-radius: var(--radius-full); }
  .btn-text:hover { background: var(--md-outline); }
  .btn-primary { background: var(--md-primary); color: var(--md-on-primary); border: none; padding: 12px 32px; border-radius: var(--radius-full); font: var(--m3-label-large); font-weight: 800; cursor: pointer; transition: all 0.2s; }
  .btn-primary:hover { transform: translateY(-1px); box-shadow: 0 4px 12px var(--md-primary-container); }

  .state-msg { text-align: center; margin-top: 80px; color: var(--md-on-surface-variant); font: var(--m3-body-large); opacity: 0.6; }
</style>
