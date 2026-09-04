import {AlertTriangle, Bot, CircleHelp, ExternalLink, FileText, FolderOpen, ShieldAlert, X} from 'lucide-react';
import type {agent, rules, scanner} from '../../wailsjs/go/models';
import {api} from '../lib/api';
import {formatBytes, recommendationLabels, riskLabels} from '../lib/format';

type HelpSource = rules.Rule | scanner.Item | agent.Finding;

interface Props {
    source: HelpSource | null;
    onClose: () => void;
}

function isItem(source: HelpSource): source is scanner.Item | agent.Finding {
    return 'rule_id' in source;
}

function isAIFinding(source: HelpSource): source is agent.Finding {
    return 'suggested_action' in source && 'confidence' in source;
}

function directDeleteAdvice(source: scanner.Item | agent.Finding): string {
    if (!source.selectable || source.action === 'analyze') return '不可以。该项仅用于分析或需要通过系统、软件自身的管理功能处理。';
    if (source.risk !== 'low') return '不能直接批量删除。请先核对路径、文件用途和备份，再手动勾选并在确认页执行。';
    return '可以在确认用途且关闭占用程序后，手动勾选并在最终确认页永久删除。';
}

function spaceSavingAdvice(source: HelpSource): string {
    const steps = isItem(source) ? source.manual_steps : source.help.manual_steps;
    if (steps && steps.length > 0) return steps.join('；');
    switch (source.category) {
        case 'browser_cache': return '关闭浏览器后，优先使用浏览器自身的“清除缓存”功能。';
        case 'app_cache': return '关闭对应软件，优先在软件设置中清理缓存、下载内容或离线文件。';
        case 'dev_cache': return '在构建任务结束后，通过对应开发工具的缓存清理功能释放空间。';
        case 'large_files': return '确认已备份或不再需要后，移动到其他磁盘、压缩归档，或仅删除明确无用的副本。';
        case 'system_storage': return '使用 Windows 存储设置、磁盘清理或对应系统功能处理，不要手动删除系统文件。';
        default: return '检查文件所属软件的存储管理功能；对于个人文件，优先移动、备份或归档，而不是直接删除。';
    }
}

export function RuleHelpModal({source, onClose}: Props) {
    if (!source) return null;
    const item = isItem(source);
    const name = item ? source.name : source.name;
    const ruleID = item ? source.rule_id : source.id;
    const help = item ? {
        summary: source.help_summary,
        details: source.help_details,
        warning: source.special_warning,
        steps: source.manual_steps,
    } : {
        summary: source.help.summary,
        details: source.help.details,
        warning: source.help.special_warning,
        steps: source.help.manual_steps,
    };
    const risk = source.risk;
    const recommendation = source.recommendation;
    const defaultSelected = source.default_selected;
    const purpose = source.purpose;
    const effect = source.clean_effect;
    const isPagefile = ruleID === 'windows.pagefile_analysis';
    const itemType = item ? source.is_directory ? '目录' : '文件' : '规则';
    const aiFinding = isAIFinding(source) ? source : null;
    const explanation = aiFinding?.reason || `${name} 是一个${itemType}，${purpose}。${effect}。`;
    const deleteAdvice = item ? directDeleteAdvice(source) : '规则页仅用于说明；请先完成扫描，再根据扫描结果决定是否处理。';
    const deleteRecommendation = `${recommendationLabels[recommendation] ?? recommendation}${source.recommendation_reason ? `：${source.recommendation_reason}` : ''}`;

    return (
        <div className="modal-backdrop" role="presentation" onMouseDown={onClose}>
            <section className="modal help-modal" role="dialog" aria-modal="true" aria-labelledby="help-title" onMouseDown={event => event.stopPropagation()}>
                <header className="modal-header">
                    <div className="modal-title">
                        <span className="icon-box"><CircleHelp size={18}/></span>
                        <div>
                            <h2 id="help-title">{name}</h2>
                            <p>{ruleID}</p>
                        </div>
                    </div>
                    <button className="icon-button" aria-label="关闭" title="关闭" onClick={onClose}><X size={18}/></button>
                </header>

                <div className="help-summary">{help.summary}</div>
                {item && <section className="ai-explanation">
                    <header><Bot size={17}/><strong>AI 释义</strong>{aiFinding && <small>置信度 {Math.round((aiFinding.confidence || 0) * 100)}%</small>}</header>
                    <p>{explanation}</p>
                </section>}
                {help.warning && (
                    <div className="warning-callout">
                        <ShieldAlert size={18}/>
                        <div><strong>特别提醒</strong><p>{help.warning}</p></div>
                    </div>
                )}

                <dl className="detail-list">
                    {item && <div><dt>项目类型</dt><dd className="item-kind">{source.is_directory ? <FolderOpen size={14}/> : <FileText size={14}/>} {itemType}</dd></div>}
                    <div><dt>当前项用途</dt><dd>{purpose}</dd></div>
                    <div><dt>清理后影响</dt><dd>{effect}</dd></div>
                    <div><dt>是否建议删除</dt><dd>{deleteRecommendation}</dd></div>
                    {item && <div><dt>是否可直接删除</dt><dd>{deleteAdvice}</dd></div>}
                    <div><dt>风险等级</dt><dd><span className={`risk-badge risk-${risk}`}>{riskLabels[risk] ?? risk}</span></dd></div>
                    <div><dt>默认选中</dt><dd>{defaultSelected ? '是' : '否'}</dd></div>
                    {item && <div><dt>磁盘占用</dt><dd>{formatBytes(source.allocated_size)}</dd></div>}
                    {item && <div><dt>文件位置</dt><dd className="path-value">{source.path}</dd></div>}
                </dl>

                {help.details && <div className="help-details"><h3>详细说明</h3><p>{help.details}</p></div>}
                <div className="manual-steps"><h3>{item && (!source.selectable || source.action === 'analyze' || source.risk !== 'low') ? '不能直接删除时，如何减少占用' : '建议处理方式'}</h3><p>{spaceSavingAdvice(source)}</p></div>

                <footer className="modal-footer">
                    {isPagefile && (
                        <button className="button secondary" onClick={() => api.openSystemSettings('virtual_memory')}>
                            <ExternalLink size={16}/>打开虚拟内存设置
                        </button>
                    )}
                    {risk === 'high' && !isPagefile && <span className="footer-note"><AlertTriangle size={15}/>需要手动确认</span>}
                    <button className="button primary" onClick={onClose}>知道了</button>
                </footer>
            </section>
        </div>
    );
}
