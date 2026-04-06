<script>
  import { fly } from 'svelte/transition';
  import { cubicOut } from 'svelte/easing';
  import PostCard from './PostCard.svelte';
  import { Search, CheckCheck, PanelRightClose, PanelRightOpen } from 'lucide-svelte';
  import { t } from './i18n.js';
  
  /** @type {Array<any>} */
  export let articles = [];
  
  /** @type {number} */
  export let selectedId;
  
  export let isExpanded = false;
  export let onToggleExpand = () => {};

  /** @type {(id: number) => void} */
  export let onSelect;

</script>

<section class="feed-list {isExpanded ? 'expanded' : ''}" transition:fly={{ x: -20, duration: 400, easing: cubicOut }}>
  <header class="header-bar">
    <div></div> <!-- Empty spacer to keep actions aligned -->
    <div class="actions">
      <button class="action-btn" title="Search feeds">
        <Search size={18} />
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
        />
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
    gap: 6px;
  }

  @media (max-width: 1000px) {
    .header-bar {
      padding-left: 12px;
      padding-right: 12px;
    }
  }

  @media (max-width: 768px) {
    .feed-list {
      width: 100% !important;
      border-right: none;
    }
    .expand-btn {
      display: none !important;
    }
  }
</style>
