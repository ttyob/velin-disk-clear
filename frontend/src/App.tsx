import {
    AlertTriangle, Bot, CheckCircle2, ChevronRight, CircleHelp, Clock3, Database, FileSearch,
    FolderOpen, HardDrive, History, Home, List, Moon, Play,
    ChevronLeft, RotateCcw, Search, Settings as SettingsIcon, ShieldCheck, Sparkles, Square, SquareCheckBig,
    Sun, Trash2, XCircle,
} from 'lucide-react';
import {type ReactNode, useEffect, useMemo, useState} from 'react';
import {createPortal} from 'react-dom';
import type {agent, cleaner, main, provider, rules, scanner} from '../wailsjs/go/models';
import './App.css';
import {FolderTree} from './components/FolderTree';
import {RuleHelpModal} from './components/RuleHelpModal';
import {api} from './lib/api';
import {categoryLabels, formatBytes, formatDate, recommendationLabels, riskLabels, ruleTypeLabels} from './lib/format';

type Page = 'home' | 'system' | 'disk' | 'ai' | 'history' | 'rules' | 'settings';
type ThemeMode = 'system' | 'light' | 'dark';

const navItems: Array<{id: Page; label: string; icon: typeof Home}> = [
    {id: 'home', label: '首页', icon: Home},
    {id: 'system', label: 'C 盘专清', icon: ShieldCheck},
    {id: 'disk', label: '磁盘清理', icon: HardDrive},
    {id: 'ai', label: 'AI 清理', icon: Sparkles},
    {id: 'history', label: '清理历史', icon: History},
    {id: 'rules', label: '规则管理', icon: List},
    {id: 'settings', label: '设置', icon: SettingsIcon},
];

export default function App() {
    const [page, setPage] = useState<Page>('home');
    const [dashboard, setDashboard] = useState<main.Dashboard | null>(null);
    const [ruleList, setRuleList] = useState<rules.Rule[]>([]);
    const [ruleStats, setRuleStats] = useState<rules.Statistics | null>(null);
    const [loadingError, setLoadingError] = useState('');
    const [latestScanID, setLatestScanID] = useState('');
    const [themeMode, setThemeMode] = useState<ThemeMode>(() => (localStorage.getItem('theme-mode') as ThemeMode) || 'system');

    useEffect(() => {
        Promise.all([api.dashboard(), api.rules(), api.ruleStatistics()])
            .then(([dashboardData, ruleData, stats]) => {
                setDashboard(dashboardData);
                setRuleList(ruleData);
                setRuleStats(stats);
            })
            .catch(error => setLoadingError(String(error)));
    }, []);

    useEffect(() => {
        const media = window.matchMedia('(prefers-color-scheme: dark)');
        const applyTheme = () => {
            document.documentElement.dataset.theme = themeMode === 'system' ? (media.matches ? 'dark' : 'light') : themeMode;
        };
        applyTheme();
        media.addEventListener('change', applyTheme);
        localStorage.setItem('theme-mode', themeMode);
        return () => media.removeEventListener('change', applyTheme);
    }, [themeMode]);

    const title = navItems.find(item => item.id === page)?.label ?? 'Velin Clear';

    return (
        <div className="app-shell">
            <header className="app-header">
                <div className="brand">
                    <div className="brand-mark"><HardDrive size={20}/><span/></div>
                    <div><strong>Velin Clear</strong><small>磁盘清理管家</small></div>
                </div>
                <nav aria-label="主导航">
                    {navItems.map(({id, label, icon: Icon}) => (
                        <button key={id} className={page === id ? 'nav-item active' : 'nav-item'} onClick={() => setPage(id)}>
                            <Icon size={18}/><span>{label}</span>
                        </button>
                    ))}
                </nav>
                <div className="app-engine" title={`本地引擎 ${dashboard?.version ?? ''}`}>
                    <span className="status-dot"/><span>引擎就绪</span>
                </div>
            </header>

            <div className="workspace">
                <header className="topbar">
                    <div className="topbar-title"><h1>{title}</h1><p>{pageSubtitle(page)}</p></div>
                    <div className="page-toolbar-actions" id="page-toolbar-actions"/>
                </header>

                <main className="content">
                    {loadingError && <div className="error-banner"><XCircle size={18}/>{loadingError}</div>}
                    {page === 'home' && <DashboardPage dashboard={dashboard} onNavigate={setPage}/>}
                    {page === 'system' && <ScanPage kind="system" onCompleted={setLatestScanID}/>}
                    {page === 'disk' && <ScanPage kind="custom" onCompleted={setLatestScanID}/>}
                    {page === 'rules' && <RulesPage rules={ruleList} stats={ruleStats} onSync={async () => {
                        const result = await api.syncRules();
                        const [latestRules, latestStats] = await Promise.all([api.rules(), api.ruleStatistics()]);
                        setRuleList(latestRules);
                        setRuleStats(latestStats);
                        setDashboard(previous => previous ? Object.assign(previous, {rule_count: latestStats.total}) : previous);
                        return result;
                    }}/>}
                    {page === 'ai' && <AIPage scanID={latestScanID}/>}
                    {page === 'settings' && <SettingsPage themeMode={themeMode} onThemeChange={setThemeMode}/>}
                    {page === 'history' && <HistoryPage/>}
                </main>
            </div>
        </div>
    );
}

function pageSubtitle(page: Page) {
    const subtitles: Record<Page, string> = {
        home: '查看磁盘状态并开始清理',
        system: 'Windows 系统与应用缓存',
        disk: '自定义范围和大文件分析',
        ai: '由 Cleaning Agent 生成可审查方案',
        history: '扫描与清理执行记录',
        rules: '规则用途、风险和默认选择',
        settings: '主题、扫描、清理与隐私',
    };
    return subtitles[page];
}

function DashboardPage({dashboard, onNavigate}: {dashboard: main.Dashboard | null; onNavigate: (page: Page) => void}) {
    const systemVolume = dashboard?.volumes.find(volume => volume.system) ?? dashboard?.volumes[0];
    const usedRatio = systemVolume && systemVolume.total_bytes > 0 ? systemVolume.used_bytes / systemVolume.total_bytes : 0;
    return (
        <div className="page-stack">
            <section className="action-strip">
                <div>
                    <span className="eyebrow">{systemVolume ? `${systemVolume.id} 系统盘` : '磁盘状态'}</span>
                    <h2>{systemVolume ? `${formatBytes(systemVolume.free_bytes)} 可用空间` : '正在读取磁盘信息'}</h2>
                    <p>{systemVolume ? `总容量 ${formatBytes(systemVolume.total_bytes)}，已使用 ${Math.round(usedRatio * 100)}%` : '正在连接本地磁盘服务'}</p>
                </div>
                <div className="storage-ring" style={{background: `conic-gradient(${usedRatio >= .9 ? 'var(--red)' : 'var(--accent)'} ${Math.round(usedRatio * 100)}%, var(--surface-3) 0)`}}>
                    <div><HardDrive size={22}/><strong>{Math.round(usedRatio * 100)}%</strong><small>已使用</small></div>
                </div>
                <div className="action-strip-buttons">
                    <button className="button primary main-action" onClick={() => onNavigate('system')}><ShieldCheck size={18}/>扫描 C 盘</button>
                    <button className="button secondary" onClick={() => onNavigate('disk')}><FolderOpen size={17}/>扫描其他位置</button>
                </div>
            </section>

            <section className="section">
                <div className="section-heading"><div><h2>磁盘概览</h2><p>实际占用与剩余容量</p></div><button className="icon-button" title="刷新磁盘信息" aria-label="刷新磁盘信息"><RotateCcw size={17}/></button></div>
                <div className="disk-list">
                    {(dashboard?.volumes ?? []).map(volume => <DiskRow key={volume.id} volume={volume} onScan={() => onNavigate(volume.system ? 'system' : 'disk')}/>)}
                    {!dashboard && <><div className="skeleton disk-skeleton"/><div className="skeleton disk-skeleton"/></>}
                </div>
            </section>

            <div className="summary-grid">
                <section className="metric-block"><span className="metric-icon blue"><Database size={18}/></span><div><small>已加载规则</small><strong>{dashboard?.rule_count ?? '—'}</strong><p>经过字段和安全策略校验</p></div></section>
                <section className="metric-block"><span className="metric-icon teal"><ShieldCheck size={18}/></span><div><small>保护状态</small><strong>已启用</strong><p>系统目录与用户数据保护</p></div></section>
                <section className="metric-block"><span className="metric-icon amber"><Clock3 size={18}/></span><div><small>最近扫描</small><strong>尚未扫描</strong><p>选择磁盘开始首次分析</p></div></section>
            </div>

            <section className="section activity-section">
                <div className="section-heading"><div><h2>最近任务</h2><p>扫描和清理状态</p></div></div>
                <div className="empty-inline"><FileSearch size={20}/><span>还没有任务记录</span></div>
            </section>
        </div>
    );
}

function DiskRow({volume, onScan}: {volume: main.Dashboard['volumes'][number]; onScan: () => void}) {
    const ratio = volume.total_bytes > 0 ? volume.used_bytes / volume.total_bytes : 0;
    const critical = ratio >= .9;
    return (
        <div className="disk-row">
            <div className={`drive-icon ${critical ? 'critical' : ''}`}><HardDrive size={21}/></div>
            <div className="disk-main">
                <div className="disk-title">
                    <div><strong>{volume.name} <span>{volume.id}</span></strong>{volume.system && <em>系统盘</em>}</div>
                    <span>{formatBytes(volume.free_bytes)} 可用，共 {formatBytes(volume.total_bytes)}</span>
                </div>
                <div className="capacity-track"><span className={critical ? 'critical' : ''} style={{width: `${Math.min(100, ratio * 100)}%`}}/></div>
                <div className="disk-meta"><span>{volume.file_system}</span><span>已使用 {Math.round(ratio * 100)}%</span></div>
            </div>
            <button className="button ghost" onClick={onScan}>扫描<ChevronRight size={16}/></button>
        </div>
    );
}

function ScanPage({kind, onCompleted}: {kind: 'system' | 'custom'; onCompleted: (scanID: string) => void}) {
    const [mode, setMode] = useState(kind === 'system' ? 'quick' : 'deep');
    const [root, setRoot] = useState('');
    const [job, setJob] = useState<scanner.Job | null>(null);
    const [items, setItems] = useState<scanner.Item[]>([]);
    const [folders, setFolders] = useState<scanner.Folder[]>([]);
    const [selected, setSelected] = useState<Set<string>>(new Set());
    const [view, setView] = useState<'files' | 'folders'>('files');
    const [query, setQuery] = useState('');
    const [helpItem, setHelpItem] = useState<scanner.Item | null>(null);
    const [cleanPlan, setCleanPlan] = useState<cleaner.Plan | null>(null);
    const [cleanResult, setCleanResult] = useState<cleaner.Result | null>(null);
    const [cleaning, setCleaning] = useState(false);
    const [error, setError] = useState('');

    const running = Boolean(job && !['completed', 'cancelled', 'failed'].includes(job.status));

    useEffect(() => {
        if (!job || !running) return;
        const timer = window.setInterval(async () => {
            try {
                const next = await api.scanJob(job.id);
                setJob(next);
                if (next.status === 'completed') {
                    onCompleted(next.id);
                    const [page, folderData] = await Promise.all([api.scanItems(next.id, 0, 500), api.scanFolders(next.id)]);
                    setItems(page.items);
                    setFolders(folderData);
                    setSelected(new Set(page.items.filter(item => item.default_selected && item.selectable && item.risk === 'low').map(item => item.id)));
                }
            } catch (reason) {
                setError(String(reason));
            }
        }, 250);
        return () => window.clearInterval(timer);
    }, [job?.id, running, onCompleted]);

    const filteredItems = useMemo(() => {
        const value = query.trim().toLowerCase();
        if (!value) return items;
        return items.filter(item => item.name.toLowerCase().includes(value) || item.path.toLowerCase().includes(value));
    }, [items, query]);
    const selectedItems = items.filter(item => selected.has(item.id));
    const selectedBytes = selectedItems.reduce((sum, item) => sum + item.estimated_reclaimable, 0);
    const selectableItems = items.filter(item => item.selectable);
    const allSelectableSelected = selectableItems.length > 0 && selectableItems.every(item => selected.has(item.id));

    async function chooseRoot() {
        try {
            const selectedRoot = await api.selectDirectory();
            if (selectedRoot) setRoot(selectedRoot);
        } catch (reason) {
            setError(String(reason));
        }
    }

    async function start() {
        setError('');
        setItems([]);
        setFolders([]);
        setSelected(new Set());
        try {
            if (kind === 'custom' && !root) {
                await chooseRoot();
                return;
            }
            const next = await api.startScan({mode, roots: kind === 'custom' ? [root] : [], rule_ids: []} as scanner.Request);
            setJob(next);
        } catch (reason) {
            setError(String(reason));
        }
    }

    async function cancel() {
        if (!job) return;
        await api.cancelScan(job.id);
    }

    async function reviewClean() {
        if (!job || selected.size === 0) return;
        setError('');
        try {
            setCleanPlan(await api.buildCleanPlan({scan_id: job.id, item_ids: [...selected]} as cleaner.BuildRequest));
        } catch (reason) {
            setError(String(reason));
        }
    }

    async function executeClean() {
        if (!cleanPlan) return;
        setCleaning(true);
        setError('');
        try {
            const result = await api.executeCleanPlan({plan_id: cleanPlan.id, confirmation_token: cleanPlan.confirmation_token} as cleaner.ExecuteRequest);
            setCleanResult(result);
            setSelected(new Set());
        } catch (reason) {
            setError(String(reason));
        } finally {
            setCleaning(false);
        }
    }

    function toggleItem(item: scanner.Item) {
        if (!item.selectable) return;
        setSelected(previous => {
            const next = new Set(previous);
            next.has(item.id) ? next.delete(item.id) : next.add(item.id);
            return next;
        });
    }

    function toggleFolder(folderItems: scanner.Item[], checked: boolean) {
        setSelected(previous => {
            const next = new Set(previous);
            for (const item of folderItems) {
                if (!item.selectable || item.action === 'analyze') continue;
                checked ? next.add(item.id) : next.delete(item.id);
            }
            return next;
        });
    }

    function toggleAllItems() {
        setSelected(allSelectableSelected ? new Set() : new Set(selectableItems.map(item => item.id)));
    }

    return (
        <div className="page-stack">
            <ToolbarPortal>
                <div className="scan-controls">
                    <div className="segmented" aria-label="扫描模式">
                        {['quick', 'standard', 'deep'].map(value => <button key={value} className={mode === value ? 'active' : ''} onClick={() => setMode(value)} disabled={running}>{modeLabel(value)}</button>)}
                    </div>
                    {kind === 'custom' && <button className="path-picker" onClick={chooseRoot} disabled={running}><FolderOpen size={17}/><span>{root || '选择扫描目录'}</span></button>}
                    {!running ? <button className="button primary" onClick={start}><Play size={17}/>{job?.status === 'completed' ? '重新扫描' : '开始扫描'}</button>
                        : <button className="button danger-outline" onClick={cancel}><Square size={14}/>取消</button>}
                </div>
            </ToolbarPortal>

            {error && <div className="error-banner"><XCircle size={18}/>{error}</div>}
            {running && job && <ScanProgress job={job}/>}
            {job?.status === 'cancelled' && <div className="status-banner"><XCircle size={18}/>扫描已取消，未完成结果不会用于清理。</div>}
            {cleanResult && <div className="success-banner"><CheckCircle2 size={18}/>清理完成：永久删除 {cleanResult.succeeded} 项，实际释放 {formatBytes(cleanResult.actually_reclaimed)}。</div>}
            {job?.status === 'completed' && (
                <section className="results-section">
                    <div className="result-summary">
                        <div><span className="eyebrow">扫描完成</span><h2>发现 {job.matched_files} 个项目</h2><p>实际占用 {formatBytes(job.allocated_bytes)}，跳过 {job.error_count} 个不可访问项</p></div>
                        <div className="selected-total"><small>已选择 {selected.size} 项</small><strong>{formatBytes(selectedBytes)}</strong><button className="button primary" disabled={selected.size === 0} onClick={reviewClean}><Trash2 size={17}/>审查清理</button></div>
                    </div>
                    <div className="result-toolbar">
                        <div className="segmented compact">
                            <button className={view === 'files' ? 'active' : ''} onClick={() => setView('files')}><List size={15}/>文件</button>
                            <button className={view === 'folders' ? 'active' : ''} onClick={() => setView('folders')}><FolderOpen size={15}/>文件夹</button>
                        </div>
                        <label className="search-field"><Search size={16}/><input value={query} onChange={event => setQuery(event.target.value)} placeholder="搜索文件或路径"/></label>
                        <button className="button ghost" onClick={() => setSelected(new Set(items.filter(item => item.default_selected && item.risk === 'low' && item.selectable).map(item => item.id)))}>选择推荐项</button>
                        <button className="button ghost" disabled={selectableItems.length === 0} onClick={toggleAllItems}><SquareCheckBig size={15}/>{allSelectableSelected ? '取消全选' : '全选'}</button>
                    </div>
                    {view === 'files' ? <FileTable items={filteredItems} selected={selected} onToggle={toggleItem} onHelp={setHelpItem}/>
                        : <FolderTree folders={folders} items={items} selected={selected} onToggleItem={toggleItem} onToggleFolder={toggleFolder} onHelp={setHelpItem}/>}
                </section>
            )}
            {!job && <ScanPrimer kind={kind}/>}
            <RuleHelpModal source={helpItem} onClose={() => setHelpItem(null)}/>
            <CleanPlanModal plan={cleanPlan} result={cleanResult} busy={cleaning} onClose={() => { setCleanPlan(null); setCleanResult(null); }} onExecute={executeClean}/>
        </div>
    );
}

function CleanPlanModal({plan, result, busy, onClose, onExecute}: {plan: cleaner.Plan | null; result: cleaner.Result | null; busy: boolean; onClose: () => void; onExecute: () => void}) {
    if (!plan) return null;
    return (
        <div className="modal-backdrop" role="presentation" onMouseDown={busy ? undefined : onClose}>
            <section className="modal clean-modal" role="dialog" aria-modal="true" aria-labelledby="clean-plan-title" onMouseDown={event => event.stopPropagation()}>
                <header className="modal-header">
                    <div className="modal-title"><span className="icon-box"><ShieldCheck size={18}/></span><div><h2 id="clean-plan-title">{result ? '清理结果' : '确认清理计划'}</h2><p>{plan.id}</p></div></div>
                    <button className="icon-button" aria-label="关闭" disabled={busy} onClick={onClose}><XCircle size={18}/></button>
                </header>
                {!result ? <>
                    <div className="plan-metrics">
                        <div><small>选中项目</small><strong>{plan.item_count}</strong></div><div><small>预计可释放</small><strong>{formatBytes(plan.estimated_reclaimable)}</strong></div><div><small>手动选择</small><strong>{plan.manual_selected_count}</strong></div>
                    </div>
                    {(plan.medium_risk_count > 0 || plan.high_risk_count > 0) && <div className="warning-callout"><AlertTriangle size={18}/><div><strong>包含需谨慎处理的项目</strong><p>中风险 {plan.medium_risk_count} 项，高风险 {plan.high_risk_count} 项。执行前请核对路径。</p></div></div>}
                    <div className="danger-callout"><AlertTriangle size={20}/><div><strong>这些文件将被永久删除</strong><p>文件不会进入回收站，删除后无法恢复。请在执行前逐项核对文件路径。</p></div></div>
                    <div className="plan-action-list"><div><span>永久删除</span><strong>{plan.permanent_delete_count} 项</strong><small>执行前会再次校验路径、文件身份和修改状态</small></div></div>
                    <div className="plan-files">{plan.items.slice(0, 8).map(item => <div key={item.id}><span title={item.path}>{item.name}<small>{item.path}</small></span><strong>{formatBytes(item.allocated_size)}</strong></div>)}{plan.items.length > 8 && <p>另有 {plan.items.length - 8} 项</p>}</div>
                </> : <div className="clean-result"><CheckCircle2 size={32}/><h3>永久删除已完成</h3><p>成功 {result.succeeded} 项，跳过 {result.skipped} 项，失败 {result.failed} 项</p><div><span>实际释放</span><strong>{formatBytes(result.actually_reclaimed)}</strong><span>永久删除</span><strong>{formatBytes(result.deleted_bytes)}</strong></div></div>}
                <footer className="modal-footer">
                    <button className="button secondary" disabled={busy} onClick={onClose}>{result ? '完成' : '取消'}</button>
                    {!result && <button className="button danger" disabled={busy} onClick={onExecute}><Trash2 size={16}/>{busy ? '正在永久删除…' : `永久删除 ${plan.item_count} 项`}</button>}
                </footer>
            </section>
        </div>
    );
}

function ScanProgress({job}: {job: scanner.Job}) {
    return (
        <section className="scan-progress">
            <div className="scan-pulse"><Search size={22}/><span/></div>
            <div className="progress-main">
                <div><strong>正在扫描</strong><span>{job.scanned_files.toLocaleString()} 个文件</span></div>
                <div className="progress-track indeterminate"><span/></div>
                <p title={job.current_path}>{job.current_path || '正在整理扫描结果…'}</p>
            </div>
            <div className="progress-stat"><small>已发现</small><strong>{job.matched_files}</strong></div>
        </section>
    );
}

function ScanPrimer({kind}: {kind: 'system' | 'custom'}) {
    const rows = kind === 'system'
        ? [['系统临时文件', '超过保留期的 Windows 临时数据'], ['浏览器缓存', 'Edge、Chrome 与 Firefox可重建缓存'], ['系统占用', '分页文件等只分析项目']]
        : [['大文件分析', '按大小列出，默认不勾选'], ['旧日志', '明确目录中的历史日志'], ['目录聚合', '可切换文件与文件夹视图']];
    return <section className="scan-primer">{rows.map(([rowTitle, body]) => <div key={rowTitle}><CheckCircle2 size={17}/><span><strong>{rowTitle}</strong><small>{body}</small></span></div>)}</section>;
}

function FileTable({items, selected, onToggle, onHelp}: {items: scanner.Item[]; selected: Set<string>; onToggle: (item: scanner.Item) => void; onHelp: (item: scanner.Item) => void}) {
    return (
        <div className="table-wrap">
            <table className="file-table">
                <thead><tr><th className="check-column"/><th>文件</th><th>分类</th><th>建议</th><th>修改时间</th><th className="number">磁盘占用</th><th className="help-column"/></tr></thead>
                <tbody>
                {items.map(item => (
                    <tr key={item.id} className={!item.selectable ? 'disabled-row' : ''}>
                        <td><input type="checkbox" checked={selected.has(item.id)} disabled={!item.selectable} aria-label={`选择 ${item.name}`} onChange={() => onToggle(item)}/></td>
                        <td><div className="file-cell"><span className="file-type">{(item.extension || 'FILE').replace('.', '').slice(0, 4).toUpperCase()}</span><div><strong title={item.name}>{item.name}</strong><small title={item.path}>{item.path}</small></div></div></td>
                        <td>{categoryLabels[item.category] ?? item.category}</td>
                        <td><div className="recommendation-cell"><span className={`risk-dot risk-${item.risk}`}/><span>{recommendationLabels[item.recommendation] ?? item.recommendation}</span></div></td>
                        <td>{formatDate(item.modified_at)}</td>
                        <td className="number"><strong>{formatBytes(item.allocated_size)}</strong>{item.link_count > 1 && <small>{item.link_count} 个硬链接</small>}</td>
                        <td><button className="icon-button subtle" title="查看清理建议" aria-label="查看清理建议" onClick={() => onHelp(item)}><CircleHelp size={16}/></button></td>
                    </tr>
                ))}
                </tbody>
            </table>
            {items.length === 0 && <div className="table-empty">没有符合当前筛选条件的文件</div>}
        </div>
    );
}

function RulesPage({rules: entries, stats, onSync}: {rules: rules.Rule[]; stats: rules.Statistics | null; onSync: () => Promise<rules.SyncResult>}) {
    const [query, setQuery] = useState('');
    const [risk, setRisk] = useState('all');
    const [ruleType, setRuleType] = useState('all');
    const [page, setPage] = useState(1);
    const pageSize = 10;
    const [syncing, setSyncing] = useState(false);
    const [syncMessage, setSyncMessage] = useState('');
    const [helpRule, setHelpRule] = useState<rules.Rule | null>(null);
    const filtered = entries.filter(rule => {
        const matchesText = !query || rule.name.toLowerCase().includes(query.toLowerCase()) || rule.id.includes(query.toLowerCase());
        return matchesText && (risk === 'all' || rule.risk === risk) && (ruleType === 'all' || rule.rule_type === ruleType);
    });
    const pageCount = Math.max(1, Math.ceil(filtered.length / pageSize));
    const visibleRules = filtered.slice((page - 1) * pageSize, page * pageSize);
    useEffect(() => setPage(1), [query, risk, ruleType]);
    useEffect(() => setPage(current => Math.min(current, pageCount)), [pageCount]);
    const sync = async () => {
        setSyncing(true);
        setSyncMessage('');
        try {
            const result = await onSync();
            setSyncMessage(`已同步 ${result.statistics.total} 条规则`);
        } catch (error) {
            setSyncMessage(String(error));
        } finally {
            setSyncing(false);
        }
    };
    return (
        <div className="page-stack">
            <ToolbarPortal>
                <div className="rules-toolbar">
                <label className="search-field wide"><Search size={16}/><input value={query} onChange={event => setQuery(event.target.value)} placeholder="搜索规则名称或 ID"/></label>
                <select value={ruleType} onChange={event => setRuleType(event.target.value)} aria-label="规则类型筛选">
                    <option value="all">全部类型</option><option value="system">系统级规则</option><option value="third_party">三方软件</option><option value="general">通用规则</option>
                </select>
                <select value={risk} onChange={event => setRisk(event.target.value)} aria-label="风险筛选">
                    <option value="all">全部风险</option><option value="low">低风险</option><option value="medium">中风险</option><option value="high">高风险</option>
                </select>
                <span className="toolbar-count">{filtered.length} 条规则</span>
                <button className="icon-button" title="从 Velin Clear 仓库同步规则" aria-label="同步仓库规则" disabled={syncing} onClick={() => void sync()}><RotateCcw size={16} className={syncing ? 'spin' : ''}/></button>
                </div>
            </ToolbarPortal>
            <section className="rules-summary" aria-label="规则统计">
                <div><small>全部规则</small><strong>{stats?.total ?? entries.length}</strong></div>
                <div className="system"><small>系统级</small><strong>{stats?.system ?? 0}</strong></div>
                <div className="third-party"><small>三方软件</small><strong>{stats?.third_party ?? 0}</strong></div>
                <div><small>通用规则</small><strong>{stats?.general ?? 0}</strong></div>
                <div><small>可执行清理</small><strong>{stats?.executable ?? 0}</strong></div>
                <div><small>仅分析</small><strong>{stats?.analysis_only ?? 0}</strong></div>
            </section>
            {syncMessage && <p className="rules-sync-message">{syncMessage}</p>}
            <section className="rules-list">
                {visibleRules.map(rule => (
                    <div className="rule-row" key={rule.id}>
                        <span className={`rule-status ${rule.enabled ? 'enabled' : ''}`}/>
                        <div className="rule-main"><strong>{rule.name}</strong><small>{rule.id}</small><p>{rule.purpose}</p></div>
                        <div className="rule-class"><span className={`rule-kind ${rule.rule_type}`}>{ruleTypeLabels[rule.rule_type] ?? rule.rule_type}</span><span className="rule-category">{categoryLabels[rule.category] ?? rule.category}</span></div>
                        <span className={`risk-badge risk-${rule.risk}`}>{riskLabels[rule.risk] ?? rule.risk}</span>
                        <span className="default-state">{rule.default_selected ? '默认选中' : '默认不选'}</span>
                        <button className="icon-button" title="查看规则说明" aria-label="查看规则说明" onClick={() => setHelpRule(rule)}><CircleHelp size={17}/></button>
                    </div>
                ))}
            </section>
            {filtered.length > 0 && <footer className="rules-pagination">
                <span>显示 {(page - 1) * pageSize + 1}-{Math.min(page * pageSize, filtered.length)} / {filtered.length}</span>
                <div><button className="icon-button subtle" title="上一页" aria-label="上一页" disabled={page <= 1} onClick={() => setPage(current => current - 1)}><ChevronLeft size={16}/></button><strong>{page} / {pageCount}</strong><button className="icon-button subtle" title="下一页" aria-label="下一页" disabled={page >= pageCount} onClick={() => setPage(current => current + 1)}><ChevronRight size={16}/></button></div>
            </footer>}
            <RuleHelpModal source={helpRule} onClose={() => setHelpRule(null)}/>
        </div>
    );
}

function AIPage({scanID}: {scanID: string}) {
    const [name, setName] = useState('Cleaning Agent');
    const [baseURL, setBaseURL] = useState('');
    const [model, setModel] = useState('');
    const [apiKey, setAPIKey] = useState('');
    const [keySaved, setKeySaved] = useState(false);
    const [capabilityOK, setCapabilityOK] = useState(false);
    const [providerConfigured, setProviderConfigured] = useState(false);
    const [providerLoading, setProviderLoading] = useState(true);
    const [configOpen, setConfigOpen] = useState(false);
    const [status, setStatus] = useState<'idle' | 'testing' | 'ready' | 'error'>('idle');
    const [message, setMessage] = useState('');
    const [objective, setObjective] = useState('');
    const [scanMode, setScanMode] = useState<'quick' | 'standard' | 'deep'>('standard');
    const [agentResult, setAgentResult] = useState<agent.Result | null>(null);
    const [selectedIDs, setSelectedIDs] = useState<Set<string>>(new Set());
    const [agentBusy, setAgentBusy] = useState(false);
    const [agentError, setAgentError] = useState('');
    const [cleanPlan, setCleanPlan] = useState<cleaner.Plan | null>(null);
    const [cleanResult, setCleanResult] = useState<cleaner.Result | null>(null);
    const [cleaning, setCleaning] = useState(false);
    const [helpFinding, setHelpFinding] = useState<scanner.Item | null>(null);
    const [presets, setPresets] = useState<agent.Preset[]>([]);
    const [resultView, setResultView] = useState<'files' | 'folders'>('files');

    useEffect(() => {
        api.aiProvider().then(config => {
            setName(config.name || 'Cleaning Agent'); setBaseURL(config.base_url || '');
            setModel(config.model || ''); setKeySaved(config.key_saved); setCapabilityOK(config.capability_ok);
            setProviderConfigured(Boolean(config.base_url && config.model));
            if (config.capability_ok) setStatus('ready');
        }).catch(reason => { setStatus('error'); setMessage(String(reason)); })
            .finally(() => setProviderLoading(false));
    }, []);

    useEffect(() => { api.cleaningAgentPresets().then(setPresets).catch(() => undefined); }, []);

    async function testProvider() {
        setStatus('testing'); setMessage('');
        const input = {name, base_url: baseURL, model, api_key: apiKey, timeout_seconds: 60, max_output_tokens: 4096} as provider.ConfigInput;
        try {
            const result = await api.testAIProvider(input);
            setStatus(result.ok ? 'ready' : 'error'); setMessage(result.message);
            setCapabilityOK(result.capability_ok);
            if (result.ok) {
                setKeySaved(Boolean(apiKey) || keySaved);
                setAPIKey('');
                setProviderConfigured(true);
                setConfigOpen(false);
                setMessage('');
            }
        } catch (reason) {
            setStatus('error'); setMessage(String(reason));
        }
    }

    async function runAgent() {
        if (!objective.trim() || !capabilityOK) return;
        setAgentBusy(true); setAgentError(''); setAgentResult(null);
        try {
            const result = await api.runCleaningAgent({objective: objective.trim(), scan_id: scanID, mode: 'scan', scan_mode: scanMode} as agent.Request);
            setAgentResult(result);
            setSelectedIDs(new Set((result.items || []).filter(item => item.default_selected && item.selectable).map(item => item.item_id)));
            setResultView('files');
            setObjective('');
        }
        catch (reason) { setAgentError(String(reason)); }
        finally { setAgentBusy(false); }
    }

    async function buildAgentPlan() {
        if (!agentResult || selectedIDs.size === 0) return;
        setAgentError('');
        try { setCleanPlan(await api.buildCleanPlan({scan_id: agentResult.scan_id, item_ids: [...selectedIDs]} as cleaner.BuildRequest)); }
        catch (reason) { setAgentError(String(reason)); }
    }

    async function executeAgentPlan() {
        if (!cleanPlan) return;
        setCleaning(true); setAgentError('');
        try { setCleanResult(await api.executeCleanPlan({plan_id: cleanPlan.id, confirmation_token: cleanPlan.confirmation_token} as cleaner.ExecuteRequest)); }
        catch (reason) { setAgentError(String(reason)); }
        finally { setCleaning(false); }
    }

    const presetOptions: agent.Preset[] = presets.length > 0 ? presets : [
        {id: 'safe-space', name: '安全释放空间', objective: '优先分析低风险、可恢复的缓存和临时文件，列出可安全清理项。', description: '仅关注低风险可清理内容'},
        {id: 'large-files', name: '查找大文件', objective: '列出占用空间较大的文件，说明用途和清理影响，默认不勾选需要人工确认的项目。', description: '发现大文件，不自动建议删除'},
        {id: 'system-deep', name: '系统深度检查', objective: '分析 Windows 系统缓存、日志、更新残留、回收站和特殊系统文件，明确建议清理或手动处理。', description: '覆盖更多 Windows 专属规则'},
    ];
    const agentSelectableItems = (agentResult?.items || []).filter(item => item.selectable);
    const allAgentItemsSelected = agentSelectableItems.length > 0 && agentSelectableItems.every(item => selectedIDs.has(item.item_id));
    const aiItems = useMemo(() => (agentResult?.items || []).map(findingToScannerItem), [agentResult?.items]);
    const aiFolders = useMemo(() => buildAIFolders(aiItems), [aiItems]);

    function toggleAllAgentItems() {
        setSelectedIDs(allAgentItemsSelected ? new Set() : new Set(agentSelectableItems.map(item => item.item_id)));
    }

    function toggleAIFolder(folderItems: scanner.Item[], checked: boolean) {
        setSelectedIDs(previous => {
            const next = new Set(previous);
            for (const item of folderItems) {
                if (!item.selectable || item.action === 'analyze') continue;
                checked ? next.add(item.id) : next.delete(item.id);
            }
            return next;
        });
    }

    if (providerLoading) {
        return <div className="ai-unconfigured"><div className="ai-unconfigured-content"><span className="ai-empty-icon"><Bot size={25}/></span><h2>正在读取 AI 配置</h2></div></div>;
    }

    return (
        <div className={providerConfigured ? (agentResult ? 'ai-results-layout' : 'ai-start-layout') : 'ai-unconfigured'}>
            {providerConfigured && <ToolbarPortal><button className="button secondary" onClick={() => setConfigOpen(true)}><SettingsIcon size={16}/>模型配置</button></ToolbarPortal>}
            {!providerConfigured ? <div className="ai-unconfigured-content">
                <span className="ai-empty-icon"><Bot size={25}/></span>
                <h2>AI 清理尚未配置</h2>
                <p>配置 OpenAI-compatible 服务后才能使用 Cleaning Agent。</p>
                <button className="button primary" onClick={() => setConfigOpen(true)}><SettingsIcon size={16}/>立即配置</button>
            </div> : !agentResult ? <section className="ai-start-view">
                <div className="ai-start-content">
                    <header className="ai-start-heading">
                        <div className="ai-start-title"><span className="ai-start-icon"><Sparkles size={22}/></span><div><h2>选择清理目标</h2><p>扫描完成后将按用途、风险和清理影响整理文件。</p></div></div>
                        <div className="ai-mode-control"><span>扫描深度</span><div className="segmented compact"><button className={scanMode === 'quick' ? 'active' : ''} onClick={() => setScanMode('quick')}>快速</button><button className={scanMode === 'standard' ? 'active' : ''} onClick={() => setScanMode('standard')}>标准</button><button className={scanMode === 'deep' ? 'active' : ''} onClick={() => setScanMode('deep')}>深度</button></div></div>
                    </header>
                    <div className="ai-target-list">{presetOptions.map((preset, index) => {
                        const Icon = [ShieldCheck, FileSearch, Search][index] ?? Sparkles;
                        return <button key={preset.id} className={objective === preset.objective ? 'ai-target active' : 'ai-target'} onClick={() => setObjective(preset.objective)}><span className="ai-target-icon"><Icon size={17}/></span><span><strong>{preset.name}</strong><small>{preset.description}</small></span><span className="ai-target-check">{objective === preset.objective ? <CheckCircle2 size={17}/> : null}</span></button>;
                    })}</div>
                    {agentError && <div className="error-banner"><XCircle size={18}/>{agentError}</div>}
                    <footer className="ai-start-footer"><span className="ai-boundary"><ShieldCheck size={13}/>仅分析当前授权范围，清理前仍需确认</span><button className="button primary ai-start-button" disabled={!objective.trim() || !capabilityOK || agentBusy} onClick={() => void runAgent()}>{agentBusy ? <span className="button-spinner"/> : <Play size={16}/>}开始智能扫描</button></footer>
                </div>
            </section> : <section className="ai-results-view">
                {agentError && <div className="error-banner"><XCircle size={18}/>{agentError}</div>}
                <header className="ai-results-header">
                    <div className="ai-results-summary"><span className="ai-start-icon"><Bot size={18}/></span><div><h2>分析完成</h2><p>{agentResult.summary || '请审核下方文件后选择清理。'}</p></div></div>
                    <div className="ai-result-actions"><button className="button secondary" onClick={() => { setAgentResult(null); setObjective(''); setSelectedIDs(new Set()); }}>重新扫描</button><button className="button primary" disabled={selectedIDs.size === 0} onClick={() => void buildAgentPlan()}><Trash2 size={15}/>清理选中 {selectedIDs.size} 项</button></div>
                </header>
                <div className="ai-results-meta"><span>发现 {agentResult.items?.length || 0} 项</span><span>扫描编号 {agentResult.scan_id}</span></div>
                <div className="ai-results-toolbar">
                    <div className="segmented compact" aria-label="AI 结果视图">
                        <button className={resultView === 'files' ? 'active' : ''} onClick={() => setResultView('files')}><List size={15}/>文件</button>
                        <button className={resultView === 'folders' ? 'active' : ''} onClick={() => setResultView('folders')}><FolderOpen size={15}/>文件夹</button>
                    </div>
                    <span aria-hidden="true" />
                    <button className="button ghost" disabled={agentSelectableItems.length === 0} onClick={toggleAllAgentItems}><SquareCheckBig size={15}/>{allAgentItemsSelected ? '取消全选' : '全选'}</button>
                </div>
                {resultView === 'files' ? <div className="ai-findings-list">{(agentResult.items || []).map(item => <AIFindingRow key={item.item_id} item={item} checked={selectedIDs.has(item.item_id)} onToggle={() => setSelectedIDs(previous => { const next = new Set(previous); next.has(item.item_id) ? next.delete(item.item_id) : next.add(item.item_id); return next; })} onHelp={() => setHelpFinding(findingToScannerItem(item))}/>)}</div>
                    : <div className="ai-findings-list ai-folder-list"><FolderTree folders={aiFolders} items={aiItems} selected={selectedIDs} onToggleItem={item => setSelectedIDs(previous => { const next = new Set(previous); next.has(item.id) ? next.delete(item.id) : next.add(item.id); return next; })} onToggleFolder={toggleAIFolder} onHelp={setHelpFinding}/></div>}
            </section>}
            {configOpen && <ProviderConfigModal
                name={name} baseURL={baseURL} model={model} apiKey={apiKey} keySaved={keySaved} status={status} message={message}
                onNameChange={setName} onBaseURLChange={setBaseURL} onModelChange={setModel} onAPIKeyChange={setAPIKey}
                onClose={() => setConfigOpen(false)} onSave={testProvider}
            />}
            {cleanPlan && <CleanPlanModal plan={cleanPlan} result={cleanResult} busy={cleaning} onClose={() => { setCleanPlan(null); setCleanResult(null); }} onExecute={() => void executeAgentPlan()}/>}
            {helpFinding && <RuleHelpModal source={helpFinding} onClose={() => setHelpFinding(null)}/>}
        </div>
    );
}

function AIFindingRow({item, checked, onToggle, onHelp}: {item: agent.Finding; checked: boolean; onToggle: () => void; onHelp: () => void}) {
    const blocked = !item.selectable || item.suggested_action === 'manual' || item.recommendation === 'review' || item.recommendation === 'keep';
    return <div className={`ai-finding-row ${blocked ? 'blocked' : ''}`}>
        <input type="checkbox" checked={checked} disabled={!item.selectable} onChange={onToggle}/>
        <div className="ai-finding-main"><div className="ai-finding-title"><strong>{item.name}</strong><span className={`risk-badge risk-${item.risk}`}>{item.risk === 'low' ? '低风险' : item.risk === 'medium' ? '中风险' : '高风险'}</span><span className="ai-recommendation">{item.recommendation === 'recommended' ? '建议清理' : item.recommendation === 'optional' ? '可选清理' : item.recommendation === 'keep' ? '建议保留' : '需人工确认'}</span></div><code>{item.path}</code><p><b>用途：</b>{item.purpose || '扫描器未提供用途说明'} <b>影响：</b>{item.clean_effect || '请确认删除影响'}</p><small>{item.reason || item.recommendation_reason || item.help_summary} · {formatBytes(item.allocated_size)} · 置信度 {Math.round((item.confidence || 0) * 100)}%</small></div>
        <button className="icon-button subtle" title="查看规则说明" aria-label="查看规则说明" onClick={onHelp}><CircleHelp size={15}/></button>
    </div>;
}

function findingToScannerItem(item: agent.Finding): scanner.Item {
    return {
        id: item.item_id,
        rule_id: item.rule_id,
        matched_rule_ids: [item.rule_id],
        name: item.name,
        path: item.path,
        directory: item.directory,
        extension: item.extension,
        category: item.category,
        purpose: item.purpose,
        clean_effect: item.clean_effect,
        recommendation: item.recommendation,
        recommendation_reason: item.recommendation_reason,
        risk: item.risk,
        default_selected: item.default_selected,
        selectable: item.selectable,
        action: item.selectable ? item.action : 'analyze',
        recovery_type: item.selectable ? 'none' : 'system_settings',
        requires_admin: false,
        requires_restart: false,
        logical_size: item.logical_size,
        allocated_size: item.allocated_size,
        estimated_reclaimable: item.selectable ? item.allocated_size : 0,
        volume_id: '',
        file_id: '',
        link_count: 1,
        modified_at: item.modified_at,
        help_summary: item.help_summary,
        help_details: item.help_details,
        special_warning: item.special_warning,
        manual_steps: item.manual_steps,
    } as unknown as scanner.Item;
}

type AIFolderNode = {
    path: string;
    name: string;
    directItems: string[];
    children: Map<string, AIFolderNode>;
    parent?: string;
};

function buildAIFolders(items: scanner.Item[]): scanner.Folder[] {
    const nodes = new Map<string, AIFolderNode>();
    const roots: AIFolderNode[] = [];
    const ensure = (path: string): AIFolderNode => {
        const key = normaliseFolderPath(path);
        const existing = nodes.get(key);
        if (existing) return existing;
        const node: AIFolderNode = {path, name: folderName(path), directItems: [], children: new Map()};
        nodes.set(key, node);
        const parentPath = folderParent(path);
        if (parentPath && normaliseFolderPath(parentPath) !== key) {
            const parent = ensure(parentPath);
            node.parent = normaliseFolderPath(parent.path);
            parent.children.set(key, node);
        } else {
            roots.push(node);
        }
        return node;
    };

    for (const item of items) {
        const directory = item.directory || folderParent(item.path) || item.path;
        ensure(directory).directItems.push(item.id);
    }

    const byID = new Map(items.map(item => [item.id, item]));
    const toFolder = (node: AIFolderNode): scanner.Folder => {
        const children = [...node.children.values()].sort((a, b) => a.path.localeCompare(b.path)).map(toFolder);
        const itemIDs = [...node.directItems];
        for (const child of children) itemIDs.push(...child.item_ids);
        const matching = itemIDs.map(id => byID.get(id)).filter((item): item is scanner.Item => Boolean(item));
        return {
            id: `ai-folder-${normaliseFolderPath(node.path)}`,
            name: node.name,
            path: node.path,
            file_count: matching.length,
            logical_bytes: matching.reduce((sum, item) => sum + item.logical_size, 0),
            allocated_bytes: matching.reduce((sum, item) => sum + item.allocated_size, 0),
            estimated_reclaimable: matching.reduce((sum, item) => sum + item.estimated_reclaimable, 0),
            highest_risk: matching.reduce((highest, item) => riskRank(item.risk) > riskRank(highest) ? item.risk : highest, 'low'),
            item_ids: itemIDs,
            children,
        } as unknown as scanner.Folder;
    };
    return roots.sort((a, b) => a.path.localeCompare(b.path)).map(toFolder);
}

function normaliseFolderPath(path: string) {
    const trimmed = path.replace(/[\\/]+$/, '');
    return (trimmed || path || '/').toLowerCase();
}

function folderParent(path: string) {
    if (!path) return '';
    const trimmed = path.replace(/[\\/]+$/, '');
    if (/^[a-zA-Z]:$/.test(trimmed) || trimmed === '/') return '';
    const index = Math.max(trimmed.lastIndexOf('\\'), trimmed.lastIndexOf('/'));
    if (index < 0) return '';
    if (index === 0) return trimmed.startsWith('/') ? '/' : '';
    if (index === 2 && trimmed[1] === ':') return trimmed.slice(0, 3);
    return trimmed.slice(0, index) || '/';
}

function folderName(path: string) {
    const trimmed = path.replace(/[\\/]+$/, '');
    if (!trimmed || trimmed === '/' || /^[a-zA-Z]:$/.test(trimmed)) return path || '/';
    return trimmed.slice(Math.max(trimmed.lastIndexOf('\\'), trimmed.lastIndexOf('/')) + 1) || trimmed;
}

function riskRank(value: string) {
    return value === 'forbidden' ? 4 : value === 'high' ? 3 : value === 'medium' ? 2 : 1;
}

function ProviderConfigModal({name, baseURL, model, apiKey, keySaved, status, message, onNameChange, onBaseURLChange, onModelChange, onAPIKeyChange, onClose, onSave}: {
    name: string; baseURL: string; model: string; apiKey: string; keySaved: boolean;
    status: 'idle' | 'testing' | 'ready' | 'error'; message: string;
    onNameChange: (value: string) => void; onBaseURLChange: (value: string) => void; onModelChange: (value: string) => void; onAPIKeyChange: (value: string) => void;
    onClose: () => void; onSave: () => void;
}) {
    const testing = status === 'testing';
    return <div className="modal-backdrop" role="presentation" onMouseDown={testing ? undefined : onClose}>
        <section className="modal provider-config-modal" role="dialog" aria-modal="true" aria-labelledby="provider-config-title" onMouseDown={event => event.stopPropagation()}>
            <header className="modal-header"><div className="modal-title"><span className="icon-box"><Bot size={18}/></span><div><h2 id="provider-config-title">模型配置</h2><p>OpenAI-compatible Provider</p></div></div><button className="icon-button" aria-label="关闭" disabled={testing} onClick={onClose}><XCircle size={18}/></button></header>
            <div className="provider-config-form">
                <label className="field"><span>配置名称</span><input value={name} onChange={event => onNameChange(event.target.value)} placeholder="例如：本地模型"/></label>
                <label className="field"><span>Base URL</span><input value={baseURL} onChange={event => onBaseURLChange(event.target.value)} placeholder="https://api.example.com/v1"/></label>
                <label className="field"><span>模型</span><input value={model} onChange={event => onModelChange(event.target.value)} placeholder="model-id"/></label>
                <label className="field"><span>API Key {keySaved && '（已安全保存）'}</span><input value={apiKey} onChange={event => onAPIKeyChange(event.target.value)} placeholder={keySaved ? '留空以继续使用现有密钥' : '本地无鉴权服务可留空'} type="password" autoComplete="new-password"/></label>
                <div className="provider-note"><ShieldCheck size={16}/><span>仅发送文件元数据（路径、用途、大小和规则说明），不发送文件内容；密钥不会出现在日志和会话中。</span></div>
                {message && <p className={`provider-message ${status}`}>{message}</p>}
            </div>
            <footer className="modal-footer"><button className="button secondary" disabled={testing} onClick={onClose}>取消</button><button className="button primary" disabled={testing || !baseURL.trim() || !model.trim()} onClick={onSave}>{testing ? '正在测试…' : '保存并测试'}</button></footer>
        </section>
    </div>;
}

function HistoryPage() {
    const [items, setItems] = useState<cleaner.Result[]>([]);
    const [error, setError] = useState('');
    const [loading, setLoading] = useState(true);
    useEffect(() => { api.cleanHistory().then(value => setItems(value ?? [])).catch(reason => setError(String(reason))).finally(() => setLoading(false)); }, []);
    if (!loading && items.length === 0) return <EmptyState icon={History} title="暂无清理记录" body="完成首次清理后，这里会显示实际释放空间和执行结果。"/>;
    return <div className="page-stack">
        <ToolbarPortal><span className="toolbar-count">{items.length} 次记录</span></ToolbarPortal>
        {error && <div className="error-banner"><XCircle size={18}/>{error}</div>}
        <section className="data-list">
            {items.map(item => <div className="history-row" key={item.id}><span className="metric-icon teal"><CheckCircle2 size={17}/></span><div><strong>{formatDate(item.completed_at)} 清理</strong><small>{item.id}</small></div><span>成功 {item.succeeded} / 跳过 {item.skipped} / 失败 {item.failed}</span><div><small>永久删除</small><strong>{formatBytes(item.deleted_bytes)}</strong></div><div><small>实际释放</small><strong>{formatBytes(item.actually_reclaimed)}</strong></div></div>)}
        </section>
    </div>;
}

function SettingsPage({themeMode, onThemeChange}: {themeMode: ThemeMode; onThemeChange: (mode: ThemeMode) => void}) {
    return (
        <div className="settings-layout">
            <section className="settings-group">
                <div><h2>外观</h2><p>选择界面主题</p></div>
                <div className="theme-options">
                    {([{id: 'system', label: '跟随系统', icon: HardDrive}, {id: 'light', label: '亮色', icon: Sun}, {id: 'dark', label: '暗色', icon: Moon}] as const).map(({id, label, icon: Icon}) => (
                        <button className={themeMode === id ? 'theme-option active' : 'theme-option'} key={id} onClick={() => onThemeChange(id)}><Icon size={18}/><span>{label}</span>{themeMode === id && <CheckCircle2 size={16}/>}</button>
                    ))}
                </div>
            </section>
            <section className="settings-group">
                <div><h2>扫描</h2><p>控制磁盘负载和分析范围</p></div>
                <div className="setting-row"><div><strong>增量扫描</strong><small>复用有效的 NTFS 扫描索引</small></div><input type="checkbox" defaultChecked aria-label="启用增量扫描"/></div>
                <div className="setting-row"><div><strong>大文件阈值</strong><small>大于该值的文件进入分析列表</small></div><select defaultValue="1"><option value=".5">500 MB</option><option value="1">1 GB</option><option value="2">2 GB</option><option value="5">5 GB</option></select></div>
            </section>
            <section className="settings-group">
                <div><h2>安全</h2><p>永久删除保护策略</p></div>
                <div className="setting-row"><div><strong>强制清理确认</strong><small>永久删除前展示风险、准确路径和预计空间</small></div><input type="checkbox" defaultChecked disabled aria-label="强制清理确认"/></div>
                <div className="setting-row"><div><strong>文件状态复核</strong><small>执行前校验路径、文件身份、大小和修改时间</small></div><input type="checkbox" defaultChecked disabled aria-label="文件状态复核"/></div>
            </section>
        </div>
    );
}

function EmptyState({icon: Icon, title, body}: {icon: typeof History; title: string; body: string}) {
    return <section className="empty-state"><span><Icon size={24}/></span><h2>{title}</h2><p>{body}</p></section>;
}

function ToolbarPortal({children}: {children: ReactNode}) {
    const [target, setTarget] = useState<HTMLElement | null>(null);
    useEffect(() => setTarget(document.getElementById('page-toolbar-actions')), []);
    return target ? createPortal(children, target) : null;
}

function modeLabel(mode: string) {
    return ({quick: '快速', standard: '标准', deep: '深度'} as Record<string, string>)[mode] ?? mode;
}
