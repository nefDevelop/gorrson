<script>
  import { fly } from 'svelte/transition';
  import { cubicOut } from 'svelte/easing';
  import PostCard from './PostCard.svelte';
  import { Search, CheckCheck } from 'lucide-svelte';
  import { t } from './i18n.js';
  
  /** @type {Array<any>} */
  export let articles = [];
  
  /** @type {number} */
  export let selectedId;
  
  /** @type {(id: number) => void} */
  export let onSelect;

  let titleKey = "latest";
</script>

<section class="feed-list scroll-area" transition:fly={{ x: -20, duration: 400, easing: cubicOut }}>
  <header class="header-bar">
    <div class="title-container">
      <h1 class="title">{$t(titleKey)}</h1>
      <p class="subtitle">{$t('monday')}, {$t('april')} 6</p>
    </div>
    <div class="actions">
      <button class="action-btn">
        <Search size={18} />
      </button>
      <button class="action-btn">
        <CheckCheck size={18} />
      </button>
    </div>
  </header>

  <div class="filter-chips">
    <button class="chip active">{$t('all')}</button>
    <button class="chip">{$t('unread')}</button>
    <button class="chip">{$t('favorites')}</button>
    <button class="chip">{$t('reddit')}</button>
  </div>

  <div class="articles-container">
    {#each articles as item}
      <PostCard 
        article={{...item, selected: selectedId === item.id}} 
        onClick={onSelect} 
      />
    {/each}
  </div>
</section>

<style>
  .feed-list {
    width: var(--feed-width);
    border-right: 1px solid var(--md-surface-variant);
    background: var(--md-background);
    display: flex;
    flex-direction: column;
    height: 100vh;
    transition: all var(--layout-transition);
  }

  .header-bar {
    display: flex;
    justify-content: space-between;
    align-items: center; /* Keep centered */
    padding: 8px 24px; /* Add vertical padding */
    background: var(--md-surface);
    /* border-bottom: 1px solid var(--md-outline); */
    box-shadow: 0 1px 0 0 var(--md-outline); /* Use box-shadow for consistent border rendering */
    height: var(--header-height); /* Explicit height */
    min-height: var(--header-height); /* Ensure minimum height */
    max-height: var(--header-height); /* Ensure maximum height */
  }

  .header-bar .title {
    font: var(--m3-headline-small);
    font-weight: 700;
    margin: 0;
    color: var(--md-on-surface);
  }

  .header-bar .subtitle {
    color: var(--md-on-surface-variant);
    font: var(--m3-label-small);
    margin: 4px 0 0 0;
    opacity: 0.7;
    letter-spacing: 0.5px;
  }

  .actions {
    display: flex;
    gap: 8px;
  }

  .action-btn {
    background: transparent;
    border: none;
    color: var(--md-on-surface-variant);
    cursor: pointer;
    padding: 8px;
    border-radius: var(--radius-m);
    transition: all 0.2s;
  }
  .action-btn:hover { background: var(--md-surface-variant); color: var(--md-on-surface); }

  .filter-chips {
    padding: 12px 24px;
    display: flex;
    gap: 8px;
    background: var(--md-background);
    border-bottom: 1px solid var(--md-outline);
  }

  .chip {
    padding: 6px 16px;
    border-radius: var(--radius-m);
    background: transparent;
    color: var(--md-on-surface-variant);
    border: 1px solid var(--md-outline);
    font: var(--m3-label-large);
    cursor: pointer;
    white-space: nowrap;
    transition: all 0.2s;
  }
  .chip:hover { background: var(--md-surface-variant); color: var(--md-on-surface); }
  .chip.active {
    background: var(--md-primary-container);
    color: var(--md-on-primary-container);
    border-color: var(--md-primary);
  }

  .articles-container {
    padding: 12px 16px;
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding-bottom: 120px;
  }

  @media (max-width: 1000px) {
    .header-bar, .filter-chips {
      padding-left: 12px;
      padding-right: 12px;
    }
  }

  @media (max-width: 768px) {
    .feed-list {
      width: 100% !important;
      border-right: none;
    }
  }
</style>
