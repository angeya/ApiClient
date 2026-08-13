package main

import "ApiClient/internal/filedialog"

// SaveResponseBody 将响应内容保存到用户选择的文件。
// 返回实际保存路径，取消时返回空字符串。
func (a *App) SaveResponseBody(content string, defaultFilename string) (string, error) {
	return filedialog.SaveText(a.ctx, content, defaultFilename)
}
