<script>
  import './app.css';
  let searchQuery = "";
  
  const news = [
    { id: 1, title: "Quantum Leaps: The Next Era of Computing", source: "The Verge", time: "14 min", category: "Tech", size: "large", img: "https://images.unsplash.com/photo-1635070041078-e363dbe005cb?auto=format&fit=crop&w=800&q=80" },
    { id: 2, title: "Market Trends: Q3 Analysis", source: "Forbes", time: "8 min", category: "Finance", size: "small", trend: "negative" },
    { id: 3, title: "Product Design: Crafting the Future", source: "Design", time: "10 min", category: "Design", size: "medium", img: "https://images.unsplash.com/photo-1558655146-d09347e92766?auto=format&fit=crop&w=600&q=80" },
    { id: 4, title: "AI in Society", source: "The Atlantic", time: "18 min", category: "Society", size: "small" },
    { id: 5, title: "The Minimalist Home", source: "Dwell", time: "12 min", category: "Architecture", size: "wide" }
  ];
</script>

<main>
  <!-- Header / Nav -->
  <header style="padding: 20px 40px; display: flex; justify-content: space-between; align-items: center; border-bottom: 1px solid var(--border-primary);">
    <div style="font-weight: 800; font-size: 24px; letter-spacing: -1px; color: var(--accent-amber);">GORRSON</div>
    <div style="display: flex; gap: 24px; align-items: center; color: var(--text-secondary); font-size: 14px;">
      <span style="color: var(--accent-amber);">Inbox</span>
      <span>Favorites</span>
      <span>History</span>
      <div style="width: 32px; height: 32px; border-radius: 50%; background: #333; border: 1px solid var(--border-primary);"></div>
    </div>
  </header>

  <!-- Bento Grid -->
  <div class="bento-grid">
    {#each news as item}
      <div class="card {item.size}">
        {#if item.img}
          <div style="height: 180px; width: 100%; border-radius: 4px; overflow: hidden; margin-bottom: 16px;">
            <img src={item.img} alt={item.title} style="width: 100%; height: 100%; object-fit: cover; opacity: 0.8;" />
          </div>
        {/if}
        <div style="font-size: 12px; color: var(--accent-amber); text-transform: uppercase; letter-spacing: 1px; margin-bottom: 8px;">
          {item.source} • {item.time}
        </div>
        <h2 style="font-size: 20px; font-weight: 700; line-height: 1.2; margin: 0 0 12px 0;">{item.title}</h2>
        <div style="margin-top: auto; display: flex; gap: 8px;">
          <span style="font-size: 10px; padding: 2px 8px; border: 1px solid var(--border-primary); border-radius: 4px; color: var(--text-secondary);">
            {item.category}
          </span>
        </div>
      </div>
    {/each}
  </div>

  <!-- Command Palette (Floating) -->
  <div class="command-palette">
    <span style="color: var(--accent-amber);">⚡</span>
    <input type="text" placeholder="Search or jump to... (⌘K)" bind:value={searchQuery} />
    <span class="kbd">⌘K</span>
  </div>
</main>

<style>
  main {
    min-height: 100vh;
    display: flex;
    flex-direction: column;
  }
</style>
