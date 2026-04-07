<script>
  import { fly } from 'svelte/transition';
  import { cubicOut } from 'svelte/easing';
  import PostCard from './PostCard.svelte';
  import { Search, CheckCheck, PanelRightClose, PanelRightOpen, Inbox, X } from 'lucide-svelte';
  import { t } from './i18n.js';
  import { fade, slide } from 'svelte/transition';
  
  /** @type {Array<any>} */
  export let articles = [];
  
  /** @type {number} */
  export let selectedId;
  
  export let isExpanded = false;
  export let onToggleExpand = () => {};

  /** @type {(id: number) => void} */
  export let onSelect;

  /** @type {(id: string) => void} */
  export let onToggleFavorite;

  /** @type {(article: any) => void} */
  export let onShare;

  export let searchQuery = "";
  let showSearch = false;
  let searchInput;

  function toggleSearch() {
    showSearch = !showSearch;
    if (!showSearch) searchQuery = "";
    else {
      setTimeout(() => searchInput?.focus(), 50);
    }
  }

  function handleKeydown(e) {
    if (e.key === 'Escape' && showSearch) toggleSearch();
  }

</script>

<section class="feed-list {isExpanded ? 'expanded' : ''} {articles.length === 0 ? 'is-empty' : ''}" transition:fly={{ x: -20, duration: 400, easing: cubicOut }}>
  <header class="header-bar">
    <div></div> <!-- Empty spacer to keep actions aligned -->
    <div class="actions">
      {#if showSearch}
        <div class="search-box" transition:slide={{ axis: 'x', duration: 300 }}>
          <input 
            bind:value={searchQuery} 
            bind:this={searchInput}
            placeholder={$t('search') || 'Search...'} 
            on:keydown={handleKeydown}
            class="search-input"
          />
        </div>
      {/if}

      <button class="action-btn {showSearch ? 'active' : ''}" on:click={toggleSearch} title="Search feeds">
        {#if showSearch}
          <X size={18} />
        {:else}
          <Search size={18} />
        {/if}
      </button>
      <button class="action-btn" title="Mark visible as read">
        <CheckCheck size={18} />
      </button>
      <button class="action-btn expand-btn" on:click={onToggleExpand} title={isExpanded ? 'Show reader panel' : 'Expand feed list'}>
        {#if isExpanded}
          <PanelRightOpen size={18} />
        {:else}
          <PanelRightClose size={18} />
        {/if}
      </button>
    </div>
  </header>

  <!-- Solid surface background for the gap -->
  <div class="corner-bg"></div>
  
  <!-- Overlay to draw the perfectly curved sharp line -->
  <div class="corner-arc"></div>

  <div class="feed-content">
    <div class="articles-container scroll-area">
      {#each articles as item}
          <PostCard 
            article={{...item, selected: selectedId === item.id}} 
            onClick={onSelect} 
            {onToggleFavorite}
            onShare={() => onShare(item)}
          />
      {:else}
        <div class="empty-state" transition:fly={{ y: 20, duration: 400 }}>
          <div class="empty-icon">
            <div class="icon-circle">
              <Inbox size={48} strokeWidth={1.5} />
            </div>
          </div>
          <h3>{$t('all_caught_up') || 'All Caught Up'}</h3>
          <p>{$t('no_articles_desc') || 'There are no articles to display in this view right now.'}</p>
        </div>
      {/each}
    </div>
  </div>
</section>

<style>
  .feed-list {
    position: relative;
    width: var(--feed-width);
    background: var(--md-background);
    display: flex;
    flex-direction: column;
    height: 100vh;
    transition: all var(--layout-transition);
  }
  .feed-list.expanded {
    width: calc(100vw - var(--sidebar-width));
    flex-shrink: 0;
  }
  .feed-list.is-empty {
    flex: 1;
    width: auto;
  }

  .header-bar { 
    display: flex; 
    justify-content: space-between; 
    align-items: center; 
    padding: 0 24px; 
    border-bottom: 1px solid var(--md-outline); 
    border-right: 1px solid var(--md-outline); 
    background: var(--md-surface);
    box-sizing: border-box; 
    height: var(--header-height); 
    min-height: var(--header-height); 
    max-height: var(--header-height);
    border-bottom-right-radius: 32px; 
    position: relative;
    z-index: 10;
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

  /* Solid background to color the gap outside the feed-content radius */
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
    z-index: 20; /* Keep stroke sharp and unclipped */
    pointer-events: none;
  }

  .feed-content {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    background: var(--md-background);
    border-top-left-radius: 32px;
    border-left: 1px solid var(--md-outline);
    position: relative;
    z-index: 10;
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
    border-radius: var(--radius-full); /* Redondeado perfecto */
    transition: all 0.2s;
  }
  .action-btn:hover { background: var(--md-surface-variant); color: var(--md-on-surface); }


  .articles-container {
    flex: 1;
    padding: 24px 16px 24px 16px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .empty-state {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 48px 24px;
    text-align: center;
    color: var(--md-on-surface-variant);
  }

  .empty-icon {
    margin-bottom: 24px;
  }

  .icon-circle {
    width: 80px;
    height: 80px;
    border-radius: 50%;
    background: var(--md-surface-variant);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--md-primary);
  }

  .empty-state h3 {
    margin: 8px 0;
    color: var(--md-on-surface);
    font: var(--m3-headline-small);
    font-weight: 600;
  }

  .empty-state p {
    font: var(--m3-body-medium);
    max-width: 240px;
    margin: 0;
    opacity: 0.7;
    line-height: 1.5;
  }

  @media (max-width: 1000px) {
    .header-bar {
      padding-left: 12px;
      padding-right: 12px;
    }
    .feed-list {
      width: 100% !important;
      border-right: none;
    }
    .expand-btn {
      display: none !important;
    }
  }
  .action-btn.active {
    color: var(--md-primary);
    background: var(--md-primary-container);
  }

  .search-box {
    display: flex;
    align-items: center;
    background: var(--md-surface-variant);
    border-radius: var(--radius-m);
    padding: 0 12px;
    margin-right: 8px;
    height: 36px;
    border: 1px solid var(--md-outline);
  }

  .search-input {
    background: transparent;
    border: none;
    color: var(--md-on-surface);
    font: var(--m3-label-large);
    outline: none;
    width: 200px;
  }

  @media (max-width: 600px) {
    .search-input { width: 120px; }
  }
</style>
