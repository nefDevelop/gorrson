import os, re
with open('internal/api/server.go', 'r') as f:
    text = f.read()

def get_block(name, is_struct=False):
    global text
    if is_struct:
        pattern = re.compile(r'^type ' + name + r' struct \{.*?\n\}', re.MULTILINE | re.DOTALL)
    elif name == "decodeJSONBody":
        pattern = re.compile(r'^func ' + name + r'\(.*?\n\}', re.MULTILINE | re.DOTALL)
    else:
        pattern = re.compile(r'^func \(s \*Server\) ' + name + r'\(.*?\n\}', re.MULTILINE | re.DOTALL)
    
    m = pattern.search(text)
    if m:
        block = m.group(0)
        text = text[:m.start()] + text[m.end():]
        return block
    return f"// WARNING: Could not find {name}"

auth = [get_block("loginRequest", True), get_block("refreshRequest", True), get_block("decodeJSONBody"), get_block("handleLogin"), get_block("handleRefresh"), get_block("handleLogout")]
news = [get_block("handleListNews"), get_block("handleGetFullContent"), get_block("handleMarkRead"), get_block("handleMarkAllRead"), get_block("handleMarkFavorite")]
feeds = [get_block("handleListFeeds"), get_block("handleAddFeed"), get_block("handleImportOPML"), get_block("handleExportOPML"), get_block("handleUpdateFeed"), get_block("handleDeleteFeed"), get_block("handleRenameCategory")]
system = [get_block("handleOpenAPI"), get_block("handleDocs"), get_block("handleFeverAPI"), get_block("handleImageProxy"), get_block("handlePageProxy"), get_block("handleSaveRecipe"), get_block("handleUpdateRecipes"), get_block("handleExportConfig"), get_block("handleImportConfig"), get_block("handleListFilters"), get_block("handleAddFilter"), get_block("handleDeleteFilter"), get_block("handleListAPIKeys"), get_block("handleCreateAPIKey"), get_block("handleDeleteAPIKey"), get_block("handleGetStats"), get_block("handleExportBackup"), get_block("handleImportBackup")]

def write(name, blocks):
    with open(f"internal/api/{name}", "w") as f:
        f.write("package api\n\n")
        f.write("\n\n".join(blocks))

write("handler_auth.go", auth)
write("handler_news.go", news)
write("handler_feeds.go", feeds)
write("handler_system.go", system)

with open("internal/api/server.go", "w") as f:
    f.write(text)
