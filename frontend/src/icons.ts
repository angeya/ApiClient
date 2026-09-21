/**
 * 全局注册的 Element Plus 图标。
 *
 * 这里必须按需显式列出：模板里直接以标签形式使用图标（如 <Download />），
 * 依赖全局注册才能渲染。
 * 早期做法是 `import * as ElementPlusIconsVue` 全量注册，会把上千个图标组件
 * 一并打进包里（约 200KB），明显拖慢启动，因此改为按需注册。
 * 新增图标时在此处补上即可。
 */
import {
  Aim,
  ArrowDown,
  ArrowUp,
  Brush,
  Check,
  CircleCheck,
  Close,
  Collection,
  CopyDocument,
  Cpu,
  Delete,
  Document,
  Download,
  Folder,
  FolderAdd,
  FolderOpened,
  Loading,
  MagicStick,
  MoreFilled,
  Plus,
  Promotion,
  QuestionFilled,
  Refresh,
  Search,
  SuccessFilled,
  Tools,
  Upload,
  WarningFilled,
} from '@element-plus/icons-vue'

export const icons = {
  Aim,
  ArrowDown,
  ArrowUp,
  Brush,
  Check,
  CircleCheck,
  Close,
  Collection,
  CopyDocument,
  Cpu,
  Delete,
  Document,
  Download,
  Folder,
  FolderAdd,
  FolderOpened,
  Loading,
  MagicStick,
  MoreFilled,
  Plus,
  Promotion,
  QuestionFilled,
  Refresh,
  Search,
  SuccessFilled,
  Tools,
  Upload,
  WarningFilled,
}
