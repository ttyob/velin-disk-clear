import {
    AIProvider,
    BuildCleanPlan,
    CheckForUpdates,
    CleaningAgentPresets,
    CancelScan,
    CleanHistory,
    Dashboard,
    ExecuteCleanPlan,
    DownloadUpdate,
    InstallUpdate,
    NetworkSettings,
    SaveNetworkSettings,
    ScanSettings,
    SaveScanSettings,
    OpenSystemSettings,
    RunCleaningAgent,
    RuleStatistics,
    Rules,
    SaveAIProvider,
    ScanFolders,
    ScanItems,
    ScanJob,
    SelectScanDirectory,
    StartScan,
    SyncRules,
    TestAIProvider,
} from '../../wailsjs/go/main/App';
import type {agent, cleaner, main, provider, rules, scanner} from '../../wailsjs/go/models';

const isWails = () => Boolean((window as Window & {go?: unknown}).go);

const gib = 1024 ** 3;
let mockStartedAt = 0;
let mockCleanHistory: cleaner.Result[] = [];
let mockPlan: cleaner.Plan | null = null;
const mockProviderStorageKey = 'ai-clear.mock.provider';
const mockProviderDefaults = {
    id: 'provider-default', name: 'AI 助手', protocol: 'openai_compatible',
    base_url: '', model: '', credential_ref: 'dpapi://ai-clear/provider-default', key_saved: false,
    timeout_seconds: 60, max_output_tokens: 4096, enabled: false,
} as unknown as provider.Config;
let mockProvider = (() => {
    if (typeof window === 'undefined') return mockProviderDefaults;
    try {
        const saved = window.localStorage.getItem(mockProviderStorageKey);
        return saved ? {...mockProviderDefaults, ...JSON.parse(saved)} as provider.Config : mockProviderDefaults;
    } catch {
        return mockProviderDefaults;
    }
})();
function persistMockProvider() {
    try { window.localStorage.setItem(mockProviderStorageKey, JSON.stringify(mockProvider)); } catch { /* private browsing may disable storage */ }
}

const goAPI = (import.meta.env.VITE_GO_API_URL || '/api').replace(/\/$/, '');
async function goRequest<T>(path: string, init: RequestInit = {}): Promise<T> {
    const response = await fetch(`${goAPI}/${path.replace(/^\//, '')}`, {
        ...init,
        headers: {'Content-Type': 'application/json', ...(init.headers || {})},
    });
    if (!response.ok) throw new Error((await response.text()) || `Go API 请求失败（${response.status}）`);
    return response.json() as Promise<T>;
}

const mockRules: rules.Rule[] = [
    rule('windows.user_temp', '用户临时文件', 'system_temp', 'low', true, 'recommended',
        '应用保存安装、解压和运行过程中的短期工作数据',
        '旧文件清理后通常无影响，活动文件会被跳过',
        '推荐清理超过 24 小时且未被占用的临时文件。'),
    rule('windows.thumbnail_cache', '缩略图缓存', 'system_cache', 'low', true, 'recommended',
        '加快图片、视频和文档缩略图显示',
        '缓存会自动重建，首次浏览文件夹时加载稍慢',
        '推荐清理，Windows 会在需要时重新生成缩略图。'),
    rule('browser.edge_cache', 'Edge 网页缓存', 'browser_cache', 'low', true, 'recommended',
        '减少网页图片、脚本和其他资源的重复下载',
        '网站资源会重新下载，首次打开可能稍慢',
        '推荐清理，不会删除登录信息、收藏夹和历史。'),
    rule('generic.old_logs', '旧日志文件', 'other', 'medium', false, 'optional',
        '记录应用运行状态与故障信息',
        '删除后会失去对应历史排障信息',
        '按需清理；旧日志可能仍有排障价值。'),
    rule('windows.recycle_bin', '回收站', 'recycle_bin', 'medium', false, 'optional',
        '暂存已删除但仍可恢复的文件',
        '清空后文件将永久删除，无法再从 Windows 回收站恢复',
        '默认不选；确认不再需要恢复其中的文件后再清空。'),
    rule('windows.downloads_installers', '下载目录旧安装包', 'large_files', 'high', false, 'analyze_only',
        '保存软件下载、离线安装和系统安装介质',
        '永久删除后需要重新下载，也可能失去唯一的离线安装介质',
        '只列出候选；确认软件已安装且无需离线副本后再逐项选择。', true),
    rule('generic.large_files', '大文件', 'large_files', 'high', false, 'analyze_only',
        '帮助定位占用大量磁盘空间的文件',
        '文件可能是个人资料、备份或应用数据，删除后可能无法恢复',
        '仅分析且默认不选；大文件不一定可以清理。', true),
    rule('windows.pagefile_analysis', 'Windows 分页文件', 'system_storage', 'high', false, 'not_recommended',
        '在物理内存不足时提供虚拟内存，并支持部分系统转储',
        '错误缩小可能导致内存不足、应用崩溃和转储不可用',
        '不建议清理；通常保持“由系统管理”。', true),
];

const mockItems: scanner.Item[] = [
    item('1', mockRules[6], 'Windows11_24H2.iso', String.raw`D:\Downloads\Windows11_24H2.iso`, 6.42 * gib, true),
    item('2', mockRules[7], 'pagefile.sys', String.raw`C:\pagefile.sys`, 4.0 * gib, false, false),
    item('3', mockRules[2], 'data_3', String.raw`C:\Users\Lin\AppData\Local\Microsoft\Edge\User Data\Default\Cache\data_3`, 824 * 1024 ** 2, true),
    item('4', mockRules[0], 'setup-8f13.tmp', String.raw`C:\Users\Lin\AppData\Local\Temp\setup-8f13.tmp`, 362 * 1024 ** 2, true),
    item('5', mockRules[6], 'footage-export-v7.zip', String.raw`D:\Projects\Exports\footage-export-v7.zip`, 2.18 * gib, true),
    item('6', mockRules[1], 'thumbcache_2560.db', String.raw`C:\Users\Lin\AppData\Local\Microsoft\Windows\Explorer\thumbcache_2560.db`, 188 * 1024 ** 2, true),
    item('7', mockRules[3], 'render-2025-11.log', String.raw`D:\Projects\Video\logs\render-2025-11.log`, 96 * 1024 ** 2, false),
    item('8', mockRules[0], 'unpack-cache.tmp', String.raw`C:\Users\Lin\AppData\Local\Temp\unpack-cache.tmp`, 74 * 1024 ** 2, true),
    item('9', mockRules[4], 'C: 回收站', 'C:\\', 1.36 * gib, true, false),
    item('10', mockRules[5], 'designer-setup-4.8.2.exe', String.raw`C:\Users\Lin\Downloads\designer-setup-4.8.2.exe`, 486 * 1024 ** 2, true, false),
];

function rule(
    id: string,
    name: string,
    category: string,
    risk: string,
    defaultSelected: boolean,
    recommendation: string,
    purpose: string,
    cleanEffect: string,
    summary: string,
    special = false,
): rules.Rule {
    return {
        id, version: 1, name, category, risk, recommendation, purpose,
        description: summary, clean_effect: cleanEffect, recommendation_reason: summary,
        platform: 'windows', enabled: true, default_selected: defaultSelected,
        requires_admin: id.includes('pagefile'), supported_windows_versions: ['10.0.19045', '11'],
        scope: 'current_user', size_mode: 'allocated', recovery_type: special ? 'system_settings' : 'none',
        requires_network_after_clean: false, may_sign_out: false, requires_restart: id.includes('pagefile'),
        process_guard: [], conflicts: [], last_verified_at: '2026-09-03', source: 'internal-review',
        modes: ['quick', 'standard', 'deep'],
        help: {
            summary,
            details: special ? `${purpose}。${cleanEffect}。` : undefined,
            special_warning: special ? '这是系统或用户重要数据，不能由普通清理流程自动处理。' : undefined,
            manual_steps: id.includes('pagefile') ? ['打开 Windows 虚拟内存设置', '保持“由系统管理”或根据实际需求调整'] : [],
        },
        scan: {roots: [], stay_on_volume: true, follow_reparse_points: false},
        action: {type: id === 'windows.recycle_bin' ? 'empty_recycle_bin' : special && !id.startsWith('windows.downloads_') ? 'analyze' : 'permanent_delete'},
        safety: {allowed_roots: [], revalidate_before_clean: true},
    } as unknown as rules.Rule;
}

function item(
    id: string,
    sourceRule: rules.Rule,
    name: string,
    path: string,
    size: number,
    selectable: boolean,
    defaultSelected = sourceRule.default_selected,
): scanner.Item {
    return {
        id,
        rule_id: sourceRule.id,
        matched_rule_ids: [sourceRule.id],
        name,
        path,
        directory: path.slice(0, path.lastIndexOf('\\')),
        extension: name.includes('.') ? name.slice(name.lastIndexOf('.')).toLowerCase() : '',
        category: sourceRule.category,
        purpose: sourceRule.purpose,
        clean_effect: sourceRule.clean_effect,
        recommendation: sourceRule.recommendation,
        recommendation_reason: sourceRule.recommendation_reason,
        risk: sourceRule.risk,
        default_selected: defaultSelected,
        selectable,
        action: selectable ? sourceRule.action.type : 'analyze',
        recovery_type: selectable ? 'none' : 'system_settings',
        requires_admin: sourceRule.requires_admin,
        requires_restart: sourceRule.requires_restart,
        logical_size: Math.round(size),
        allocated_size: Math.round(size),
        estimated_reclaimable: selectable ? Math.round(size) : 0,
        volume_id: path.slice(0, 2),
        file_id: id,
        link_count: 1,
        modified_at: new Date(Date.now() - Number(id) * 8.64e7).toISOString(),
        help_summary: sourceRule.help.summary,
        help_details: sourceRule.help.details,
        special_warning: sourceRule.help.special_warning,
        manual_steps: sourceRule.help.manual_steps,
    } as scanner.Item;
}

function mockDashboard(): main.Dashboard {
    return {
        version: '0.1.0-dev',
        rule_count: 50,
        volumes: [
            {id: 'C:', name: 'Windows', mount_point: 'C:\\', file_system: 'NTFS', total_bytes: 237 * gib, used_bytes: 213.6 * gib, free_bytes: 23.4 * gib, system: true, ready: true},
            {id: 'D:', name: 'Data', mount_point: 'D:\\', file_system: 'NTFS', total_bytes: 931 * gib, used_bytes: 524.2 * gib, free_bytes: 406.8 * gib, system: false, ready: true},
        ],
    } as main.Dashboard;
}

function mockJob(status?: string): scanner.Job {
    const elapsed = Date.now() - mockStartedAt;
    const completed = status === 'completed' || elapsed > 1800;
    const progress = completed ? 1 : Math.min(.92, elapsed / 1800);
    return {
        id: 'mock-scan-1',
        status: completed ? 'completed' : (status ?? 'running'),
        mode: 'deep',
        current_path: completed ? '' : `C:\\Users\\Lin\\AppData\\Local\\Temp\\segment-${Math.floor(progress * 28)}`,
        scanned_files: Math.floor(progress * 128430),
        matched_files: completed ? mockItems.length : Math.floor(progress * mockItems.length),
        logical_bytes: completed ? mockItems.reduce((sum, value) => sum + value.logical_size, 0) : 0,
        allocated_bytes: completed ? mockItems.reduce((sum, value) => sum + value.allocated_size, 0) : 0,
        estimated_reclaimable: completed ? mockItems.reduce((sum, value) => sum + value.estimated_reclaimable, 0) : 0,
        error_count: 3,
        errors: [],
        started_at: new Date(mockStartedAt || Date.now()).toISOString(),
        completed_at: completed ? new Date().toISOString() : undefined,
    } as unknown as scanner.Job;
}

function mockFolders(): scanner.Folder[] {
    const byVolume = new Map<string, scanner.Item[]>();
    for (const entry of mockItems) {
        const volume = entry.path.slice(0, 2);
        byVolume.set(volume, [...(byVolume.get(volume) ?? []), entry]);
    }
    return Array.from(byVolume, ([volume, items]) => ({
        id: volume,
        name: volume,
        path: `${volume}\\`,
        file_count: items.length,
        logical_bytes: items.reduce((sum, value) => sum + value.logical_size, 0),
        allocated_bytes: items.reduce((sum, value) => sum + value.allocated_size, 0),
        estimated_reclaimable: items.reduce((sum, value) => sum + value.estimated_reclaimable, 0),
        highest_risk: items.some(value => value.risk === 'high') ? 'high' : 'low',
        item_ids: items.map(value => value.id),
        children: groupMockChildren(volume, items),
    } as unknown as scanner.Folder));
}

function groupMockChildren(volume: string, items: scanner.Item[]): scanner.Folder[] {
    const groups = new Map<string, scanner.Item[]>();
    for (const entry of items) {
        const rest = entry.path.slice(3);
        const first = rest.split('\\')[0] || 'Root';
        groups.set(first, [...(groups.get(first) ?? []), entry]);
    }
    return Array.from(groups, ([name, values]) => ({
        id: `${volume}-${name}`,
        name,
        path: `${volume}\\${name}`,
        file_count: values.length,
        logical_bytes: values.reduce((sum, value) => sum + value.logical_size, 0),
        allocated_bytes: values.reduce((sum, value) => sum + value.allocated_size, 0),
        estimated_reclaimable: values.reduce((sum, value) => sum + value.estimated_reclaimable, 0),
        highest_risk: values.some(value => value.risk === 'high') ? 'high' : values.some(value => value.risk === 'medium') ? 'medium' : 'low',
        item_ids: values.map(value => value.id),
        children: [],
    } as unknown as scanner.Folder));
}

export const api = {
    isDesktop: isWails,
    dashboard: async () => isWails() ? Dashboard() : goRequest<main.Dashboard>('dashboard'),
    rules: async () => isWails() ? Rules() : goRequest<rules.Rule[]>('rules'),
    ruleStatistics: async () => isWails() ? RuleStatistics() : goRequest<rules.Statistics>('rules/stats'),
    checkForUpdates: async () => isWails() ? CheckForUpdates() : goRequest<main.UpdateInfo>('update/check'),
    downloadUpdate: async () => isWails() ? DownloadUpdate() : goRequest<main.UpdateDownload>('update/download', {method: 'POST'}),
    installUpdate: async () => isWails() ? InstallUpdate() : goRequest('update/install', {method: 'POST'}).then(() => undefined),
    networkSettings: async () => isWails() ? NetworkSettings() : goRequest<main.NetworkSettings>('network/settings'),
    saveNetworkSettings: async (value: main.NetworkSettings) => {
        if (isWails()) return SaveNetworkSettings(value);
        return goRequest<main.NetworkSettings>('network/settings', {method: 'POST', body: JSON.stringify(value)});
    },
    scanSettings: async () => isWails() ? ScanSettings() : goRequest<main.ScanSettings>('scan/settings'),
    saveScanSettings: async (value: main.ScanSettings) => {
        if (isWails()) return SaveScanSettings(value);
        return goRequest<main.ScanSettings>('scan/settings', {method: 'POST', body: JSON.stringify(value)});
    },
    syncRules: async () => isWails() ? SyncRules() : goRequest<rules.SyncResult>('rules/sync', {method: 'POST'}),
    selectDirectory: async () => isWails() ? SelectScanDirectory() : 'D:\\Downloads',
    startScan: async (request: scanner.Request) => {
        if (!isWails()) return goRequest<scanner.Job>('scan', {method: 'POST', body: JSON.stringify(request)});
        return StartScan(request);
    },
    scanJob: async (id: string) => isWails() ? ScanJob(id) : goRequest<scanner.Job>(`scan/${encodeURIComponent(id)}`),
    scanItems: async (id: string, offset = 0, limit = 200) => {
        if (isWails()) return ScanItems(id, offset, limit);
        return goRequest<scanner.ItemPage>(`scan/${encodeURIComponent(id)}/items?offset=${offset}&limit=${limit}`);
    },
    scanFolders: async (id: string) => isWails() ? ScanFolders(id) : goRequest<scanner.Folder[]>(`scan/${encodeURIComponent(id)}/folders`),
    cancelScan: async (id: string) => isWails() ? CancelScan(id) : goRequest(`scan/${encodeURIComponent(id)}`, {method: 'DELETE'}).then(() => undefined),
    openSystemSettings: async (action: string) => isWails() ? OpenSystemSettings(action) : goRequest('settings', {method: 'POST', body: JSON.stringify({action})}).then(() => undefined),
    buildCleanPlan: async (request: cleaner.BuildRequest) => {
        if (!isWails()) return goRequest<cleaner.Plan>('clean/plan', {method: 'POST', body: JSON.stringify(request)});
        return BuildCleanPlan(request);
    },
    executeCleanPlan: async (request: cleaner.ExecuteRequest) => {
        if (!isWails()) return goRequest<cleaner.Result>('clean/execute', {method: 'POST', body: JSON.stringify(request)});
        return ExecuteCleanPlan(request);
    },
    cleanHistory: async () => isWails() ? CleanHistory() : goRequest<cleaner.Result[]>('clean/history'),
    aiProvider: async () => isWails() ? AIProvider() : goRequest<provider.Config>('provider'),
    saveAIProvider: async (input: provider.ConfigInput) => {
        if (!isWails()) return goRequest<provider.Config>('provider/save', {method: 'POST', body: JSON.stringify(input)});
        return SaveAIProvider(input);
    },
    testAIProvider: async (input: provider.ConfigInput) => {
        if (!isWails()) return goRequest<provider.TestResult>('provider/test', {method: 'POST', body: JSON.stringify(input)});
        return TestAIProvider(input);
    },
    runCleaningAgent: async (request: agent.Request) => {
        if (!isWails()) return goRequest<agent.Result>('agent', {method: 'POST', body: JSON.stringify(request)});
        return RunCleaningAgent(request);
    },
    cleaningAgentPresets: async () => isWails() ? CleaningAgentPresets() : goRequest<agent.Preset[]>('agent/presets'),
};
