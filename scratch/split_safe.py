import os

with open('internal/api/server.go', 'r') as f:
    lines = f.readlines()

def find_block(func_name, is_struct=False):
    global lines
    start_idx = -1
    for i, line in enumerate(lines):
        if is_struct:
            if line.startswith(f"type {func_name} struct"):
                start_idx = i
                break
        elif func_name == "decodeJSONBody":
            if line.startswith(f"func {func_name}("):
                start_idx = i
                break
        else:
            if line.startswith(f"func (s *Server) {func_name}("):
                start_idx = i
                break

    if start_idx == -1:
        return ""

    brace_count = 0
    end_idx = -1
    for i in range(start_idx, len(lines)):
        line = lines[i]
        brace_count += line.count('{')
        brace_count -= line.count('}')
        if brace_count == 0 and '{' in ''.join(lines[start_idx:i+1]):
            end_idx = i
            break

    if end_idx != -1:
        block = "".join(lines[start_idx:end_idx+1])
        # remove from lines
        del lines[start_idx:end_idx+1]
        return block
    return ""

auth = [find_block("loginRequest", True), find_block("refreshRequest", True), find_block("decodeJSONBody"), find_block("handleLogin"), find_block("handleRefresh"), find_block("handleLogout")]
news = [find_block("handleListNews"), find_block("handleGetFullContent"), find_block("handleMarkRead"), find_block("handleMarkAllRead"), find_block("handleMarkFavorite")]
feeds = [find_block("handleListFeeds"), find_block("handleAddFeed"), find_block("handleImportOPML"), find_block("handleExportOPML"), find_block("handleUpdateFeed"), find_block("handleDeleteFeed"), find_block("handleRenameCategory")]
system = [find_block("handleOpenAPI"), find_block("handleDocs"), find_block("handleFeverAPI"), find_block("handleImageProxy"), find_block("handlePageProxy"), find_block("handleSaveRecipe"), find_block("handleUpdateRecipes"), find_block("handleExportConfig"), find_block("handleImportConfig"), find_block("handleListFilters"), find_block("handleAddFilter"), find_block("handleDeleteFilter"), find_block("handleListAPIKeys"), find_block("handleCreateAPIKey"), find_block("handleDeleteAPIKey"), find_block("handleGetStats"), find_block("handleExportBackup"), find_block("handleImportBackup")]

def write(name, blocks):
    with open(f"internal/api/{name}", "w") as f:
        f.write("package api\n\n")
        f.write("\n\n".join(blocks))

write("handler_auth.go", auth)
write("handler_news.go", news)
write("handler_feeds.go", feeds)
write("handler_system.go", system)

with open("internal/api/server.go", "w") as f:
    f.writelines(lines)
