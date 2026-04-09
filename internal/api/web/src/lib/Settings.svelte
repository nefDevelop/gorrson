<script>
  import { t, locale } from './i18n.js';
  import { Globe, Shield, Info, LogOut, Palette, Moon, Sun, Monitor, Settings as SettingsIcon } from 'lucide-svelte';
  import Filters from './Filters.svelte';
  import { onMount } from 'svelte';
  import { Share2, Plus, Trash2, ExternalLink as ExtIcon, Download, Upload, FileJson, FileCode } from 'lucide-svelte';
  import { fly, fade } from 'svelte/transition';

  export let onLogout = () => {};
  export let onRefresh = () => {};
  export let feeds = [];
  export let username = '';
  export let scrollRead = false;
  export let cardStyle = 'default';
  export let isMobile = false;

  import { updateFeed, deleteFeed, fetchAPIKeys, createAPIKey, deleteAPIKey } from './api.js';

  async function handleUpdateFeed(id, title, category) {
    try {
      await updateFeed(id, { title, category });
      onRefresh();
    } catch (e) {
      console.error(e);
    }
  }

  let feedToDelete = null;

  async function confirmDeleteFeed() {
    if (!feedToDelete) return;
    try {
      const idToDelete = feedToDelete.id;
      await deleteFeed(idToDelete);
      feeds = feeds.filter(f => f.id !== idToDelete);
      feedToDelete = null;
      onRefresh();
    } catch (e) {
      console.error(e);
    }
  }

  async function handleDeleteFeed(feed) {
    feedToDelete = feed;
  }

  $: uniqueCategories = Array.from(new Set(feeds.map(f => f.category).filter(Boolean))).sort();

  let activeTab = 'general';
  
  // Theme state
  let currentTheme = localStorage.getItem('gorrson-theme') || 'lavender';
  let currentMode = localStorage.getItem('gorrson-mode') || 'dark';
  let customColor = localStorage.getItem('gorrson-custom-color') || '#6750A4';


  const modes = [
    { id: 'light', icon: Sun },
    { id: 'dark', icon: Moon },
    { id: 'system', icon: Monitor }
  ];

  function hexToHsl(hex) {
    let r = parseInt(hex.slice(1, 3), 16) / 255;
    let g = parseInt(hex.slice(3, 5), 16) / 255;
    let b = parseInt(hex.slice(5, 7), 16) / 255;
    let max = Math.max(r, g, b), min = Math.min(r, g, b);
    let h, s, l = (max + min) / 2;
    if (max === min) h = s = 0;
    else {
      let d = max - min;
      s = l > 0.5 ? d / (2 - max - min) : d / (max + min);
      switch (max) {
        case r: h = (g - b) / d + (g < b ? 6 : 0); break;
        case g: h = (b - r) / d + 2; break;
        case b: h = (r - g) / d + 4; break;
      }
      h /= 6;
    }
    return [h * 360, s * 100, l * 100];
  }

  function hslToHex(h, s, l) {
    l /= 100;
    const a = s * Math.min(l, 1 - l) / 100;
    const f = n => {
      const k = (n + h / 30) % 12;
      const color = l - a * Math.max(Math.min(k - 3, 9 - k, 1), -1);
      return Math.round(255 * color).toString(16).padStart(2, '0');
    };
    return `#${f(0)}${f(8)}${f(4)}`;
  }

  function generateTonal(hex, mode) {
    const [h, s, l] = hexToHsl(hex);
    const isDark = mode === 'dark';

    // MD3 Expressive: Vibrant palettes with higher chroma
    const primary = isDark ? hslToHex(h, Math.max(s, 50), 80) : hex;
    const secondary = hslToHex((h + 25) % 360, Math.max(s - 15, 30), isDark ? 70 : 40);
    const secondaryContainer = isDark 
      ? hslToHex((h + 25) % 360, Math.max(s - 15, 30), 25)
      : hslToHex((h + 25) % 360, Math.max(s - 15, 30), 90);
    const onSecondaryContainer = isDark
      ? hslToHex((h + 25) % 360, Math.max(s - 15, 30), 90)
      : hslToHex((h + 25) % 360, Math.max(s - 15, 30), 10);
    const tertiary = hslToHex((h + 80) % 360, Math.max(s, 40), isDark ? 75 : 45);

    
    // Adwaita 4 Standard Surfaces (Tinted slightly with Seed Hue)
    const adwaita = {
      dark: { 
        bg: hslToHex(h, 5, 12), 
        surface: hslToHex(h, 6, 16), 
        card: hslToHex(h, 7, 20), 
        border: hslToHex(h, 10, 24) 
      },
      light: { 
        bg: '#FFFFFF', 
        surface: hslToHex(h, 10, 96), 
        card: '#FFFFFF', 
        border: hslToHex(h, 15, 88) 
      }
    };
    const currentA = isDark ? adwaita.dark : adwaita.light;

    if (!isDark) {
      return {
        primary: hex,
        onPrimary: '#FFFFFF',
        container: hslToHex(h, Math.max(s, 70), 92), // Very saturated for "Expressive"
        onContainer: hslToHex(h, s, 10),
        secondary: secondary,
        secondaryContainer: secondaryContainer,
        onSecondaryContainer: onSecondaryContainer,
        tertiary: tertiary,
        surface: currentA.surface,
        onSurface: hslToHex(h, 20, 10),
        onSurfaceVariant: hslToHex(h, 15, 40),
        background: currentA.bg,
        variant: currentA.card,
        outline: currentA.border
      };
    } else {
      return {
        primary: primary,
        onPrimary: hslToHex(h, s, 15),
        container: hslToHex(h, Math.max(s, 60), 25), // Saturated highlight
        onContainer: hslToHex(h, Math.max(s, 40), 90),
        secondary: secondary,
        secondaryContainer: secondaryContainer,
        onSecondaryContainer: onSecondaryContainer,
        tertiary: tertiary,
        surface: currentA.surface,
        onSurface: hslToHex(h, 10, 85),
        onSurfaceVariant: hslToHex(h, 15, 65),
        background: currentA.bg,
        variant: currentA.card,
        outline: currentA.border
      };
    }
  }

  function applyAppearance(themeId, modeId, seed) {
    if (typeof document === 'undefined') return;
    const root = document.documentElement;
    
    let mode = modeId;
    if (mode === 'system') {
      mode = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
    }

    // Since we only use Seed Color now
    const tonals = generateTonal(seed, mode);
    
    // MD3 Core Tokens
    root.style.setProperty('--md-primary', tonals.primary);
    root.style.setProperty('--md-on-primary', tonals.onPrimary);
    root.style.setProperty('--md-primary-container', tonals.container);
    root.style.setProperty('--md-on-primary-container', tonals.onContainer);
    
    root.style.setProperty('--md-secondary', tonals.secondary);
    root.style.setProperty('--md-secondary-container', tonals.secondaryContainer);
    root.style.setProperty('--md-on-secondary-container', tonals.onSecondaryContainer);
    root.style.setProperty('--md-tertiary', tonals.tertiary);

    // Adwaita + Tonal Alignment
    root.style.setProperty('--md-surface', tonals.surface);
    root.style.setProperty('--md-on-surface', tonals.onSurface);
    root.style.setProperty('--md-on-surface-variant', tonals.onSurfaceVariant);
    
    root.style.setProperty('--md-background', tonals.background);
    root.style.setProperty('--md-surface-variant', tonals.variant);
    root.style.setProperty('--md-outline', tonals.outline);
    
    // UI Effects
    root.style.setProperty('--shadow-color', mode === 'light' ? 'rgba(0,0,0,0.06)' : 'rgba(0,0,0,0.3)');

    localStorage.setItem('gorrson-theme', themeId);
    localStorage.setItem('gorrson-mode', modeId);
    localStorage.setItem('gorrson-custom-color', seed);
    localStorage.setItem('gorrson-card-style', cardStyle);
  }

  $: if (typeof document !== 'undefined') {
     applyAppearance('custom', currentMode, customColor);
  }

  const toggleLanguage = () => {
    locale.update(l => l === 'en' ? 'es' : 'en');
  };



  $: categoriesList = Array.from(new Set(feeds.map(f => f.category || 'General'))).sort();

  let editingCat = null;
  let newName = "";

  async function renameCategory(oldName) {
    if (!newName || newName === oldName) {
       editingCat = null;
       return;
    }
    const res = await fetch(`/categories/${encodeURIComponent(oldName)}`, {
      method: 'PUT',
      headers: { 
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      },
      body: JSON.stringify({ name: newName })
    });
    if (res.ok) {
      window.location.reload(); // Hard reload to refresh everything
    }
    editingCat = null;
  }

  // Export Services state
  let exportServices = JSON.parse(localStorage.getItem('gorrson-export-services') || '[]');
  let newServiceName = "";
  let newServiceTemplate = "";

  function saveServices() {
    localStorage.setItem('gorrson-export-services', JSON.stringify(exportServices));
    // Trigger update in app
    window.dispatchEvent(new Event('storage'));
  }

  function addService() {
    if (!newServiceName || !newServiceTemplate) return;
    exportServices = [...exportServices, { 
      id: Date.now().toString(), 
      name: newServiceName, 
      template: newServiceTemplate 
    }];
    newServiceName = "";
    newServiceTemplate = "";
    saveServices();
  }

  function removeService(id) {
    exportServices = exportServices.filter(s => s.id !== id);
    saveServices();
  }

  // Data Import/Export
  async function exportConfig() {
    try {
      const res = await fetch('/config/export', {
        headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` }
      });
      const serverConfig = await res.json();
      
      const config = {
        ...serverConfig,
        local: {
          theme: localStorage.getItem('gorrson-theme'),
          mode: localStorage.getItem('gorrson-mode'),
          color: localStorage.getItem('gorrson-custom-color'),
          services: exportServices,
          locale: localStorage.getItem('gorrson-locale'),
          scrollRead: scrollRead
        },
        exportedAt: new Date().toISOString()
      };
      
      const blob = new Blob([JSON.stringify(config, null, 2)], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `gorrson-config-${new Date().toISOString().split('T')[0]}.json`;
      a.click();
    } catch (e) {
      console.error(e);
      alert('Export failed');
    }
  }

  async function importConfig(event) {
    const file = event.target.files[0];
    if (!file) return;
    
    const reader = new FileReader();
    reader.onload = async (e) => {
      try {
        if (typeof e.target.result !== 'string') return;
        const config = JSON.parse(e.target.result);
        
        // Save local settings
        if (config.local) {
          const l = config.local;
          if (l.theme) localStorage.setItem('gorrson-theme', l.theme);
          if (l.mode) localStorage.setItem('gorrson-mode', l.mode);
          if (l.color) localStorage.setItem('gorrson-custom-color', l.color);
          if (l.services) {
             exportServices = l.services;
             saveServices();
          }
          if (l.locale) locale.set(l.locale);
          if (l.scrollRead !== undefined) scrollRead = l.scrollRead;
        }

        // Upload to backend
        const res = await fetch('/config/import', {
          method: 'POST',
          headers: { 
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${localStorage.getItem('token')}` 
          },
          body: JSON.stringify({ filters: config.filters })
        });

        if (res.ok) {
           alert('Configuration imported. Page will refresh.');
           window.location.reload();
        }
      } catch (err) {
        console.error(err);
        alert('Invalid config file');
      }
    };
    reader.readAsText(file);
  }

  async function exportOPMLBlob() {
    const res = await fetch('/feeds/export', {
      headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` }
    });
    const blob = await res.blob();
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'gorrson.opml';
    a.click();
  }

  async function importOPML(event) {
    const file = event.target.files[0];
    if (!file) return;
    
    const formData = new FormData();
    formData.append('file', file);
    
    try {
      const res = await fetch('/feeds/import', {
        method: 'POST',
        headers: { 
          'Authorization': `Bearer ${localStorage.getItem('token')}` 
        },
        body: formData
      });
      if (res.ok) {
        alert('OPML Import started. Feeds will appear shortly.');
        onRefresh();
      }
    } catch (e) {
      console.error(e);
      alert('OPML import failed');
    }
  }

  const examples = [
    { name: 'Obsidian', template: 'obsidian://new?title={title}&content={url}' },
    { name: 'Pocket', template: 'https://getpocket.com/edit?url={url}' },
    { name: 'WhatsApp', template: 'https://wa.me/?text={title}%20{url}' },
    { name: 'Telegram', template: 'https://t.me/share/url?url={url}&text={title}' }
  ];

  // API Keys state
  let apiKeys = [];
  let newKeyName = "";
  let generatedKey = null;

  async function loadAPIKeys() {
    try {
      apiKeys = await fetchAPIKeys();
    } catch (e) {
      console.error(e);
    }
  }

  $: if (activeTab === 'keys') {
    loadAPIKeys();
  }

  async function handleCreateKey() {
    if (!newKeyName) return;
    try {
      const data = await createAPIKey(newKeyName);
      generatedKey = data.api_key;
      newKeyName = "";
      loadAPIKeys();
    } catch (e) {
      alert(e.message);
    }
  }

  async function handleDeleteKey(id) {
    if (!confirm("Are you sure you want to revoke this API key? Devices using it will lose access.")) return;
    try {
      await deleteAPIKey(id);
      loadAPIKeys();
    } catch (e) {
      alert(e.message);
    }
  }
</script>

<div class="view-container">
  <header class="header-bar">
    {#if !isMobile}
      <div class="header-left">
        <SettingsIcon size={20} />
        <h1 class="title">{$t('settings')}</h1>
      </div>
    {/if}
    <div class="tabs">
      <button class="tab {activeTab === 'general' ? 'active' : ''}" on:click={() => activeTab = 'general'}>General</button>
      <button class="tab {activeTab === 'sources' ? 'active' : ''}" on:click={() => activeTab = 'sources'}>{$t('sources')}</button>
      <button class="tab {activeTab === 'filters' ? 'active' : ''}" on:click={() => activeTab = 'filters'}>{$t('filters')}</button>
      <button class="tab {activeTab === 'sharing' ? 'active' : ''}" on:click={() => activeTab = 'sharing'}>{$t('sharing')}</button>
      <button class="tab {activeTab === 'keys' ? 'active' : ''}" on:click={() => activeTab = 'keys'}>API Keys</button>
      <button class="tab {activeTab === 'data' ? 'active' : ''}" on:click={() => activeTab = 'data'}>Data</button>
    </div>
  </header>

  <!-- Sharp edge fix for unibody rounding -->
  <div class="corner-bg"></div>
  <div class="corner-arc"></div>

  <div class="content-scroll scroll-area">
    {#if activeTab === 'general'}
      <div class="settings-section">
        <div class="section-header">
          <Palette size={18} />
          <span>Appearance</span>
        </div>
        
        <div class="setting-item">
          <div class="text">
            <div class="label">Accent Color</div>
            <div class="desc">Tints the entire workspace with your brand/style.</div>
          </div>
          <div class="seed-compact-picker">
            <span class="seed-hex">{customColor.toUpperCase()}</span>
            <div class="color-preview-circle" style="background: {customColor}">
              <input type="color" bind:value={customColor} />
            </div>
          </div>
        </div>

        <div class="setting-item">
          <div class="text">
            <div class="label">Theme Mode</div>
            <div class="desc">Switch between light and dark backgrounds.</div>
          </div>
          <div class="mode-selector">
            {#each modes as mode}
              <button 
                class="mode-btn {currentMode === mode.id ? 'active' : ''}" 
                on:click={() => currentMode = mode.id}
                title={mode.id}
              >
                <svelte:component this={mode.icon} size={18} />
              </button>
            {/each}
          </div>
        </div>

        <div class="setting-item">
          <div class="text">
            <div class="label">Language / Idioma</div>
            <div class="desc">Chose your preferred interface language.</div>
          </div>
          <button class="btn-toggle" on:click={toggleLanguage}>
            {$locale === 'en' ? 'English' : 'Español'}
          </button>
        </div>
        <div class="setting-item">
          <div class="text">
            <div class="label">Auto-mark Read</div>
            <div class="desc">Mark articles as read when you scroll past them.</div>
          </div>
          <button class="btn-toggle" on:click={() => scrollRead = !scrollRead}>
            {scrollRead ? 'ON' : 'OFF'}
          </button>
        </div>

        <div class="setting-item vertical">
          <div class="text">
            <div class="label">News Card Style</div>
            <div class="desc">Choose how articles are displayed in the feed.</div>
          </div>
          <div class="style-selector">
            <button class="style-btn {cardStyle === 'default' ? 'active' : ''}" on:click={() => cardStyle = 'default'}>
              <div class="style-preview def"></div>
              <span>Default</span>
            </button>
            <button class="style-btn {cardStyle === 'image' ? 'active' : ''}" on:click={() => cardStyle = 'image'}>
              <div class="style-preview img"></div>
              <span>Image</span>
            </button>
            <button class="style-btn {cardStyle === 'compact' ? 'active' : ''}" on:click={() => cardStyle = 'compact'}>
              <div class="style-preview com"></div>
              <span>Compact</span>
            </button>
            <button class="style-btn {cardStyle === 'minimal' ? 'active' : ''}" on:click={() => cardStyle = 'minimal'}>
              <div class="style-preview min"></div>
              <span>Minimal</span>
            </button>
          </div>
        </div>
      </div>

      <div class="settings-section">
        <div class="section-header">
          <Shield size={18} />
          <span>Account</span>
        </div>
        <div class="setting-item">
          <div class="text">
            <div class="label">User Profile</div>
            <div class="desc">Logged in as <strong>{username}</strong></div>
          </div>
          <button class="btn-danger" on:click={onLogout}>
            <LogOut size={16} />
            <span>{$t('logout')}</span>
          </button>
        </div>
      </div>

      <div class="settings-section">
        <div class="section-header">
          <Info size={18} />
          <span>System</span>
        </div>
        <div class="info-grid">
          <div class="info-item">
            <span class="key">Version</span>
            <span class="value">v1.2.0-stable</span>
          </div>
          <div class="info-item">
            <span class="key">Engine</span>
            <span class="value">Go 1.22 + SQLite FTS5</span>
          </div>
          <div class="info-item">
            <span class="key">Architecture</span>
            <span class="value">Multi-tenant (Isolated databases)</span>
          </div>
        </div>
      </div>
    {:else if activeTab === 'categories'}
      <div class="settings-section">
        <div class="section-header">
          <Globe size={18} />
          <span>Manage Categories</span>
        </div>
        <div class="category-manager">
          {#each categoriesList as cat}
            <div class="cat-row">
              {#if editingCat === cat}
                <input 
                  class="edit-input"
                  bind:value={newName}
                  on:keydown={e => e.key === 'Enter' && renameCategory(cat)}
                  on:blur={() => editingCat = null}
                />
                <button class="btn-primary-small" on:click={() => renameCategory(cat)}>Save</button>
              {:else}
                <span class="cat-name">{cat}</span>
                <div class="row-actions">
                  <button class="btn-ghost" on:click={() => { editingCat = cat; newName = cat; }}>Rename</button>
                </div>
              {/if}
            </div>
          {/each}
        </div>
      </div>
    {:else if activeTab === 'sharing'}
      <div class="settings-section">
        <div class="section-header">
          <Share2 size={18} />
          <span>Custom Sharing & Export</span>
        </div>
        
        <div class="service-form">
          <input bind:value={newServiceName} placeholder={$t('service_name')} class="modern-input" />
          <input bind:value={newServiceTemplate} placeholder={$t('service_template')} class="modern-input wide" />
          <button class="btn-primary" on:click={addService}>
            <Plus size={18} />
            <span>{$t('add_service')}</span>
          </button>
        </div>

        <div class="examples-row">
          <span class="label">Presets:</span>
          {#each examples as ex}
            <button class="example-chip" on:click={() => { newServiceName = ex.name; newServiceTemplate = ex.template; }}>
              {ex.name}
            </button>
          {/each}
        </div>

        <div class="services-list">
          {#each exportServices as svc}
            <div class="svc-card">
              <div class="svc-info">
                <span class="svc-name">{svc.name}</span>
                <span class="svc-template">{svc.template}</span>
              </div>
              <button class="btn-icon-danger" on:click={() => removeService(svc.id)}>
                <Trash2 size={18} />
              </button>
            </div>
          {:else}
            <div class="empty-list">No custom services added yet. Use presets to get started.</div>
          {/each}
        </div>
      </div>
    {:else if activeTab === 'sources'}
      <div class="settings-section">
        <div class="section-header">
          <Globe size={18} />
          <span>{$t('sources')} Management</span>
        </div>
        <div class="feeds-admin-list">
          {#each feeds as feed}
            <div class="admin-feed-card" in:fly={{ y: 10, duration: 200 }}>
              <div class="feed-main-info">
                 <div class="input-stack">
                   <span class="tiny-label">Title</span>
                   <input 
                      class="modern-input title-input" 
                      bind:value={feed.title} 
                      on:change={() => handleUpdateFeed(feed.id, feed.title, feed.category)} 
                    />
                 </div>
                 <div class="input-stack cate-stack">
                   <span class="tiny-label">Category</span>
                   <input 
                      class="modern-input cat-input" 
                      bind:value={feed.category} 
                      list="category-suggestions"
                      on:change={() => handleUpdateFeed(feed.id, feed.title, feed.category)} 
                    />
                 </div>
              </div>
              <button class="btn-icon-danger" on:click={() => handleDeleteFeed(feed)} title="Delete Feed">
                 <Trash2 size={18} />
              </button>
            </div>
          {:else}
            <div class="empty-list">No feeds found. Add some from the sidebar.</div>
          {/each}
        </div>
      </div>
      <datalist id="category-suggestions">
        {#each uniqueCategories as cat}
          <option value={cat}></option>
        {/each}
      </datalist>
    {:else if activeTab === 'filters'}
      <div class="settings-section">
        <div class="section-header">
          <Shield size={18} />
          <span>Filters & Content Cleaning</span>
        </div>
        <div class="filters-view-embedded">
          <Filters {feeds} headless={true} />
        </div>
      </div>
    {:else if activeTab === 'data'}
      <div class="settings-section">
        <div class="section-header">
          <FileCode size={18} />
          <span>Import / Export OPML</span>
        </div>
        <div class="data-actions">
           <div class="data-card">
              <div class="icon-box"><Globe size={24} /></div>
              <div class="card-text">
                 <strong>Feeds (OPML)</strong>
                 <p>Standard format to move your subscriptions between apps.</p>
              </div>
              <div class="actions">
                 <button class="btn-secondary" on:click={exportOPMLBlob}>
                    <Download size={16} />
                    <span>Export</span>
                 </button>
                 <label class="btn-primary file-label">
                    <Upload size={16} />
                    <span>Import</span>
                    <input type="file" accept=".opml,.xml" on:change={importOPML} style="display:none" />
                 </label>
              </div>
           </div>
        </div>
      </div>

      <div class="settings-section">
        <div class="section-header">
          <FileJson size={18} />
          <span>Full Configuration (JSON)</span>
        </div>
        <div class="data-actions">
           <div class="data-card">
              <div class="icon-box"><Shield size={24} /></div>
              <div class="card-text">
                 <strong>Filters & Settings</strong>
                 <p>Backup all your filtering rules, custom sharing services, and theme preferences.</p>
              </div>
              <div class="actions">
                 <button class="btn-secondary" on:click={exportConfig}>
                    <Download size={16} />
                    <span>Export</span>
                 </button>
                 <label class="btn-primary file-label">
                    <Upload size={16} />
                    <span>Import</span>
                    <input type="file" accept=".json" on:change={importConfig} style="display:none" />
                 </label>
              </div>
           </div>
        </div>
      </div>
    {:else if activeTab === 'keys'}
      <div class="settings-section">
        <div class="section-header">
          <Shield size={18} />
          <span>Device API Keys</span>
        </div>
        <p class="section-hint">Use these keys to authenticate devices (mobile apps, terminal clients) without using your main password. Format for header: <code>X-API-Key: user:key</code></p>
        
        <div class="key-form">
          <input 
            bind:value={newKeyName} 
            placeholder="Key name (e.g. My iPhone)" 
            class="modern-input"
            on:keydown={e => e.key === 'Enter' && handleCreateKey()}
          />
          <button class="btn-primary" on:click={handleCreateKey} disabled={!newKeyName}>
            <Plus size={18} />
            <span>Create Key</span>
          </button>
        </div>

        <div class="keys-list">
          {#each apiKeys as key}
            <div class="key-card" in:fly={{ y: 10, duration: 200 }}>
              <div class="key-info">
                <span class="key-name">{key.name}</span>
                <span class="key-date">Created on {new Date(key.created_at).toLocaleDateString()}</span>
              </div>
              <button class="btn-icon-danger" on:click={() => handleDeleteKey(key.id)} title="Revoke Key">
                <Trash2 size={18} />
              </button>
            </div>
          {:else}
            <div class="empty-list">No API keys created yet.</div>
          {/each}
        </div>
      </div>
    {/if}
  </div>

  {#if generatedKey}
    <div class="modal-overlay" transition:fade={{ duration: 200 }}>
      <div class="modal-card" transition:fly={{ y: 20, duration: 300 }}>
        <div class="modal-icon-header">
           <div class="success-circle">
              <Shield size={24} color="white" />
           </div>
        </div>
        <div class="modal-body">
          <h2>New API Key Created</h2>
          <p>Copy this key now. For security reasons, <strong>it will not be shown again</strong>.</p>
          <div class="key-display">
            <code>{generatedKey}</code>
            <button class="btn-ghost-small" on:click={() => { navigator.clipboard.writeText(generatedKey); alert('Copied!'); }}>Copy</button>
          </div>
        </div>
        <div class="modal-footer">
          <button class="btn-primary" on:click={() => generatedKey = null}>I've saved it</button>
        </div>
      </div>
    </div>
  {/if}

  {#if feedToDelete}
    <div 
      class="modal-overlay" 
      role="button" 
      tabindex="0"
      transition:fade={{ duration: 200 }} 
      on:click={() => feedToDelete = null}
      on:keydown={(e) => e.key === 'Escape' && (feedToDelete = null)}
    >
      <div 
        class="modal-card" 
        role="dialog"
        aria-modal="true"
        tabindex="-1"
        transition:fly={{ y: 20, duration: 300 }} 
        on:click|stopPropagation
        on:keydown|stopPropagation
      >
        <div class="modal-icon-header">
           <div class="warning-circle">
              <Trash2 size={24} />
           </div>
        </div>
        <div class="modal-body">
          <h2>Delete Feed?</h2>
          <p>Are you sure you want to remove <strong>{feedToDelete.title}</strong>? This will delete all its articles from your database. This action cannot be undone.</p>
        </div>
        <div class="modal-footer">
          <button class="btn-secondary" on:click={() => feedToDelete = null}>Cancel</button>
          <button class="btn-danger" on:click={confirmDeleteFeed}>Delete Feed</button>
        </div>
      </div>
    </div>
  {/if}
</div>

<style>
  .view-container { 
    display: flex; 
    flex-direction: column; 
    height: 100vh; 
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
    background: var(--md-surface); 
    gap: 48px; 
    box-sizing: border-box;
    position: relative;
    z-index: 20;
    border-bottom: 1px solid var(--md-outline);
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
    z-index: 20;
    pointer-events: none;
  }

  .header-left { display: flex; align-items: center; gap: 12px; }
  .header-left :global(svg) { color: var(--md-primary); }
  .title { font: var(--m3-headline-small); font-weight: 700; color: var(--md-primary); }

  .tabs { display: flex; gap: 8px; height: 100%; align-items: center; padding-left: 24px; }
  .tab { 
    background: transparent; 
    border: none; 
    color: var(--md-on-surface-variant); 
    font: var(--m3-label-large); 
    padding: 8px 16px; 
    height: 36px;
    display: flex;
    align-items: center;
    cursor: pointer; 
    border-radius: var(--radius-full);
    transition: all 0.2s;
    opacity: 0.8;
  }
  .tab:hover { 
    background: var(--md-surface-variant);
    opacity: 1; 
    color: var(--md-on-surface); 
  }
  .tab.active { 
    background: var(--md-primary-container);
    color: var(--md-primary); 
    font-weight: 800;
    opacity: 1;
  }

  .content-scroll { 
    padding: 48px; 
    flex: 1; 
    width: 100%; 
    box-sizing: border-box; 
    overflow-y: auto; 
    background: var(--md-background);
    border-top-left-radius: 32px;
    border-left: 1px solid var(--md-outline);
    position: relative;
    z-index: 10;
  }
  
  .settings-section { margin-bottom: 48px; max-width: 900px; }
  .section-header { 
    display: flex; 
    align-items: center; 
    gap: 12px; 
    font: var(--m3-title-medium); 
    color: var(--md-primary); 
    margin-bottom: 24px; 
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .setting-item {
    background: var(--md-surface);
    border: 1px solid var(--md-outline);
    border-radius: var(--radius-xl);
    padding: 20px 24px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16px;
    box-shadow: 0 1px 3px rgba(0,0,0,0.05);
  }

  .setting-item.vertical {
    flex-direction: column;
    align-items: stretch;
    gap: 16px;
  }

  .text .label { font: var(--m3-title-medium); color: var(--md-on-surface); margin-bottom: 4px; }
  .text .desc { font: var(--m3-body-small); color: var(--md-on-surface-variant); }

  .btn-toggle {
    background: var(--md-primary-container);
    color: var(--md-on-primary-container);
    border: 1px solid var(--md-primary);
    padding: 8px 16px;
    border-radius: var(--radius-full);
    font: var(--m3-label-large);
    cursor: pointer;
    font-weight: 700;
  }

  .btn-danger {
    background: #ff525222;
    color: #ff5252;
    border: 1px solid #ff525244;
    padding: 8px 20px;
    border-radius: var(--radius-full);
    font: var(--m3-label-large);
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 8px;
    font-weight: 700;
  }

  .info-grid {
    background: var(--md-surface);
    border: 1px solid var(--md-outline);
    border-radius: var(--radius-xl);
    padding: 24px;
    display: grid;
    gap: 12px;
  }

  .info-item { display: flex; justify-content: space-between; font: var(--m3-body-medium); }
  .info-item .key { color: var(--md-on-surface-variant); }
  .info-item .value { color: var(--md-on-surface); font-weight: 600; font-family: var(--font-mono); font-size: 13px; }



  .seed-compact-picker { 
    display: flex; 
    align-items: center; 
    gap: 12px; 
    background: var(--md-surface-variant); 
    padding: 6px 12px; 
    border-radius: var(--radius-full); 
    border: 1px solid var(--md-outline);
  }
  .seed-hex { 
    font-family: var(--font-mono); 
    font-size: 13px; 
    font-weight: 700; 
    color: var(--md-primary); 
    letter-spacing: 0.5px;
  }
  .color-preview-circle { 
    width: 24px; 
    height: 24px; 
    border-radius: 50%; 
    border: 2px solid white; 
    box-shadow: 0 0 0 1px var(--md-outline); 
    position: relative; 
    cursor: pointer;
    transition: transform 0.2s;
  }
  .color-preview-circle:hover { transform: scale(1.15); }
  .color-preview-circle input { 
    position: absolute; 
    inset: 0; 
    width: 100%; 
    height: 100%; 
    opacity: 0; 
    cursor: pointer; 
  }

  .mode-selector { display: flex; background: var(--md-surface); padding: 4px; border-radius: var(--radius-full); border: 1px solid var(--md-outline); }
  .mode-btn { 
    background: transparent; 
    border: none; 
    color: var(--md-on-surface-variant); 
    width: 36px; 
    height: 36px; 
    display: flex; 
    align-items: center; 
    justify-content: center; 
    border-radius: var(--radius-full);
    cursor: pointer;
    transition: all 0.2s;
  }
  .mode-btn.active { background: var(--md-primary); color: var(--md-on-primary); }
  .category-manager {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-top: 16px;
  }
  .cat-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px 24px;
    background: var(--md-surface);
    box-sizing: border-box;
    border: 1px solid var(--md-outline);
    border-radius: var(--radius-xl);
    transition: all 0.2s;
  }
  .cat-row:hover {
    border-color: var(--md-primary);
    background: var(--md-surface-variant);
  }
  .btn-ghost {
    background: transparent;
    border: 1px solid var(--md-outline);
    color: var(--md-primary);
    padding: 6px 16px;
    border-radius: var(--radius-full);
    font: var(--m3-label-large);
    font-weight: 700;
    cursor: pointer;
    transition: all 0.2s;
  }
  .btn-ghost:hover {
    background: var(--md-primary-container);
    border-color: var(--md-primary);
  }
  .cat-row:hover {
    border-color: var(--md-primary);
  }
  .btn-ghost {
    background: transparent;
    border: 1px solid var(--md-outline);
    color: var(--md-primary);
    padding: 6px 16px;
    border-radius: var(--radius-full);
    font: var(--m3-label-large);
    font-weight: 700;
    cursor: pointer;
    transition: all 0.2s;
  }
  .btn-ghost:hover {
    background: var(--md-primary-container);
    border-color: var(--md-primary);
  }
  .cat-name {
    font: var(--m3-body-large);
    font-weight: 700;
    color: var(--md-on-surface);
  }
  .filters-view-embedded {
    margin-top: 16px;
    border: 1px solid var(--md-outline);
    border-radius: 12px;
    padding: 12px;
    background: var(--md-surface-variant);
  }

  /* Sources Management Styles */
  .feeds-admin-list {
    display: flex;
    flex-direction: column;
    gap: 12px;
    margin-top: 16px;
  }

  .admin-feed-card {
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 16px;
    background: var(--md-surface-variant);
    border: 1px solid var(--md-outline);
    border-radius: 16px;
    transition: transform 0.2s, box-shadow 0.2s;
  }

  .admin-feed-card:hover {
    transform: translateY(-2px);
    box-shadow: 0 4px 12px var(--shadow-color);
  }

  .feed-main-info {
    flex: 1;
    display: flex;
    gap: 12px;
  }

  .input-stack {
    flex: 2;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .cate-stack {
    flex: 1;
  }

  .tiny-label {
    font-size: 10px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    font-weight: 700;
    opacity: 0.5;
    color: var(--md-on-surface-variant);
    margin-left: 4px;
  }

  .btn-icon-danger {
    background: transparent;
    border: none;
    color: #ff5252;
    width: 44px;
    height: 44px;
    border-radius: 12px;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn-icon-danger:hover {
    background: rgba(255, 82, 82, 0.1);
    transform: scale(1.05);
  }

  @media (max-width: 800px) {
    .feed-main-info {
      flex-direction: column;
    }
  }

  /* Shared UI Components */
  .modern-input {
    background: var(--md-surface);
    border: 1px solid var(--md-outline);
    border-radius: var(--radius-full);
    padding: 10px 20px;
    color: var(--md-on-surface);
    font: var(--m3-body-medium);
    outline: none;
    transition: all 0.2s;
    width: 100%;
    box-sizing: border-box;
  }

  .modern-input:focus { 
    border-color: var(--md-primary); 
    box-shadow: 0 0 0 2px var(--md-primary-container); 
  }

  .btn-icon-danger {
    background: transparent;
    border: none;
    color: var(--md-on-surface-variant);
    cursor: pointer;
    width: 40px;
    height: 40px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.2s;
  }

  .btn-icon-danger:hover { 
    background: #ff525222; 
    color: #ff5252; 
  }

  .btn-primary-small {
    background: var(--md-primary);
    color: var(--md-on-primary);
    border: none;
    padding: 8px 16px;
    border-radius: var(--radius-full);
    font: var(--m3-label-medium);
    font-weight: 700;
    cursor: pointer;
  }
  .service-form { display: flex; gap: 12px; margin-bottom: 12px; }
  .modern-input.wide { flex: 1; }
  
  .btn-primary {
    background: var(--md-primary);
    color: var(--md-on-primary);
    border: none;
    padding: 12px 24px;
    border-radius: var(--radius-full);
    font: var(--m3-label-large);
    font-weight: 700;
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 8px;
    transition: all 0.2s;
  }
  .btn-primary:hover {
    background: var(--md-primary-container);
    color: var(--md-primary);
    transform: translateY(-1px);
  }

  .examples-row { display: flex; align-items: center; gap: 8px; margin-bottom: 32px; }
  .examples-row .label { font: var(--m3-label-medium); color: var(--md-on-surface-variant); }
  .example-chip {
    background: var(--md-surface-variant);
    border: 1px solid var(--md-outline);
    padding: 4px 12px;
    border-radius: var(--radius-full);
    font: var(--m3-label-small);
    color: var(--md-primary);
    cursor: pointer;
    transition: all 0.2s;
  }
  .example-chip:hover { border-color: var(--md-primary); background: var(--md-primary-container); }

  .services-list { display: flex; flex-direction: column; gap: 12px; }
  .svc-card {
    background: var(--md-surface);
    border: 1px solid var(--md-outline);
    border-radius: var(--radius-xl);
    padding: 16px 24px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    transition: all 0.2s;
  }
  .svc-card:hover { border-color: var(--md-primary); transform: translateX(4px); }
  .svc-info { display: flex; flex-direction: column; gap: 4px; overflow: hidden; }
  .svc-name { font: var(--m3-title-small); color: var(--md-on-surface); font-weight: 700; }
  .svc-template { font: var(--m3-body-small); color: var(--md-on-surface-variant); white-space: nowrap; text-overflow: ellipsis; overflow: hidden; font-family: var(--font-mono); }


  .empty-list { text-align: center; padding: 40px; color: var(--md-on-surface-variant); font: var(--m3-body-large); opacity: 0.6; }

  /* Modal Styles */
  .modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    width: 100vw;
    height: 100vh;
    background: rgba(0, 0, 0, 0.4);
    backdrop-filter: blur(8px);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }

  .modal-card {
    background: var(--md-surface);
    width: 100%;
    max-width: 400px;
    border-radius: 28px;
    border: 1px solid var(--md-outline);
    box-shadow: 0 24px 48px rgba(0,0,0,0.3);
    padding: 32px;
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: center;
    gap: 16px;
  }

  .warning-circle {
    width: 64px;
    height: 64px;
    border-radius: 50%;
    background: rgba(255, 82, 82, 0.1);
    color: #ff5252;
    display: flex;
    align-items: center;
    justify-content: center;
    margin-bottom: 8px;
  }

  .modal-body h2 {
    margin: 0 0 8px 0;
    color: var(--md-on-surface);
    font: var(--m3-headline-small);
  }

  .modal-body p {
    margin: 0;
    color: var(--md-on-surface-variant);
    font: var(--m3-body-medium);
    line-height: 1.6;
  }

  .modal-footer {
    display: flex;
    gap: 12px;
    width: 100%;
    margin-top: 16px;
  }

  .modal-footer button {
    flex: 1;
    height: 48px;
    border-radius: var(--radius-full);
    font-weight: 700;
    cursor: pointer;
    transition: all 0.2s;
    font-family: var(--font-main);
  }

  .btn-secondary {
    background: var(--md-surface-variant);
    border: 1px solid var(--md-outline);
    color: var(--md-on-surface);
    padding: 12px 24px;
    border-radius: var(--radius-full);
    font: var(--m3-label-large);
    font-weight: 700;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    transition: all 0.2s;
  }

  .btn-secondary:hover {
    background: var(--md-outline);
    transform: translateY(-1px);
  }

  .btn-danger {
    background: #ff5252;
    border: none;
    color: white;
  }

  .btn-danger:hover {
    background: #ff3333;
    box-shadow: 0 4px 12px rgba(255, 82, 82, 0.3);
  }

  .data-actions {
    display: flex;
    flex-direction: column;
    gap: 16px;
    margin-top: 24px;
    max-width: 900px;
  }
  .data-card {
    background: var(--md-surface-variant);
    border: 1px solid var(--md-outline);
    border-radius: 20px;
    padding: 24px;
    display: flex;
    align-items: center;
    gap: 20px;
    transition: all 0.2s;
  }
  .data-card:hover {
    border-color: var(--md-primary);
    background: var(--md-secondary-container);
  }
  .icon-box {
    width: 64px;
    height: 64px;
    background: var(--md-primary-container);
    color: var(--md-on-primary-container);
    border-radius: 16px;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }
  .card-text {
    flex: 1;
  }
  .card-text strong {
    display: block;
    font-size: 18px;
    color: var(--md-on-surface);
    margin-bottom: 4px;
    font-family: var(--font-display);
  }
  .card-text p {
    font-size: 14px;
    color: var(--md-on-surface-variant);
    margin: 0;
    opacity: 0.8;
  }
  .data-card .actions {
    display: flex;
    gap: 12px;
  }
  .file-label {
    cursor: pointer;
    margin: 0;
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .style-selector {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 12px;
    margin-top: 16px;
    width: 100%;
  }
  .style-btn {
    background: var(--md-surface-variant);
    border: none;
    border-radius: 16px;
    padding: 16px 12px;
    cursor: pointer;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 10px;
    transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
    color: var(--md-on-surface-variant);
  }
  .style-btn:hover { background: var(--md-secondary-container); color: var(--md-primary); transform: translateY(-2px); }
  .style-btn.active { background: var(--md-primary-container); color: var(--md-primary); }
  
  .style-preview {
    width: 64px;
    height: 48px;
    background: var(--md-surface);
    border-radius: 8px;
    border: none;
    position: relative;
    overflow: hidden;
    box-shadow: inset 0 0 10px rgba(0,0,0,0.05);
  }
  .style-preview.def::after { content: ""; position: absolute; top: 10px; left: 10px; right: 10px; height: 5px; background: var(--md-primary); opacity: 0.4; border-radius: 2px; }
  .style-preview.def::before { content: ""; position: absolute; top: 20px; left: 10px; right: 20px; height: 4px; background: var(--md-on-surface-variant); opacity: 0.15; border-radius: 2px; }
  
  .style-preview.img::after { content: ""; position: absolute; top: 10px; left: 10px; right: 28px; height: 5px; background: var(--md-primary); opacity: 0.4; border-radius: 2px; }
  .style-preview.img::before { content: ""; position: absolute; top: 10px; right: 10px; width: 14px; height: 14px; background: var(--md-secondary); opacity: 0.3; border-radius: 4px; }
  
  .style-preview.com::after { content: ""; position: absolute; top: 8px; left: 8px; right: 8px; height: 4px; background: var(--md-primary); opacity: 0.4; border-radius: 2px; }
  .style-preview.com::before { content: ""; position: absolute; top: 16px; left: 8px; right: 8px; height: 4px; background: var(--md-on-surface-variant); opacity: 0.15; border-radius: 2px; }
  
  .style-preview.min::after { content: ""; position: absolute; top: 10px; left: 10px; right: 10px; height: 5px; background: var(--md-primary); opacity: 0.4; border-radius: 2px; }

  /* API Keys Styles */
  .section-hint { font: var(--m3-label-medium); color: var(--md-on-surface-variant); margin-bottom: 24px; opacity: 0.8; }
  .key-form { display: flex; gap: 12px; margin-bottom: 32px; max-width: 600px; }
  .keys-list { display: flex; flex-direction: column; gap: 12px; max-width: 600px; }
  .key-card { 
    background: var(--md-surface-variant); 
    border: 1px solid var(--md-outline); 
    padding: 16px 20px; 
    border-radius: 16px; 
    display: flex; 
    align-items: center; 
    justify-content: space-between;
    transition: all 0.2s;
  }
  .key-card:hover { border-color: var(--md-primary); background: var(--md-secondary-container); }
  .key-info { display: flex; flex-direction: column; gap: 4px; }
  .key-name { font: var(--m3-title-medium); color: var(--md-on-surface); font-weight: 700; }
  .key-date { font: var(--m3-label-small); color: var(--md-on-surface-variant); opacity: 0.7; }
  
  .success-circle { 
    width: 64px; 
    height: 64px; 
    background: #4CAF50; 
    border-radius: 50%; 
    display: flex; 
    align-items: center; 
    justify-content: center; 
    margin: 0 auto 16px; 
    box-shadow: 0 8px 16px rgba(76, 175, 80, 0.3);
  }
  .key-display { 
    background: var(--md-surface); 
    padding: 16px; 
    border-radius: 12px; 
    border: 1px solid var(--md-outline); 
    display: flex; 
    align-items: center; 
    justify-content: space-between; 
    margin-top: 16px;
    word-break: break-all;
  }
  .key-display code { font-family: 'JetBrains Mono', monospace; font-size: 14px; color: var(--md-primary); flex: 1; margin-right: 12px; }
</style>
