<script>
  import { stripHtml } from './utils.js';
  /** @type {{id: number, title: string, source: string, time: string, category: string, content?: string, selected?: boolean}} */
  export let article;
  
  /** @type {(id: number) => void} */
  export let onClick;

  $: summary = stripHtml(article.content || article.summary || "");
</script>

<button 
  class="post-card {article.selected ? 'selected' : ''}" 
  on:click={() => onClick(article.id)}
>
  <div class="header">
    <span class="source">{article.source}</span>
    <span class="time">{article.time}</span>
  </div>
  <h3 class="title" class:active-title={article.selected}>{article.title}</h3>
  <p class="summary">
    {summary || "No preview available..."}
  </p>
</button>

<style>
  .post-card {
    background: var(--md-surface-variant);
    border: 1px solid var(--md-outline);
    border-radius: var(--radius-m);
    padding: 16px 20px;
    margin-bottom: 8px;
    cursor: pointer;
    transition: all 0.2s cubic-bezier(0, 0, 0.2, 1);
    width: 100%;
    outline: none;
    display: block;
    text-align: left;
  }

  .post-card:hover {
    background: var(--md-secondary-container);
    border-color: var(--md-primary);
    transform: translateX(2px);
  }

  .post-card.selected {
    background: var(--md-primary-container);
    border-color: var(--md-primary);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  }

  .header {
    display: flex;
    justify-content: space-between;
    margin-bottom: 6px;
  }

  .source {
    font: var(--m3-label-small);
    font-weight: 700;
    color: var(--md-primary);
    text-transform: uppercase;
    letter-spacing: 1px;
  }

  .time {
    font: var(--m3-label-small);
    color: var(--md-on-surface-variant);
    opacity: 0.6;
  }

  .title {
    font: var(--m3-title-medium);
    margin: 0 0 6px 0;
    line-height: 1.4;
    color: var(--md-on-surface);
  }

  .summary {
    font: var(--m3-body-small);
    color: var(--md-on-surface-variant);
    margin: 0;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
    line-height: 1.5;
    opacity: 0.8;
  }
</style>
