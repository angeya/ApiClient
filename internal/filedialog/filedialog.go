package filedialog

import (
	"context"
	"fmt"
	"os"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// SaveText 弹出保存文件对话框并将内容写入用户选择的路径。
// 若用户取消则返回空字符串。
func SaveText(ctx context.Context, content string, defaultFilename string) (string, error) {
	path, err := wailsRuntime.SaveFileDialog(ctx, wailsRuntime.SaveDialogOptions{
		Title:           "保存响应内容",
		DefaultFilename: defaultFilename,
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "所有文件 (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil {
		return "", fmt.Errorf("打开保存对话框失败: %w", err)
	}
	if path == "" {
		return "", nil
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("写入文件失败: %w", err)
	}
	return path, nil
}
