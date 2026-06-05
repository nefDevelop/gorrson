<script>
  import { fly } from "svelte/transition";
  import { cubicOut } from "svelte/easing";
  import PostCard from "./PostCard.svelte";
  import {
    Search,
    CheckCheck,
    PanelRightClose,
    PanelRightOpen,
    Inbox,
    X,
    ListFilter,
    Clock,
    CheckCircle2,
    LayoutGrid,
    List,
    AlignJustify,
    BookOpen,
    LayoutTemplate,
    Image as ImageIcon,
    ChevronDown
  } from "lucide-svelte";
  import { t } from "./i18n.js";
  import { fade, slide } from "svelte/transition";
  import { onMount } from "svelte";
  import { extractFirstImage, stripHtml } from "./utils.js";

  /** @type {Array<any>} */
  export let articles = [];

  /** @type {string} */
  export let title = "Inbox";

  /** @type {string | number | null | undefined} */
  export let selectedId;

  /** @type {boolean} */
  export let hasSelection = false;

  /** @type {string} */
  export let readerMode = 'side-by-side';

  /** @type {() => void} */
  export let onLoadMore = () => {};

  /** @type {boolean} */
  export let loadingMore = false;

  /** @type {boolean} */
  export let allLoaded = false;

  /** @type {(id: string | number) => void} */
  export let onSelect;

  /** @type {(id: string) => void} */
  export let onToggleFavorite;

  /** @type {(id: string) => void} */
  export let onMarkAsRead;

  /** @type {(article: any) => void} */
  export let onShare;

  /** @type {boolean} */
  export let unreadOnly = false;

  /** @type {string} */
  export let sortMode = "newest";

  /** @type {string} */
  export let cardStyle = "default";

  export let searchQuery = "";
  let showFilters = false;
  let showViewMenu = false;
  let showMarkMenu = false;

  /** @type {(isGlobal?: boolean) => void} */
  export let onMarkAllRead;

  /** @type {boolean} */
  export let scrollRead = false;

  /** @type {IntersectionObserver | undefined} */
  let statusObserver;
  /** @type {IntersectionObserver | undefined} */
  let scrollObserver;
  /** @type {Array<HTMLElement>} */
  let pendingCards = [];
  /** @type {HTMLElement | null} */
  let pendingSentinel = null;

  onMount(() => {
    const isMobile = window.innerWidth <= 768;
    const scrollRoot = isMobile ? null : document.querySelector(".articles-container");

    // Observer for marking articles as read (top 20% of screen)
    statusObserver = new IntersectionObserver(
      (entries) => {
        if (scrollRead) {
          entries.forEach((entry) => {
            if (
              entry.isIntersecting &&
              entry.target.classList.contains("post-card") &&
              !entry.target.classList.contains("read")
            ) {
              const id = entry.target.getAttribute("data-id");
              if (id) onMarkAsRead(id);
            }
          });
        }
      },
      {
        root: scrollRoot,
        rootMargin: "0px 0px -80% 0px",
        threshold: 0.1,
      },
    );

    // Observer for infinite scroll (bottom of the list)
    scrollObserver = new IntersectionObserver(
      (entries) => {
        const sentinel = entries[0];
        if (sentinel.isIntersecting && !loadingMore && !allLoaded) {
          onLoadMore();
        }
      },
      {
        root: scrollRoot,
        threshold: 0.1,
      },
    );

    pendingCards.forEach(node => statusObserver.observe(node));
    pendingCards = [];
    if (pendingSentinel) scrollObserver.observe(pendingSentinel);

    return () => {
      if (statusObserver) statusObserver.disconnect();
      if (scrollObserver) scrollObserver.disconnect();
    };
  });

  $: renderedArticles = buildLayout(articles, cardStyle, readerMode, selectedId);

  /**
   * @param {Array<any>} arts
   * @param {string} style
   * @param {string} mode
   * @param {number|string|null|undefined} selId
   */
  function buildLayout(arts, style, mode, selId) {
    if (!arts || arts.length === 0) return [];
    
    if (selId && mode === 'side-by-side') return arts.map(a => ({ ...a, _role: 'bento-full' }));
    if (style === 'compact' || style === 'minimal') return arts.map(a => ({ ...a, _role: 'bento-full' }));
    if (style === 'default' || style === 'image') return arts.map((a, i) => ({ ...a, _role: i === 0 ? 'bento-hero' : 'bento-standard' }));
    
    // Magazine style (Dynamic Bento Grid)
    let result = [];
    let pool = [...arts];
    let cycle = 0;

    const hasImage = (a) => !!(a.image_url || extractFirstImage(a.content));
    const isTextRich = (a) => stripHtml(a.content || a.summary || "").length >= 200;

    while (pool.length > 0) {
      const pattern = cycle % 4;
      
      if (pattern === 0 && pool.length >= 2) {
        // Hero (8) + Side (4)
        let main = pool.shift();
        let side = pool.shift();
        main._role = 'mag-hero';
        side._role = isTextRich(side) && !hasImage(side) ? 'mag-side-text' : 'mag-side-img';
        result.push(main, side);
      } 
      else if (pattern === 1 && pool.length >= 2) {
        // Side (4) + Horizontal (8)
        let side = pool.shift();
        let main = pool.shift();
        side._role = hasImage(side) ? 'mag-side-img' : 'mag-side-text';
        main._role = 'mag-horizontal';
        result.push(side, main);
      }
      else if (pattern === 2 && pool.length >= 3) {
        // Triple (4+4+4)
        let a1 = pool.shift();
        let a2 = pool.shift();
        let a3 = pool.shift();
        a1._role = 'mag-side-img';
        a2._role = 'mag-side-img';
        a3._role = 'mag-side-img';
        result.push(a1, a2, a3);
      }
      else if (pattern === 3 && pool.length >= 2) {
        // Horizontal (8) + Side (4)
        let main = pool.shift();
        let side = pool.shift();
        main._role = 'mag-horizontal';
        side._role = 'mag-side-img';
        result.push(main, side);
      }
      else {
        // Fallback or single remaining
        let item = pool.shift();
        item._role = hasImage(item) ? 'bento-hero' : 'bento-standard';
        result.push(item);
      }
      cycle++;
    }
    return result;
  }

  // Svelte Action para enlazar la tarjeta al IntersectionObserver nativamente
  /**
   * @param {HTMLElement} node
   * @param {boolean} isRead
   */
  function observeCard(node, isRead) {
    if (!isRead) {
      if (statusObserver) statusObserver.observe(node);
      else pendingCards.push(node);
    }
    return {
      update(newIsRead) {
        if (newIsRead && statusObserver) statusObserver.unobserve(node);
      },
      destroy() {
        if (statusObserver) statusObserver.unobserve(node);
        pendingCards = pendingCards.filter(n => n !== node);
      }
    };
  }

  // Svelte Action para el scroll infinito
  /**
   * @param {HTMLElement} node
   */
  function observeSentinel(node) {
    if (scrollObserver) scrollObserver.observe(node);
    else pendingSentinel = node;
    return {
      destroy() {
        if (scrollObserver) scrollObserver.unobserve(node);
        pendingSentinel = null;
      }
    };
  }
</script>

<section
  class="dashboard-view {articles.length === 0
    ? 'is-empty'
    : ''} style-{cardStyle}"
  transition:fly={{ x: -20, duration: 400, easing: cubicOut }}
>
  <header class="header-bar" class:mobile-reading={hasSelection}>
    <div class="actions">
      <!-- Buscador integrado -->
      <div class="inline-search">
        <span class="search-icon-inner">
          <Search size={18} />
        </span>
        <input bind:value={searchQuery} placeholder={$t("search") || "Search..."} class="search-input" />
        {#if searchQuery}
          <button class="clear-btn" on:click={() => (searchQuery = "")}>
            <X size={14} />
          </button>
        {/if}
      </div>

      <!-- Menú Desplegable de Vista y Lector -->
      <div class="filter-wrapper">
        <button class="action-btn {showViewMenu ? 'active' : ''}" on:click={() => {showViewMenu = !showViewMenu; showFilters = false;}} title="View Options">
          {#if cardStyle === 'default'}
            <LayoutGrid size={18} />
          {:else if cardStyle === 'compact'}
            <AlignJustify size={18} />
          {:else if cardStyle === 'magazine'}
            <LayoutTemplate size={18} />
          {:else}
            <List size={18} />
          {/if}
        </button>
        {#if showViewMenu}
          <div class="filter-menu" in:fly={{ y: 10, duration: 200 }} out:fade={{ duration: 100 }}>
            <div class="menu-section">
              <span class="section-title">Layout</span>
              <button class="menu-item {cardStyle === 'default' ? 'selected' : ''}" on:click={() => {cardStyle = 'default'; showViewMenu = false;}}>
                <LayoutGrid size={16} />
                <span>Grid (Bento)</span>
              </button>
            <button class="menu-item {cardStyle === 'magazine' ? 'selected' : ''}" on:click={() => {cardStyle = 'magazine'; showViewMenu = false;}}>
              <LayoutTemplate size={16} />
              <span>Magazine</span>
            </button>
            <button class="menu-item {cardStyle === 'image' ? 'selected' : ''}" on:click={() => {cardStyle = 'image'; showViewMenu = false;}}>
              <ImageIcon size={16} />
              <span>Image</span>
            </button>
              <button class="menu-item {cardStyle === 'compact' ? 'selected' : ''}" on:click={() => {cardStyle = 'compact'; showViewMenu = false;}}>
                <AlignJustify size={16} />
                <span>Compact</span>
              </button>
              <button class="menu-item {cardStyle === 'minimal' ? 'selected' : ''}" on:click={() => {cardStyle = 'minimal'; showViewMenu = false;}}>
                <List size={16} />
                <span>Minimal</span>
              </button>
            </div>
            <div class="menu-divider expand-divider"></div>
            <div class="menu-section expand-section">
              <span class="section-title">Reading Pane</span>
              <button class="menu-item {readerMode === 'side-by-side' ? 'selected' : ''}" on:click={() => { readerMode = 'side-by-side'; showViewMenu = false; }}>
                <PanelRightClose size={16} />
                <span>Side-by-Side</span>
              </button>
              <button class="menu-item {readerMode === 'full-list' ? 'selected' : ''}" on:click={() => { readerMode = 'full-list'; showViewMenu = false; }}>
                <PanelRightOpen size={16} />
                <span>Full List</span>
              </button>
              <button class="menu-item {readerMode === 'overlay' ? 'selected' : ''}" on:click={() => { readerMode = 'overlay'; showViewMenu = false; }}>
                <BookOpen size={16} />
                <span>Overlay</span>
              </button>
            </div>
          </div>
        {/if}
      </div>

      <div class="filter-wrapper">
        <button class="action-btn {unreadOnly || sortMode !== 'newest' ? 'active' : ''}" on:click={() => {showFilters = !showFilters; showViewMenu = false;}} title="Filter and Sort">
          <ListFilter size={18} />
        </button>
        {#if showFilters}
          <div class="filter-menu" in:fly={{ y: 10, duration: 200 }} out:fade={{ duration: 100 }}>
            <div class="menu-section">
              <span class="section-title">Filter</span>
              <button class="menu-item {unreadOnly ? 'selected' : ''}" on:click={() => (unreadOnly = !unreadOnly)}>
                <CheckCircle2 size={16} />
                <span>Unread Only</span>
              </button>
            </div>
            <div class="menu-divider"></div>
            <div class="menu-section">
              <span class="section-title">Sort</span>
              <button class="menu-item {sortMode === 'newest' ? 'selected' : ''}" on:click={() => (sortMode = "newest")}>
                <Clock size={16} />
                <span>Newest</span>
              </button>
              <button class="menu-item {sortMode === 'oldest' ? 'selected' : ''}" on:click={() => (sortMode = "oldest")}>
                <Clock size={16} style="transform: rotate(180deg)" />
                <span>Oldest</span>
              </button>
            </div>
          </div>
        {/if}
      </div>
      <div class="filter-wrapper split-button-wrapper">
        <div class="split-button">
          <button 
            class="action-btn split-main" 
            on:click={() => onMarkAllRead(false)}
            title="Mark loaded as read"
          >
            <CheckCheck size={18} />
          </button>
          <button 
            class="action-btn split-trigger {showMarkMenu ? 'active' : ''}" 
            on:click={() => {showMarkMenu = !showMarkMenu; showFilters = false; showViewMenu = false;}}
            title="More options"
          >
            <ChevronDown size={14} />
          </button>
        </div>
        {#if showMarkMenu}
          <div class="filter-menu right-aligned" in:fly={{ y: 10, duration: 200 }} out:fade={{ duration: 100 }}>
            <div class="menu-section">
              <span class="section-title">Mark as read</span>
              <button class="menu-item" on:click={() => { onMarkAllRead(false); showMarkMenu = false; }}>
                <CheckCheck size={16} />
                <span>Mark Loaded</span>
              </button>
              <button class="menu-item" on:click={() => { onMarkAllRead(true); showMarkMenu = false; }}>
                <CheckCheck size={16} style="color: var(--md-primary)" />
                <span style="color: var(--md-primary); font-weight: 600;">Mark ALL in System</span>
              </button>
            </div>
          </div>
        {/if}
      </div>
    </div>
  </header>

  <div class="dashboard-split">
    <div class="feed-column {hasSelection && readerMode === 'side-by-side' ? 'narrow' : 'wide'}" class:mobile-hidden={hasSelection} class:desktop-hidden={hasSelection && readerMode === 'full-list'}>
      <div class="articles-container scroll-area">

      <!-- Hero Page Header -->
      <div class="feed-page-header">
        <div class="header-text-block">
          <h2 class="page-title">{title}</h2>
          <p class="page-subtitle">Your curated feed insights and updates.</p>
        </div>
        <div class="header-badges desktop-only">
          <span class="unread-badge">Unread ({articles.filter(a => !a.is_read).length})</span>
        </div>
      </div>

      {#each renderedArticles as item (item.id)}
        <div
          data-id={item.id}
          class="card-wrapper {item.is_read ? 'read' : ''} {item._role}"
          use:observeCard={item.is_read}
        >
          <PostCard
            article={{ ...item, selected: selectedId === item.id }}
            viewMode={cardStyle}
            bentoSize={item._role}
            onClick={onSelect}
            {onToggleFavorite}
            onShare={() => onShare(item)}
          />
        </div>
      {:else}
        <div class="empty-state" transition:fly={{ y: 20, duration: 400 }}>
          <div class="empty-icon">
            <div class="icon-circle">
              <Inbox size={48} strokeWidth={1.5} />
            </div>
          </div>
          <h3>{$t("all_caught_up") || "All Caught Up"}</h3>
          <p>
            {$t("no_articles_desc") ||
              "There are no articles to display in this view right now."}
          </p>
        </div>
      {/each}

      {#if articles.length > 0}
        <div class="sentinel" use:observeSentinel>
          {#if loadingMore}
            <div class="loading-more-spinner"></div>
          {:else if !allLoaded}
            <button class="load-more-btn" on:click={onLoadMore}>
              Load More
            </button>
          {:else}
            <span class="all-loaded-text">All articles loaded</span>
          {/if}
        </div>
      {/if}
    </div>
  </div>

    {#if hasSelection}
      <div class="reader-column" class:overlay-mode={readerMode === 'overlay'} transition:fly={{x: 40, duration: 400, easing: cubicOut}}>
        <slot name="reader"></slot>
      </div>
    {/if}
  </div>
</section>

<style>
  .dashboard-view {
    position: relative;
    width: 100%;
    background: var(--md-background);
    display: flex;
    flex-direction: column;
    height: 100vh;
    transition: all var(--layout-transition);
  }
  .dashboard-view.style-image {
    --feed-width: calc(var(--feed-width-base) * 1.25);
  }

  .dashboard-split {
    display: flex;
    flex: 1;
    min-height: 0;
  }

  .feed-column {
    display: flex;
    flex-direction: column;
    height: 100%;
    transition: width var(--layout-transition);
    border-right: 1px solid var(--md-outline);
    background: transparent;
  }

  .feed-column.narrow {
    width: var(--feed-width);
    flex-shrink: 0;
  }

  .feed-column.wide {
    width: 100%;
    flex: 1;
    border-right: none;
  }

  .reader-column {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    height: 100%;
    background: var(--md-background);
  }

  @media (min-width: 1001px) {
    .reader-column.overlay-mode {
      position: absolute;
      top: 0;
      left: 0;
      right: 0;
      bottom: 0;
      z-index: 200;
      box-shadow: -20px 0 60px rgba(0,0,0,0.4);
      border-left: 1px solid var(--md-outline);
    }
    .feed-column.desktop-hidden {
      display: none !important;
    }
  }

  .header-bar {
    display: flex;
    justify-content: flex-end;
    align-items: center;
    padding: 0 32px;
    border-bottom: 1px solid color-mix(in srgb, var(--md-outline) 50%, transparent);
    background: color-mix(in srgb, var(--md-background) 85%, transparent);
    backdrop-filter: blur(24px);
    -webkit-backdrop-filter: blur(24px);
    box-sizing: border-box;
    height: var(--header-height);
    min-height: var(--header-height);
    max-height: var(--header-height);
    position: relative;
    flex-shrink: 0;
    z-index: 50;
    width: 100%;
  }

  .actions {
    display: flex;
    gap: 8px;
    align-items: center;
  }

  .action-btn {
    background: transparent;
    border: none;
    color: var(--md-on-surface-variant);
    cursor: pointer;
    width: 40px;
    height: 40px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: var(--radius-full);
    transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
    flex-shrink: 0;
  }
  .action-btn:hover {
    background: color-mix(in srgb, var(--md-on-surface) 8%, transparent);
    color: var(--md-on-surface);
  }

  .articles-container {
    flex: 1;
    overflow-y: auto;
    padding: 24px;
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
    gap: 24px;
    align-content: start;
    transition: padding-top 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  }

  .dashboard-view.style-magazine .articles-container {
    grid-template-columns: repeat(12, 1fr);
  }

  /* Forzar modo lista para tarjetas delgadas */
  .dashboard-view.style-compact .articles-container,
  .dashboard-view.style-minimal .articles-container {
    grid-template-columns: 1fr;
    gap: 8px;
  }

  .feed-page-header {
    grid-column: 1 / -1;
    display: flex;
    justify-content: space-between;
    align-items: flex-end;
    margin-bottom: 8px;
    padding-bottom: 16px;
  }
  .page-title {
    font-family: var(--font-display);
    font-size: 32px;
    font-weight: 800;
    color: var(--md-on-surface);
    margin: 0 0 4px 0;
    letter-spacing: -0.5px;
    text-transform: capitalize;
  }
  .page-subtitle {
    font-size: 14px;
    color: var(--md-on-surface-variant);
    margin: 0;
  }
  .unread-badge {
    background: var(--md-surface-variant);
    color: var(--md-on-surface-variant);
    padding: 6px 16px;
    border-radius: var(--radius-full);
    font-size: 10px;
    font-weight: 800;
    text-transform: uppercase;
    letter-spacing: 1px;
    white-space: nowrap;
  }

  /* Split Mark All button */
  .split-button-wrapper {
    position: relative;
    display: inline-flex;
  }

  .split-button {
    display: flex;
    background: var(--md-surface);
    border-radius: var(--radius-full);
    border: 1px solid var(--md-outline);
    padding: 2px;
    align-items: center;
  }

  .split-button .action-btn {
    width: 36px;
    height: 36px;
  }

  .split-button .split-trigger {
    width: 28px;
  }

  .filter-menu.right-aligned {
    right: 0;
    left: auto;
  }

  .card-wrapper {
    display: flex;
    flex-direction: column;
  }

  /* Si está en modo lista o hay noticia seleccionada, ocupa toda la fila */
  .card-wrapper.bento-full {
    grid-column: 1 / -1;
  }

  /* Magazine Grid Rules */
  .dashboard-view.style-magazine .card-wrapper { grid-column: 1 / -1; }
  @media (min-width: 800px) {
    .dashboard-view.style-magazine .card-wrapper.mag-hero { grid-column: span 8; }
    .dashboard-view.style-magazine .card-wrapper.mag-side-text { grid-column: span 4; }
    .dashboard-view.style-magazine .card-wrapper.mag-side-img { grid-column: span 4; }
    .dashboard-view.style-magazine .card-wrapper.mag-horizontal { grid-column: span 8; }
  }

  /* Diseño Bento Grid Responsivo */
  @media (min-width: 800px) {
    .card-wrapper.bento-hero { grid-column: 1 / -1; }
  }

  .articles-container.search-open {
    padding-top: 80px;
  }

  .empty-state {
    grid-column: 1 / -1;
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 48px 24px;
    text-align: center;
    color: var(--md-on-surface-variant);
    min-height: 300px;
  }

  @media (max-width: 768px) {
    .empty-state {
      padding-bottom: 120px; /* Extra space for mobile nav */
    }
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
    .expand-section, .expand-divider {
      display: none !important;
    }
    .feed-column.mobile-hidden {
      display: none;
    }
    .header-bar.mobile-reading {
      display: none;
    }
  }
  .action-btn.active {
    color: var(--md-primary);
    background: var(--md-primary-container);
  }

  .inline-search {
    display: flex;
    align-items: center;
    background: var(--md-surface);
    border: 1px solid var(--md-outline);
    border-radius: var(--radius-full);
    padding: 0 16px;
    height: 40px;
    width: 280px;
    gap: 10px;
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
    box-shadow: 0 2px 8px rgba(0,0,0,0.04);
  }

  .inline-search:focus-within {
    border-color: var(--md-primary);
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--md-primary) 20%, transparent);
    width: 320px;
  }

  .search-icon-inner {
    color: var(--md-on-surface-variant);
    display: flex;
    align-items: center;
    transition: color 0.2s;
  }

  .inline-search:focus-within .search-icon-inner {
    color: var(--md-primary);
  }

  .search-input {
    background: transparent;
    border: none;
    color: var(--md-on-surface);
    font: var(--m3-body-medium);
    outline: none;
    flex: 1;
  }

  .search-input::placeholder {
    color: var(--md-on-surface-variant);
    opacity: 0.6;
  }

  .clear-btn {
    background: var(--md-surface-variant);
    border: none;
    color: var(--md-on-surface-variant);
    width: 22px;
    height: 22px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    transition: all 0.2s;
  }
  .clear-btn:hover {
    background: var(--md-outline);
    color: var(--md-on-surface);
  }

  @media (max-width: 800px) {
    .inline-search {
      width: auto;
      flex: 1;
      max-width: 140px;
    }
    .inline-search:focus-within {
      max-width: 180px;
    }
  }

  .sentinel {
    grid-column: 1 / -1;
    height: 48px;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 100%;
    margin-top: 12px;
  }

  .filter-wrapper {
    position: relative;
  }

  .filter-menu {
    position: absolute;
    top: calc(100% + 12px);
    right: 0;
    width: 200px;
    background: var(--md-surface);
    border-radius: 16px;
    box-shadow: 0 12px 32px var(--shadow-color);
    border: 1px solid var(--md-outline);
    z-index: 100;
    padding: 8px;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .menu-section {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .section-title {
    font-size: 10px;
    font-weight: 800;
    text-transform: uppercase;
    color: var(--md-on-surface-variant);
    padding: 6px 12px;
    letter-spacing: 1px;
  }

  .menu-divider {
    height: 1px;
    background: var(--md-outline);
    margin: 4px 0;
    opacity: 0.5;
  }

  .menu-item {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 12px;
    border: none;
    background: transparent;
    color: var(--md-on-surface);
    font-size: 13px;
    font-family: var(--font-main);
    font-weight: 500;
    cursor: pointer;
    border-radius: 10px;
    transition: all 0.2s;
    text-align: left;
    width: 100%;
  }

  .menu-item:hover {
    background: color-mix(in srgb, var(--md-on-surface) 5%, transparent);
  }

  .menu-item.selected {
    background: var(--md-primary);
    color: var(--md-on-primary);
    font-weight: 700;
  }

  .load-more-btn {
    background: var(--md-surface-variant);
    border: 1px solid var(--md-outline);
    color: var(--md-on-surface);
    padding: 8px 24px;
    border-radius: var(--radius-full);
    font-size: 13px;
    font-family: var(--font-main);
    cursor: pointer;
    transition: all 0.2s;
  }

  .load-more-btn:hover {
    background: var(--md-primary-container);
    border-color: var(--md-primary);
    color: var(--md-on-primary-container);
  }

  .all-loaded-text {
    font-size: 12px;
    color: var(--md-on-surface-variant);
    opacity: 0.5;
    font-weight: 500;
  }

  .loading-more-spinner {
    width: 20px;
    height: 20px;
    border: 2px solid var(--md-primary);
    border-top-color: transparent;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }

  @media (max-width: 768px) {
    .dashboard-view {
      height: auto;
      display: block;
    }
    .articles-container {
      overflow-y: visible;
      height: auto;
      display: flex;
      flex-direction: column;
      gap: 16px;
      padding-top: 12px;
    }
    .header-bar {
      position: sticky;
      top: 0;
      z-index: 100;
      padding: 0 16px;
    }
    .feed-page-header {
      padding: 0 16px 16px 16px;
    }
  }
</style>
