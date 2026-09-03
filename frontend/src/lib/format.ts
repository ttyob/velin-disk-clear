export function formatBytes(value: number, precision = 1): string {
    if (!Number.isFinite(value) || value <= 0) return '0 B';
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    const index = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1);
    const amount = value / 1024 ** index;
    return `${amount.toFixed(index === 0 ? 0 : precision)} ${units[index]}`;
}

export function formatDate(value: unknown): string {
    if (!value) return '—';
    const date = new Date(String(value));
    if (Number.isNaN(date.getTime())) return '—';
    return new Intl.DateTimeFormat('zh-CN', {
        year: 'numeric', month: '2-digit', day: '2-digit',
    }).format(date);
}

export const riskLabels: Record<string, string> = {
    low: '低风险',
    medium: '中风险',
    high: '高风险',
    forbidden: '禁止清理',
};

export const recommendationLabels: Record<string, string> = {
    recommended: '推荐清理',
    optional: '按需清理',
    analyze_only: '仅分析',
    not_recommended: '不建议清理',
    forbidden: '禁止清理',
};

export const categoryLabels: Record<string, string> = {
    system_temp: '系统临时文件',
    system_cache: '系统缓存',
    system_storage: '系统占用',
    diagnostics: '错误报告',
    browser_cache: '浏览器缓存',
    recycle_bin: '回收站',
    large_files: '大文件',
    protected_data: '受保护数据',
    other: '其他文件',
};

export const ruleTypeLabels: Record<string, string> = {
    system: '系统级',
    third_party: '三方软件',
    general: '通用',
};
