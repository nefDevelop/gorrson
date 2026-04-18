<script>
  import { stripHtml, extractFirstImage } from './utils.js';
  import { Star, Share2, Image as ImageIcon } from 'lucide-svelte';
  import { fetchFullContent } from './api.js';
  /** @type {{id: string, title: string, source: string, time: string, category: string, is_read?: boolean, content?: string, summary?: string, selected?: boolean, is_favorite?: boolean, matched_action?: string, image_url?: string}} */
  export let article;
  
  /** @type {(id: string | number) => void} */
  export let onClick;
  
  /** @type {(id: string) => void} */
  export let onToggleFavorite;
  
  /** @type {() => void} */
  export let onShare;

  /** @type {string} */
  export let viewMode = 'default'; // 'default', 'compact', 'minimal', 'image'
 
  /** @type {string} */
  export let bentoSize = 'bento-full';

  $: summary = stripHtml(article.content || article.summary || "");
  $: imageUrl = article.image_url || extractFirstImage(article.content);
  $: showThumbnail = (viewMode !== 'minimal' && viewMode !== 'compact') && imageUrl;

  let loadingFull = false;
  $: if (bentoSize === 'mag-side-text' && !article.is_full_content && !loadingFull) {
    loadExtraText();
  }

  async function loadExtraText() {
    loadingFull = true;
    try {
      const text = await fetchFullContent(article.id);
      article = { ...article, content: text, is_full_content: true };
    } catch (e) {
      console.error("Failed to load full content for vertical card", e);
    }
  }
</script>

<div 
  data-id={article.id}
  class="post-card {article.selected ? 'selected' : ''} {article.is_read ? 'read' : ''} {article.matched_action?.toLowerCase()} mode-{viewMode} {bentoSize}" 
  on:click={() => onClick(article.id)}
  role="button"
  tabindex="0"
  on:keydown={(e) => e.key === 'Enter' && onClick(article.id)}
>
  <div class="card-layout">
    <div class="content-side">
      <div class="title-row">
        <h3 class="title" class:active-title={article.selected}>
          {article.title}
        </h3>
        {#if viewMode !== 'compact' && viewMode !== 'minimal'}
          <span class="time">{article.time}</span>
        {/if}
      </div>

      {#if viewMode !== 'minimal'}
        <p class="summary">
          {summary || "No preview available..."}
        </p>
      {/if}

      <div class="footer-row">
        <div class="meta">
          <span class="source-tag">{article.source}</span>
          {#if viewMode === 'compact' || viewMode === 'minimal'}
            <span class="dot">•</span>
            <span class="time-mini">{article.time}</span>
          {/if}
        </div>
        
        <div class="actions">
          <button 
            class="action-btn" 
            on:click|stopPropagation={onShare} 
            title="Share"
          >
            <Share2 size={14} strokeWidth={2} />
          </button>
          <button 
            class="action-btn" 
            class:starred={article.is_favorite} 
            on:click|stopPropagation={() => onToggleFavorite(article.id)}
          >
            <Star size={14} strokeWidth={2} />
          </button>
        </div>
      </div>

      {#if bentoSize === 'mag-hero'}
        <div class="hero-read-more">
          Read Full Article <span class="arrow">→</span>
        </div>
      {/if}
    </div>

    {#if showThumbnail}
      <div class="image-side">
        <img src={imageUrl} alt="Article thumbnail" loading="lazy" />
        {#if bentoSize === 'mag-hero'}
          <div class="hero-badge">Newest</div>
        {/if}
      </div>
    {/if}
  </div>
</div>

<style>
  .post-card {
    background: var(--md-surface);
    border: 1px solid var(--md-outline);
    border-radius: 20px;
    padding: 16px 20px 12px 20px;
    margin-bottom: 8px;
    cursor: pointer;
    transition: all 0.2s cubic-bezier(0, 0, 0.2, 1);
    width: 100%;
    outline: none;
    display: block;
    text-align: left;
    position: relative;
    opacity: 0.9;
    height: 100%;
    display: flex;
    flex-direction: column;
    box-shadow: 0 4px 12px rgba(0,0,0,0.04);
  }

  .post-card:not(.read) { opacity: 1; }

  .post-card:hover {
    background: var(--md-surface);
    border-color: var(--md-primary);
    transform: translateY(-4px);
    box-shadow: 0 12px 32px var(--shadow-color);
    opacity: 1;
  }

  /* Layout Base */
  .card-layout { display: flex; gap: 16px; width: 100%; flex: 1; }
  .content-side { flex: 1; min-width: 0; display: flex; flex-direction: column; justify-content: space-between; }
  
  .image-side { 
    width: 84px; 
    height: 84px; 
    flex-shrink: 0; 
    border-radius: 16px; 
    overflow: hidden; 
    background: var(--md-surface); 
    border: none;
    margin-top: 4px;
  }
  .image-side img { width: 100%; height: 100%; object-fit: cover; transition: transform 0.3s; }
  .post-card:hover .image-side img { transform: scale(1.03); }

  /* Modes */
  .mode-compact { padding: 12px 16px 8px 16px; border-radius: 16px; margin-bottom: 4px; }
  .mode-compact .title { font-size: 15px; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; display: -webkit-box; }
  .mode-compact .summary { font-size: 12px; -webkit-line-clamp: 1; -webkit-box-orient: vertical; overflow: hidden; display: -webkit-box; margin-top: 2px; }
  
  .mode-minimal { padding: 10px 14px; border-radius: 12px; margin-bottom: 4px; }
  .mode-minimal .title { font-size: 14px; font-weight: 500; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; display: -webkit-box; margin-bottom: 2px; }
  .mode-minimal .footer-row { margin-top: 2px; }
  .mode-minimal .action-btn { padding: 2px; }

  /* Layout Estandar (Grid) */
  .post-card.bento-standard .card-layout {
    flex-direction: column-reverse; /* Pone la imagen arriba */
  }
  
  .post-card.bento-standard .image-side {
    width: 100%;
    height: auto;
    aspect-ratio: 16 / 10;
    margin-top: 0;
    margin-bottom: 16px;
  }

  /* Hero (Artículo Destacado) */
  .post-card.bento-hero {
    padding: 24px;
    background: var(--md-surface-variant); /* Le damos un fondo sutilmente distinto para destacar */
  }

  @media (min-width: 800px) {
    .post-card.bento-hero .card-layout { align-items: center; }
    .post-card.bento-hero .image-side {
      width: 45%;
      height: auto;
      aspect-ratio: 16 / 9;
      margin: 0;
    }
    .post-card.bento-hero .title {
      font-size: 32px;
      font-weight: 900;
      line-height: 1.2;
      margin-bottom: 12px;
      -webkit-line-clamp: 4;
    }
    .post-card.bento-hero .summary { font-size: 16px; -webkit-line-clamp: 3; }
  }
  @media (max-width: 799px) {
    .post-card.bento-hero .card-layout { flex-direction: column-reverse; }
    .post-card.bento-hero .image-side { width: 100%; aspect-ratio: 16 / 9; margin-bottom: 16px; }
    .post-card.bento-hero .title { font-size: 24px; line-height: 1.2; }
  }

  /* Magazine Mode (Exact Match code.html) */
  .post-card.mag-hero, .post-card.mag-side-img {
    padding: 0;
    overflow: hidden;
  }
  .post-card.mag-hero .content-side { padding: 32px; justify-content: flex-start; }
  .post-card.mag-side-img .content-side { padding: 24px; }
  
  .post-card.mag-hero .card-layout, 
  .post-card.mag-side-img .card-layout {
    flex-direction: column-reverse;
    gap: 0;
  }
  
  .post-card.mag-hero .image-side {
    width: 100%;
    height: auto;
    aspect-ratio: 16 / 9;
    margin: 0;
    border-radius: 0;
    position: relative;
  }
  
  .post-card.mag-side-img .image-side {
    width: 100%;
    height: 220px;
    margin: 0;
    border-radius: 0;
  }
  
  .post-card.mag-side-text {
    padding: 32px;
    background: var(--md-surface-variant);
  }
  .post-card.mag-side-text .image-side {
    display: none !important;
  }
  
  .post-card.mag-horizontal {
    padding: 24px;
  }
  .post-card.mag-horizontal .card-layout {
    flex-direction: row-reverse;
    align-items: center;
    gap: 24px;
  }
  .post-card.mag-horizontal .image-side {
    width: 220px;
    height: 180px;
    margin: 0;
    border-radius: 12px;
    flex-shrink: 0;
  }
  
  .post-card.mag-side-text .content-side {
    justify-content: space-between;
  }

  /* Magazine Typography */
  .post-card.mag-hero .title { font-size: 28px; font-weight: 800; margin-bottom: 16px; line-height: 1.3; }
  .post-card.mag-hero .source-tag { color: var(--md-primary); font-size: 12px; font-weight: 800; opacity: 1; }
  .post-card.mag-hero .summary { font-size: 16px; -webkit-line-clamp: 3; }
  
  .post-card.mag-horizontal .title { font-size: 20px; }
  .post-card.mag-side-text .title { font-size: 20px; margin-bottom: 12px; }
  .post-card.mag-side-text .summary { 
    -webkit-line-clamp: 12; 
    font-size: 14px; 
    line-height: 1.6; 
    margin-bottom: auto; 
    opacity: 0.8;
    margin-top: 8px;
  }
  .post-card.mag-side-text .source-tag { color: var(--md-tertiary); opacity: 1; }

  .hero-badge { position: absolute; top: 16px; left: 16px; background: var(--md-primary); color: var(--md-on-primary); font-size: 10px; font-weight: 800; padding: 4px 12px; border-radius: var(--radius-full); text-transform: uppercase; letter-spacing: 1px; }
  .hero-read-more { margin-top: auto; padding-top: 24px; font-size: 12px; font-weight: 800; color: var(--md-primary); text-transform: uppercase; letter-spacing: 1px; display: flex; align-items: center; gap: 8px; }
  .hero-read-more .arrow { font-size: 16px; }

  @media (max-width: 799px) {
    .post-card.mag-horizontal .card-layout { flex-direction: column-reverse; }
    .post-card.mag-horizontal .image-side { width: 100%; aspect-ratio: 16/9; height: auto; }
    .post-card.mag-hero .content-side { padding: 20px; }
    .post-card.mag-hero .title { font-size: 22px; }
  }

  /* Shared Components */
  .title-row {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 12px;
    margin-bottom: 8px;
  }

  .title {
    font-family: var(--font-display);
    font-size: 18px;
    font-weight: 500;
    margin: 0;
    flex: 1;
    line-height: 1.3;
    color: var(--md-on-surface-variant);
    display: -webkit-box;
    -webkit-line-clamp: 3;
    -webkit-box-orient: vertical;
    overflow: hidden;
    transition: all 0.2s;
  }

  .post-card:not(.read) .title { font-weight: 800; color: var(--md-on-surface); }
  .post-card.selected .title { color: var(--md-primary); }

  .time {
    font-family: var(--font-main);
    font-size: 11px;
    color: var(--md-on-surface-variant);
    opacity: 0.6;
    flex-shrink: 0;
    white-space: nowrap;
    margin-top: 4px;
    font-weight: 500;
  }

  .post-card:not(.read) .time { color: var(--md-primary); opacity: 1; font-weight: 700; }

  .summary {
    font-family: var(--font-main);
    font-size: 13px;
    color: var(--md-on-surface-variant);
    margin: 0;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
    line-height: 1.5;
    opacity: 0.6;
  }

  .post-card:not(.read) .summary { opacity: 0.85; }

  .footer-row {
    margin-top: 10px;
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .meta { display: flex; align-items: center; gap: 8px; min-width: 0; }
  .dot { opacity: 0.3; font-size: 12px; }
  .time-mini { font-size: 11px; opacity: 0.6; white-space: nowrap; }

  .source-tag {
    font-size: 10px;
    font-weight: 600;
    color: var(--md-on-surface-variant);
    opacity: 0.5;
    text-transform: uppercase;
    letter-spacing: 0.6px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 120px;
  }

  .actions { display: flex; gap: 4px; }

  .action-btn {
    background: none;
    border: none;
    padding: 6px;
    border-radius: 50%;
    color: var(--md-on-surface-variant);
    opacity: 0.4;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.2s ease;
  }

  .action-btn:hover { opacity: 1; background: var(--md-surface-variant); color: var(--md-primary); }
  .action-btn.starred { color: #FFB300; opacity: 1; fill: #FFB300; }

  /* Specialty States */
  .post-card.highlight { background: var(--md-secondary-container); border-color: var(--md-secondary); border-width: 2px; }
  .post-card.silent { opacity: 0.25 !important; filter: grayscale(1); transform: scale(0.98); }
</style>
